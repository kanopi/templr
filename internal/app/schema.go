package app

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"gopkg.in/yaml.v3"
)

const (
	// DefaultSchemaFile is the default schema filename to look for
	DefaultSchemaFile = ".templr.schema.yml"
)

// SchemaError represents a schema validation error
type SchemaError struct {
	Path       string // JSON path (e.g., ".service.replicas")
	Message    string // Error message
	Value      string // The actual value that failed
	Suggestion string // Helpful suggestion for fixing
}

// SchemaValidationResult contains validation results
type SchemaValidationResult struct {
	Errors   []SchemaError
	Warnings []SchemaError
	Passed   bool
}

// ValidateWithSchema validates data against a YAML schema file
func ValidateWithSchema(data map[string]interface{}, schemaPath string, mode string) (*SchemaValidationResult, error) {
	// Read schema file
	schemaBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("read schema file: %w", err)
	}

	// Parse schema YAML to map
	var schemaMap map[string]interface{}
	if err := yaml.Unmarshal(schemaBytes, &schemaMap); err != nil {
		return nil, fmt.Errorf("parse schema YAML: %w", err)
	}

	// Compile the schema directly from the map
	compiler := jsonschema.NewCompiler()

	// Add the schema using the map directly
	if err := compiler.AddResource("schema.json", schemaMap); err != nil {
		return nil, fmt.Errorf("add schema resource: %w", err)
	}

	schema, err := compiler.Compile("schema.json")
	if err != nil {
		return nil, fmt.Errorf("compile schema: %w", err)
	}

	// Validate the data
	if err := schema.Validate(data); err != nil {
		return parseValidationErrors(err, mode), nil
	}

	return &SchemaValidationResult{
		Passed: true,
		Errors: []SchemaError{},
	}, nil
}

// parseValidationErrors converts jsonschema validation errors to SchemaErrors
func parseValidationErrors(err error, mode string) *SchemaValidationResult {
	result := &SchemaValidationResult{
		Errors: []SchemaError{},
		Passed: false,
	}

	// Handle validation error
	if validationErr, ok := err.(*jsonschema.ValidationError); ok {
		result.Errors = flattenValidationErrors(validationErr, mode)
	} else {
		// Generic error
		result.Errors = append(result.Errors, SchemaError{
			Path:    "",
			Message: err.Error(),
		})
	}

	return result
}

// flattenValidationErrors recursively flattens nested validation errors
func flattenValidationErrors(err *jsonschema.ValidationError, mode string) []SchemaError {
	var errors []SchemaError

	// Convert instance location to readable path
	path := formatPathSegments(err.InstanceLocation)

	// Build error message and suggestion
	message := err.Error()
	suggestion := buildSuggestion(err, mode)

	// Value is shown in the error message
	value := ""

	errors = append(errors, SchemaError{
		Path:       path,
		Message:    message,
		Value:      value,
		Suggestion: suggestion,
	})

	// Recursively add nested errors
	for _, cause := range err.Causes {
		errors = append(errors, flattenValidationErrors(cause, mode)...)
	}

	return errors
}

// formatPathSegments converts path segments to readable dot notation
func formatPathSegments(segments []string) string {
	if len(segments) == 0 {
		return "(root)"
	}
	// Convert ["service", "replicas"] to .service.replicas
	path := "." + strings.Join(segments, ".")
	return path
}

// buildSuggestion creates helpful suggestions based on error type
func buildSuggestion(err *jsonschema.ValidationError, mode string) string {
	errStr := err.Error()

	// Type mismatch errors
	if strings.Contains(errStr, "expected") && strings.Contains(errStr, "but got") {
		if strings.Contains(errStr, "integer") {
			return "set as number without quotes, e.g. `replicas: 2`"
		}
		if strings.Contains(errStr, "string") {
			return "set as string with quotes, e.g. `name: \"value\"`"
		}
		if strings.Contains(errStr, "boolean") {
			return "set as boolean: true or false (no quotes)"
		}
	}

	// Required property errors
	if strings.Contains(errStr, "required") || strings.Contains(errStr, "missing") {
		return "add this required field to your values file"
	}

	// Enum errors
	if strings.Contains(errStr, "enum") || strings.Contains(errStr, "one of") {
		return "use one of the allowed values specified in the schema"
	}

	// Additional properties error
	if strings.Contains(errStr, "additionalProperties") {
		return "remove this field or update schema to allow additional properties"
	}

	return ""
}

