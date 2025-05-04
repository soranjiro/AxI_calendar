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

// UseCaseInterface defines the methods for all use cases.
type UseCaseInterface interface {
	// Auth
	GetAuthMe(ctx context.Context, userID uuid.UUID) (*api.User, error)

	// Entries
	CreateEntry(ctx context.Context, userID uuid.UUID, req api.CreateEntryRequest) (*api.Entry, error)
	GetEntries(ctx context.Context, userID uuid.UUID, params api.GetEntriesParams) ([]api.Entry, error)
	GetEntryByID(ctx context.Context, userID uuid.UUID, entryID uuid.UUID) (*api.Entry, error)
	UpdateEntry(ctx context.Context, userID uuid.UUID, entryID uuid.UUID, req api.UpdateEntryRequest) (*api.Entry, error)
	DeleteEntry(ctx context.Context, userID uuid.UUID, entryID uuid.UUID) error

	// Themes
	CreateTheme(ctx context.Context, userID uuid.UUID, req api.CreateThemeRequest) (*api.Theme, error)
	GetThemes(ctx context.Context, userID uuid.UUID) ([]api.Theme, error)
	GetThemeByID(ctx context.Context, userID uuid.UUID, themeID uuid.UUID) (*api.Theme, error)
	UpdateTheme(ctx context.Context, userID uuid.UUID, themeID uuid.UUID, req api.UpdateThemeRequest) (*api.Theme, error)
	DeleteTheme(ctx context.Context, userID uuid.UUID, themeID uuid.UUID) error
}

// UseCase implements the UseCaseInterface.
type UseCase struct {
	themeRepo repo.ThemeRepository
	entryRepo repo.EntryRepository
	// Add other repositories or services as needed
}

// NewUseCase creates a new UseCase with dependencies.
func NewUseCase(themeRepo repo.ThemeRepository, entryRepo repo.EntryRepository) UseCaseInterface {
	return &UseCase{
		themeRepo: themeRepo,
		entryRepo: entryRepo,
	}
}

// GetThemeByID handles the logic for getting a single theme by its ID.
func (uc *UseCase) GetThemeByID(ctx context.Context, userID uuid.UUID, themeID uuid.UUID) (*api.Theme, error) {
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
