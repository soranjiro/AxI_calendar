package usecase

import (
	"context"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	dynamodbrepo "github.com/soranjiro/axicalendar/internal/adapter/persistence/dynamodb"
	"github.com/soranjiro/axicalendar/internal/domain/entry"
	"github.com/soranjiro/axicalendar/internal/domain/theme"
	"github.com/soranjiro/axicalendar/internal/domain/user"
	"github.com/soranjiro/axicalendar/internal/presentation/api"
)

// UseCaseInterface defines the methods for all use cases.
type UseCaseInterface interface {
	// Auth
	GetAuthMe(ctx context.Context, userID uuid.UUID) (*user.User, error)

	// Entries
	// Accepts domain entry, returns domain entry
	CreateEntry(ctx context.Context, newEntry entry.Entry) (*entry.Entry, error)
	// Accepts API params for now, returns domain entries
	GetEntries(ctx context.Context, userID uuid.UUID, params api.GetEntriesParams) ([]entry.Entry, error)
	// Accepts IDs, returns domain entry
	GetEntryByID(ctx context.Context, userID uuid.UUID, entryID uuid.UUID) (*entry.Entry, error)
	// Accepts IDs and API request (needs internal conversion), returns domain entry
	UpdateEntry(ctx context.Context, userID uuid.UUID, entryID uuid.UUID, req api.UpdateEntryRequest) (*entry.Entry, error)
	// Accepts IDs
	DeleteEntry(ctx context.Context, userID uuid.UUID, entryID uuid.UUID) error

	// Themes
	// Accepts domain theme, returns domain theme
	CreateTheme(ctx context.Context, newTheme theme.Theme) (*theme.Theme, error)
	// Accepts ID, returns domain themes
	GetThemes(ctx context.Context, userID uuid.UUID) ([]theme.Theme, error)
	// Accepts IDs, returns domain theme
	GetThemeByID(ctx context.Context, userID uuid.UUID, themeID uuid.UUID) (*theme.Theme, error)
	// Accepts IDs and domain theme, returns domain theme
	UpdateTheme(ctx context.Context, userID uuid.UUID, themeID uuid.UUID, updatedThemeData theme.Theme) (*theme.Theme, error)
	// Accepts IDs
	DeleteTheme(ctx context.Context, userID uuid.UUID, themeID uuid.UUID) error

	// GetEntryCountForTheme returns the number of entries for a specific theme within a date range.
	GetEntryCountForTheme(ctx context.Context, userID uuid.UUID, themeID uuid.UUID, startDate, endDate openapi_types.Date) (int64, error)
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