// FindSchemaFile looks for schema file in order of precedence
func FindSchemaFile(configSchemaPath string) string {
	// 1. If explicit path provided in config, use that
	if configSchemaPath != "" {
		if _, err := os.Stat(configSchemaPath); err == nil {
			return configSchemaPath
		}
	}

	// 2. Look for default schema file in current directory
	if _, err := os.Stat(DefaultSchemaFile); err == nil {
		return DefaultSchemaFile
	}

	// 3. Look in .templr/ subdirectory
	templrSchema := filepath.Join(".templr", "schema.yml")
	if _, err := os.Stat(templrSchema); err == nil {
		return templrSchema
	}

	return ""
}

// GenerateSchema generates a JSON Schema from data
func GenerateSchema(data map[string]interface{}, config SchemaGenerateConfig) (map[string]interface{}, error) {
	schema := map[string]interface{}{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type":    "object",
	}

	// Generate properties
	properties := make(map[string]interface{})
	var required []string

	for key, value := range data {
		propSchema := generatePropertySchema(value, config)
		properties[key] = propSchema

		// Determine if required
		if shouldBeRequired(key, value, config.Required) {
			required = append(required, key)
		}
	}

	schema["properties"] = properties

	if len(required) > 0 {
		sort.Strings(required) // Consistent ordering
		schema["required"] = required
	}

	if !config.AdditionalProps {
		schema["additionalProperties"] = false
	}

	return schema, nil
}

// generatePropertySchema generates schema for a single property
func generatePropertySchema(value interface{}, config SchemaGenerateConfig) map[string]interface{} {
	propSchema := make(map[string]interface{})

	if !config.InferTypes {
		return propSchema
	}

	switch v := value.(type) {
	case string:
		propSchema["type"] = "string"
		if v != "" {
			propSchema["description"] = fmt.Sprintf("String value (example: %s)", truncate(v, 30))
		}

	case int, int32, int64, float32, float64:
		propSchema["type"] = "number"
		propSchema["description"] = fmt.Sprintf("Numeric value (example: %v)", v)

	case bool:
		propSchema["type"] = "boolean"

	case []interface{}:
		propSchema["type"] = "array"
		if len(v) > 0 {
			// Infer item schema from first element
			propSchema["items"] = generatePropertySchema(v[0], config)
		}

	case map[string]interface{}:
		propSchema["type"] = "object"
		nestedProps := make(map[string]interface{})
		var nestedRequired []string

		for key, val := range v {
			nestedProps[key] = generatePropertySchema(val, config)
			if shouldBeRequired(key, val, config.Required) {
				nestedRequired = append(nestedRequired, key)
			}
		}

		propSchema["properties"] = nestedProps
		if len(nestedRequired) > 0 {
			sort.Strings(nestedRequired)
			propSchema["required"] = nestedRequired
		}

		if !config.AdditionalProps {
			propSchema["additionalProperties"] = false
		}

	default:
		// Unknown type
		propSchema["description"] = fmt.Sprintf("Type: %s", reflect.TypeOf(value))
	}

	return propSchema
}

// shouldBeRequired determines if a field should be marked as required
func shouldBeRequired(key string, value interface{}, mode string) bool {
	switch mode {
	case "all":
		return true
	case "none":
		return false
	case "auto":
		// Auto mode: require if value is non-zero/non-empty
		if value == nil {
			return false
		}
		switch v := value.(type) {
		case string:
			return v != ""
		case []interface{}:
			return len(v) > 0
		case map[string]interface{}:
			return len(v) > 0
		default:
			return true
		}
	default:
		return false
	}
}

// truncate truncates a string to maxLen characters
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// FormatSchemaErrors formats schema errors for display
func FormatSchemaErrors(result *SchemaValidationResult, mode string) string {
	if result.Passed {
		return ""
	}

	var output strings.Builder
	prefix := "[templr:error:schema]"
	if mode == "warn" {
		prefix = "[templr:warn:schema]"
	}

	for _, err := range result.Errors {
		output.WriteString(fmt.Sprintf("%s %s: %s\n", prefix, err.Path, err.Message))
		if err.Suggestion != "" {
			output.WriteString(fmt.Sprintf("  Suggestion: %s\n", err.Suggestion))
		}
	}

	return output.String()
}
