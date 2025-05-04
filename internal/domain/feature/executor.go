package feature

import (
	"context"

	"github.com/soranjiro/axicalendar/internal/domain/entry"
)

// AnalysisResult represents the generic result of a feature execution.
// The actual structure will vary depending on the feature.
type AnalysisResult map[string]interface{}

// FeatureExecutor defines the interface for executing a theme-specific feature.
type FeatureExecutor interface {
	// Execute performs the feature logic on the provided entries.
	// ctx can be used for cancellation or passing request-scoped values.
	// entries are the relevant entries for the feature (e.g., for a specific theme and user).
	Execute(ctx context.Context, entries []entry.Entry) (AnalysisResult, error)
}
