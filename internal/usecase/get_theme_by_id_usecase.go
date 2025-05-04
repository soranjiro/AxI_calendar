package usecase

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/soranjiro/axicalendar/internal/api" // Keep for error response
	"github.com/soranjiro/axicalendar/internal/domain"
	"github.com/soranjiro/axicalendar/internal/domain/theme"
)

// GetThemeByID handles the logic for getting a single theme by its ID.
// Returns a domain theme.
func (uc *UseCase) GetThemeByID(ctx context.Context, userID uuid.UUID, themeID uuid.UUID) (*theme.Theme, error) {
	th, err := uc.themeRepo.GetThemeByID(ctx, userID, themeID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) || errors.Is(err, domain.ErrForbidden) { // Use domain errors
			// Treat forbidden as not found from the user's perspective for GET
			return nil, echo.NewHTTPError(http.StatusNotFound, api.Error{Message: "Theme not found or access denied"})
		}
		// Log internal error if needed
		return nil, echo.NewHTTPError(http.StatusInternalServerError, api.Error{Message: "Failed to retrieve theme"})
	}

	// Return domain model directly
	return th, nil
}
