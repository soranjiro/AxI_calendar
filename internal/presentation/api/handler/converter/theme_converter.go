package converter

import (
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/soranjiro/axicalendar/internal/domain/theme"
	"github.com/soranjiro/axicalendar/internal/presentation/api"
)

// --- Theme Converters ---

// FieldTypeFromApi converts API ThemeFieldType to domain FieldType
func FieldTypeFromApi(apiType api.ThemeFieldType) (theme.FieldType, error) {
	switch apiType {
	case api.Text:
		return theme.FieldTypeText, nil
	case api.Date:
		return theme.FieldTypeDate, nil
	case api.Datetime:
		return theme.FieldTypeDateTime, nil
	case api.Number:
		return theme.FieldTypeNumber, nil
	case api.Boolean:
		return theme.FieldTypeBoolean, nil
	case api.Textarea:
		return theme.FieldTypeTextarea, nil
	case api.Select:
		return theme.FieldTypeSelect, nil
	default:
		return "", fmt.Errorf("unknown API field type: %s", apiType)
	}
}

// FieldTypeToApi converts domain FieldType to API ThemeFieldType
func FieldTypeToApi(domainType theme.FieldType) (api.ThemeFieldType, error) {
	switch domainType {
	case theme.FieldTypeText:
		return api.Text, nil
	case theme.FieldTypeDate:
		return api.Date, nil
	case theme.FieldTypeDateTime:
		return api.Datetime, nil
	case theme.FieldTypeNumber:
		return api.Number, nil
	case theme.FieldTypeBoolean:
		return api.Boolean, nil
	case theme.FieldTypeTextarea:
		return api.Textarea, nil
	case theme.FieldTypeSelect:
		return api.Select, nil
	default:
		return "", fmt.Errorf("unknown domain field type: %s", domainType)
	}
}

// FromApiThemeField converts api.ThemeField to domain ThemeField
func FromApiThemeField(af api.ThemeField) (theme.ThemeField, error) {
	domainType, err := FieldTypeFromApi(af.Type)
	if err != nil {
		return theme.ThemeField{}, err
	}
	required := false // Default to false if nil
	if af.Required != nil {
		required = *af.Required
	}
	return theme.ThemeField{
		Name:     af.Name,
		Label:    af.Label,
		Type:     domainType,
		Required: required,
	}, nil
}

// FromApiThemeFields converts a slice of api.ThemeField to domain ThemeField
func FromApiThemeFields(afs []api.ThemeField) ([]theme.ThemeField, error) {
	dfs := make([]theme.ThemeField, len(afs))
	for i, af := range afs {
		df, err := FromApiThemeField(af)
		if err != nil {
			return nil, fmt.Errorf("error converting field %d ('%s'): %w", i, af.Name, err)
		}
		dfs[i] = df
	}
	return dfs, nil
}

// ToApiThemeField converts domain ThemeField to api.ThemeField
func ToApiThemeField(df theme.ThemeField) (api.ThemeField, error) {
	apiType, err := FieldTypeToApi(df.Type)
	if err != nil {
		return api.ThemeField{}, err // Return api.ThemeField{} on error
	}
	required := df.Required // Copy bool value
	return api.ThemeField{
		Name:     df.Name,
		Label:    df.Label,
		Type:     apiType,
		Required: &required, // Assign pointer to the copied value
	}, nil
}

// ToApiThemeFields converts a slice of domain ThemeField to api.ThemeField
func ToApiThemeFields(dfs []theme.ThemeField) ([]api.ThemeField, error) {
	afs := make([]api.ThemeField, len(dfs))
	for i, df := range dfs {
		af, err := ToApiThemeField(df)
		if err != nil {
			return nil, fmt.Errorf("error converting field %d ('%s'): %w", i, df.Name, err)
		}
		afs[i] = af
	}
	return afs, nil
}

