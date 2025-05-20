package usecase

import (
	dynamodbrepo "github.com/soranjiro/axicalendar/internal/adapter/persistence/dynamodb"
	"github.com/soranjiro/axicalendar/internal/domain/feature"   // feature をインポート
	"github.com/soranjiro/axicalendar/internal/usecase/services" // Import services
)

// UseCase implements the UseCaseInterface.
type UseCase struct {
	themeRepo       dynamodbrepo.ThemeRepository
	entryRepo       dynamodbrepo.EntryRepository
	featureRegistry feature.ExecutorRegistry // featureRegistry を追加
	entryService    *services.EntryService   // Add EntryService
	themeService    *services.ThemeService   // Add ThemeService
	// Add other repositories or services as needed
}

// NewUseCase creates a new UseCase with dependencies.
func NewUseCase(
	themeRepo dynamodbrepo.ThemeRepository,
	entryRepo dynamodbrepo.EntryRepository,
	featureRegistry feature.ExecutorRegistry,
	entryService *services.EntryService, // Add EntryService
	themeService *services.ThemeService, // Add ThemeService
) *UseCase { // featureRegistry を引数に追加
	return &UseCase{
		themeRepo:       themeRepo,
		entryRepo:       entryRepo,
		featureRegistry: featureRegistry, // featureRegistry を設定
		entryService:    entryService,    // Set EntryService
		themeService:    themeService,    // Set ThemeService
	}
}
