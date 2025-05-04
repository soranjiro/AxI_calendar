package handler

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/soranjiro/axicalendar/internal/api"
	"github.com/soranjiro/axicalendar/internal/domain"
	"github.com/soranjiro/axicalendar/internal/domain/entry"
	"github.com/soranjiro/axicalendar/internal/domain/theme"
	repo "github.com/soranjiro/axicalendar/internal/repository/dynamodb"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

type contextKey string

const UserIDContextKey contextKey = "userID"

// DummyAuthMiddleware is for local testing only.
// It injects a hardcoded UserID into the request context.
// DO NOT USE IN PRODUCTION.
func DummyAuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Use an environment variable for the dummy user ID, or fallback to a default
		dummyUserIDStr := os.Getenv("DUMMY_USER_ID")
		if dummyUserIDStr == "" {
			dummyUserIDStr = "00000000-0000-0000-0000-000000000001" // Default dummy UUID
			log.Println("DUMMY_USER_ID environment variable not set, using default:", dummyUserIDStr)
		}

		userID, err := uuid.Parse(dummyUserIDStr)
		if err != nil {
			log.Printf("Error parsing DUMMY_USER_ID: %v. Using default.", err)
			userID = uuid.MustParse("00000000-0000-0000-0000-000000000001")
		}

		// Add userID to context
		req := c.Request()
		ctxWithUser := context.WithValue(req.Context(), UserIDContextKey, userID)
		c.SetRequest(req.WithContext(ctxWithUser))

		log.Printf("DummyAuthMiddleware: Injected UserID %s into context", userID.String())
		return next(c)
	}
}

// GetUserIDFromContext retrieves the UserID from the context.
// In a real application, this would be populated by a proper authentication middleware.
func GetUserIDFromContext(ctx context.Context) (uuid.UUID, error) {
	userIDVal := ctx.Value(UserIDContextKey)
	if userIDVal == nil {
		// Return a standard echo HTTP error which the framework handles
		return uuid.Nil, echo.NewHTTPError(http.StatusUnauthorized, "User ID not found in context")
	}
	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		return uuid.Nil, echo.NewHTTPError(http.StatusInternalServerError, "User ID in context is not a valid UUID")
	}
	if userID == uuid.Nil {
		return uuid.Nil, echo.NewHTTPError(http.StatusUnauthorized, "Invalid User ID (Nil UUID) in context")
	}
	return userID, nil
}

// Helper to return standard API error response
func newApiError(statusCode int, message string, err error) error {
	logMsg := message
	if err != nil {
		logMsg = fmt.Sprintf("%s: %v", message, err)
	}
	log.Printf("API Error (%d): %s", statusCode, logMsg) // Log the detailed error
	// Return a generic message to the client in the standard error format
	apiErr := api.Error{Message: message}
	return echo.NewHTTPError(statusCode, apiErr)
}

// ApiHandler implements the api.ServerInterface
type ApiHandler struct {
	EntryRepo repo.EntryRepository
	ThemeRepo repo.ThemeRepository
	// TODO: Add UserRepository if needed for GetAuthMe
}

// NewApiHandler creates a new ApiHandler
func NewApiHandler(entryRepo repo.EntryRepository, themeRepo repo.ThemeRepository) *ApiHandler {
	return &ApiHandler{EntryRepo: entryRepo, ThemeRepo: themeRepo}
}

// --- Auth Handlers ---

// GetAuthMe retrieves information about the currently authenticated user.
func (h *ApiHandler) GetAuthMe(ctx echo.Context) error {
	userID, err := GetUserIDFromContext(ctx.Request().Context())
	if err != nil {
		// GetUserIDFromContext already returns a formatted echo.HTTPError
		return err
	}

	// In a real application:
	// 1. Use the userID to fetch user details (e.g., email, name) from a user repository or Cognito.
	// 2. For now, we'll return a dummy response based on the userID.

	// Placeholder implementation: Return the UserID.
	// You might want to fetch more user details from a UserRepository here.
	emailStr := openapi_types.Email(fmt.Sprintf("user-%s@example.com", userID.String())) // Dummy email
	dummyUser := api.User{
		UserId: &userID,   // Use address of userID
		Email:  &emailStr, // Use address of emailStr
		// Add other fields as defined in your openapi.yaml User schema
	}

	log.Printf("GetAuthMe called for UserID: %s", userID.String())
	return ctx.JSON(http.StatusOK, dummyUser)
}

