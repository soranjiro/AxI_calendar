package features

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/soranjiro/axicalendar/internal/domain/entry"
	"github.com/soranjiro/axicalendar/internal/domain/theme"
)

func TestFeatures_Count_Summation(t *testing.T) {
	// Setup
	features := NewFeatures()

	// Create a theme with summation feature
	testTheme := theme.Theme{
		ThemeID:           uuid.New(),
		ThemeName:         "Test Budget Theme",
		SupportedFeatures: []string{SummationFeature},
		Fields: []theme.ThemeField{
			{Name: "amount", Label: "Amount", Type: theme.FieldTypeNumber, Required: true},
		},
	}

	// Create test entries with amount data
	testEntries := entry.Entries{
		{
			EntryID: uuid.New(),
			ThemeID: testTheme.ThemeID,
			Data: map[string]interface{}{
				"amount": 100.5,
			},
		},
		{
			EntryID: uuid.New(),
			ThemeID: testTheme.ThemeID,
			Data: map[string]interface{}{
				"amount": 250.75,
			},
		},
		{
			EntryID: uuid.New(),
			ThemeID: testTheme.ThemeID,
			Data: map[string]interface{}{
				"amount": -50.25, // negative amount
			},
		},
	}

	// Execute
	result, err := features.Count(context.Background(), testTheme, testEntries)

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expected := 100.5 + 250.75 + (-50.25) // = 301.0
	if result != expected {
		t.Errorf("Expected count to be %f, got %f", expected, result)
	}
}

func TestFeatures_Count_NoSupportedFeatures(t *testing.T) {
	// Setup
	features := NewFeatures()

	// Create a theme without supported features
	testTheme := theme.Theme{
		ThemeID:           uuid.New(),
		ThemeName:         "Test Theme",
		SupportedFeatures: []string{}, // No supported features
	}

	testEntries := entry.Entries{
		{
			EntryID: uuid.New(),
			ThemeID: testTheme.ThemeID,
			Data: map[string]interface{}{
				"amount": 100.0,
			},
		},
	}

	// Execute
	result, err := features.Count(context.Background(), testTheme, testEntries)

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result != 0 {
		t.Errorf("Expected count to be 0 for theme without supported features, got %f", result)
	}
}

func TestFeatures_Count_DiscountSummation(t *testing.T) {
	// Setup
	features := NewFeatures()

	// Create a theme with discount summation feature (not implemented yet)
	testTheme := theme.Theme{
		ThemeID:           uuid.New(),
		ThemeName:         "Test Discount Theme",
		SupportedFeatures: []string{DiscountSummationFeature},
	}

	testEntries := entry.Entries{
		{
			EntryID: uuid.New(),
			ThemeID: testTheme.ThemeID,
			Data: map[string]interface{}{
				"amount": 100.0,
			},
		},
	}

	// Execute
	result, err := features.Count(context.Background(), testTheme, testEntries)

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Currently returns 0 as it's not implemented
	if result != 0 {
		t.Errorf("Expected count to be 0 for unimplemented feature, got %f", result)
	}
}
