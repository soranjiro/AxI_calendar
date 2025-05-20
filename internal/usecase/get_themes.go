package usecase

import (
	"context"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/soranjiro/axicalendar/internal/domain/theme"
	"github.com/soranjiro/axicalendar/internal/presentation/api"
)

// GetThemes handles the logic for getting all themes accessible by the user.
// Returns domain themes.
func (uc *UseCase) GetThemes(ctx context.Context, userID uuid.UUID) ([]theme.Theme, error) {
	themes, err := uc.themeRepo.ListThemes(ctx, userID)
	if err != nil {
		// Log internal error if needed
		log.Printf("Error fetching themes from repository: %v", err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, api.Error{Message: "Failed to retrieve themes"})
	}

	// Return domain models directly
	return themes, nil
}