// PostAuthConfirmForgotPassword handles the confirmation of a password reset.
// Placeholder implementation.
func (h *ApiHandler) PostAuthConfirmForgotPassword(ctx echo.Context) error {
	log.Println("PostAuthConfirmForgotPassword called (Not Implemented)")
	// In a real implementation:
	// 1. Bind request body (confirmation code, new password, email/username)
	// 2. Call Cognito ConfirmForgotPassword
	// 3. Handle success or errors (e.g., invalid code, expired code)
	return newApiError(http.StatusNotImplemented, "Confirm Forgot Password not implemented", nil)
}

// PostAuthConfirmSignup handles the confirmation of a user signup.
// Placeholder implementation.
func (h *ApiHandler) PostAuthConfirmSignup(ctx echo.Context) error {
	log.Println("PostAuthConfirmSignup called (Not Implemented)")
	// In a real implementation:
	// 1. Bind request body (confirmation code, email/username)
	// 2. Call Cognito ConfirmSignUp
	// 3. Handle success or errors (e.g., invalid code, expired code, user already confirmed)
	return newApiError(http.StatusNotImplemented, "Confirm Signup not implemented", nil)
}

// PostAuthForgotPassword initiates the password reset process.
// Placeholder implementation.
func (h *ApiHandler) PostAuthForgotPassword(ctx echo.Context) error {
	log.Println("PostAuthForgotPassword called (Not Implemented)")
	// In a real implementation:
	// 1. Bind request body (email/username)
	// 2. Call Cognito ForgotPassword
	// 3. Handle success (code sent) or errors (user not found)
	return newApiError(http.StatusNotImplemented, "Forgot Password not implemented", nil)
}

// PostAuthLogout handles user logout.
// Placeholder implementation.
func (h *ApiHandler) PostAuthLogout(ctx echo.Context) error {
	log.Println("PostAuthLogout called (Not Implemented)")
	// In a real implementation:
	// 1. If using session-based auth, invalidate the session.
	// 2. If using JWTs, potentially add the token to a blacklist (depending on strategy).
	// 3. Cognito: Call GlobalSignOut or RevokeToken.
	return newApiError(http.StatusNotImplemented, "Logout not implemented", nil)
}

// PostAuthRefresh handles token refresh requests.
// Placeholder implementation.
func (h *ApiHandler) PostAuthRefresh(ctx echo.Context) error {
	log.Println("PostAuthRefresh called (Not Implemented)")
	// In a real implementation:
	// 1. Bind request body (refresh token)
	// 2. Call Cognito InitiateAuth (REFRESH_TOKEN_AUTH flow)
	// 3. Return new ID and Access tokens on success
	return newApiError(http.StatusNotImplemented, "Token Refresh not implemented", nil)
}

func (h *ApiHandler) PostAuthLogin(ctx echo.Context) error {
	// In a real implementation:
	// 1. Bind request body (email, password)
	// 2. Call Cognito InitiateAuth (USER_PASSWORD_AUTH flow)
	// 3. Handle challenges if necessary (e.g., NEW_PASSWORD_REQUIRED)
	// 4. Return tokens (ID, Access, Refresh) on success
	log.Println("Auth Login endpoint called (Not Implemented)")
	return newApiError(http.StatusNotImplemented, "Login not implemented", nil)
}

