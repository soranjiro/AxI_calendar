package handler

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/soranjiro/axicalendar/internal/domain/entry"
	"github.com/soranjiro/axicalendar/internal/domain/feature"
	"github.com/soranjiro/axicalendar/internal/domain/theme"
	"github.com/soranjiro/axicalendar/internal/domain/user"
)

type UseCase interface {
	// Auth
	GetAuthMe(ctx context.Context, userID uuid.UUID) (*user.User, error)

	// Entries
	// Accepts domain entry, returns domain entry
	CreateEntry(ctx context.Context, newEntry entry.Entry) (*entry.Entry, error)
	// Accepts IDs and date range, returns domain entries
	GetEntries(ctx context.Context, userID uuid.UUID, themeID uuid.UUID, startDate time.Time, endDate time.Time) ([]entry.Entry, error)
	// Accepts IDs, returns domain entry
	GetEntryByID(ctx context.Context, userID uuid.UUID, entryID uuid.UUID) (*entry.Entry, error)
	// Accepts IDs and domain entry, returns domain entry
	UpdateEntry(ctx context.Context, userID uuid.UUID, entryID uuid.UUID, updatedEntry entry.Entry) (*entry.Entry, error)
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

	// Features
	// Executes a registered feature and returns its analysis result
	ExecuteFeature(ctx context.Context, userID uuid.UUID, themeID uuid.UUID, featureName string) (feature.AnalysisResult, error)
}
