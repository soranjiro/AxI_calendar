package usecase

import (
	"context"
	"github.com/google/uuid"
	"github.com/soranjiro/axicalendar/internal/api"
	dynamodbrepo "github.com/soranjiro/axicalendar/internal/repository/dynamodb"
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
	themeRepo dynamodbrepo.ThemeRepository
	entryRepo dynamodbrepo.EntryRepository
	// Add other repositories or services as needed
}

// NewUseCase creates a new UseCase with dependencies.
func NewUseCase(themeRepo dynamodbrepo.ThemeRepository, entryRepo dynamodbrepo.EntryRepository) UseCaseInterface {
	return &UseCase{
		themeRepo: themeRepo,
		entryRepo: entryRepo,
	}
}