func (h *ApiHandler) PostAuthSignup(ctx echo.Context) error {
	// In a real implementation:
	// 1. Bind request body (email, password, other attributes)
	// 2. Call Cognito SignUp
	// 3. Handle response (user confirmation needed?)
	// 4. Optionally, create associated user profile data in DynamoDB if needed
	log.Println("Auth Signup endpoint called (Not Implemented)")
	return newApiError(http.StatusNotImplemented, "Signup not implemented", nil)
}

// --- Entry Handlers ---

func (h *ApiHandler) GetEntries(ctx echo.Context, params api.GetEntriesParams) error {
	userID, err := GetUserIDFromContext(ctx.Request().Context())
	if err != nil {
		return err // Error already formatted by GetUserIDFromContext
	}

	// Parse dates (OpenAPI spec ensures format, but time.Time conversion needed)
	startDate := params.StartDate.Time
	endDate := params.EndDate.Time

	if endDate.Before(startDate) {
		return newApiError(http.StatusBadRequest, "end_date cannot be before start_date", nil)
	}

	// Parse theme IDs
	var themeIDs []uuid.UUID
	if params.ThemeIds != nil && *params.ThemeIds != "" {
		themeIDStrs := strings.Split(*params.ThemeIds, ",")
		for _, idStr := range themeIDStrs {
			id, err := uuid.Parse(strings.TrimSpace(idStr))
			if err != nil {
				return newApiError(http.StatusBadRequest, fmt.Sprintf("Invalid theme_id format: %s", idStr), err)
			}
			themeIDs = append(themeIDs, id)
		}
	}

	entries, err := h.EntryRepo.ListEntriesByDateRange(ctx.Request().Context(), userID, startDate, endDate, themeIDs)
	if err != nil {
		return newApiError(http.StatusInternalServerError, "Failed to retrieve entries", err)
	}

	apiEntries := make([]api.Entry, len(entries))
	for i, e := range entries {
		apiEntries[i] = entry.ToApiEntry(e)
	}

	return ctx.JSON(http.StatusOK, apiEntries)
}

func (h *ApiHandler) PostEntries(ctx echo.Context) error {
	userID, err := GetUserIDFromContext(ctx.Request().Context())
	if err != nil {
		return err
	}

	var req api.CreateEntryRequest
	if err := ctx.Bind(&req); err != nil {
		return newApiError(http.StatusBadRequest, "Invalid request body", err)
	}

	// Validate theme exists and user has access
	th, err := h.ThemeRepo.GetThemeByID(ctx.Request().Context(), userID, req.ThemeId)
	if err != nil {
		if errors.Is(err, domain.ErrThemeNotFound) || errors.Is(err, domain.ErrForbidden) {
			return newApiError(http.StatusNotFound, "Theme not found or access denied", err)
		}
		return newApiError(http.StatusInternalServerError, "Failed to validate theme", err)
	}

	// Validate data against theme fields
	if err := validateEntryData(req.Data, th.Fields); err != nil {
		return newApiError(http.StatusBadRequest, fmt.Sprintf("Entry data validation failed: %v", err), nil)
	}

	newEntry := entry.Entry{
		EntryID:   uuid.New(), // Generate new ID
		ThemeID:   req.ThemeId,
		UserID:    userID,
		EntryDate: req.EntryDate.Format("2006-01-02"), // Store as YYYY-MM-DD string
		Data:      req.Data,
		// CreatedAt, UpdatedAt, PK, SK, GSI keys set by repository
	}

	if err := h.EntryRepo.CreateEntry(ctx.Request().Context(), &newEntry); err != nil {
		// Handle potential conditional check failure (already exists) if needed
		if strings.Contains(err.Error(), "ConditionalCheckFailed") {
			return newApiError(http.StatusConflict, "Entry potentially already exists (conditional check failed)", err)
		}
		return newApiError(http.StatusInternalServerError, "Failed to create entry", err)
	}

	// Fetch the created entry to return the full object with generated fields
	createdEntry, err := h.EntryRepo.GetEntryByID(ctx.Request().Context(), userID, newEntry.EntryID)
	if err != nil {
		// Log the inconsistency, but maybe return the initial object? Or a simpler success message?
		log.Printf("WARN: Failed to fetch newly created entry %s: %v", newEntry.EntryID, err)
		// For now, return the data we have, converting the input date format
		newEntry.CreatedAt = time.Now() // Approximate
		newEntry.UpdatedAt = newEntry.CreatedAt
		return ctx.JSON(http.StatusCreated, entry.ToApiEntry(newEntry))
	}

	return ctx.JSON(http.StatusCreated, entry.ToApiEntry(*createdEntry))
}

