package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// GetThemesThemeIdEntriesCount retrieves the count of entries for a specific user and theme.
func (uc *UseCase) GetThemesThemeIdEntriesCount(ctx context.Context, userID uuid.UUID, themeID uuid.UUID) (int64, error) {
	// 1. Get Theme and Entries from repository in a single query
	th, entries, err := uc.entryRepo.GetThemeAndEntries(ctx, userID, themeID)
	if err != nil {
		return 0, fmt.Errorf("error getting theme and entries for user %s, theme %s: %w", userID, themeID, err)
	}
	if th == nil {
		return 0, fmt.Errorf("theme %s not found for user %s", themeID, userID)
	}

	// 2. Call the count feature
	count, err := uc.feature.Count(ctx, *th, entries)
	if err != nil {
		return 0, fmt.Errorf("error getting entries count for user %s: %w", userID, err)
	}

	return count, nil
}
