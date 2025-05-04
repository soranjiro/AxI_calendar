package validation

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/soranjiro/axicalendar/internal/api"
	"github.com/soranjiro/axicalendar/internal/domain/theme"
)

// --- Theme Validation Helpers ---

// ValidateApiThemeFields performs basic validation on API theme field definitions.
func ValidateApiThemeFields(fields []api.ThemeField) error {
	if len(fields) == 0 {
		return errors.New("theme must have at least one field")
	}
	names := make(map[string]bool)
	for i, field := range fields {
		if field.Name == "" {
			return fmt.Errorf("field %d: name is required", i)
		}
		if !IsValidFieldName(field.Name) {
			return fmt.Errorf("field %d ('%s'): name contains invalid characters (allowed: a-z, 0-9, _ starting with letter or _)", i, field.Name)
		}
		if field.Label == "" {
			return fmt.Errorf("field %d ('%s'): label is required", i, field.Name)
		}
		if _, exists := names[field.Name]; exists {
			return fmt.Errorf("field name '%s' is duplicated", field.Name)
		}
		names[field.Name] = true

		isValidType := false
		validTypes := []api.ThemeFieldType{
			api.Text,
			api.Date,
			api.Datetime,
			api.Number,
			api.Boolean,
			api.Textarea,
			api.Select,
		}
		for _, vt := range validTypes {
			if field.Type == vt {
				isValidType = true
				break
			}
		}
		if !isValidType {
			return fmt.Errorf("field '%s': invalid type '%s'", field.Name, field.Type)
		}

		if field.Required == nil {
			return fmt.Errorf("field %d ('%s'): required attribute must be explicitly set to true or false", i, field.Name)
		}
	}
	return nil
}

// IsValidFieldName checks if a field name is valid (e.g., snake_case).
func IsValidFieldName(name string) bool {
	if name == "" || (!((name[0] >= 'a' && name[0] <= 'z') || name[0] == '_')) {
		return false
	}
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_') {
			return false
		}
	}
	return true
}

// ValidateSupportedFeatures performs basic validation on supported feature names.
func ValidateSupportedFeatures(features []string) error {
	validFeatures := map[string]bool{
		"monthly_summary":      true,
		"category_aggregation": true,
		// Add other known valid features here
	}
	names := make(map[string]bool)
	for i, feature := range features {
		if feature == "" {
			return fmt.Errorf("feature %d: name cannot be empty", i)
		}
		if !IsValidFeatureName(feature) {
			return fmt.Errorf("feature %d ('%s'): name contains invalid characters or format (use snake_case)", i, feature)
		}
		if !validFeatures[feature] {
			log.Printf("WARN: Potentially unsupported feature '%s' included in theme definition.", feature)
		}
		if _, exists := names[feature]; exists {
			return fmt.Errorf("feature name '%s' is duplicated", feature)
		}
		names[feature] = true
	}
	return nil
}

// IsValidFeatureName checks if a feature name is valid (e.g., snake_case).
func IsValidFeatureName(name string) bool {
	if name == "" || !(name[0] >= 'a' && name[0] <= 'z') {
		return false
	}
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_') {
			return false
		}
	}
	return true
}

// --- Entry Validation Helpers ---

// ValidateEntryDataAgainstTheme checks if the provided data matches the theme's field definitions.
// Uses domain theme fields (theme.ThemeField).
func ValidateEntryDataAgainstTheme(data map[string]interface{}, fields []theme.ThemeField) error {
	definedFields := make(map[string]theme.ThemeField)
	for _, f := range fields {
		definedFields[f.Name] = f
	}

	// Check required fields are present and not empty
	for _, field := range fields {
		if field.Required {
			val, exists := data[field.Name]
			if !exists {
				return fmt.Errorf("required field '%s' is missing", field.Name)
			}
			if val == nil {
				return fmt.Errorf("required field '%s' cannot be null", field.Name)
			}
			// Add more specific checks based on type if needed (e.g., empty string)
			if field.Type == theme.FieldTypeText || field.Type == theme.FieldTypeTextarea || field.Type == theme.FieldTypeSelect {
				if strVal, ok := val.(string); !ok || strVal == "" {
					return fmt.Errorf("required field '%s' cannot be empty", field.Name)
				}
			}
		}
	}

	// Check types of provided data and presence of undefined fields
	for key, value := range data {
		fieldDef, exists := definedFields[key]
		if !exists {
			return fmt.Errorf("field '%s' is not defined in the theme", key)
		}

		// Skip type validation if value is nil (unless required, checked above)
		if value == nil {
			continue
		}

		// Type validation logic
		switch fieldDef.Type {
		case theme.FieldTypeText, theme.FieldTypeTextarea, theme.FieldTypeSelect:
			if _, ok := value.(string); !ok {
				return fmt.Errorf("field '%s' expects a string, got %T", key, value)
			}
		case theme.FieldTypeNumber:
			// Allow int or float64 from JSON unmarshalling
			switch value.(type) {
			case float64, int, int32, int64:
				// OK
			default:
				return fmt.Errorf("field '%s' expects a number, got %T", key, value)
			}
		case theme.FieldTypeBoolean:
			if _, ok := value.(bool); !ok {
				return fmt.Errorf("field '%s' expects a boolean, got %T", key, value)
			}
		case theme.FieldTypeDate:
			valStr, ok := value.(string)
			if !ok {
				return fmt.Errorf("field '%s' expects a date string (YYYY-MM-DD), got %T", key, value)
			}
			if _, err := time.Parse("2006-01-02", valStr); err != nil {
				return fmt.Errorf("field '%s' has invalid date format: %v. Expected YYYY-MM-DD", key, err)
			}
		case theme.FieldTypeDateTime:
			valStr, ok := value.(string)
			if !ok {
				return fmt.Errorf("field '%s' expects a datetime string (RFC3339), got %T", key, value)
			}
			if _, err := time.Parse(time.RFC3339, valStr); err != nil {
				return fmt.Errorf("field '%s' has invalid datetime format: %v. Expected RFC3339", key, err)
			}
		default:
			return fmt.Errorf("internal error: unknown field type '%s' for field '%s'", fieldDef.Type, key)
		}
	}

	return nil
}