// ToApiTheme converts internal Theme to API Theme
func ToApiTheme(dt theme.Theme) (api.Theme, error) {
	apiFields, err := ToApiThemeFields(dt.Fields)
	if err != nil {
		return api.Theme{}, fmt.Errorf("error converting theme fields for theme %s: %w", dt.ThemeID, err)
	}

	createdAt := dt.CreatedAt
	updatedAt := dt.UpdatedAt
	themeID := dt.ThemeID
	isDefault := dt.IsDefault
	var ownerUserID *uuid.UUID
	if dt.OwnerUserID != nil {
		ownerUserID = dt.OwnerUserID // Copy pointer
	}

	var supportedFeatures *[]string
	if dt.SupportedFeatures != nil {
		// Create a new slice and copy elements to avoid aliasing issues if dt.SupportedFeatures changes
		tempFeatures := make([]string, len(dt.SupportedFeatures))
		copy(tempFeatures, dt.SupportedFeatures)
		supportedFeatures = &tempFeatures
	}

	return api.Theme{
		ThemeId:           &themeID, // Corrected field name
		ThemeName:         dt.ThemeName,
		Fields:            apiFields,
		IsDefault:         &isDefault,
		OwnerUserId:       ownerUserID,
		SupportedFeatures: supportedFeatures,
		CreatedAt:         &createdAt,
		UpdatedAt:         &updatedAt,
	}, nil
}

// ToApiThemes converts a slice of internal Theme to API Theme
func ToApiThemes(dts []theme.Theme) ([]api.Theme, error) {
	ats := make([]api.Theme, len(dts))
	for i, dt := range dts {
		at, err := ToApiTheme(dt)
		if err != nil {
			// Log the error for the specific theme but continue converting others
			log.Printf("WARN: Failed to convert theme %s to API format: %v", dt.ThemeID, err)
			// Optionally, you could return the error immediately or skip this theme
			// For now, we skip adding the problematic theme to the result
			continue
		}
		ats[i] = at
	}
	// Filter out zero-value themes if any were skipped due to errors
	result := make([]api.Theme, 0, len(ats))
	for _, at := range ats {
		if at.ThemeId != nil { // Corrected field name check
			result = append(result, at)
		}
	}
	return result, nil // Return nil error even if some themes failed, log indicates issues
}

// FromApiCreateThemeRequest converts API CreateThemeRequest to domain Theme
func FromApiCreateThemeRequest(req api.CreateThemeRequest, userID uuid.UUID) (theme.Theme, error) {
	domainFields, err := FromApiThemeFields(req.Fields)
	if err != nil {
		return theme.Theme{}, fmt.Errorf("invalid theme fields in request: %w", err)
	}

	var supportedFeatures []string
	if req.SupportedFeatures != nil {
		supportedFeatures = *req.SupportedFeatures
	} else {
		supportedFeatures = []string{} // Default to empty slice
	}

	newTheme := theme.Theme{
		ThemeID:           uuid.New(), // Generate new ID
		ThemeName:         req.ThemeName,
		Fields:            domainFields,
		IsDefault:         false, // User-created themes are not default
		OwnerUserID:       &userID,
		SupportedFeatures: supportedFeatures,
		// CreatedAt, UpdatedAt, PK, SK set by repository
	}
	return newTheme, nil
}

// FromApiUpdateThemeRequest converts API UpdateThemeRequest to domain Theme
// Requires existing theme to preserve fields not allowed to be updated.
func FromApiUpdateThemeRequest(req api.UpdateThemeRequest, themeID uuid.UUID, userID uuid.UUID, existingTheme theme.Theme) (theme.Theme, error) {
	domainFields, err := FromApiThemeFields(req.Fields)
	if err != nil {
		return theme.Theme{}, fmt.Errorf("invalid theme fields in request: %w", err)
	}

	var supportedFeatures []string
	if req.SupportedFeatures != nil {
		supportedFeatures = *req.SupportedFeatures
	} else {
		supportedFeatures = existingTheme.SupportedFeatures // Keep existing if not provided
	}

	updatedTheme := theme.Theme{
		ThemeID:           themeID,
		ThemeName:         req.ThemeName,
		Fields:            domainFields,
		IsDefault:         existingTheme.IsDefault, // Cannot change this flag via update
		OwnerUserID:       &userID,                 // Should match existing owner
		SupportedFeatures: supportedFeatures,
		CreatedAt:         existingTheme.CreatedAt, // Preserve original creation time
		// UpdatedAt, PK, SK handled by repository
	}
	return updatedTheme, nil
}
