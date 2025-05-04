package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/soranjiro/axicalendar/internal/api" // Keep for request/error types
	"github.com/soranjiro/axicalendar/internal/domain"
	"github.com/soranjiro/axicalendar/internal/domain/entry" // Use domain entry
	// "github.com/soranjiro/axicalendar/internal/validation" // Validation moved to domain
)

// UpdateEntry handles the logic for updating an entry.
// Accepts IDs and API request, returns domain entry
func (uc *UseCase) UpdateEntry(ctx context.Context, userID uuid.UUID, entryID uuid.UUID, req api.UpdateEntryRequest) (*entry.Entry, error) {
	// 1. Get existing entry to find ThemeID and validate ownership/existence
	existingEntry, err := uc.entryRepo.GetEntryByID(ctx, userID, entryID)
	if err != nil {
		if errors.Is(err, domain.ErrEntryNotFound) {
			return nil, echo.NewHTTPError(http.StatusNotFound, api.Error{Message: "Entry not found"})
		}
		// Check for Forbidden error which might be returned by repo if userID doesn't match
		if errors.Is(err, domain.ErrForbidden) {
			return nil, echo.NewHTTPError(http.StatusForbidden, api.Error{Message: "Access denied to entry"})
		}
		log.Printf("Error retrieving existing entry %s for update: %v", entryID, err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, api.Error{Message: "Failed to retrieve existing entry"})
	}

	// 2. Validate theme exists (it should, but check anyway)
	th, err := uc.themeRepo.GetThemeByID(ctx, userID, existingEntry.ThemeID)
	if err != nil {
		// This indicates data inconsistency if the entry existed but the theme doesn't
		log.Printf("ERROR: Entry %s references non-existent/inaccessible theme %s", entryID, existingEntry.ThemeID)
		if errors.Is(err, domain.ErrThemeNotFound) || errors.Is(err, domain.ErrForbidden) {
			// Use 400 Bad Request as the entry references an invalid theme
			return nil, echo.NewHTTPError(http.StatusBadRequest, api.Error{Message: "Associated theme not found or access denied"})
		}
		log.Printf("Error validating theme %s for entry %s update: %v", existingEntry.ThemeID, entryID, err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, api.Error{Message: "Failed to validate associated theme"})
	}

	// 3. Prepare updated entry domain model from request and existing data
	updatedEntry := entry.Entry{
		EntryID:   entryID,
		ThemeID:   existingEntry.ThemeID, // Theme cannot be changed
		UserID:    userID,
		EntryDate: req.EntryDate.Format("2006-01-02"), // Store as YYYY-MM-DD string
		Data:      req.Data,
		CreatedAt: existingEntry.CreatedAt, // Preserve original creation time
		// UpdatedAt, PK, SK handled by repository
	}

	// 4. Validate new data against theme fields using domain method
	if err := updatedEntry.ValidateDataAgainstTheme(th.Fields); err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, api.Error{Message: fmt.Sprintf("Entry data validation failed: %v", err)})
	}

	// 5. Call repository to update entry
	err = uc.entryRepo.UpdateEntry(ctx, &updatedEntry)
	if err != nil {
		if errors.Is(err, domain.ErrEntryNotFound) {
			// This could happen if the entry was deleted between the Get and Update calls
			return nil, echo.NewHTTPError(http.StatusNotFound, api.Error{Message: "Entry not found during update attempt"})
		}
		// Check for Forbidden error from repo update (e.g., conditional check on user ID failed)
		if errors.Is(err, domain.ErrForbidden) {
			return nil, echo.NewHTTPError(http.StatusForbidden, api.Error{Message: "Access denied during entry update"})
		}
		log.Printf("Error updating entry %s in repository: %v", entryID, err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, api.Error{Message: "Failed to update entry"})
	}

	// 6. Fetch the updated entry to return the full object with updated timestamp
	finalEntry, err := uc.entryRepo.GetEntryByID(ctx, userID, entryID)
	if err != nil {
		// Log the inconsistency, but return the data we sent for update as approximation
		log.Printf("WARN: Failed to fetch updated entry %s after successful update: %v", entryID, err)
		updatedEntry.UpdatedAt = time.Now() // Approximate update time
		return &updatedEntry, nil
	}

	// 7. Return the fetched domain entry
	return finalEntry, nil
}
