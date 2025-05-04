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
	"github.com/soranjiro/axicalendar/internal/domain"
	"github.com/soranjiro/axicalendar/internal/domain/theme"
	repo "github.com/soranjiro/axicalendar/internal/repository/dynamodb"
)

// UpdateThemeUseCase defines the interface for the update theme use case.
type UpdateThemeUseCase interface {
	Execute(ctx context.Context, userID uuid.UUID, themeID uuid.UUID, req api.UpdateThemeRequest) (*api.Theme, error)
}

// updateThemeUseCase implements the UpdateThemeUseCase interface.
type updateThemeUseCase struct {
	themeRepo repo.ThemeRepository
}

// NewUpdateThemeUseCase creates a new UpdateThemeUseCase.
func NewUpdateThemeUseCase(themeRepo repo.ThemeRepository) UpdateThemeUseCase {
	return &updateThemeUseCase{themeRepo: themeRepo}
}

// Execute handles the logic for updating an existing theme.
func (uc *updateThemeUseCase) Execute(ctx context.Context, userID uuid.UUID, themeID uuid.UUID, req api.UpdateThemeRequest) (*api.Theme, error) {
	// 1. Validate incoming theme fields definition
	if err := validateThemeFields(req.Fields); err != nil { // Assumes validateThemeFields is available
		return nil, echo.NewHTTPError(http.StatusBadRequest, api.Error{Message: fmt.Sprintf("Theme fields validation failed: %v", err)})
	}

	// 2. Validate incoming supported features
	if req.SupportedFeatures != nil {
		if err := validateSupportedFeatures(*req.SupportedFeatures); err != nil { // Assumes validateSupportedFeatures is available
			return nil, echo.NewHTTPError(http.StatusBadRequest, api.Error{Message: fmt.Sprintf("Supported features validation failed: %v", err)})
		}
	}

	// 3. Check if theme exists, is owned by user, and is not default *before* attempting update
	existingTheme, err := uc.themeRepo.GetThemeByID(ctx, userID, themeID)
	if err != nil {
		if errors.Is(err, domain.ErrThemeNotFound) || errors.Is(err, domain.ErrForbidden) {
			return nil, echo.NewHTTPError(http.StatusNotFound, api.Error{Message: "Theme not found or access denied"})
		}
		log.Printf("Error retrieving theme %s before update: %v", themeID, err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, api.Error{Message: "Failed to retrieve theme before update"})
	}
	if existingTheme.IsDefault {
		return nil, echo.NewHTTPError(http.StatusForbidden, api.Error{Message: "Cannot modify a default theme"})
	}
	// Ownership is implicitly checked by GetThemeByID returning the theme for the given userID

	// 4. Convert API fields to domain fields
	modelFields := make([]theme.ThemeField, len(req.Fields))
	for i, f := range req.Fields {
		modelFields[i] = theme.FromApiThemeField(f)
	}

	// 5. Handle supported features - keep existing if not provided in request
	var supportedFeatures []string
	if req.SupportedFeatures != nil {
		supportedFeatures = *req.SupportedFeatures
	} else {
		supportedFeatures = existingTheme.SupportedFeatures // Keep existing if not provided
	}

	// 6. Create updated domain theme object
	updatedTheme := theme.Theme{
		ThemeID:           themeID,
		ThemeName:         req.ThemeName,
		Fields:            modelFields,
		IsDefault:         false, // Cannot change this flag via update
		OwnerUserID:       &userID,
		SupportedFeatures: supportedFeatures,
		// CreatedAt, UpdatedAt, PK, SK handled by repository
	}

	// 7. Call repository to update theme
	if err := uc.themeRepo.UpdateTheme(ctx, &updatedTheme); err != nil {
		// The repository's UpdateTheme might return ErrForbidden for not found/default/not owner
		if errors.Is(err, domain.ErrForbidden) {
			// This could happen if deleted/changed between Get and Update, or repo internal check failed
			log.Printf("Forbidden error during theme update %s: %v", themeID, err)
			return nil, echo.NewHTTPError(http.StatusForbidden, api.Error{Message: "Failed to update theme: not found, is default, or not owned by user"})
		}
		log.Printf("Error updating theme %s in repository: %v", themeID, err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, api.Error{Message: "Failed to update theme"})
	}

	// 8. Fetch the updated theme to return the full object
	finalTheme, err := uc.themeRepo.GetThemeByID(ctx, userID, themeID)
	if err != nil {
		// Log the inconsistency, return approximated data
		log.Printf("WARN: Failed to fetch updated theme %s after successful update: %v", themeID, err)
		updatedTheme.CreatedAt = existingTheme.CreatedAt // Keep original creation time
		updatedTheme.UpdatedAt = time.Now()              // Approximate update time
		apiTheme := theme.ToApiTheme(updatedTheme)
		return &apiTheme, nil
	}

	// 9. Convert to API model and return
	apiTheme := theme.ToApiTheme(*finalTheme)
	return &apiTheme, nil
}

// Note: Assumes validation helper functions (validateThemeFields, validateSupportedFeatures, isValidFieldName, isValidFeatureName)
// are available in this package or imported from a shared location (e.g., create_theme_usecase.go or validation package).
// Consider moving them to a shared place.
