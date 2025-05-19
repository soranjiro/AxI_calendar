package monthly_summary

import (
	"context"
	"fmt"
	"time"

	"github.com/soranjiro/axicalendar/internal/domain/entry"
	"github.com/soranjiro/axicalendar/internal/domain/feature"
)

// MonthlySummaryExecutor implements the FeatureExecutor for monthly summaries.
type MonthlySummaryExecutor struct {
	// Add any dependencies needed for this feature, e.g., configuration.
}

// NewMonthlySummaryExecutor creates a new instance of MonthlySummaryExecutor.
func NewMonthlySummaryExecutor() *MonthlySummaryExecutor {
	return &MonthlySummaryExecutor{}
}

// Execute calculates the monthly summary (e.g., count of entries per month).
func (e *MonthlySummaryExecutor) Execute(ctx context.Context, entries []entry.Entry) (feature.AnalysisResult, error) {
	summary := make(map[string]int) // Map of "YYYY-MM" to count

	for _, ent := range entries {
		// Parse the EntryDate string to time.Time
		entryDate, err := time.Parse("2006-01-02", ent.EntryDate)
		if err != nil {
			return nil, fmt.Errorf("could not parse entry date '%s': %w", ent.EntryDate, err)
		}
		monthKey := entryDate.Format("2006-01") // Format as YYYY-MM
		summary[monthKey]++
	}

	// Convert the summary map to AnalysisResult format
	result := make(feature.AnalysisResult)
	for key, value := range summary {
		result[key] = value
	}

	return result, nil
}

// Compile-time check to ensure MonthlySummaryExecutor implements FeatureExecutor
var _ feature.FeatureExecutor = (*MonthlySummaryExecutor)(nil)
