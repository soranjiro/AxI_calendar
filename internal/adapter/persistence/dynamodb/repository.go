package dynamodbrepo

import (
	"context"
	"time"

	"github.com/soranjiro/axicalendar/internal/domain/entry"
	"github.com/soranjiro/axicalendar/internal/domain/theme"

	"github.com/google/uuid"
)

// EntryRepository defines the interface for entry data operations.
type EntryRepository interface {
	GetEntryByID(ctx context.Context, userID uuid.UUID, entryID uuid.UUID) (*entry.Entry, error)
	ListEntriesByDateRange(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time, themeID uuid.UUID) ([]entry.Entry, error)
	// GetEntriesForSummary retrieves entries for a specific user, theme, and year-month (YYYY-MM).
	GetEntriesForSummary(ctx context.Context, userID uuid.UUID, themeID uuid.UUID, yearMonth string) ([]entry.Entry, error)
	CreateEntry(ctx context.Context, entry *entry.Entry) error
	UpdateEntry(ctx context.Context, entry *entry.Entry) error
	// DeleteEntry requires entryDate because it's part of the SK.
	DeleteEntry(ctx context.Context, userID uuid.UUID, entryID uuid.UUID, entryDate string) error
}

// ThemeRepository defines the interface for theme data operations.
type ThemeRepository interface {
	GetThemeByID(ctx context.Context, userID uuid.UUID, themeID uuid.UUID) (*theme.Theme, error)
	ListThemes(ctx context.Context, userID uuid.UUID) ([]theme.Theme, error)
	CreateTheme(ctx context.Context, theme *theme.Theme) error
	UpdateTheme(ctx context.Context, theme *theme.Theme) error
	DeleteTheme(ctx context.Context, userID uuid.UUID, themeID uuid.UUID) error
	// AddUserThemeLink creates a link item allowing a user to access a theme.
	AddUserThemeLink(ctx context.Context, link *theme.UserThemeLink) error
	// RemoveUserThemeLink removes the link item, revoking user access to a theme.
	RemoveUserThemeLink(ctx context.Context, userID, themeID uuid.UUID) error
	// ListUserThemes retrieves the UserThemeLink items for a user.
	ListUserThemes(ctx context.Context, userID uuid.UUID) ([]theme.UserThemeLink, error)
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
// GSI1SK: ENTRY_DATE#<date>#<theme_id> (Corrected based on V1 design)
func entryGSI1SK(date string, themeID string) string {
	return entryDateSKPrefix(date) + "#" + themeID
}

// --- Theme Key Functions ---

// themePK generates the PK for a theme item.
// PK: THEME#<theme_id>
func themePK(themeID string) string {
	return "THEME#" + themeID
}

// themeMetadataSK generates the SK for a theme metadata item.
// SK: METADATA
func themeMetadataSK() string {
	return "METADATA"
}

// userThemeLinkSK generates the SK for a user-theme link item.
// SK: THEME#<theme_id>
func userThemeLinkSK(themeID string) string {
	return "THEME#" + themeID
}