func (h *ApiHandler) DeleteEntriesEntryId(ctx echo.Context, entryId openapi_types.UUID) error {
	userID, err := GetUserIDFromContext(ctx.Request().Context())
	if err != nil {
		return err
	}

	// Need EntryDate to delete. Get the entry first.
	e, err := h.EntryRepo.GetEntryByID(ctx.Request().Context(), userID, entryId)
	if err != nil {
		if errors.Is(err, domain.ErrEntryNotFound) {
			return newApiError(http.StatusNotFound, "Entry not found", err)
		}
		return newApiError(http.StatusInternalServerError, "Failed to retrieve entry before delete", err)
	}

	// Now delete using the retrieved date
	err = h.EntryRepo.DeleteEntry(ctx.Request().Context(), userID, entryId, e.EntryDate)
	if err != nil {
		if errors.Is(err, domain.ErrEntryNotFound) { // Should not happen if GetEntryByID succeeded, but check anyway
			return newApiError(http.StatusNotFound, "Entry not found", err)
		}
		return newApiError(http.StatusInternalServerError, "Failed to delete entry", err)
	}

	return ctx.NoContent(http.StatusNoContent)
}

func (h *ApiHandler) GetEntriesEntryId(ctx echo.Context, entryId openapi_types.UUID) error {
	userID, err := GetUserIDFromContext(ctx.Request().Context())
	if err != nil {
		return err
	}

	e, err := h.EntryRepo.GetEntryByID(ctx.Request().Context(), userID, entryId)
	if err != nil {
		if errors.Is(err, domain.ErrEntryNotFound) {
			return newApiError(http.StatusNotFound, "Entry not found", err)
		}
		return newApiError(http.StatusInternalServerError, "Failed to retrieve entry", err)
	}

	return ctx.JSON(http.StatusOK, entry.ToApiEntry(*e))
}

