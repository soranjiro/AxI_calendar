package usecase

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/soranjiro/axicalendar/internal/api"
	"github.com/soranjiro/axicalendar/internal/domain"
	"github.com/soranjiro/axicalendar/internal/domain/theme"
	repo "github.com/soranjiro/axicalendar/internal/repository/dynamodb"
)

// GetThemeByIDUseCase defines the interface for the get theme by ID use case.
type GetThemeByIDUseCase interface {
	Execute(ctx context.Context, userID uuid.UUID, themeID uuid.UUID) (*api.Theme, error)
}

// getThemeByIDUseCase implements the GetThemeByIDUseCase interface.
type getThemeByIDUseCase struct {
	themeRepo repo.ThemeRepository
}

// NewGetThemeByIDUseCase creates a new GetThemeByIDUseCase.
func NewGetThemeByIDUseCase(themeRepo repo.ThemeRepository) GetThemeByIDUseCase {
	return &getThemeByIDUseCase{themeRepo: themeRepo}
}

// Execute handles the logic for getting a single theme by its ID.
func (uc *getThemeByIDUseCase) Execute(ctx context.Context, userID uuid.UUID, themeID uuid.UUID) (*api.Theme, error) {
	th, err := uc.themeRepo.GetThemeByID(ctx, userID, themeID)
	if err != nil {
		if errors.Is(err, domain.ErrThemeNotFound) || errors.Is(err, domain.ErrForbidden) {
			// Treat forbidden as not found from the user's perspective for GET
			return nil, echo.NewHTTPError(http.StatusNotFound, api.Error{Message: "Theme not found or access denied"})
		}
		// Log internal error if needed
		return nil, echo.NewHTTPError(http.StatusInternalServerError, api.Error{Message: "Failed to retrieve theme"})
	}

	apiTheme := theme.ToApiTheme(*th)
	return &apiTheme, nil
}
