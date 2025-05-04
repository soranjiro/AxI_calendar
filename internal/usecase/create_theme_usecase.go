package usecase

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/soranjiro/axicalendar/internal/api"
	"github.com/soranjiro/axicalendar/internal/domain/theme"
	"github.com/soranjiro/axicalendar/internal/validation" // Import validation package
)

// CreateTheme handles the logic for creating a new theme.
func (uc *UseCase) CreateTheme(ctx context.Context, userID uuid.UUID, req api.CreateThemeRequest) (*api.Theme, error) {
	// 1. Validate theme fields definition using validation package
	if err := validation.ValidateApiThemeFields(req.Fields); err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, api.Error{Message: fmt.Sprintf("Theme fields validation failed: %v", err)})
	}

	// 2. Validate supported features (basic validation) using validation package
	if req.SupportedFeatures != nil {
		if err := validation.ValidateSupportedFeatures(*req.SupportedFeatures); err != nil {
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
