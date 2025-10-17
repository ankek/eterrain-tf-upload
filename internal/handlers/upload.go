package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/eterrain/tf-backend-service/internal/auth"
	"github.com/eterrain/tf-backend-service/internal/storage"
	"github.com/eterrain/tf-backend-service/internal/validation"
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

// InstanceUpload represents a single instance within a resource
type InstanceUpload struct {
	Attributes map[string]interface{} `json:"attributes"`
}

// ResourceUpload represents the hierarchical structure for resource uploads
type ResourceUpload struct {
	Provider     string           `json:"provider"`
	Category     string           `json:"category"`
	ResourceType string           `json:"resource_type"`
	Name         string           `json:"name,omitempty"` // Optional name for the report/upload
	Instances    []InstanceUpload `json:"instances"`
}

// UploadData handles POST requests for data uploads from Terraform provider
func (h *UploadHandler) UploadData(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Read request body with size limit (already limited by middleware, but double-check)
	bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, 10<<20)) // 10MB limit
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Validate JSON size and format
	if err := validation.ValidateJSONString(bodyBytes, 10<<20); err != nil {
		log.Printf("SECURITY: Invalid JSON data from org %s - IP: %s, Error: %v", orgID, r.RemoteAddr, err)
		http.Error(w, "Invalid JSON data", http.StatusBadRequest)
		return
	}

	// Parse JSON data from request body
	var upload ResourceUpload
	if err := json.Unmarshal(bodyBytes, &upload); err != nil {
		http.Error(w, "Failed to decode request body", http.StatusBadRequest)
		return
	}

	// Validate JSON depth (max 10 levels deep)
	if err := validation.ValidateJSONDepth(upload, 10); err != nil {
		log.Printf("SECURITY: JSON depth violation from org %s - IP: %s, Error: %v", orgID, r.RemoteAddr, err)
		http.Error(w, "JSON structure too deeply nested", http.StatusBadRequest)
		return
	}

	// Validate JSON complexity (max 1000 total elements)
	if err := validation.ValidateJSONComplexity(upload, 1000); err != nil {
		log.Printf("SECURITY: JSON complexity violation from org %s - IP: %s, Error: %v", orgID, r.RemoteAddr, err)
		http.Error(w, "JSON structure too complex", http.StatusBadRequest)
		return
	}

	// Validate required fields with specific validators
	if err := validation.ValidateProvider(upload.Provider); err != nil {
		http.Error(w, fmt.Sprintf("Invalid provider: %v", err), http.StatusBadRequest)
		return
	}

	if err := validation.ValidateCategory(upload.Category); err != nil {
		http.Error(w, fmt.Sprintf("Invalid category: %v", err), http.StatusBadRequest)
		return
	}

	if err := validation.ValidateResourceType(upload.ResourceType); err != nil {
		http.Error(w, fmt.Sprintf("Invalid resource_type: %v", err), http.StatusBadRequest)
		return
	}

	// Validate instances array
	if len(upload.Instances) == 0 {
		http.Error(w, "At least one instance is required in the instances array", http.StatusBadRequest)
		return
	}

	// Limit number of instances to prevent resource exhaustion
	if len(upload.Instances) > 100 {
		http.Error(w, "Too many instances: maximum 100 instances per request", http.StatusBadRequest)
		return
	}

	// Process each instance and store separately
	for idx, instance := range upload.Instances {
		// Limit number of attributes per instance
		if len(instance.Attributes) > 100 {
			http.Error(w, fmt.Sprintf("Instance %d has too many attributes: maximum 100 attributes per instance", idx), http.StatusBadRequest)
			return
		}

		// Validate all attributes before processing
		for k, v := range instance.Attributes {
			if err := validation.ValidateAttributeKey(k); err != nil {
				http.Error(w, fmt.Sprintf("Invalid attribute key '%s' in instance %d: %v", k, idx, err), http.StatusBadRequest)
				return
			}
			if err := validation.ValidateAttributeValue(v); err != nil {
				http.Error(w, fmt.Sprintf("Invalid attribute value for '%s' in instance %d: %v", k, idx, err), http.StatusBadRequest)
				return
			}
		}

		// Convert to flat map for CSV storage
		data := map[string]interface{}{
			"provider":      upload.Provider,
			"category":      upload.Category,
			"resource_type": upload.ResourceType,
		}

		// Add report name if provided
		if upload.Name != "" {
			data["report_name"] = upload.Name
		}

		// Determine resource name from attributes or use index
		resourceName := ""
		if name, ok := instance.Attributes["name"].(string); ok && name != "" {
			resourceName = name
		} else if id, ok := instance.Attributes["id"].(string); ok && id != "" {
			resourceName = id
		} else {
			resourceName = fmt.Sprintf("%s-%d", upload.ResourceType, idx)
		}
		data["resource_name"] = resourceName

		// Add all attributes to the flat map (already validated above)
		for k, v := range instance.Attributes {
			data[k] = v
		}

		// Append data to CSV file
		if err := h.csvStorage.AppendData(orgID, data); err != nil {
			http.Error(w, fmt.Sprintf("Failed to store data: %v", err), http.StatusInternalServerError)
			return
		}
	}

	// Log successful upload
	logMsg := fmt.Sprintf("DATA: Successful upload - OrgID: %s, Provider: %s, Category: %s, ResourceType: %s, Instances: %d, IP: %s",
		orgID, upload.Provider, upload.Category, upload.ResourceType, len(upload.Instances), r.RemoteAddr)
	if upload.Name != "" {
		logMsg += fmt.Sprintf(", ReportName: %s", upload.Name)
	}
	log.Print(logMsg)

	// Return success response
	response := map[string]interface{}{
		"status":          "success",
		"message":         fmt.Sprintf("Successfully uploaded %d instance(s)", len(upload.Instances)),
		"org_id":          orgID.String(),
		"instances_count": len(upload.Instances),
	}

	// Include report name in response if provided
	if upload.Name != "" {
		response["report_name"] = upload.Name
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
		log.Printf("ERROR: Failed to retrieve data for org %s - Error: %v", orgID, err)
		http.Error(w, "Failed to retrieve data", http.StatusInternalServerError)
		return
	}

	// Log data retrieval
	log.Printf("DATA: Data retrieval - OrgID: %s, RecordCount: %d, IP: %s", orgID, len(uploads), r.RemoteAddr)

	// Return data as JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"org_id": orgID.String(),
		"count":  len(uploads),
		"data":   uploads,
	})
}