func (h *ApiHandler) PutEntriesEntryId(ctx echo.Context, entryId openapi_types.UUID) error {
	userID, err := GetUserIDFromContext(ctx.Request().Context())
	if err != nil {
		return err
	}

	var req api.UpdateEntryRequest
	if err := ctx.Bind(&req); err != nil {
		return newApiError(http.StatusBadRequest, "Invalid request body", err)
	}

	// Get existing entry to find ThemeID and original date (needed for update key if date changes)
	existingEntry, err := h.EntryRepo.GetEntryByID(ctx.Request().Context(), userID, entryId)
	if err != nil {
		if errors.Is(err, domain.ErrEntryNotFound) {
			return newApiError(http.StatusNotFound, "Entry not found", err)
		}
		return newApiError(http.StatusInternalServerError, "Failed to retrieve existing entry", err)
	}

	// Validate theme exists (it should, but check anyway)
	th, err := h.ThemeRepo.GetThemeByID(ctx.Request().Context(), userID, existingEntry.ThemeID)
	if err != nil {
		// This indicates data inconsistency if the entry existed but the theme doesn't
		log.Printf("ERROR: Entry %s references non-existent/inaccessible theme %s", entryId, existingEntry.ThemeID)
		if errors.Is(err, domain.ErrThemeNotFound) || errors.Is(err, domain.ErrForbidden) {
			return newApiError(http.StatusNotFound, "Associated theme not found or access denied", err)
		}
		return newApiError(http.StatusInternalServerError, "Failed to validate associated theme", err)
	}

	// Validate new data against theme fields
	if err := validateEntryData(req.Data, th.Fields); err != nil {
		return newApiError(http.StatusBadRequest, fmt.Sprintf("Entry data validation failed: %v", err), nil)
	}

	// Prepare updated entry model
	updatedEntry := entry.Entry{
		EntryID:   entryId,
		ThemeID:   existingEntry.ThemeID, // Theme cannot be changed
		UserID:    userID,
		EntryDate: req.EntryDate.Format("2006-01-02"), // Store as YYYY-MM-DD string
		Data:      req.Data,
		// CreatedAt, UpdatedAt, PK, SK, GSI keys handled by repository
	}

	// Pass the *original* date to UpdateEntry to find the item if the date might change
	// Assuming repo.UpdateEntry handles potential date changes (PK/SK modification)
	err = h.EntryRepo.UpdateEntry(ctx.Request().Context(), &updatedEntry)
	if err != nil {
		if errors.Is(err, domain.ErrEntryNotFound) {
			return newApiError(http.StatusNotFound, "Entry not found during update", err)
		}
		return newApiError(http.StatusInternalServerError, "Failed to update entry", err)
	}

	// Fetch the updated entry to return the full object
	// Use the *new* date if it could have changed
	finalEntry, err := h.EntryRepo.GetEntryByID(ctx.Request().Context(), userID, entryId)
	if err != nil {
		log.Printf("WARN: Failed to fetch updated entry %s: %v", entryId, err)
		// Return the data we sent for update as approximation
		updatedEntry.CreatedAt = existingEntry.CreatedAt // Keep original creation time
		updatedEntry.UpdatedAt = time.Now()              // Approximate
		return ctx.JSON(http.StatusOK, entry.ToApiEntry(updatedEntry))
	}

	return ctx.JSON(http.StatusOK, entry.ToApiEntry(*finalEntry))
}

// --- Theme Handlers ---

func (h *ApiHandler) GetThemes(ctx echo.Context) error {
	userID, err := GetUserIDFromContext(ctx.Request().Context())
	if err != nil {
		return err
	}

	themes, err := h.ThemeRepo.ListThemes(ctx.Request().Context(), userID)
	if err != nil {
		return newApiError(http.StatusInternalServerError, "Failed to retrieve themes", err)
	}

	apiThemes := make([]api.Theme, len(themes))
	for i, th := range themes {
		apiThemes[i] = theme.ToApiTheme(th)
	}

	return ctx.JSON(http.StatusOK, apiThemes)
}

func (h *ApiHandler) PostThemes(ctx echo.Context) error {
	userID, err := GetUserIDFromContext(ctx.Request().Context())
	if err != nil {
		return err
	}

	var req api.CreateThemeRequest
	if err := ctx.Bind(&req); err != nil {
		return newApiError(http.StatusBadRequest, "Invalid request body", err)
	}

	// Validate theme fields definition
	if err := validateThemeFields(req.Fields); err != nil {
		return newApiError(http.StatusBadRequest, fmt.Sprintf("Theme fields validation failed: %v", err), nil)
	}
	// Validate supported features (basic validation for now)
	if req.SupportedFeatures != nil {
		if err := validateSupportedFeatures(*req.SupportedFeatures); err != nil {
			return newApiError(http.StatusBadRequest, fmt.Sprintf("Supported features validation failed: %v", err), nil)
		}
	}

	modelFields := make([]theme.ThemeField, len(req.Fields))
	for i, f := range req.Fields {
		modelFields[i] = theme.FromApiThemeField(f)
	}

	// Handle supported features
	var supportedFeatures []string
	if req.SupportedFeatures != nil {
		supportedFeatures = *req.SupportedFeatures
	} else {
		supportedFeatures = []string{} // Default to empty list if nil
	}

	newTheme := theme.Theme{
		ThemeID:           uuid.New(), // Generate new ID
		ThemeName:         req.ThemeName,
		Fields:            modelFields,
		IsDefault:         false, // User-created themes are not default
		OwnerUserID:       &userID,
		SupportedFeatures: supportedFeatures,
		// CreatedAt, UpdatedAt, PK, SK set by repository
	}

	if err := h.ThemeRepo.CreateTheme(ctx.Request().Context(), &newTheme); err != nil {
		return newApiError(http.StatusInternalServerError, "Failed to create theme", err)
	}

	// Fetch the created theme to return the full object
	createdTheme, err := h.ThemeRepo.GetThemeByID(ctx.Request().Context(), userID, newTheme.ThemeID)
	if err != nil {
		log.Printf("WARN: Failed to fetch newly created theme %s: %v", newTheme.ThemeID, err)
		// Return the data we have, approximating timestamps
		newTheme.CreatedAt = time.Now()
		newTheme.UpdatedAt = newTheme.CreatedAt
		return ctx.JSON(http.StatusCreated, theme.ToApiTheme(newTheme))
	}

	return ctx.JSON(http.StatusCreated, theme.ToApiTheme(*createdTheme))
}

