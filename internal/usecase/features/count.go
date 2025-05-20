package features

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/soranjiro/axicalendar/internal/domain"
	"github.com/soranjiro/axicalendar/internal/domain/entry"
	"github.com/soranjiro/axicalendar/internal/domain/theme"
	"github.com/soranjiro/axicalendar/internal/presentation/api"
	"github.com/soranjiro/axicalendar/internal/repository"
)

type CountFeature string

const (
	Summation CountFeature = "summation"
	DiscountSummation CountFeature = "discount_summation"
)


func (f *Features) Count(ctx context.Context, theme theme.Theme, entries entries) (int64, error) {
	// 1. Check supported_features
	switch (theme.SupportedFeatures) {
	case Summation:
		// Handle summation
		entries.summation()
	case DiscountSummation:
		// Handle discount summation
		entries.discountSummation()
	}

	return count, nil
}
