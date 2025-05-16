package usecase

import (
	"context"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// GetEntryCountForTheme returns the number of entries for a specific theme within a date range.
func (uc *UseCase) GetEntryCountForTheme(ctx context.Context, userID uuid.UUID, themeID uuid.UUID, startDate openapi_types.Date, endDate openapi_types.Date) (int64, error) {
	// Validate that the theme exists and belongs to the user or is a default theme
	_, err := uc.GetThemeByID(ctx, userID, themeID) // This already checks ownership/default status
	if err != nil {
		return 0, err // Error from GetThemeByID (e.g., NotFound, Unauthorized)
	}

	// Call the repository method to count entries
	count, err := uc.entryRepo.CountEntriesByThemeAndDateRange(ctx, userID, themeID, startDate.Time, endDate.Time)
	if err != nil {
		// Handle specific errors from repository if needed, or return a generic server error
		return 0, err // Assuming repository returns errors that can be directly passed or wrapped
	}

	return count, nil
}
