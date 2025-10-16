package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/eterrain/tf-backend-service/internal/auth"
	"github.com/eterrain/tf-backend-service/internal/storage"
	"github.com/eterrain/tf-backend-service/internal/validation"
	"github.com/go-chi/chi/v5"
)

// StateHandler handles Terraform state operations
type StateHandler struct {
	storage storage.Storage
}

// NewStateHandler creates a new state handler
func NewStateHandler(storage storage.Storage) *StateHandler {
	return &StateHandler{
		storage: storage,
	}
}

// GetState handles GET requests for state retrieval
func (h *StateHandler) GetState(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	stateName := chi.URLParam(r, "name")
	if err := validation.ValidateStateName(stateName); err != nil {
		http.Error(w, "Invalid state name", http.StatusBadRequest)
		log.Printf("SECURITY: Invalid state name from org %s: %v", orgID, err)
		return
	}

	state, err := h.storage.GetState(orgID, stateName)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			http.Error(w, "State not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to retrieve state: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(state.Data)
}

// PutState handles POST/PUT requests for state updates
func (h *StateHandler) PutState(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	stateName := chi.URLParam(r, "name")
	if err := validation.ValidateStateName(stateName); err != nil {
		http.Error(w, "Invalid state name", http.StatusBadRequest)
		log.Printf("SECURITY: Invalid state name from org %s: %v", orgID, err)
		return
	}

	// Read state data from request body
	data, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read request body: %v", err), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Validate that the data is valid JSON
	if !json.Valid(data) {
		http.Error(w, "Invalid JSON state data", http.StatusBadRequest)
		return
	}

	// Store the state
	if err := h.storage.PutState(orgID, stateName, data); err != nil {
		http.Error(w, fmt.Sprintf("Failed to store state: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// DeleteState handles DELETE requests for state removal
func (h *StateHandler) DeleteState(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	stateName := chi.URLParam(r, "name")
	if err := validation.ValidateStateName(stateName); err != nil {
		http.Error(w, "Invalid state name", http.StatusBadRequest)
		log.Printf("SECURITY: Invalid state name from org %s: %v", orgID, err)
		return
	}

	err := h.storage.DeleteState(orgID, stateName)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			http.Error(w, "State not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, storage.ErrAlreadyLocked) {
			http.Error(w, "State is locked", http.StatusLocked)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to delete state: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// LockState handles LOCK requests for state locking
func (h *StateHandler) LockState(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	stateName := chi.URLParam(r, "name")
	if err := validation.ValidateStateName(stateName); err != nil {
		http.Error(w, "Invalid state name", http.StatusBadRequest)
		log.Printf("SECURITY: Invalid state name from org %s: %v", orgID, err)
		return
	}

	// Read lock info from request body
	var lockInfo storage.LockInfo
	if err := json.NewDecoder(r.Body).Decode(&lockInfo); err != nil {
		http.Error(w, fmt.Sprintf("Failed to decode lock info: %v", err), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Lock the state
	err := h.storage.LockState(orgID, stateName, &lockInfo)
	if err != nil {
		if errors.Is(err, storage.ErrAlreadyLocked) {
			// Return current lock info
			currentLock, _ := h.storage.GetLock(orgID, stateName)
			if currentLock != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusLocked)
				json.NewEncoder(w).Encode(currentLock)
				return
			}
			http.Error(w, "State is already locked", http.StatusLocked)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to lock state: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// UnlockState handles UNLOCK requests for state unlocking
func (h *StateHandler) UnlockState(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	stateName := chi.URLParam(r, "name")
	if err := validation.ValidateStateName(stateName); err != nil {
		http.Error(w, "Invalid state name", http.StatusBadRequest)
		log.Printf("SECURITY: Invalid state name from org %s: %v", orgID, err)
		return
	}

	// Read lock info from request body to get lock ID
	var lockInfo storage.LockInfo
	if err := json.NewDecoder(r.Body).Decode(&lockInfo); err != nil {
		http.Error(w, fmt.Sprintf("Failed to decode lock info: %v", err), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Unlock the state
	err := h.storage.UnlockState(orgID, stateName, lockInfo.ID)
	if err != nil {
		if errors.Is(err, storage.ErrNotLocked) {
			http.Error(w, "State is not locked", http.StatusConflict)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to unlock state: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
