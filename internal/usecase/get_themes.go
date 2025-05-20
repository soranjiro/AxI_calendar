package usecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/soranjiro/axicalendar/internal/domain/theme"
)

// GetThemes handles the logic for getting all themes accessible by the user.
// Returns domain themes.
func (uc *UseCase) GetThemes(ctx context.Context, userID uuid.UUID) ([]theme.Theme, error) {
	// Call the ThemeService
	return uc.themeService.GetThemes(ctx, userID)
}
