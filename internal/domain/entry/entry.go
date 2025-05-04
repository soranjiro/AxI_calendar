package entry

import (
	"time"

	"github.com/soranjiro/axicalendar/internal/api" // Assuming api.gen.go is in internal/api

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
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
		EntryId:   &entryID, // Assign pointer to UUID
		ThemeId:   themeID,  // Not a pointer in API spec
		UpdatedAt: &updatedAt,
		UserId:    &userID, // Assign pointer to UUID
	}
}

// FromApiEntry converts API Entry to internal Entry (partial conversion, might need adjustments)
func FromApiEntry(ae api.Entry) Entry {
	// Note: PK, SK, GSI keys are not present in api.Entry and need to be constructed elsewhere.
	// UserID might be nil in the API response, handle appropriately.
	var userID uuid.UUID
	if ae.UserId != nil {
		userID = *ae.UserId
	}
	return Entry{
		EntryID:   *ae.EntryId, // Assuming EntryId is never nil in contexts where this is called
		ThemeID:   ae.ThemeId,
		UserID:    userID,
		EntryDate: ae.EntryDate.Format("2006-01-02"),
		Data:      ae.Data,
		// CreatedAt and UpdatedAt might be nil in API request, handle appropriately.
		// CreatedAt: *ae.CreatedAt,
		// UpdatedAt: *ae.UpdatedAt,
	}
}

// EntryRepository defines the interface for entry data persistence.
type Repository interface {
	// Define methods for entry CRUD operations, e.g.:
	// GetEntryByID(ctx context.Context, userID, entryID uuid.UUID) (*Entry, error)
	// ListEntriesByDateRange(ctx context.Context, userID uuid.UUID, startDate, endDate string, themeIDs []uuid.UUID) ([]Entry, error)
	// CreateEntry(ctx context.Context, entry *Entry) error
	// UpdateEntry(ctx context.Context, entry *Entry) error
	// DeleteEntry(ctx context.Context, userID, entryID uuid.UUID) error
}
