package theme

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/google/uuid"
)

// FieldType defines the possible types for a theme field.
type FieldType string

const (
	FieldTypeText     FieldType = "text"
	FieldTypeDate     FieldType = "date"
	FieldTypeDateTime FieldType = "datetime"
	FieldTypeNumber   FieldType = "number"
	FieldTypeBoolean  FieldType = "boolean"
	FieldTypeTextarea FieldType = "textarea"
	FieldTypeSelect   FieldType = "select"
)

// ThemeField represents a single field definition within a theme.
// Corresponds to api.ThemeField.
type ThemeField struct {
	Name     string    `dynamodbav:"Name"`     // Internal field name
	Label    string    `dynamodbav:"Label"`    // Display label
	Type     FieldType `dynamodbav:"Type"`     // Data type
	Required bool      `dynamodbav:"Required"` // Whether the field is required
}

// Theme represents a calendar theme definition.
// Corresponds to api.Theme but includes DynamoDB keys and uses domain types.
type Theme struct {
	PK                string       `dynamodbav:"PK"` // Partition Key: USER#<user_id> or DEFAULT#THEME
	SK                string       `dynamodbav:"SK"` // Sort Key: THEME#<theme_id>
	ThemeID           uuid.UUID    `dynamodbav:"ThemeID"`
	ThemeName         string       `dynamodbav:"ThemeName"`
	Fields            []ThemeField `dynamodbav:"Fields"`
	IsDefault         bool         `dynamodbav:"IsDefault"`
	OwnerUserID       *uuid.UUID   `dynamodbav:"OwnerUserID,omitempty"` // Pointer to allow null for default themes
	SupportedFeatures []string     `dynamodbav:"SupportedFeatures"`
	CreatedAt         time.Time    `dynamodbav:"CreatedAt"`
	UpdatedAt         time.Time    `dynamodbav:"UpdatedAt"`
}

// UserThemeLink represents the association between a user and a theme they can use.
// This is used for DynamoDB storage to quickly find themes accessible by a user.
type UserThemeLink struct {
	PK      string    `dynamodbav:"PK"`      // Partition Key: USER#<user_id>
	SK      string    `dynamodbav:"SK"`      // Sort Key: LINK#THEME#<theme_id>
	UserID  uuid.UUID `dynamodbav:"UserID"`  // For potential GSI queries if needed
	ThemeID uuid.UUID `dynamodbav:"ThemeID"` // For potential GSI queries if needed
}

// --- Validation Logic ---

var validFieldNameRegex = regexp.MustCompile(`^[a-z_][a-z0-9_]*$`)

// Validate checks the theme's own fields for validity.
func (t *Theme) Validate() error {
	if t.ThemeName == "" {
		return errors.New("theme name is required")
	}
	if err := ValidateThemeFields(t.Fields); err != nil {
		return fmt.Errorf("invalid theme fields: %w", err)
	}
	if err := ValidateSupportedFeatures(t.SupportedFeatures); err != nil {
		return fmt.Errorf("invalid supported features: %w", err)
	}
	return nil
}

// ValidateThemeFields performs basic validation on domain theme field definitions.
func ValidateThemeFields(fields []ThemeField) error {
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
		validTypes := []FieldType{
			FieldTypeText,
			FieldTypeDate,
			FieldTypeDateTime,
			FieldTypeNumber,
			FieldTypeBoolean,
			FieldTypeTextarea,
			FieldTypeSelect,
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
		// Required is a boolean, no need for nil check like in API model
	}
	return nil
}

// IsValidFieldName checks if a field name is valid (e.g., snake_case).
func IsValidFieldName(name string) bool {
	if name == "" {
		return false
	}
	return validFieldNameRegex.MatchString(name)
}

// ValidateSupportedFeatures performs basic validation on supported feature names.
func ValidateSupportedFeatures(features []string) error {
	// Allow empty or nil features list
	if len(features) == 0 {
		return nil
	}

	validFeatures := map[string]bool{
		"monthly_summary":      true,
		"category_aggregation": true,
		"SumAll":               true, // Add "SumAll" here
		// Add other known valid features here
	}
	names := make(map[string]bool)
	for i, feature := range features {
		if feature == "" {
			return fmt.Errorf("feature %d: name cannot be empty", i)
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

// --- End Validation Logic ---

// ThemeRepository defines the interface for theme data persistence.
type Repository interface {
	// Define methods for theme CRUD operations, e.g.:
	GetThemeByID(ctx context.Context, userID, themeID uuid.UUID) (*Theme, error) // Needs adjustment for default themes
	ListThemes(ctx context.Context, userID uuid.UUID) ([]Theme, error)
	CreateTheme(ctx context.Context, theme *Theme) error
	UpdateTheme(ctx context.Context, theme *Theme) error
	DeleteTheme(ctx context.Context, userID, themeID uuid.UUID) error
}
