package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type contextKey string

const (
	OrgIDContextKey contextKey = "orgid"
)

// Credentials represents the authentication credentials
type Credentials struct {
	OrgID  uuid.UUID
	APIKey string
}

// CredentialStore defines the interface for validating credentials
type CredentialStore interface {
	ValidateCredentials(orgID uuid.UUID, apiKey string) (bool, error)
}

// Middleware creates an authentication middleware that validates orgid and apikey
func Middleware(store CredentialStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract orgid from header
			orgIDStr := r.Header.Get("X-Org-ID")
			if orgIDStr == "" {
				http.Error(w, "Missing X-Org-ID header", http.StatusUnauthorized)
				return
			}

			// Parse orgid as UUID
			orgID, err := uuid.Parse(orgIDStr)
			if err != nil {
				http.Error(w, "Invalid X-Org-ID format: must be a valid UUID", http.StatusUnauthorized)
				return
			}

			// Extract apikey from header
			apiKey := r.Header.Get("X-API-Key")
			if apiKey == "" {
				http.Error(w, "Missing X-API-Key header", http.StatusUnauthorized)
				return
			}

			// Validate credentials
			valid, err := store.ValidateCredentials(orgID, apiKey)
			if err != nil {
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			if !valid {
				http.Error(w, "Invalid credentials", http.StatusUnauthorized)
				return
			}

			// Store orgID in context for use by handlers
			ctx := context.WithValue(r.Context(), OrgIDContextKey, orgID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetOrgIDFromContext retrieves the orgID from the request context
func GetOrgIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	orgID, ok := ctx.Value(OrgIDContextKey).(uuid.UUID)
	return orgID, ok
}

// ExtractBearerToken extracts a bearer token from the Authorization header
func ExtractBearerToken(r *http.Request) string {
	bearerToken := r.Header.Get("Authorization")
	if len(bearerToken) > 7 && strings.ToUpper(bearerToken[0:7]) == "BEARER " {
		return bearerToken[7:]
	}
	return ""
}
