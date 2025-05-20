package usecase

import (
	dynamodbrepo "github.com/soranjiro/axicalendar/internal/adapter/persistence/dynamodb"
)

// UseCase implements the UseCaseInterface.
type UseCase struct {
	themeRepo dynamodbrepo.ThemeRepository
	entryRepo dynamodbrepo.EntryRepository
	// Add other repositories or services as needed
}

// NewUseCase creates a new UseCase with dependencies.
func NewUseCase(themeRepo dynamodbrepo.ThemeRepository, entryRepo dynamodbrepo.EntryRepository) *UseCase {
	return &UseCase{
		themeRepo: themeRepo,
		entryRepo: entryRepo,
	}
}
