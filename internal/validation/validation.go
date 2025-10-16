package validation

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

var (
	// stateNameRegex allows only alphanumeric, hyphens, underscores, and dots
	stateNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_.-]+$`)

	// attributeKeyRegex for validating attribute keys
	attributeKeyRegex = regexp.MustCompile(`^[a-zA-Z0-9_.-]+$`)
)

// ValidateStateName validates a Terraform state name to prevent path traversal
func ValidateStateName(name string) error {
	if name == "" {
		return fmt.Errorf("state name is required")
	}

	// Check for path traversal attempts
	if strings.Contains(name, "..") || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return fmt.Errorf("invalid state name: path traversal detected")
	}

	// Validate against allowed characters
	if !stateNameRegex.MatchString(name) {
		return fmt.Errorf("invalid state name: only alphanumeric characters, hyphens, underscores, and dots allowed")
	}

	// Limit length to prevent abuse
	if len(name) > 255 {
		return fmt.Errorf("state name too long: maximum 255 characters")
	}

	return nil
}

// ValidateAttributeKey validates an attribute key in upload data
func ValidateAttributeKey(key string) error {
	if key == "" {
		return fmt.Errorf("attribute key cannot be empty")
	}

	if len(key) > 100 {
		return fmt.Errorf("attribute key too long: maximum 100 characters")
	}

	if !attributeKeyRegex.MatchString(key) {
		return fmt.Errorf("invalid attribute key: only alphanumeric characters, hyphens, underscores, and dots allowed")
	}

	return nil
}

// ValidateAttributeValue validates an attribute value
func ValidateAttributeValue(val interface{}) error {
	str := fmt.Sprintf("%v", val)

	// Prevent extremely long values
	if len(str) > 10000 {
		return fmt.Errorf("attribute value too long: maximum 10000 characters")
	}

	return nil
}

// ValidateResourceType validates a resource type field
func ValidateResourceType(resourceType string) error {
	if resourceType == "" {
		return fmt.Errorf("resource_type is required")
	}

	if len(resourceType) > 200 {
		return fmt.Errorf("resource_type too long: maximum 200 characters")
	}

	// Allow alphanumeric, underscores, and hyphens
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, resourceType)
	if !matched {
		return fmt.Errorf("invalid resource_type: only alphanumeric characters, hyphens, and underscores allowed")
	}

	return nil
}

// ValidateProvider validates a provider field
func ValidateProvider(provider string) error {
	if provider == "" {
		return fmt.Errorf("provider is required")
	}

	if len(provider) > 100 {
		return fmt.Errorf("provider too long: maximum 100 characters")
	}

	// Allow alphanumeric, underscores, and hyphens
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, provider)
	if !matched {
		return fmt.Errorf("invalid provider: only alphanumeric characters, hyphens, and underscores allowed")
	}

	return nil
}

// ValidateCategory validates a category field
func ValidateCategory(category string) error {
	if category == "" {
		return fmt.Errorf("category is required")
	}

	if len(category) > 100 {
		return fmt.Errorf("category too long: maximum 100 characters")
	}

	// Allow alphanumeric, underscores, and hyphens
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, category)
	if !matched {
		return fmt.Errorf("invalid category: only alphanumeric characters, hyphens, and underscores allowed")
	}

	return nil
}

// ValidateJSONDepth validates that JSON data doesn't exceed maximum nesting depth
func ValidateJSONDepth(data interface{}, maxDepth int) error {
	return validateDepthRecursive(data, 0, maxDepth)
}

// validateDepthRecursive recursively checks the depth of nested structures
func validateDepthRecursive(data interface{}, currentDepth, maxDepth int) error {
	if currentDepth > maxDepth {
		return fmt.Errorf("JSON exceeds maximum nesting depth of %d", maxDepth)
	}

	v := reflect.ValueOf(data)
	switch v.Kind() {
	case reflect.Map:
		iter := v.MapRange()
		for iter.Next() {
			if err := validateDepthRecursive(iter.Value().Interface(), currentDepth+1, maxDepth); err != nil {
				return err
			}
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			if err := validateDepthRecursive(v.Index(i).Interface(), currentDepth+1, maxDepth); err != nil {
				return err
			}
		}
	case reflect.Interface, reflect.Ptr:
		if !v.IsNil() {
			return validateDepthRecursive(v.Elem().Interface(), currentDepth, maxDepth)
		}
	}

	return nil
}

// ValidateJSONComplexity validates JSON doesn't have too many total elements
func ValidateJSONComplexity(data interface{}, maxElements int) error {
	count := 0
	if err := countElementsRecursive(data, &count, maxElements); err != nil {
		return err
	}
	if count > maxElements {
		return fmt.Errorf("JSON has too many elements: %d (max: %d)", count, maxElements)
	}
	return nil
}

// countElementsRecursive counts total number of elements in nested structures
func countElementsRecursive(data interface{}, count *int, maxElements int) error {
	*count++
	if *count > maxElements {
		return fmt.Errorf("JSON complexity limit exceeded")
	}

	v := reflect.ValueOf(data)
	switch v.Kind() {
	case reflect.Map:
		iter := v.MapRange()
		for iter.Next() {
			if err := countElementsRecursive(iter.Value().Interface(), count, maxElements); err != nil {
				return err
			}
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			if err := countElementsRecursive(v.Index(i).Interface(), count, maxElements); err != nil {
				return err
			}
		}
	case reflect.Interface, reflect.Ptr:
		if !v.IsNil() {
			return countElementsRecursive(v.Elem().Interface(), count, maxElements)
		}
	}

	return nil
}

// ValidateJSONString validates a JSON string for size and complexity before parsing
func ValidateJSONString(jsonData []byte, maxSize int) error {
	// Check size
	if len(jsonData) > maxSize {
		return fmt.Errorf("JSON data too large: %d bytes (max: %d)", len(jsonData), maxSize)
	}

	// Validate it's well-formed JSON
	if !json.Valid(jsonData) {
		return fmt.Errorf("invalid JSON format")
	}

	return nil
}