func (h *ApiHandler) DeleteThemesThemeId(ctx echo.Context, themeId openapi_types.UUID) error {
	userID, err := GetUserIDFromContext(ctx.Request().Context())
	if err != nil {
		return err
	}

	// Repository's DeleteTheme already checks ownership and if it's default
	err = h.ThemeRepo.DeleteTheme(ctx.Request().Context(), userID, themeId)
	if err != nil {
		if errors.Is(err, domain.ErrThemeNotFound) {
			return newApiError(http.StatusNotFound, "Theme not found", err)
		}
		if errors.Is(err, domain.ErrForbidden) || errors.Is(err, domain.ErrCannotDeleteDefaultTheme) {
			return newApiError(http.StatusForbidden, "Cannot delete this theme", err)
		}
		return newApiError(http.StatusInternalServerError, "Failed to delete theme", err)
	}

	return ctx.NoContent(http.StatusNoContent)
}

func (h *ApiHandler) GetThemesThemeId(ctx echo.Context, themeId openapi_types.UUID) error {
	userID, err := GetUserIDFromContext(ctx.Request().Context())
	if err != nil {
		return err
	}

	th, err := h.ThemeRepo.GetThemeByID(ctx.Request().Context(), userID, themeId)
	if err != nil {
		if errors.Is(err, domain.ErrThemeNotFound) || errors.Is(err, domain.ErrForbidden) {
			// Treat forbidden as not found from the user's perspective
			return newApiError(http.StatusNotFound, "Theme not found or access denied", err)
		}
		return newApiError(http.StatusInternalServerError, "Failed to retrieve theme", err)
	}

	return ctx.JSON(http.StatusOK, theme.ToApiTheme(*th))
}

