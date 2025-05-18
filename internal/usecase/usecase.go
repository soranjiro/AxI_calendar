package usecase

import (
	dynamodbrepo "github.com/soranjiro/axicalendar/internal/adapter/persistence/dynamodb"
	"github.com/soranjiro/axicalendar/internal/domain/feature" // feature をインポート
)

// UseCase implements the UseCaseInterface.
type UseCase struct {
	themeRepo       dynamodbrepo.ThemeRepository
	entryRepo       dynamodbrepo.EntryRepository
	featureRegistry feature.ExecutorRegistry // featureRegistry を追加
	// Add other repositories or services as needed
}

// NewUseCase creates a new UseCase with dependencies.
func NewUseCase(themeRepo dynamodbrepo.ThemeRepository, entryRepo dynamodbrepo.EntryRepository, featureRegistry feature.ExecutorRegistry) *UseCase { // featureRegistry を引数に追加
	return &UseCase{
		themeRepo:       themeRepo,
		entryRepo:       entryRepo,
		featureRegistry: featureRegistry, // featureRegistry を設定
	}
}
