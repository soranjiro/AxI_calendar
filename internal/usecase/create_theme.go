package usecase

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/soranjiro/axicalendar/internal/domain/theme"
	"github.com/soranjiro/axicalendar/internal/presentation/api"
)

// CreateTheme handles the logic for creating a new theme.
// Accepts domain theme, returns domain theme
func (uc *UseCase) CreateTheme(ctx context.Context, newTheme theme.Theme) (*theme.Theme, error) {
	// 1. Validate the domain theme object itself
	if err := newTheme.Validate(); err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, api.Error{Message: fmt.Sprintf("Theme validation failed: %v", err)})
	}

	// 2. Ensure OwnerUserID is set (should be set by converter/handler)
	if newTheme.OwnerUserID == nil || *newTheme.OwnerUserID == uuid.Nil {
		log.Printf("ERROR: CreateTheme called with nil or zero OwnerUserID for theme %s", newTheme.ThemeName)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, api.Error{Message: "Internal error: User ID missing for theme creation"})
	}
	userID := *newTheme.OwnerUserID

	// 3. Ensure ThemeID is set (should be set by converter/handler)
	if newTheme.ThemeID == uuid.Nil {
		newTheme.ThemeID = uuid.New()
	}

	// 4. Ensure IsDefault is false for user-created themes
	newTheme.IsDefault = false

	// 5. Call repository to create theme
	if err := uc.themeRepo.CreateTheme(ctx, &newTheme); err != nil {
		log.Printf("Error creating theme in repository for user %s: %v", userID, err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, api.Error{Message: "Failed to create theme"})
	}

	// 6. Fetch the created theme to return the full object with timestamps
	createdTheme, err := uc.themeRepo.GetThemeByID(ctx, userID, newTheme.ThemeID)
	if err != nil {
		// Log the inconsistency, but return the data we have as a fallback
		log.Printf("WARN: Failed to fetch newly created theme %s for user %s: %v", newTheme.ThemeID, userID, err)
		// Approximate timestamps and return
		now := time.Now()
		newTheme.CreatedAt = now
		newTheme.UpdatedAt = now
		return &newTheme, nil // Return success even if fetch failed, but with approximated data
	}

	// 7. Return the fetched domain theme
	return createdTheme, nil
}
