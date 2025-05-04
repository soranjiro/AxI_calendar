package usecase

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/soranjiro/axicalendar/internal/api"
	"github.com/soranjiro/axicalendar/internal/domain"
)

// DeleteTheme handles the logic for deleting a theme.
func (uc *UseCase) DeleteTheme(ctx context.Context, userID uuid.UUID, themeID uuid.UUID) error {
	// Repository's DeleteTheme already checks ownership and if it's default
	err := uc.themeRepo.DeleteTheme(ctx, userID, themeID)
	if err != nil {
		if errors.Is(err, domain.ErrThemeNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, api.Error{Message: "Theme not found"})
		}
		if errors.Is(err, domain.ErrForbidden) || errors.Is(err, domain.ErrCannotDeleteDefaultTheme) {
			// Combine forbidden/cannot delete default into a single 403 for the API
			return echo.NewHTTPError(http.StatusForbidden, api.Error{Message: "Cannot delete this theme (not owner or is default)"})
		}
		// Log internal error if needed
		return echo.NewHTTPError(http.StatusInternalServerError, api.Error{Message: "Failed to delete theme"})
	}

	return nil // Success indicates no content (204)
}
