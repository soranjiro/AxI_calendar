package converter

import (
	"log"
	"time"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/soranjiro/axicalendar/internal/domain/entry"
	"github.com/soranjiro/axicalendar/internal/presentation/api"
)

// --- Entry Converters ---

// ToApiEntry converts internal Entry to API Entry
func ToApiEntry(de entry.Entry) (api.Entry, error) {
	entryID := de.EntryID
	userID := de.UserID
	themeID := de.ThemeID
	createdAt := de.CreatedAt
	updatedAt := de.UpdatedAt

	// Parse the YYYY-MM-DD date string into time.Time
	entryDateTime, err := time.Parse("2006-01-02", de.EntryDate)
	if err != nil {
		// Log error and potentially return a default or zero date
		log.Printf("ERROR: Failed to parse EntryDate string '%s' for entry %s: %v", de.EntryDate, entryID, err)
		// Depending on requirements, you might return error or a zero date
		// return api.Entry{}, fmt.Errorf("invalid entry date format: %w", err)
		entryDateTime = time.Time{} // Use zero time as fallback
	}
	apiEntryDate := openapi_types.Date{Time: entryDateTime}

	return api.Entry{
		EntryId:   &entryID,
		UserId:    &userID,
		ThemeId:   themeID,
		EntryDate: apiEntryDate,
		Data:      de.Data,
		CreatedAt: &createdAt,
		UpdatedAt: &updatedAt,
	}, nil // Return nil error even if date parsing failed (logged)
}

// ToApiEntries converts a slice of internal Entry to API Entry
func ToApiEntries(des []entry.Entry) ([]api.Entry, error) {
	aes := make([]api.Entry, len(des))
	for i, de := range des {
		ae, err := ToApiEntry(de)
		if err != nil {
			log.Printf("WARN: Failed to convert entry %s to API format: %v", de.EntryID, err)
			// Skip adding the problematic entry
			continue
		}
		aes[i] = ae
	}
	// Filter out zero-value entries if any were skipped
	result := make([]api.Entry, 0, len(aes))
	for _, ae := range aes {
		if ae.EntryId != nil { // Check if it's a valid entry
			result = append(result, ae)
		}
	}
	return result, nil // Return nil error even if some entries failed
}

// FromApiCreateEntryRequest converts API CreateEntryRequest to domain Entry
func FromApiCreateEntryRequest(req api.CreateEntryRequest, userID uuid.UUID) (entry.Entry, error) {
	newEntry := entry.Entry{
		EntryID:   uuid.New(), // Generate new ID
		ThemeID:   req.ThemeId,
		UserID:    userID,
		EntryDate: req.EntryDate.Format("2006-01-02"), // Store as YYYY-MM-DD string
		Data:      req.Data,
		// CreatedAt, UpdatedAt, PK, SK, GSI keys set by repository
	}
	return newEntry, nil
}

// FromApiUpdateEntryRequest converts API UpdateEntryRequest to domain Entry
// Requires existing entry to preserve fields not allowed to be updated.
func FromApiUpdateEntryRequest(req api.UpdateEntryRequest, entryID uuid.UUID, userID uuid.UUID, existingEntry entry.Entry) (entry.Entry, error) {
	updatedEntry := entry.Entry{
		EntryID:   entryID,
		ThemeID:   existingEntry.ThemeID, // Theme cannot be changed
		UserID:    userID,
		EntryDate: req.EntryDate.Format("2006-01-02"), // Store as YYYY-MM-DD string
		Data:      req.Data,
		CreatedAt: existingEntry.CreatedAt, // Preserve original creation time
		// UpdatedAt, PK, SK handled by repository
	}
	return updatedEntry, nil
}
