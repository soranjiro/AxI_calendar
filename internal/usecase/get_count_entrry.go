package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	// entry をインポート
	// theme をインポート
)

// GetEntriesCount retrieves the count of entries for a specific user and theme.
func (uc *UseCase) GetEntriesCount(ctx context.Context, userID uuid.UUID, themeID uuid.UUID) (int64, error) {
	// 1. Get Theme and Entries from repository
	th, err := uc.repo.GetThemeByID(ctx, userID, themeID) // userID を渡すように修正
	if err != nil {
		return 0, fmt.Errorf("error getting theme %s for user %s: %w", themeID, userID, err)
	}
	if th == nil {
		return 0, fmt.Errorf("theme %s not found for user %s", themeID, userID)
	}

	// DynamoDBではパーティションキーとソートキーの範囲で効率的にデータを取得できるため、
	// themeID に紐づく entries を取得する repository メソッドを呼び出すことを想定します。
	// ここでは仮に GetEntriesByThemeID のようなメソッドが存在するとします。
	// 実際のrepositoryのメソッドに合わせて修正してください。
	entries, err := uc.repo.GetEntriesByThemeID(ctx, userID, themeID) // userID と themeID を渡す
	if err != nil {
		return 0, fmt.Errorf("error getting entries for theme %s and user %s: %w", themeID, userID, err)
	}

	// 2. Call the count feature
	count, err := uc.feature.Count(ctx, *th, entries)
	if err != nil {
		return 0, fmt.Errorf("error getting entries count for user %s: %w", userID, err)
	}

	return count, nil
}
