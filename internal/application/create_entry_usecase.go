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
	"github.com/soranjiro/axicalendar/internal/interfaces/api"
	"github.com/soranjiro/axicalendar/internal/domain"
	"github.com/soranjiro/axicalendar/internal/domain/entry"
)

// CreateEntry handles the logic for creating an entry.
// Accepts domain entry, returns domain entry
func (uc *UseCase) CreateEntry(ctx context.Context, newEntry entry.Entry) (*entry.Entry, error) {
	// 1. Validate theme exists and user has access
	// Extract userID and themeID from the domain entry
	userID := newEntry.UserID
	themeID := newEntry.ThemeID

	th, err := uc.themeRepo.GetThemeByID(ctx, userID, themeID)
	if err != nil {
		if errors.Is(err, domain.ErrThemeNotFound) || errors.Is(err, domain.ErrForbidden) {
			return nil, echo.NewHTTPError(http.StatusNotFound, api.Error{Message: "Theme not found or access denied"})
		}
		log.Printf("Error validating theme %s for user %s: %v", themeID, userID, err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, api.Error{Message: "Failed to validate theme"})
	}

	// 2. Validate data against theme fields using domain method
	if err := newEntry.ValidateDataAgainstTheme(th.Fields); err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, api.Error{Message: fmt.Sprintf("Entry data validation failed: %v", err)})
	}

	// 3. Domain entry object is already prepared (passed as argument)
	// Ensure EntryID is set (should be done by converter or here)
	if newEntry.EntryID == uuid.Nil {
		newEntry.EntryID = uuid.New()
	}

	// 4. Call repository to create entry
	if err := uc.entryRepo.CreateEntry(ctx, &newEntry); err != nil {
		// Handle potential conditional check failure (already exists) if needed
		if strings.Contains(err.Error(), "ConditionalCheckFailed") {
			log.Printf("ConditionalCheckFailed when creating entry for user %s, theme %s, date %s: %v", userID, themeID, newEntry.EntryDate, err)
			return nil, echo.NewHTTPError(http.StatusConflict, api.Error{Message: "Entry potentially already exists"})
		}
		log.Printf("Error creating entry in repository for user %s: %v", userID, err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, api.Error{Message: "Failed to create entry"})
	}

	// 5. Fetch the created entry to return the full object with timestamps
	createdEntry, err := uc.entryRepo.GetEntryByID(ctx, userID, newEntry.EntryID)
	if err != nil {
		// Log the inconsistency, but return the data we have as a fallback
		log.Printf("WARN: Failed to fetch newly created entry %s for user %s: %v", newEntry.EntryID, userID, err)
		// Approximate timestamps and return
		now := time.Now()
		newEntry.CreatedAt = now
		newEntry.UpdatedAt = now
		return &newEntry, nil // Return success even if fetch failed, but with approximated data
	}

	// 6. Return the fetched domain entry
	return createdEntry, nil
}
