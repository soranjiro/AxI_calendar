package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/soranjiro/axicalendar/internal/domain"
	"github.com/soranjiro/axicalendar/internal/domain/entry"
	"github.com/soranjiro/axicalendar/internal/presentation/api"
	"github.com/soranjiro/axicalendar/internal/repository"
)

// GetEntriesCount retrieves the count of entries for a specific user and theme.
func (uc *UseCase) GetEntriesCount(ctx context.Context, userID uuid.UUID, themeID uuid.UUID) (int64, error) {
	// 1. Set up the GSI to retrieve both the theme and entries corresponding to the theme ID
	data = ...
	theme = data...
	entries = data...

	// 2. Call the count feature
	count, err := uc.feature.Count(ctx, theme, entries)
	if err != nil {
		return 0, fmt.Errorf("error getting entries count for user %s: %w", userID, err)
	}

	return count, nil
}
