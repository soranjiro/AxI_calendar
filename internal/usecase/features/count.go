package features

import (
	"context"
	"log"

	"github.com/soranjiro/axicalendar/internal/domain/entry"
	"github.com/soranjiro/axicalendar/internal/domain/theme"
)

const (
	SummationFeature         = "summation"
	DiscountSummationFeature = "discount_summation"
)

// Features provides functionality for processing entries based on theme features
type Features struct{}

// NewFeatures creates a new Features instance
func NewFeatures() *Features {
	return &Features{}
}

// Count processes entries based on the theme's supported features and returns a count
func (f *Features) Count(ctx context.Context, theme theme.Theme, entries entry.Entries) (float64, error) {
	log.Printf("Features: Count called for theme %s with %d entries and features %v", theme.ThemeName, len(entries), theme.SupportedFeatures)

	// Check supported_features and process accordingly
	for _, feature := range theme.SupportedFeatures {
		switch feature {
		case SummationFeature:
			// Handle summation using the existing SumAll method
			sum, err := entries.SumAll()
			if err != nil {
				log.Printf("ERROR: Failed to calculate summation: %v", err)
				return 0, err
			}
			log.Printf("Features: Summation result: %f", sum)
			return sum, nil

		case DiscountSummationFeature:
			// Handle discount summation (placeholder for future implementation)
			log.Printf("Features: DiscountSummation not yet implemented")
			return 0, nil
		}
	}

	// Default case: return 0 if no supported features are found
	log.Printf("Features: No supported features found, returning default count 0")
	return 0, nil
}
