package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/eterrain/tf-backend-service/internal/auth"
	"github.com/eterrain/tf-backend-service/internal/storage"
)

// UploadHandler handles data upload operations from Terraform provider
type UploadHandler struct {
	csvStorage *storage.CSVStorage
}

// NewUploadHandler creates a new upload handler
func NewUploadHandler(csvStorage *storage.CSVStorage) *UploadHandler {
	return &UploadHandler{
		csvStorage: csvStorage,
	}
}

// ResourceUpload represents the hierarchical structure for resource uploads
type ResourceUpload struct {
	Provider     string                 `json:"provider"`
	Category     string                 `json:"category"`
	ResourceType string                 `json:"resource_type"`
	ResourceName string                 `json:"resource_name"`
	Properties   map[string]interface{} `json:"properties"`
}

// UploadData handles POST requests for data uploads from Terraform provider
func (h *UploadHandler) UploadData(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse JSON data from request body
	var upload ResourceUpload
	if err := json.NewDecoder(r.Body).Decode(&upload); err != nil {
		http.Error(w, fmt.Sprintf("Failed to decode request body: %v", err), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Validate required fields
	if upload.Provider == "" || upload.Category == "" || upload.ResourceType == "" || upload.ResourceName == "" {
		http.Error(w, "Missing required fields: provider, category, resource_type, and resource_name are required", http.StatusBadRequest)
		return
	}

	// Convert to flat map for CSV storage
	data := map[string]interface{}{
		"provider":      upload.Provider,
		"category":      upload.Category,
		"resource_type": upload.ResourceType,
		"resource_name": upload.ResourceName,
	}

	// Add properties to the flat map
	for k, v := range upload.Properties {
		data[k] = v
	}

	// Append data to CSV file
	if err := h.csvStorage.AppendData(orgID, data); err != nil {
		http.Error(w, fmt.Sprintf("Failed to store data: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success response
	response := map[string]interface{}{
		"status":  "success",
		"message": "Data uploaded successfully",
		"org_id":  orgID.String(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetOrgData handles GET requests to retrieve all data for an organization
func (h *UploadHandler) GetOrgData(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Retrieve data from CSV file
	uploads, err := h.csvStorage.GetOrgData(orgID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to retrieve data: %v", err), http.StatusInternalServerError)
		return
	}

	// Return data as JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"org_id": orgID.String(),
		"count":  len(uploads),
		"data":   uploads,
	})
}
