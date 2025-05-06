package usecase

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/soranjiro/axicalendar/internal/interfaces/api"
	"github.com/soranjiro/axicalendar/internal/domain/entry"
)

// GetEntries handles the logic for getting entries.
// Returns domain entries.
func (uc *UseCase) GetEntries(ctx context.Context, userID uuid.UUID, params api.GetEntriesParams) ([]entry.Entry, error) {
	// Parse dates (YYYY-MM-DD string format expected by repository)
	startDateStr := params.StartDate.Format("2006-01-02")
	endDateStr := params.EndDate.Format("2006-01-02")

	// Basic date validation (optional, repo might handle it)
	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, api.Error{Message: "Invalid start_date format"})
	}
	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, api.Error{Message: "Invalid end_date format"})
	}
	if endDate.Before(startDate) {
		return nil, echo.NewHTTPError(http.StatusBadRequest, api.Error{Message: "end_date cannot be before start_date"})
	}

	// Parse theme IDs
	var themeIDs []uuid.UUID
	if params.ThemeIds != nil && *params.ThemeIds != "" {
		themeIDStrs := strings.Split(*params.ThemeIds, ",")
		for _, idStr := range themeIDStrs {
			id, err := uuid.Parse(strings.TrimSpace(idStr))
			if err != nil {
				return nil, echo.NewHTTPError(http.StatusBadRequest, api.Error{Message: fmt.Sprintf("Invalid theme_id format: %s", idStr)})
			}
			themeIDs = append(themeIDs, id)
		}
	}

	// Call repository with time.Time dates
	entries, err := uc.entryRepo.ListEntriesByDateRange(ctx, userID, startDate, endDate, themeIDs)
	if err != nil {
		// Log the internal error if needed
		// log.Printf("Error fetching entries from repository: %v", err)
		// Return a generic error to the handler
		return nil, echo.NewHTTPError(http.StatusInternalServerError, api.Error{Message: "Failed to retrieve entries"})
	}

	// Return domain models directly
	return entries, nil
}