func (h *ApiHandler) PutThemesThemeId(ctx echo.Context, themeId openapi_types.UUID) error {
	userID, err := GetUserIDFromContext(ctx.Request().Context())
	if err != nil {
		return err
	}

	var req api.UpdateThemeRequest
	if err := ctx.Bind(&req); err != nil {
		return newApiError(http.StatusBadRequest, "Invalid request body", err)
	}

	// Validate theme fields definition
	if err := validateThemeFields(req.Fields); err != nil {
		return newApiError(http.StatusBadRequest, fmt.Sprintf("Theme fields validation failed: %v", err), nil)
	}
	// Validate supported features
	if req.SupportedFeatures != nil {
		if err := validateSupportedFeatures(*req.SupportedFeatures); err != nil {
			return newApiError(http.StatusBadRequest, fmt.Sprintf("Supported features validation failed: %v", err), nil)
		}
	}

	// Check if theme exists, is owned by user, and is not default *before* updating
	existingTheme, err := h.ThemeRepo.GetThemeByID(ctx.Request().Context(), userID, themeId)
	if err != nil {
		if errors.Is(err, domain.ErrThemeNotFound) || errors.Is(err, domain.ErrForbidden) {
			return newApiError(http.StatusNotFound, "Theme not found or access denied", err)
		}
		return newApiError(http.StatusInternalServerError, "Failed to retrieve theme before update", err)
	}
	if existingTheme.IsDefault {
		return newApiError(http.StatusForbidden, "Cannot modify a default theme", nil)
	}

	modelFields := make([]theme.ThemeField, len(req.Fields))
	for i, f := range req.Fields {
		modelFields[i] = theme.FromApiThemeField(f)
	}

	// Handle supported features - if not provided in request, keep existing ones
	var supportedFeatures []string
	if req.SupportedFeatures != nil {
		supportedFeatures = *req.SupportedFeatures
	} else {
		supportedFeatures = existingTheme.SupportedFeatures // Keep existing if not provided
	}

	updatedTheme := theme.Theme{
		ThemeID:           themeId,
		ThemeName:         req.ThemeName,
		Fields:            modelFields,
		IsDefault:         false,
		OwnerUserID:       &userID,
		SupportedFeatures: supportedFeatures,
		// CreatedAt, UpdatedAt, PK, SK handled by repository
	}

	if err := h.ThemeRepo.UpdateTheme(ctx.Request().Context(), &updatedTheme); err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			// The repository's UpdateTheme now returns ErrForbidden for not found/default/not owner
			return newApiError(http.StatusForbidden, "Failed to update theme: not found, is default, or not owned by user", err)
		}
		return newApiError(http.StatusInternalServerError, "Failed to update theme", err)
	}

	// Fetch the updated theme to return the full object
	finalTheme, err := h.ThemeRepo.GetThemeByID(ctx.Request().Context(), userID, themeId)
	if err != nil {
		log.Printf("WARN: Failed to fetch updated theme %s: %v", themeId, err)
		updatedTheme.CreatedAt = existingTheme.CreatedAt
		updatedTheme.UpdatedAt = time.Now()
		return ctx.JSON(http.StatusOK, theme.ToApiTheme(updatedTheme))
	}

	return ctx.JSON(http.StatusOK, theme.ToApiTheme(*finalTheme))
}

// GetThemesThemeIdFeaturesFeatureName retrieves details about a specific feature supported by a theme.
// Placeholder implementation.
func (h *ApiHandler) GetThemesThemeIdFeaturesFeatureName(ctx echo.Context, themeId openapi_types.UUID, featureName string) error {
	userID, err := GetUserIDFromContext(ctx.Request().Context())
	if err != nil {
		return err
	}

	log.Printf("GetThemesThemeIdFeaturesFeatureName called for ThemeID: %s, Feature: %s, UserID: %s (Not Implemented)", themeId, featureName, userID)

	// In a real implementation:
	// 1. Get the theme by ID using h.ThemeRepo.GetThemeByID(ctx.Request().Context(), userID, themeId)
	// 2. Check if the theme exists and the user has access.
	// 3. Check if the requested featureName exists in the theme's SupportedFeatures list.
	// 4. If the feature exists, return details about it (currently, the spec might just need a 200 OK or specific feature details).
	// 5. If the theme or feature is not found, return appropriate errors (404 Not Found).

	// For now, return Not Implemented.
	return newApiError(http.StatusNotImplemented, fmt.Sprintf("Feature '%s' details not implemented for theme '%s'", featureName, themeId), nil)
}

// --- Validation Helpers ---

