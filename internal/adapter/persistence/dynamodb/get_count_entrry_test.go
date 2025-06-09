package usecase

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/soranjiro/axicalendar/internal/domain/entry"
	"github.com/soranjiro/axicalendar/internal/domain/theme"
)

// mockThemeRepo is a simple mock for testing
type mockThemeRepo struct {
	themes map[string]*theme.Theme
}

func (m *mockThemeRepo) GetThemeByID(ctx context.Context, userID uuid.UUID, themeID uuid.UUID) (*theme.Theme, error) {
	if theme, exists := m.themes[themeID.String()]; exists {
		return theme, nil
	}
	return nil, fmt.Errorf("theme not found")
}

func (m *mockThemeRepo) CreateTheme(ctx context.Context, theme *theme.Theme) error {
	m.themes[theme.ThemeID.String()] = theme
	return nil
}

func (m *mockThemeRepo) ListThemes(ctx context.Context, userID uuid.UUID) ([]theme.Theme, error) {
	var result []theme.Theme
	for _, th := range m.themes {
		result = append(result, *th)
	}
	return result, nil
}

func (m *mockThemeRepo) UpdateTheme(ctx context.Context, theme *theme.Theme) error {
	m.themes[theme.ThemeID.String()] = theme
	return nil
}

func (m *mockThemeRepo) DeleteTheme(ctx context.Context, userID uuid.UUID, themeID uuid.UUID) error {
	delete(m.themes, themeID.String())
	return nil
}

func (m *mockThemeRepo) AddUserThemeLink(ctx context.Context, link *theme.UserThemeLink) error {
	// Mock implementation - not needed for count tests
	return nil
}

func (m *mockThemeRepo) RemoveUserThemeLink(ctx context.Context, userID, themeID uuid.UUID) error {
	// Mock implementation - not needed for count tests
	return nil
}

func (m *mockThemeRepo) ListUserThemes(ctx context.Context, userID uuid.UUID) ([]theme.UserThemeLink, error) {
	// Mock implementation - not needed for count tests
	return nil, nil
}

// mockEntryRepo is a simple mock for testing
type mockEntryRepo struct {
	entries []entry.Entry
}

func (m *mockEntryRepo) GetEntryByID(ctx context.Context, userID uuid.UUID, entryID uuid.UUID) (*entry.Entry, error) {
	for _, e := range m.entries {
		if e.EntryID == entryID && e.UserID == userID {
			return &e, nil
		}
	}
	return nil, fmt.Errorf("entry not found")
}

func (m *mockEntryRepo) CreateEntry(ctx context.Context, entry *entry.Entry) error {
	m.entries = append(m.entries, *entry)
	return nil
}

func (m *mockEntryRepo) ListEntriesByDateRange(ctx context.Context, userID uuid.UUID, startDate time.Time, endDate time.Time, themeID uuid.UUID) ([]entry.Entry, error) {
	var result []entry.Entry
	for _, e := range m.entries {
		if e.UserID == userID && e.ThemeID == themeID {
			// Simple date filtering (assuming EntryDate is in YYYY-MM-DD format)
			entryDate, _ := time.Parse("2006-01-02", e.EntryDate)
			if (entryDate.Equal(startDate) || entryDate.After(startDate)) &&
				(entryDate.Equal(endDate) || entryDate.Before(endDate)) {
				result = append(result, e)
			}
		}
	}
	return result, nil
}

func (m *mockEntryRepo) UpdateEntry(ctx context.Context, entry *entry.Entry) error {
	for i, e := range m.entries {
		if e.EntryID == entry.EntryID {
			m.entries[i] = *entry
			return nil
		}
	}
	return fmt.Errorf("entry not found")
}

func (m *mockEntryRepo) DeleteEntry(ctx context.Context, userID uuid.UUID, entryID uuid.UUID, entryDate string) error {
	for i, e := range m.entries {
		if e.EntryID == entryID && e.UserID == userID {
			m.entries = append(m.entries[:i], m.entries[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("entry not found")
}

func (m *mockEntryRepo) GetEntriesForSummary(ctx context.Context, userID uuid.UUID, themeID uuid.UUID, yearMonth string) ([]entry.Entry, error) {
	// Mock implementation - not needed for count tests
	return nil, nil
}

func TestUseCase_CountThemeEntries(t *testing.T) {
	// Setup
	userID := uuid.New()
	themeID := uuid.New()

	// Create mock repositories
	mockThemes := &mockThemeRepo{
		themes: make(map[string]*theme.Theme),
	}
	mockEntries := &mockEntryRepo{
		entries: []entry.Entry{},
	}

	// Create test theme with summation feature
	testTheme := &theme.Theme{
		ThemeID:           themeID,
		ThemeName:         "Budget Theme",
		SupportedFeatures: []string{"summation"},
		Fields: []theme.ThemeField{
			{Name: "amount", Label: "Amount", Type: theme.FieldTypeNumber, Required: true},
		},
	}
	mockThemes.themes[themeID.String()] = testTheme

	// Create test entries
	testEntries := []entry.Entry{
		{
			EntryID:   uuid.New(),
			ThemeID:   themeID,
			UserID:    userID,
			EntryDate: "2025-05-01",
			Data: map[string]interface{}{
				"amount": 100.0,
			},
		},
		{
			EntryID:   uuid.New(),
			ThemeID:   themeID,
			UserID:    userID,
			EntryDate: "2025-05-02",
			Data: map[string]interface{}{
				"amount": 200.5,
			},
		},
		{
			EntryID:   uuid.New(),
			ThemeID:   themeID,
			UserID:    userID,
			EntryDate: "2025-05-03",
			Data: map[string]interface{}{
				"amount": -50.0,
			},
		},
	}
	mockEntries.entries = testEntries

	// Create UseCase
	uc := NewUseCase(mockThemes, mockEntries)

	// Test date range
	startDate := time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2025, 5, 31, 0, 0, 0, 0, time.UTC)

	// Execute
	result, err := uc.CountThemeEntries(context.Background(), userID, themeID, startDate, endDate)

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expected := 100.0 + 200.5 + (-50.0) // = 250.5
	if result != expected {
		t.Errorf("Expected count to be %f, got %f", expected, result)
	}
}
