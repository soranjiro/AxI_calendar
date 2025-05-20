package usecase

import (
	"context"
	"errors" // Added import for errors package
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/soranjiro/axicalendar/internal/domain/entry"
)

func (uc *UseCase) CountThemeEntries(ctx context.Context, userID uuid.UUID, themeID uuid.UUID, startDate time.Time, endDate time.Time) (float64, error) {
	log.Printf("UseCase: CountThemeEntries called for UserID: %s, ThemeID: %s, StartDate: %s, EndDate: %s", userID, themeID, startDate, endDate)

	// Get Theme information
	th, err := uc.themeService.GetThemeByID(ctx, userID, themeID)
	if err != nil {
		log.Printf("ERROR: Failed to get theme %s for user %s: %v", themeID, userID, err)
		var httpErr *echo.HTTPError
		if errors.As(err, &httpErr) {
			return 0, err // Return the original HTTPError
		}
		return 0, fmt.Errorf("failed to retrieve theme: %w", err)
	}

	// Check if "SumAll" feature is supported
	isSumAllSupported := false
	for _, feature := range th.SupportedFeatures {
		if feature == "SumAll" { // Assuming "SumAll" is the feature identifier
			isSumAllSupported = true
			break
		}
	}

	if !isSumAllSupported {
		log.Printf("ERROR: Theme %s does not support SumAll feature for UserID: %s", themeID, userID)
		return 0, fmt.Errorf("theme %s does not support SumAll feature", themeID)
	}

	rawEntries, err := uc.entryService.GetEntries(ctx, userID, themeID, startDate, endDate)
	if err != nil {
		log.Printf("ERROR: Failed to get entries for user %s, theme %s: %v", userID, themeID, err)
		var httpErr *echo.HTTPError
		if errors.As(err, &httpErr) {
			return 0, err
		}
		return 0, fmt.Errorf("failed to retrieve entries for counting: %w", err)
	}

	entries := entry.Entries(rawEntries)

	sum, err := entries.SumAll()
	if err != nil {
		log.Printf("ERROR: Failed to sum entries for UserID: %s, ThemeID: %s: %v", userID, themeID, err)
		return 0, fmt.Errorf("failed to sum entries: %w", err)
	}

	return sum, nil
}
