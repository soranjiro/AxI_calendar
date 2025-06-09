package usecase

import (
	dynamodbrepo "github.com/soranjiro/axicalendar/internal/adapter/persistence/dynamodb"
	"github.com/soranjiro/axicalendar/internal/usecase/features"
)

// UseCase implements the UseCaseInterface.
type UseCase struct {
	themeRepo dynamodbrepo.ThemeRepository
	entryRepo dynamodbrepo.EntryRepository
	feature   *features.Features
	// Add other repositories or services as needed
}

// NewUseCase creates a new UseCase with dependencies.
func NewUseCase(themeRepo dynamodbrepo.ThemeRepository, entryRepo dynamodbrepo.EntryRepository) *UseCase { // feature を引数に追加
	return &UseCase{
		themeRepo: themeRepo,
		entryRepo: entryRepo,
		feature:   features.NewFeatures(),
	}
}
