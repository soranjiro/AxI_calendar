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
	"github.com/soranjiro/axicalendar/internal/api"
	"github.com/soranjiro/axicalendar/internal/domain"
	"github.com/soranjiro/axicalendar/internal/domain/entry" // Needed for validation
	repo "github.com/soranjiro/axicalendar/internal/repository/dynamodb"
	"github.com/soranjiro/axicalendar/internal/validation" // Import validation package
)

// UpdateEntryUseCase defines the interface for the update entry use case.
type UpdateEntryUseCase interface {
	Execute(ctx context.Context, userID uuid.UUID, entryID uuid.UUID, req api.UpdateEntryRequest) (*api.Entry, error)
}

// updateEntryUseCase implements the UpdateEntryUseCase interface.
type updateEntryUseCase struct {
	entryRepo repo.EntryRepository
	themeRepo repo.ThemeRepository // Needed to validate data against theme
}

// NewUpdateEntryUseCase creates a new UpdateEntryUseCase.
func NewUpdateEntryUseCase(entryRepo repo.EntryRepository, themeRepo repo.ThemeRepository) UpdateEntryUseCase {
	return &updateEntryUseCase{entryRepo: entryRepo, themeRepo: themeRepo}
}

// Execute handles the logic for updating an entry.
func (uc *updateEntryUseCase) Execute(ctx context.Context, userID uuid.UUID, entryID uuid.UUID, req api.UpdateEntryRequest) (*api.Entry, error) {
	// 1. Get existing entry to find ThemeID and validate ownership/existence
	existingEntry, err := uc.entryRepo.GetEntryByID(ctx, userID, entryID)
	if err != nil {
		if errors.Is(err, domain.ErrEntryNotFound) {
			return nil, echo.NewHTTPError(http.StatusNotFound, api.Error{Message: "Entry not found"})
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
			return nil, echo.NewHTTPError(http.StatusNotFound, api.Error{Message: "Associated theme not found or access denied"})
		}
		log.Printf("Error validating theme %s for entry %s update: %v", existingEntry.ThemeID, entryID, err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, api.Error{Message: "Failed to validate associated theme"})
	}

	// 3. Validate new data against theme fields using validation package
	if err := validation.ValidateEntryDataAgainstTheme(req.Data, th.Fields); err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, api.Error{Message: fmt.Sprintf("Entry data validation failed: %v", err)})
	}

	// 4. Prepare updated entry domain model
	updatedEntry := entry.Entry{
		EntryID:   entryID,
		ThemeID:   existingEntry.ThemeID, // Theme cannot be changed
		UserID:    userID,
		EntryDate: req.EntryDate.Format("2006-01-02"), // Store as YYYY-MM-DD string
		Data:      req.Data,
		// CreatedAt, UpdatedAt, PK, SK handled by repository
	}

	// 5. Call repository to update entry
	err = uc.entryRepo.UpdateEntry(ctx, &updatedEntry)
	if err != nil {
		if errors.Is(err, domain.ErrEntryNotFound) {
			// This could happen if the entry was deleted between the Get and Update calls
			return nil, echo.NewHTTPError(http.StatusNotFound, api.Error{Message: "Entry not found during update attempt"})
		}
		log.Printf("Error updating entry %s in repository: %v", entryID, err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, api.Error{Message: "Failed to update entry"})
	}

	// 6. Fetch the updated entry to return the full object with updated timestamp
	finalEntry, err := uc.entryRepo.GetEntryByID(ctx, userID, entryID)
	if err != nil {
		// Log the inconsistency, but return the data we sent for update as approximation
		log.Printf("WARN: Failed to fetch updated entry %s after successful update: %v", entryID, err)
		updatedEntry.CreatedAt = existingEntry.CreatedAt // Keep original creation time
		updatedEntry.UpdatedAt = time.Now()              // Approximate update time
		apiEntry := entry.ToApiEntry(updatedEntry)
		return &apiEntry, nil
	}

	// 7. Convert to API model and return
	apiEntry := entry.ToApiEntry(*finalEntry)
	return &apiEntry, nil
}
