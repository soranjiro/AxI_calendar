package services

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/soranjiro/axicalendar/internal/domain"
	"github.com/soranjiro/axicalendar/internal/domain/theme"
	dynamodbrepo "github.com/soranjiro/axicalendar/internal/adapter/persistence/dynamodb"
	"github.com/soranjiro/axicalendar/internal/presentation/api"
)

type ThemeService struct {
	themeRepo dynamodbrepo.ThemeRepository
}

func NewThemeService(themeRepo dynamodbrepo.ThemeRepository) *ThemeService {
	return &ThemeService{themeRepo: themeRepo}
}

func (s *ThemeService) GetThemeByID(ctx context.Context, userID uuid.UUID, themeID uuid.UUID) (*theme.Theme, error) {
	th, err := s.themeRepo.GetThemeByID(ctx, userID, themeID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) || errors.Is(err, domain.ErrForbidden) {
			return nil, echo.NewHTTPError(http.StatusNotFound, api.Error{Message: "Theme not found or access denied"})
		}
		log.Printf("Error fetching theme from repository: %v", err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, api.Error{Message: "Failed to retrieve theme"})
	}
	return th, nil
}

func (s *ThemeService) GetThemes(ctx context.Context, userID uuid.UUID) ([]theme.Theme, error) {
	themes, err := s.themeRepo.ListThemes(ctx, userID)
	if err != nil {
		log.Printf("Error fetching themes from repository: %v", err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, api.Error{Message: "Failed to retrieve themes"})
	}
	return themes, nil
}
