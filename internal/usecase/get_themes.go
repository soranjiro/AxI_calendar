package usecase

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/soranjiro/axicalendar/internal/domain"
	"github.com/soranjiro/axicalendar/internal/domain/theme"
	"github.com/soranjiro/axicalendar/internal/presentation/api"
)

// GetThemes handles the logic for getting all themes accessible by the user.
// Returns domain themes.
func (uc *UseCase) GetThemes(ctx context.Context, userID uuid.UUID) ([]theme.Theme, error) {
	themes, err := uc.themeRepo.ListThemes(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) || errors.Is(err, domain.ErrForbidden) { // Use domain errors
			// Treat forbidden as not found from the user's perspective for GET
			return nil, echo.NewHTTPError(http.StatusNotFound, api.Error{Message: "Themes not found or access denied"})
		}
		// Log internal error if needed
		return nil, echo.NewHTTPError(http.StatusInternalServerError, api.Error{Message: "Failed to retrieve themes"})
	}

	// Return domain model directly
	return themes, nil
}
