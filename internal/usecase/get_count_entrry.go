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
	"github.com/soranjiro/axicalendar/internal/usecase/features"
)

func (uc *UseCase) CountThemeEntries(ctx context.Context, userID uuid.UUID, themeID uuid.UUID, startDate time.Time, endDate time.Time) (float64, error) {
	log.Printf("UseCase: CountThemeEntries called for UserID: %s, ThemeID: %s, StartDate: %s, EndDate: %s", userID, themeID, startDate, endDate)

	// Get Theme information
	th, err := uc.themeRepo.GetThemeByID(ctx, userID, themeID)
	if err != nil {
		log.Printf("ERROR: Failed to get theme %s for user %s: %v", themeID, userID, err)
		var httpErr *echo.HTTPError
		if errors.As(err, &httpErr) {
			return 0, err // Return the original HTTPError
		}
		return 0, fmt.Errorf("failed to retrieve theme: %w", err)
	}

	// Get entries for the specified theme and date range
	entriesList, err := uc.entryRepo.ListEntriesByDateRange(ctx, userID, startDate, endDate, themeID)
	if err != nil {
		log.Printf("ERROR: Failed to get entries for theme %s, user %s: %v", themeID, userID, err)
		var httpErr *echo.HTTPError
		if errors.As(err, &httpErr) {
			return 0, err // Return the original HTTPError
		}
		return 0, fmt.Errorf("failed to retrieve entries: %w", err)
	}

	// Convert []entry.Entry to entry.Entries for processing
	entries := entry.Entries(entriesList)
	log.Printf("UseCase: Retrieved %d entries for processing", len(entries))

	// Create Features instance and process entries based on supported_features
	featuresProcessor := features.NewFeatures()
	count, err := featuresProcessor.Count(ctx, *th, entries)
	if err != nil {
		log.Printf("ERROR: Failed to process features for theme %s: %v", themeID, err)
		return 0, fmt.Errorf("failed to process features: %w", err)
	}

	log.Printf("UseCase: Count result for theme %s: %f", themeID, count)
	return count, nil
}
