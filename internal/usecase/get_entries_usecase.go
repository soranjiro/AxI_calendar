package usecase

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/soranjiro/axicalendar/internal/api"
	"github.com/soranjiro/axicalendar/internal/domain/entry"
)

// GetEntries handles the logic for getting entries.
func (uc *UseCase) GetEntries(ctx context.Context, userID uuid.UUID, params api.GetEntriesParams) ([]api.Entry, error) {
	// Parse dates
	startDate := params.StartDate.Time
	endDate := params.EndDate.Time

	if endDate.Before(startDate) {
		// Use a domain-specific error or a standard error type
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

	// Call repository
	entries, err := uc.entryRepo.ListEntriesByDateRange(ctx, userID, startDate, endDate, themeIDs)
	if err != nil {
		// Log the internal error if needed
		// log.Printf("Error fetching entries from repository: %v", err)
		// Return a generic error to the handler
		return nil, echo.NewHTTPError(http.StatusInternalServerError, api.Error{Message: "Failed to retrieve entries"})
	}

	// Convert domain models to API models
	apiEntries := make([]api.Entry, len(entries))
	for i, e := range entries {
		apiEntries[i] = entry.ToApiEntry(e) // Assuming ToApiEntry exists in domain/entry
	}

	return apiEntries, nil
}
