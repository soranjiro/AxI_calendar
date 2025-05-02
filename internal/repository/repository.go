package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/soranjiro/axicalendar/internal/models"
)

// EntryRepository defines the interface for calendar entry data operations.
type EntryRepository interface {
	GetEntryByID(ctx context.Context, userID uuid.UUID, entryID uuid.UUID) (*models.Entry, error)
	ListEntriesByDateRange(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time, themeIDs []uuid.UUID) ([]models.Entry, error)
	CreateEntry(ctx context.Context, entry *models.Entry) error
	UpdateEntry(ctx context.Context, entry *models.Entry) error
	DeleteEntry(ctx context.Context, userID uuid.UUID, entryID uuid.UUID, entryDate string) error
}

// ThemeRepository defines the interface for theme data operations.
type ThemeRepository interface {
	GetThemeByID(ctx context.Context, userID uuid.UUID, themeID uuid.UUID) (*models.Theme, error)
	ListThemes(ctx context.Context, userID uuid.UUID) ([]models.Theme, error)
	CreateTheme(ctx context.Context, theme *models.Theme) error
	UpdateTheme(ctx context.Context, theme *models.Theme) error
	DeleteTheme(ctx context.Context, userID uuid.UUID, themeID uuid.UUID) error
}

// --- Helper Functions for Key Generation ---

// userPK generates the PK for a user's items.
// PK: USER#<user_id>
func userPK(userID string) string {
	return "USER#" + userID
}

// --- Entry Key Functions ---

// entrySK generates the SK for an entry item.
// SK: ENTRY#<date>#<entry_id>
func entrySK(date string, entryID string) string {
	return "ENTRY#" + date + "#" + entryID
}

// entryDateSKPrefix generates the prefix for date-based SK queries on GSI1.
// GSI1 SK prefix: ENTRY_DATE#<date>
func entryDateSKPrefix(date string) string {
	return "ENTRY_DATE#" + date
}

// userGSI1PK generates the GSI1PK for an entry item.
// GSI1PK: USER#<user_id>
func userGSI1PK(userID string) string {
	return userPK(userID) // GSI1 uses UserID as PK
}

// entryGSI1SK generates the GSI1SK for an entry item.
// GSI1SK: ENTRY_DATE#<date>#<theme_id>
func entryGSI1SK(date string, themeID string) string {
	return entryDateSKPrefix(date) + "#" + themeID
}

// --- Theme Key Functions ---

// themeSK generates the SK for a theme item.
// SK: THEME#<theme_id>
func themeSK(themeID string) string {
	return "THEME#" + themeID
}

// --- Entry Repository Implementation ---
// Duplicate implementation removed; see entry_repository.go for actual methods
