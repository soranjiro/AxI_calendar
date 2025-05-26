package usecase

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/soranjiro/axicalendar/internal/presentation/api"
)

// GetEntriesCount handles the logic for getting the count of entries for a specific theme.
// Returns the count as int64.
func (uc *UseCase) GetEntriesCount(ctx context.Context, userID uuid.UUID, themeID uuid.UUID) (int64, error) {
	// Set a wide date range to get all entries for the theme
	// Using a practical range that covers all reasonable dates
	startDate := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2099, 12, 31, 23, 59, 59, 0, time.UTC)

	// Call the repository to get all entries for this theme and user
	entries, err := uc.entryRepo.ListEntriesByDateRange(ctx, userID, startDate, endDate, themeID)
	if err != nil {
		log.Printf("Error fetching entries count from repository: %v", err)
		return 0, echo.NewHTTPError(http.StatusInternalServerError, api.Error{Message: "Failed to retrieve entries count"})
	}

	// Return the count of entries
	return int64(len(entries)), nil
}
