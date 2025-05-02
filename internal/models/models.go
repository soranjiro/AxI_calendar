package models

import (
	"time"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/soranjiro/axicalendar/internal/api" // Import generated API types
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
	PK        string       `dynamodbav:"PK"` // Partition Key: THEME#<theme_id>
	SK        string       `dynamodbav:"SK"` // Sort Key: METADATA
	ThemeID   uuid.UUID    `dynamodbav:"ThemeID"`
	ThemeName string       `dynamodbav:"ThemeName"`
	Fields    []ThemeField `dynamodbav:"Fields"`
	IsDefault bool         `dynamodbav:"IsDefault"`
	UserID    *uuid.UUID   `dynamodbav:"UserID,omitempty"` // Null for default themes
	CreatedAt time.Time    `dynamodbav:"CreatedAt"`
	UpdatedAt time.Time    `dynamodbav:"UpdatedAt"`
	// GSI1 Keys for listing themes by user
	GSI1PK *string `dynamodbav:"GSI1PK,omitempty"` // USER#<user_id>
	GSI1SK *string `dynamodbav:"GSI1SK,omitempty"` // THEME#<theme_id>
}

// Entry represents a single calendar entry.
// Corresponds to api.Entry but includes DynamoDB keys.
type Entry struct {
	PK        string                 `dynamodbav:"PK"` // Partition Key: USER#<user_id>
	SK        string                 `dynamodbav:"SK"` // Sort Key: ENTRY#<entry_date>#<entry_id>
	EntryID   uuid.UUID              `dynamodbav:"EntryID"`
	ThemeID   uuid.UUID              `dynamodbav:"ThemeID"`
	UserID    uuid.UUID              `dynamodbav:"UserID"`
	EntryDate string                 `dynamodbav:"EntryDate"` // YYYY-MM-DD format for easier querying
	Data      map[string]interface{} `dynamodbav:"Data"`      // Custom fields data
	CreatedAt time.Time              `dynamodbav:"CreatedAt"`
	UpdatedAt time.Time              `dynamodbav:"UpdatedAt"`
	// GSI1 Keys for querying by date range
	GSI1PK string `dynamodbav:"GSI1PK"` // Same as PK: USER#<user_id>
	GSI1SK string `dynamodbav:"GSI1SK"` // ENTRY_DATE#<entry_date>#<entry_id>
}

// UserThemeLink represents the relationship between a user and a theme they can use.
// Used for GSI query to find all themes for a user.
type UserThemeLink struct {
	PK      string    `dynamodbav:"PK"` // Partition Key: USER#<user_id>
	SK      string    `dynamodbav:"SK"` // Sort Key: THEME#<theme_id>
	UserID  uuid.UUID `dynamodbav:"UserID"`
	ThemeID uuid.UUID `dynamodbav:"ThemeID"`
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
	isDefault := mt.IsDefault // Copy bool value
	themeID := mt.ThemeID     // Copy UUID
	createdAt := mt.CreatedAt // Copy time
	updatedAt := mt.UpdatedAt // Copy time
	var userID *uuid.UUID
	if mt.UserID != nil {
		uidCopy := *mt.UserID
		userID = &uidCopy
	}

	return api.Theme{
		CreatedAt: &createdAt,
		Fields:    apiFields,
		IsDefault: &isDefault, // Assign pointer
		ThemeId:   &themeID,
		ThemeName: mt.ThemeName,
		UpdatedAt: &updatedAt,
		UserId:    userID,
	}
}

// ToApiEntry converts internal Entry to API Entry
func ToApiEntry(me Entry) api.Entry {
	entryID := me.EntryID // Copy UUID
	themeID := me.ThemeID // Copy UUID
	userID := me.UserID   // Copy UUID
	createdAt := me.CreatedAt
	updatedAt := me.UpdatedAt

	// Convert YYYY-MM-DD string back to openapi_types.Date
	entryDateTime, _ := time.Parse("2006-01-02", me.EntryDate) // Handle error appropriately in real code
	apiEntryDate := openapi_types.Date{Time: entryDateTime}

	return api.Entry{
		CreatedAt: &createdAt,
		Data:      me.Data,
		EntryDate: apiEntryDate,
		EntryId:   &entryID,
		ThemeId:   themeID, // Not a pointer in API spec
		UpdatedAt: &updatedAt,
		UserId:    &userID,
	}
}
