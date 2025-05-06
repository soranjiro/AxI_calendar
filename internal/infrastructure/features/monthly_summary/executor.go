package monthly_summary

import (
	"context"
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
		// Assuming entries have a date field we can use for grouping.
		// We need a way to reliably get the primary date field from an entry.
		// For now, let's assume a field named "date" exists and is a time.Time.
		// This part might need refinement based on how entry data is structured.
		dateValue, ok := ent.Data["date"]
		if !ok {
			// Skip entries without a date field or handle differently
			continue
		}

		entryDate, ok := dateValue.(time.Time)
		if !ok {
			// Skip entries where 'date' is not a time.Time or handle conversion
			// Consider parsing from string if stored as string
			continue
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