// validateSupportedFeatures performs basic validation on supported feature names.
func validateSupportedFeatures(features []string) error {
	validFeatures := map[string]bool{
		"monthly_summary":      true,
		"category_aggregation": true,
	}
	names := make(map[string]bool)
	for i, feature := range features {
		if feature == "" {
			return fmt.Errorf("feature %d: name cannot be empty", i)
		}
		if !isValidFeatureName(feature) {
			return fmt.Errorf("feature %d ('%s'): name contains invalid characters or format", i, feature)
		}
		if !validFeatures[feature] {
			log.Printf("WARN: Unsupported feature '%s' included in theme definition.", feature)
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
			return fmt.Errorf("field %d ('%s'): name contains invalid characters (allowed: a-z, 0-9, _)", i, field.Name)
		}
		if field.Label == "" {
			return fmt.Errorf("field %d ('%s'): label is required", i, field.Name)
		}
		if _, exists := names[field.Name]; exists {
			return fmt.Errorf("field name '%s' is duplicated", field.Name)
		}
		names[field.Name] = true

		isValidType := false
		// Corrected enum constant names based on generated code
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

// validateEntryData checks if the provided data matches the theme's field definitions.
func validateEntryData(data map[string]interface{}, fields []theme.ThemeField) error {
	definedFields := make(map[string]theme.ThemeField)
	for _, f := range fields {
		definedFields[f.Name] = f
	}

	// Check required fields are present and not empty
	for _, field := range fields {
		if field.Required {
			val, exists := data[field.Name]
			if !exists {
				return fmt.Errorf("required field '%s' is missing", field.Name)
			}
			// Check for nil or empty string for text-based types specifically
			if val == nil {
				return fmt.Errorf("required field '%s' cannot be null", field.Name)
			}
			if field.Type == theme.FieldTypeText || field.Type == theme.FieldTypeTextarea || field.Type == theme.FieldTypeSelect {
				if strVal, ok := val.(string); !ok || strVal == "" {
					return fmt.Errorf("required field '%s' cannot be empty", field.Name)
				}
			}
			// Add checks for other types if empty means something different (e.g., 0 for number?)
			// For now, just checking presence and non-nil is sufficient for non-string types.
		}
	}

	// Check types of provided data
	for key, value := range data {
		fieldDef, exists := definedFields[key]
		if !exists {
			// Allow extra fields? Or return error? Design decision. Let's return error for now.
			return fmt.Errorf("field '%s' is not defined in the theme", key)
		}

		// Skip validation if value is nil (unless it was required, checked above)
		if value == nil {
			continue
		}

		switch fieldDef.Type {
		case theme.FieldTypeText, theme.FieldTypeTextarea, theme.FieldTypeSelect:
			if _, ok := value.(string); !ok {
				return fmt.Errorf("field '%s' expects a string, got %T", key, value)
			}
		case theme.FieldTypeNumber:
			// DynamoDB typically unmarshals numbers as float64
			if _, ok := value.(float64); !ok {
				// Allow integers too? Let's be strict for now.
				return fmt.Errorf("field '%s' expects a number (float64), got %T", key, value)
			}
		case theme.FieldTypeBoolean:
			if _, ok := value.(bool); !ok {
				return fmt.Errorf("field '%s' expects a boolean, got %T", key, value)
			}
		case theme.FieldTypeDate:
			valStr, ok := value.(string)
			if !ok {
				return fmt.Errorf("field '%s' expects a date string (YYYY-MM-DD), got %T", key, value)
			}
			if _, err := time.Parse("2006-01-02", valStr); err != nil {
				return fmt.Errorf("field '%s' has invalid date format: %v. Expected YYYY-MM-DD", key, err)
			}
		case theme.FieldTypeDateTime: // Corrected case name from previous thought process if needed
			valStr, ok := value.(string)
			if !ok {
				return fmt.Errorf("field '%s' expects a datetime string (RFC3339), got %T", key, value)
			}
			// Use RFC3339 which is more common than RFC3339Nano unless nano precision is required by spec
			if _, err := time.Parse(time.RFC3339, valStr); err != nil {
				// Try RFC3339Nano as fallback? Or stick to one? Let's stick to RFC3339 for now.
				return fmt.Errorf("field '%s' has invalid datetime format: %v. Expected RFC3339 format (e.g., 2006-01-02T15:04:05Z07:00)", key, err)
			}
		default:
			// This case should ideally not be reached if theme validation is correct
			return fmt.Errorf("internal error: unknown field type '%s' for field '%s'", fieldDef.Type, key)
		}
	}

	return nil
}
