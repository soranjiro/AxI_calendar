package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/soranjiro/axicalendar/internal/domain/feature"
)

// ExecuteFeature retrieves an executor from the registry and runs it.
// It fetches entries for the specified user and theme.
func (uc *UseCase) ExecuteFeature(ctx context.Context, userID uuid.UUID, themeID uuid.UUID, featureName string) (feature.AnalysisResult, error) {
	// Get the theme to check if the feature is supported
	theme, err := uc.GetThemeByID(ctx, userID, themeID)
	if err != nil {
		// Consider returning a more specific error, e.g., not found or forbidden
		return nil, fmt.Errorf("failed to get theme %s: %w", themeID, err)
	}

	isSupported := false
	for _, sf := range theme.SupportedFeatures { // Check if the feature is supported by the theme
		if sf == featureName {
			isSupported = true
			break
		}
	}

	if !isSupported {
		return nil, fmt.Errorf("feature '%s' is not supported by theme '%s'", featureName, themeID)
	}

	// Get the feature executor from the registry
	executor, err := uc.featureRegistry.GetExecutor(featureName)
	if err != nil {
		return nil, fmt.Errorf("failed to get executor for feature '%s': %w", featureName, err)
	}
	
	startDate := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC) // Far past
	endDate := time.Now().AddDate(100, 0, 0)                 // Far future

	// Use the GetEntries method from the use case, which should delegate to the entry repository.
	// This assumes GetEntries can filter by userID and themeID.
	entries, err := uc.GetEntries(ctx, userID, themeID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch entries for feature '%s' (theme: %s): %w", featureName, themeID, err)
	}

	// Execute the feature with the fetched entries
	result, err := executor.Execute(ctx, entries)
	if err != nil {
		return nil, fmt.Errorf("failed to execute feature '%s': %w", featureName, err)
	}

	return result, nil
}
