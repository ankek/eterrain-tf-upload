package middleware

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// TokenBucket implements a token bucket rate limiter
type TokenBucket struct {
	tokens         float64
	maxTokens      float64
	refillRate     float64 // tokens per second
	lastRefillTime time.Time
	mu             sync.Mutex
}

// NewTokenBucket creates a new token bucket
func NewTokenBucket(maxTokens, refillRate float64) *TokenBucket {
	return &TokenBucket{
		tokens:         maxTokens,
		maxTokens:      maxTokens,
		refillRate:     refillRate,
		lastRefillTime: time.Now(),
	}
}

// Allow checks if a request is allowed and consumes a token if so
func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefillTime).Seconds()

	// Refill tokens based on elapsed time
	tb.tokens += elapsed * tb.refillRate
	if tb.tokens > tb.maxTokens {
		tb.tokens = tb.maxTokens
	}
	tb.lastRefillTime = now

	// Check if we have tokens available
	if tb.tokens >= 1.0 {
		tb.tokens -= 1.0
		return true
	}

	return false
}

// PerOrgRateLimiter implements per-organization rate limiting
type PerOrgRateLimiter struct {
	buckets        map[uuid.UUID]*TokenBucket
	mu             sync.RWMutex
	maxTokens      float64
	refillRate     float64
	cleanupTicker  *time.Ticker
	stopCleanup    chan struct{}
	maxIdleTime    time.Duration
}

// NewPerOrgRateLimiter creates a new per-organization rate limiter
// maxRequestsPerMinute: maximum requests allowed per organization per minute
func NewPerOrgRateLimiter(maxRequestsPerMinute float64) *PerOrgRateLimiter {
	refillRate := maxRequestsPerMinute / 60.0 // convert to per-second rate

	limiter := &PerOrgRateLimiter{
		buckets:     make(map[uuid.UUID]*TokenBucket),
		maxTokens:   maxRequestsPerMinute,
		refillRate:  refillRate,
		stopCleanup: make(chan struct{}),
		maxIdleTime: 10 * time.Minute,
	}

	// Start cleanup goroutine to remove idle buckets
	limiter.cleanupTicker = time.NewTicker(5 * time.Minute)
	go limiter.cleanupRoutine()

	return limiter
}

// cleanupRoutine removes idle rate limit buckets to prevent memory leaks
func (rl *PerOrgRateLimiter) cleanupRoutine() {
	for {
		select {
		case <-rl.cleanupTicker.C:
			rl.mu.Lock()
			now := time.Now()
			for orgID, bucket := range rl.buckets {
				// Remove buckets that haven't been used recently
				if now.Sub(bucket.lastRefillTime) > rl.maxIdleTime {
					delete(rl.buckets, orgID)
				}
			}
			rl.mu.Unlock()
		case <-rl.stopCleanup:
			return
		}
	}
}

// Stop stops the cleanup goroutine
func (rl *PerOrgRateLimiter) Stop() {
	rl.cleanupTicker.Stop()
	close(rl.stopCleanup)
}

// getBucket gets or creates a token bucket for an organization
func (rl *PerOrgRateLimiter) getBucket(orgID uuid.UUID) *TokenBucket {
	rl.mu.RLock()
	bucket, exists := rl.buckets[orgID]
	rl.mu.RUnlock()

	if exists {
		return bucket
	}

	// Create new bucket
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Double-check after acquiring write lock
	bucket, exists = rl.buckets[orgID]
	if exists {
		return bucket
	}

	bucket = NewTokenBucket(rl.maxTokens, rl.refillRate)
	rl.buckets[orgID] = bucket
	return bucket
}

// Allow checks if a request from the given organization is allowed
func (rl *PerOrgRateLimiter) Allow(orgID uuid.UUID) bool {
	bucket := rl.getBucket(orgID)
	return bucket.Allow()
}

// OrgIDContextKey is the context key for storing org ID
type contextKey string

const OrgIDContextKey contextKey = "orgid"

// RateLimitMiddleware creates a middleware that applies per-organization rate limiting
func RateLimitMiddleware(limiter *PerOrgRateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract org ID from context (set by auth middleware)
			orgIDValue := r.Context().Value(OrgIDContextKey)
			if orgIDValue == nil {
				// No org ID in context, skip rate limiting (shouldn't happen with auth)
				next.ServeHTTP(w, r)
				return
			}

			orgID, ok := orgIDValue.(uuid.UUID)
			if !ok {
				// Invalid org ID type, skip rate limiting
				next.ServeHTTP(w, r)
				return
			}

			// Check rate limit
			if !limiter.Allow(orgID) {
				log.Printf("SECURITY: Rate limit exceeded for org %s, IP: %s", orgID, r.RemoteAddr)
				w.Header().Set("X-RateLimit-Limit", "60")
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.Header().Set("Retry-After", "60")
				http.Error(w, "Rate limit exceeded. Please try again later.", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
