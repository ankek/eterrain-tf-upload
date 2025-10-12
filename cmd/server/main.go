package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/eterrain/tf-backend-service/internal/auth"
	"github.com/eterrain/tf-backend-service/internal/config"
	"github.com/eterrain/tf-backend-service/internal/handlers"
	"github.com/eterrain/tf-backend-service/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

const version = "1.0.0"

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Starting Terraform Backend Service v%s", version)
	log.Printf("Server will listen on %s", cfg.Address())

	// Initialize storage
	var store storage.Storage
	var csvStore *storage.CSVStorage
	switch cfg.StorageType {
	case "memory":
		store = storage.NewMemoryStorage()
		log.Println("Using in-memory storage")
	case "csv":
		var err error
		csvStore, err = storage.NewCSVStorage(cfg.StoragePath)
		if err != nil {
			log.Fatalf("Failed to initialize CSV storage: %v", err)
		}
		log.Printf("Using CSV storage at: %s", cfg.StoragePath)
	default:
		log.Fatalf("Unsupported storage type: %s", cfg.StorageType)
	}

	// Initialize credential store from auth.cfg file
	credStore, err := auth.NewFileStore("./auth.cfg")
	if err != nil {
		log.Fatalf("Failed to load authentication config: %v", err)
	}
	log.Println("Authentication credentials loaded from ./auth.cfg")

	// Initialize handlers
	var stateHandler *handlers.StateHandler
	var uploadHandler *handlers.UploadHandler

	if store != nil {
		stateHandler = handlers.NewStateHandler(store)
	}
	if csvStore != nil {
		uploadHandler = handlers.NewUploadHandler(csvStore)
	}
	healthHandler := handlers.NewHealthHandler(version)

	// Setup router
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// Health check endpoint (no auth required)
	r.Get("/health", healthHandler.Check)

	// Protected routes with authentication
	r.Route("/api/v1", func(r chi.Router) {
		// Apply authentication middleware
		r.Use(auth.Middleware(credStore))

		// Data upload endpoints (for Terraform provider)
		if uploadHandler != nil {
			r.Post("/upload", uploadHandler.UploadData)
			r.Get("/data", uploadHandler.GetOrgData)
		}

		// State management endpoints (if using memory storage)
		if stateHandler != nil {
			// Terraform backend API endpoints
			r.Route("/state/{name}", func(r chi.Router) {
				r.Get("/", stateHandler.GetState)
				r.Post("/", stateHandler.PutState)
				r.Delete("/", stateHandler.DeleteState)
			})

			// Lock endpoints
			r.Post("/state/{name}/lock", stateHandler.LockState)
			r.Delete("/state/{name}/lock", stateHandler.UnlockState)
		}
	})

	// Create HTTP server
	srv := &http.Server{
		Addr:         cfg.Address(),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Server starting on %s", cfg.Address())

		if cfg.EnableTLS {
			log.Printf("TLS enabled with cert=%s key=%s", cfg.CertFile, cfg.KeyFile)
			if err := srv.ListenAndServeTLS(cfg.CertFile, cfg.KeyFile); err != nil && err != http.ErrServerClosed {
				log.Fatalf("Failed to start HTTPS server: %v", err)
			}
		} else {
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("Failed to start HTTP server: %v", err)
			}
		}
	}()

	log.Println("Server started successfully")
	log.Println("Press Ctrl+C to stop")

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}
