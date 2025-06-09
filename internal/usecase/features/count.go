package features

import (
	"context"
	"fmt"

	"github.com/soranjiro/axicalendar/internal/domain/entry"
	"github.com/soranjiro/axicalendar/internal/domain/theme"
)

type CountFeature string

const (
	Summation         CountFeature = "summation"
	DiscountSummation CountFeature = "discount_summation"
)

func (f CountFeature) String() string {
	return string(f)
}

func (f *Features) Count(ctx context.Context, th theme.Theme, entries entry.Entries) (int64, error) {
	// 1. Check supported_features
	for _, feature := range th.SupportedFeatures {
		switch feature {
		case Summation.String():
			// Handle summation
			return entries.Summation(), nil
		case DiscountSummation.String():
			// Handle discount summation
			return entries.DiscountSummation(), nil
		}
	}
	return 0, fmt.Errorf("no supported count feature found in theme: %v", th.SupportedFeatures)
}
