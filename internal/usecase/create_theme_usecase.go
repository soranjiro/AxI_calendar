package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/soranjiro/axicalendar/internal/api"
	"github.com/soranjiro/axicalendar/internal/domain/theme"
	repo "github.com/soranjiro/axicalendar/internal/repository/dynamodb"
)

// CreateThemeUseCase defines the interface for the create theme use case.
type CreateThemeUseCase interface {
	Execute(ctx context.Context, userID uuid.UUID, req api.CreateThemeRequest) (*api.Theme, error)
}

// createThemeUseCase implements the CreateThemeUseCase interface.
type createThemeUseCase struct {
	themeRepo repo.ThemeRepository
}

// NewCreateThemeUseCase creates a new CreateThemeUseCase.
func NewCreateThemeUseCase(themeRepo repo.ThemeRepository) CreateThemeUseCase {
	return &createThemeUseCase{themeRepo: themeRepo}
}

// Execute handles the logic for creating a new theme.
func (uc *createThemeUseCase) Execute(ctx context.Context, userID uuid.UUID, req api.CreateThemeRequest) (*api.Theme, error) {
	// 1. Validate theme fields definition
	if err := validateThemeFields(req.Fields); err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, api.Error{Message: fmt.Sprintf("Theme fields validation failed: %v", err)})
	}

	// 2. Validate supported features (basic validation)
	if req.SupportedFeatures != nil {
		if err := validateSupportedFeatures(*req.SupportedFeatures); err != nil {
			return nil, echo.NewHTTPError(http.StatusBadRequest, api.Error{Message: fmt.Sprintf("Supported features validation failed: %v", err)})
		}
	}

	// 3. Convert API fields to domain fields
	modelFields := make([]theme.ThemeField, len(req.Fields))
	for i, f := range req.Fields {
		modelFields[i] = theme.FromApiThemeField(f)
	}

	// 4. Handle supported features (default to empty list if nil)
	var supportedFeatures []string
	if req.SupportedFeatures != nil {
		supportedFeatures = *req.SupportedFeatures
	} else {
		supportedFeatures = []string{}
	}

	// 5. Create domain theme object
	newTheme := theme.Theme{
		ThemeID:           uuid.New(), // Generate new ID
		ThemeName:         req.ThemeName,
		Fields:            modelFields,
		IsDefault:         false, // User-created themes are not default
		OwnerUserID:       &userID,
		SupportedFeatures: supportedFeatures,
		// CreatedAt, UpdatedAt, PK, SK set by repository
	}

	// 6. Call repository to create theme
	if err := uc.themeRepo.CreateTheme(ctx, &newTheme); err != nil {
		log.Printf("Error creating theme in repository for user %s: %v", userID, err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, api.Error{Message: "Failed to create theme"})
	}

	// 7. Fetch the created theme to return the full object (optional but good practice)
	createdTheme, err := uc.themeRepo.GetThemeByID(ctx, userID, newTheme.ThemeID)
	if err != nil {
		// Log the inconsistency, but return the data we have as a fallback
		log.Printf("WARN: Failed to fetch newly created theme %s for user %s: %v", newTheme.ThemeID, userID, err)
		// Approximate timestamps and return
		now := time.Now()
		newTheme.CreatedAt = now
		newTheme.UpdatedAt = now
		apiTheme := theme.ToApiTheme(newTheme)
		return &apiTheme, nil // Return success even if fetch failed, but with approximated data
	}

	// 8. Convert to API model and return
	apiTheme := theme.ToApiTheme(*createdTheme)
	return &apiTheme, nil
}

// --- Validation Helpers (Consider moving to a shared validation package) ---

// validateSupportedFeatures performs basic validation on supported feature names.
func validateSupportedFeatures(features []string) error {
	validFeatures := map[string]bool{
		"monthly_summary":      true,
		"category_aggregation": true,
		// Add other known valid features here
	}
	names := make(map[string]bool)
	for i, feature := range features {
		if feature == "" {
			return fmt.Errorf("feature %d: name cannot be empty", i)
		}
		if !isValidFeatureName(feature) {
			return fmt.Errorf("feature %d ('%s'): name contains invalid characters or format (use snake_case)", i, feature)
		}
		if !validFeatures[feature] {
			log.Printf("WARN: Potentially unsupported feature '%s' included in theme definition.", feature)
		}
		if _, exists := names[feature]; exists {
			return fmt.Errorf("feature name '%s' is duplicated", feature)
		}
		names[feature] = true
	}
	return nil
}

// isValidFeatureName checks if a feature name is valid (e.g., snake_case).
func isValidFeatureName(name string) bool {
	if name == "" || !(name[0] >= 'a' && name[0] <= 'z') {
		return false
	}
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_') {
			return false
		}
	}
	return true
}

// validateThemeFields performs basic validation on theme field definitions.
func validateThemeFields(fields []api.ThemeField) error {
	if len(fields) == 0 {
		return errors.New("theme must have at least one field")
	}
	names := make(map[string]bool)
	for i, field := range fields {
		if field.Name == "" {
			return fmt.Errorf("field %d: name is required", i)
		}
		if !isValidFieldName(field.Name) {
			return fmt.Errorf("field %d ('%s'): name contains invalid characters (allowed: a-z, 0-9, _ starting with letter or _)", i, field.Name)
		}
		if field.Label == "" {
			return fmt.Errorf("field %d ('%s'): label is required", i, field.Name)
		}
		if _, exists := names[field.Name]; exists {
			return fmt.Errorf("field name '%s' is duplicated", field.Name)
		}
		names[field.Name] = true

		isValidType := false
		validTypes := []api.ThemeFieldType{
			api.Text,
			api.Date,
			api.Datetime,
			api.Number,
			api.Boolean,
			api.Textarea,
			api.Select,
		}
		for _, vt := range validTypes {
			if field.Type == vt {
				isValidType = true
				break
			}
		}
		if !isValidType {
			return fmt.Errorf("field '%s': invalid type '%s'", field.Name, field.Type)
		}

		if field.Required == nil {
			return fmt.Errorf("field %d ('%s'): required attribute must be explicitly set to true or false", i, field.Name)
		}
	}
	return nil
}

// isValidFieldName checks if a field name is valid (e.g., snake_case).
func isValidFieldName(name string) bool {
	if name == "" || (!((name[0] >= 'a' && name[0] <= 'z') || name[0] == '_')) {
		return false
	}
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_') {
			return false
		}
	}
	return true
}
