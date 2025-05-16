package usecase

import (
	"context"
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
	if endDate.Before(startDate) {
		return nil, echo.NewHTTPError(http.StatusBadRequest, api.Error{Message: "end_date cannot be before start_date"})
	}

	// Call repository with time.Time dates and themeID as a slice
	entries, err := uc.entryRepo.ListEntriesByDateRange(ctx, userID, startDate, endDate, themeID)
	if err != nil {
		// Log the internal error if needed
		// log.Printf("Error fetching entries from repository: %v", err)
		// Return a generic error to the handler
		return nil, echo.NewHTTPError(http.StatusInternalServerError, api.Error{Message: err.Error()})
	}

	// Return domain models directly
	return entries, nil
}
