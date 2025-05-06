package feature

import "github.com/google/uuid"

// Feature represents the metadata of a feature available in the system.
type Feature struct {
	ID          uuid.UUID `dynamodbav:"ID"`          // Unique identifier for the feature definition (optional, might not be stored directly)
	Name        string    `dynamodbav:"Name"`        // Internal name (e.g., "monthly_summary") used for identification
	DisplayName string    `dynamodbav:"DisplayName"` // User-facing name (e.g., "Monthly Summary")
	Description string    `dynamodbav:"Description"` // Brief description of what the feature does
	// Add other relevant metadata if needed, e.g., required theme fields, configuration options
}

// --- Potentially add a FeatureRepository interface if features need to be managed dynamically ---
// type Repository interface {
// 	GetFeatureByName(ctx context.Context, name string) (*Feature, error)
// 	ListFeatures(ctx context.Context) ([]Feature, error)
// }
