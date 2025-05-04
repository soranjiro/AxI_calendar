package usecase

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/soranjiro/axicalendar/internal/api"
	"github.com/soranjiro/axicalendar/internal/domain/theme"
)

// GetThemes handles the logic for getting all themes accessible by the user.
func (uc *UseCase) GetThemes(ctx context.Context, userID uuid.UUID) ([]api.Theme, error) {
	themes, err := uc.themeRepo.ListThemes(ctx, userID)
	if err != nil {
		// Log internal error if needed
		return nil, echo.NewHTTPError(http.StatusInternalServerError, api.Error{Message: "Failed to retrieve themes"})
	}

	apiThemes := make([]api.Theme, len(themes))
	for i, th := range themes {
		apiThemes[i] = theme.ToApiTheme(th)
	}

	return apiThemes, nil
}
