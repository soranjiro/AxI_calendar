package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/soranjiro/axicalendar/internal/api"
	"github.com/soranjiro/axicalendar/internal/domain"
	"github.com/soranjiro/axicalendar/internal/domain/entry" // Needed for ToApiEntry
	"github.com/soranjiro/axicalendar/internal/validation"   // Import validation package
)

// CreateEntry handles the logic for creating an entry.
func (uc *UseCase) CreateEntry(ctx context.Context, userID uuid.UUID, req api.CreateEntryRequest) (*api.Entry, error) {
	// 1. Validate theme exists and user has access
	th, err := uc.themeRepo.GetThemeByID(ctx, userID, req.ThemeId)
	if err != nil {
		if errors.Is(err, domain.ErrThemeNotFound) || errors.Is(err, domain.ErrForbidden) {
			return nil, echo.NewHTTPError(http.StatusNotFound, api.Error{Message: "Theme not found or access denied"})
		}
		log.Printf("Error validating theme %s for user %s: %v", req.ThemeId, userID, err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, api.Error{Message: "Failed to validate theme"})
	}

	// 2. Validate data against theme fields using validation package
	domainFields := th.Fields // Assuming th.Fields are of type []theme.ThemeField
	if err := validation.ValidateEntryDataAgainstTheme(req.Data, domainFields); err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, api.Error{Message: fmt.Sprintf("Entry data validation failed: %v", err)})
	}

	// 3. Create domain entry object
	newEntry := entry.Entry{
		EntryID:   uuid.New(), // Generate new ID
		ThemeID:   req.ThemeId,
		UserID:    userID,
		EntryDate: req.EntryDate.Format("2006-01-02"), // Store as YYYY-MM-DD string
		Data:      req.Data,
		// CreatedAt, UpdatedAt, PK, SK, GSI keys set by repository
	}

	// 4. Call repository to create entry
	if err := uc.entryRepo.CreateEntry(ctx, &newEntry); err != nil {
		// Handle potential conditional check failure (already exists) if needed
		if strings.Contains(err.Error(), "ConditionalCheckFailed") {
			log.Printf("ConditionalCheckFailed when creating entry for user %s, theme %s, date %s: %v", userID, req.ThemeId, req.EntryDate.Format("2006-01-02"), err)
			return nil, echo.NewHTTPError(http.StatusConflict, api.Error{Message: "Entry potentially already exists"})
		}
		log.Printf("Error creating entry in repository for user %s: %v", userID, err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, api.Error{Message: "Failed to create entry"})
	}

	// 5. Fetch the created entry to return the full object (optional but good practice)
	createdEntry, err := uc.entryRepo.GetEntryByID(ctx, userID, newEntry.EntryID)
	if err != nil {
		// Log the inconsistency, but return the data we have as a fallback
		log.Printf("WARN: Failed to fetch newly created entry %s for user %s: %v", newEntry.EntryID, userID, err)
		// Approximate timestamps and return
		now := time.Now()
		newEntry.CreatedAt = now
		newEntry.UpdatedAt = now
		apiEntry := entry.ToApiEntry(newEntry) // Assuming ToApiEntry exists
		return &apiEntry, nil                  // Return success even if fetch failed, but with approximated data
	}

	// 6. Convert to API model and return
	apiEntry := entry.ToApiEntry(*createdEntry)
	return &apiEntry, nil
}
