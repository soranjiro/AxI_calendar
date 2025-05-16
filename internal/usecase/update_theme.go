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
	"github.com/soranjiro/axicalendar/internal/domain"
	"github.com/soranjiro/axicalendar/internal/domain/theme"
	"github.com/soranjiro/axicalendar/internal/presentation/api"
	// No longer need validation package here
)

// UpdateTheme handles the logic for updating an existing theme.
// Accepts a domain theme object.
// Returns the updated domain theme object.
func (uc *UseCase) UpdateTheme(ctx context.Context, userID uuid.UUID, themeID uuid.UUID, updatedThemeData theme.Theme) (*theme.Theme, error) {
	// 1. Basic ID checks
	if themeID == uuid.Nil || userID == uuid.Nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, api.Error{Message: "Invalid theme ID or user ID"})
	}
	if themeID != updatedThemeData.ThemeID {
		return nil, echo.NewHTTPError(http.StatusBadRequest, api.Error{Message: "Theme ID mismatch between path and request body"})
	}
	if updatedThemeData.OwnerUserID == nil || *updatedThemeData.OwnerUserID != userID {
		// Ensure the OwnerUserID in the provided data matches the authenticated user
		return nil, echo.NewHTTPError(http.StatusBadRequest, api.Error{Message: "Owner user ID mismatch"})
	}

	// 2. Validate the incoming domain theme object itself
	if err := updatedThemeData.Validate(); err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, api.Error{Message: fmt.Sprintf("Theme data validation failed: %v", err)})
	}

	// 3. Check if theme exists, is owned by user, and is not default *before* attempting update
	existingTheme, err := uc.themeRepo.GetThemeByID(ctx, userID, themeID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) || errors.Is(err, domain.ErrForbidden) {
			return nil, echo.NewHTTPError(http.StatusNotFound, api.Error{Message: "Theme not found or access denied"})
		}
		log.Printf("Error retrieving theme %s before update: %v", themeID, err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, api.Error{Message: "Failed to retrieve theme before update"})
	}
	if existingTheme.IsDefault {
		return nil, echo.NewHTTPError(http.StatusForbidden, api.Error{Message: "Cannot modify a default theme"})
	}
	// Ownership is implicitly checked by GetThemeByID returning the theme for the given userID

	// 4. Prepare the theme object for the repository update
	// Preserve fields that cannot be updated
	updateInput := updatedThemeData                 // Copy the validated input data
	updateInput.IsDefault = existingTheme.IsDefault // Ensure IsDefault is not changed
	updateInput.CreatedAt = existingTheme.CreatedAt // Preserve original creation time
	// UpdatedAt will be set by the repository

	// 5. Call repository to update theme
	if err := uc.themeRepo.UpdateTheme(ctx, &updateInput); err != nil {
		// The repository's UpdateTheme might return ErrForbidden or ErrNotFound
		if errors.Is(err, domain.ErrForbidden) || errors.Is(err, domain.ErrNotFound) {
			// This could happen if deleted/changed between Get and Update, or repo internal check failed
			log.Printf("Forbidden/NotFound error during theme update %s: %v", themeID, err)
			// Return NotFound as the theme is either gone or inaccessible for update
			return nil, echo.NewHTTPError(http.StatusNotFound, api.Error{Message: "Failed to update theme: not found, is default, or not owned by user"})
		}
		log.Printf("Error updating theme %s in repository: %v", themeID, err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, api.Error{Message: "Failed to update theme"})
	}

	// 6. Fetch the updated theme to return the full object with updated timestamp
	finalTheme, err := uc.themeRepo.GetThemeByID(ctx, userID, themeID)
	if err != nil {
		// Log the inconsistency, but return the data we attempted to save
		log.Printf("WARN: Failed to fetch updated theme %s after successful update: %v", themeID, err)
		updateInput.UpdatedAt = time.Now() // Approximate update time
		return &updateInput, nil
	}

	// 7. Return the fetched domain theme
	return finalTheme, nil
}
