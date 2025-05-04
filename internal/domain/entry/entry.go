package entry

import (
	"context"
	"fmt"
	"time"

	"github.com/soranjiro/axicalendar/internal/domain/theme" // For validation

	"github.com/google/uuid"
)

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
	GSI1SK string `dynamodbav:"GSI1SK"` // ENTRY_DATE#<entry_date>#<theme_id>#<entry_id> (Updated based on design doc GSI-1)
}

// ValidateDataAgainstTheme checks if the entry's data matches the theme's field definitions.
func (e *Entry) ValidateDataAgainstTheme(fields []theme.ThemeField) error {
	definedFields := make(map[string]theme.ThemeField)
	for _, f := range fields {
		definedFields[f.Name] = f
	}

	// Check required fields are present and not empty
	for _, field := range fields {
		if field.Required {
			val, exists := e.Data[field.Name]
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
	for key, value := range e.Data {
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

// EntryRepository defines the interface for entry data persistence.
type Repository interface {
	// Define methods for entry CRUD operations, e.g.:
	GetEntryByID(ctx context.Context, userID, entryID uuid.UUID) (*Entry, error)
	ListEntriesByDateRange(ctx context.Context, userID uuid.UUID, startDate, endDate string, themeIDs []uuid.UUID) ([]Entry, error)
	CreateEntry(ctx context.Context, entry *Entry) error
	UpdateEntry(ctx context.Context, entry *Entry) error
	DeleteEntry(ctx context.Context, userID, entryID uuid.UUID) error
}
