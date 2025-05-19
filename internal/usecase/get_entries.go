package usecase

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/soranjiro/axicalendar/internal/domain/entry"
	"github.com/soranjiro/axicalendar/internal/presentation/api" // api.Errorのため
)

// GetEntries handles the logic for getting entries.
// Returns domain entries.
func (uc *UseCase) GetEntries(ctx context.Context, userID uuid.UUID, themeID uuid.UUID, startDate time.Time, endDate time.Time) ([]entry.Entry, error) {
	// Basic date validation
	if startDate.IsZero() || endDate.IsZero() {
		return nil, echo.NewHTTPError(http.StatusBadRequest, api.Error{Message: "start_date and end_date cannot be zero"})
	}
	if endDate.Before(startDate) {
		return nil, echo.NewHTTPError(http.StatusBadRequest, api.Error{Message: "end_date cannot be before start_date"})
	}

	// Call the EntryService
	entries, err := uc.entryService.GetEntries(ctx, userID, themeID, startDate, endDate)
	if err != nil {
		// Log the internal error if needed
		// log.Printf("Error fetching entries from repository: %v", err)
		// Return a generic error to the handler
		log.Printf("Error fetching entries from service: %v", err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, api.Error{Message: "Failed to retrieve entries"})
	}

	// Return domain models directly
	return entries, nil
}
