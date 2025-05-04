package theme

import (
	"time"

	"github.com/soranjiro/axicalendar/internal/api" // Assuming api.gen.go is in internal/api

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// ThemeFieldType defines the allowed types for a theme field.
type ThemeFieldType string

const (
	FieldTypeText     ThemeFieldType = "text"
	FieldTypeDate     ThemeFieldType = "date"
	FieldTypeDateTime ThemeFieldType = "datetime"
	FieldTypeNumber   ThemeFieldType = "number"
	FieldTypeBoolean  ThemeFieldType = "boolean"
	FieldTypeTextarea ThemeFieldType = "textarea"
	FieldTypeSelect   ThemeFieldType = "select"
)

// ThemeField represents a single field definition within a theme.
// Corresponds to api.ThemeField but used internally.
type ThemeField struct {
	Name     string         `dynamodbav:"name"`     // Internal field name (unique within theme, snake_case)
	Label    string         `dynamodbav:"label"`    // Display label
	Type     ThemeFieldType `dynamodbav:"type"`     // Data type
	Required bool           `dynamodbav:"required"` // Whether the field is required
	// Add type-specific validation attributes here if needed (e.g., Options for select)
}

// Theme represents a calendar theme (default or custom).
// Corresponds to api.Theme but includes DynamoDB keys.
type Theme struct {
	PK                string       `dynamodbav:"PK"` // Partition Key: THEME#<theme_id>
	SK                string       `dynamodbav:"SK"` // Sort Key: METADATA
	ThemeID           uuid.UUID    `dynamodbav:"ThemeID"`
	ThemeName         string       `dynamodbav:"ThemeName"`
	Fields            []ThemeField `dynamodbav:"Fields"`
	IsDefault         bool         `dynamodbav:"IsDefault"`
	OwnerUserID       *uuid.UUID   `dynamodbav:"OwnerUserID,omitempty"`       // Null for default themes
	SupportedFeatures []string     `dynamodbav:"SupportedFeatures,omitempty"` // V1 Add: List of supported features
	CreatedAt         time.Time    `dynamodbav:"CreatedAt"`
	UpdatedAt         time.Time    `dynamodbav:"UpdatedAt"`
}

// UserThemeLink represents the relationship between a user and a theme they can use.
// Used for DynamoDB item: PK=USER#<user_id>, SK=THEME#<theme_id>
type UserThemeLink struct {
	PK        string    `dynamodbav:"PK"` // Partition Key: USER#<user_id>
	SK        string    `dynamodbav:"SK"` // Sort Key: THEME#<theme_id>
	UserID    uuid.UUID `dynamodbav:"UserID"`
	ThemeID   uuid.UUID `dynamodbav:"ThemeID"`
	ThemeName string    `dynamodbav:"ThemeName"` // Denormalized for query efficiency
	CreatedAt time.Time `dynamodbav:"CreatedAt"`
}

// --- Conversion Helpers ---

// ToApiThemeField converts internal ThemeField to API ThemeField
func ToApiThemeField(mf ThemeField) api.ThemeField {
	req := mf.Required // Copy bool value
	return api.ThemeField{
		Label:    mf.Label,
		Name:     mf.Name,
		Required: &req,                        // Assign pointer
		Type:     api.ThemeFieldType(mf.Type), // Convert string type
	}
}

// FromApiThemeField converts API ThemeField to internal ThemeField
func FromApiThemeField(af api.ThemeField) ThemeField {
	required := false
	if af.Required != nil {
		required = *af.Required
	}
	return ThemeField{
		Label:    af.Label,
		Name:     af.Name,
		Required: required,
		Type:     ThemeFieldType(af.Type), // Convert string type
	}
}

// ToApiTheme converts internal Theme to API Theme
func ToApiTheme(mt Theme) api.Theme {
	apiFields := make([]api.ThemeField, len(mt.Fields))
	for i, f := range mt.Fields {
		apiFields[i] = ToApiThemeField(f)
	}
	isDefault := mt.IsDefault           // Copy bool value
	themeID := mt.ThemeID               // Copy UUID
	createdAt := mt.CreatedAt           // Copy time
	updatedAt := mt.UpdatedAt           // Copy time
	var ownerUserID *openapi_types.UUID // Use openapi_types.UUID for pointer
	if mt.OwnerUserID != nil {
		uidCopy := *mt.OwnerUserID
		ownerUserID = &uidCopy // Assign pointer
	}

	// Ensure SupportedFeatures is not nil before assigning
	supportedFeatures := mt.SupportedFeatures
	if supportedFeatures == nil {
		supportedFeatures = []string{} // Return empty array instead of null
	}
	// Convert slice to pointer for the API struct
	apiSupportedFeatures := &supportedFeatures

	return api.Theme{
		CreatedAt:         &createdAt,
		Fields:            apiFields,
		IsDefault:         &isDefault, // Assign pointer
		ThemeId:           &themeID,   // Assign pointer to UUID
		ThemeName:         mt.ThemeName,
		UpdatedAt:         &updatedAt,
		OwnerUserId:       ownerUserID,          // Map OwnerUserID to OwnerUserId
		SupportedFeatures: apiSupportedFeatures, // Assign pointer to features slice
	}
}

// FromApiTheme converts API Theme to internal Theme (partial conversion)
func FromApiTheme(at api.Theme) Theme {
	// Note: PK, SK are not present in api.Theme.
	// IsDefault, OwnerUserID might be nil in API request, handle appropriately.
	var themeID uuid.UUID
	if at.ThemeId != nil {
		themeID = *at.ThemeId
	}
	var ownerUserID *uuid.UUID
	if at.OwnerUserId != nil {
		uidCopy := *at.OwnerUserId
		ownerUserID = &uidCopy
	}
	isDefault := false
	if at.IsDefault != nil {
		isDefault = *at.IsDefault
	}
	fields := make([]ThemeField, len(at.Fields))
	for i, f := range at.Fields {
		fields[i] = FromApiThemeField(f)
	}
	supportedFeatures := []string{}
	if at.SupportedFeatures != nil {
		supportedFeatures = *at.SupportedFeatures
	}

	return Theme{
		ThemeID:           themeID,
		ThemeName:         at.ThemeName,
		Fields:            fields,
		IsDefault:         isDefault,
		OwnerUserID:       ownerUserID,
		SupportedFeatures: supportedFeatures,
		// CreatedAt, UpdatedAt might be nil
	}
}

// ThemeRepository defines the interface for theme data persistence.
// Implementations will handle the actual database interactions.
type Repository interface {
	// Define methods for theme CRUD operations, e.g.:
	// GetThemeByID(ctx context.Context, themeID uuid.UUID) (*Theme, error)
	// ListThemes(ctx context.Context, userID uuid.UUID) ([]Theme, error)
	// CreateTheme(ctx context.Context, theme *Theme) error
	// UpdateTheme(ctx context.Context, theme *Theme) error
	// DeleteTheme(ctx context.Context, themeID uuid.UUID, userID uuid.UUID) error
	// AddUserThemeLink(ctx context.Context, link *UserThemeLink) error
	// RemoveUserThemeLink(ctx context.Context, userID, themeID uuid.UUID) error
	// ListUserThemes(ctx context.Context, userID uuid.UUID) ([]UserThemeLink, error)
}
