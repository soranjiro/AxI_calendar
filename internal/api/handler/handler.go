package handler

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/soranjiro/axicalendar/internal/api"
	"github.com/soranjiro/axicalendar/internal/usecase"

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

// ApiHandler implements the api.ServerInterface using a single UseCaseInterface
type ApiHandler struct {
	useCase usecase.UseCaseInterface // Use the consolidated interface
}

// NewApiHandler creates a new ApiHandler with the injected use case service
func NewApiHandler(uc usecase.UseCaseInterface) *ApiHandler {
	return &ApiHandler{
		useCase: uc,
	}
}

// --- Health Check ---
func (h *ApiHandler) GetHealth(ctx echo.Context) error {
	// Perform a simple health check
	// In a real application, you might check database connections, external services, etc.
	// For now, just return a 200 OK with a simple message.
	statusOK := "OK"
	return ctx.JSON(http.StatusOK, api.HealthCheckResponse{Status: &statusOK})
}

// --- Auth Handlers ---

// GetAuthMe retrieves information about the currently authenticated user.
func (h *ApiHandler) GetAuthMe(ctx echo.Context) error {
	userID, err := GetUserIDFromContext(ctx.Request().Context())
	if err != nil {
		return err // Already formatted echo.HTTPError
	}

	user, err := h.useCase.GetAuthMe(ctx.Request().Context(), userID) // Call method on useCase
	if err != nil {
		// Check if the error is already an echo.HTTPError from the use case
		var httpErr *echo.HTTPError
		if errors.As(err, &httpErr) {
			return httpErr // Return the error directly
		}
		// Otherwise, wrap it as a generic internal server error
		return newApiError(http.StatusInternalServerError, "Failed to get user details", err)
	}

	log.Printf("GetAuthMe called for UserID: %s", userID.String())
	return ctx.JSON(http.StatusOK, *user) // Dereference pointer from use case
}

// PostAuthConfirmForgotPassword handles the confirmation of a password reset.
// Placeholder implementation.
func (h *ApiHandler) PostAuthConfirmForgotPassword(ctx echo.Context) error {
	log.Println("PostAuthConfirmForgotPassword called (Not Implemented)")
	return newApiError(http.StatusNotImplemented, "Confirm Forgot Password not implemented", nil)
}

// PostAuthConfirmSignup handles the confirmation of a user signup.
// Placeholder implementation.
func (h *ApiHandler) PostAuthConfirmSignup(ctx echo.Context) error {
	log.Println("PostAuthConfirmSignup called (Not Implemented)")
	return newApiError(http.StatusNotImplemented, "Confirm Signup not implemented", nil)
}

// PostAuthForgotPassword initiates the password reset process.
// Placeholder implementation.
func (h *ApiHandler) PostAuthForgotPassword(ctx echo.Context) error {
	log.Println("PostAuthForgotPassword called (Not Implemented)")
	return newApiError(http.StatusNotImplemented, "Forgot Password not implemented", nil)
}

// PostAuthLogout handles user logout.
// Placeholder implementation.
func (h *ApiHandler) PostAuthLogout(ctx echo.Context) error {
	log.Println("PostAuthLogout called (Not Implemented)")
	return newApiError(http.StatusNotImplemented, "Logout not implemented", nil)
}

// PostAuthRefresh handles token refresh requests.
// Placeholder implementation.
func (h *ApiHandler) PostAuthRefresh(ctx echo.Context) error {
	log.Println("PostAuthRefresh called (Not Implemented)")
	return newApiError(http.StatusNotImplemented, "Token Refresh not implemented", nil)
}

func (h *ApiHandler) PostAuthLogin(ctx echo.Context) error {
	log.Println("Auth Login endpoint called (Not Implemented)")
	return newApiError(http.StatusNotImplemented, "Login not implemented", nil)
}

func (h *ApiHandler) PostAuthSignup(ctx echo.Context) error {
	log.Println("Auth Signup endpoint called (Not Implemented)")
	return newApiError(http.StatusNotImplemented, "Signup not implemented", nil)
}

// --- Entry Handlers ---

func (h *ApiHandler) GetEntries(ctx echo.Context, params api.GetEntriesParams) error {
	userID, err := GetUserIDFromContext(ctx.Request().Context())
	if err != nil {
		return err // Error already formatted
	}

	// Call the use case method
	entries, err := h.useCase.GetEntries(ctx.Request().Context(), userID, params)
	if err != nil {
		var httpErr *echo.HTTPError
		if errors.As(err, &httpErr) {
			return httpErr // Return the error directly from use case
		}
		return newApiError(http.StatusInternalServerError, "Failed to retrieve entries", err)
	}

	// Use case now returns []api.Entry directly
	return ctx.JSON(http.StatusOK, entries)
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

	// Call the use case method
	createdEntry, err := h.useCase.CreateEntry(ctx.Request().Context(), userID, req)
	if err != nil {
		var httpErr *echo.HTTPError
		if errors.As(err, &httpErr) {
			return httpErr // Return the error directly from use case
		}
		return newApiError(http.StatusInternalServerError, "Failed to create entry", err)
	}

	// Use case returns *api.Entry
	return ctx.JSON(http.StatusCreated, *createdEntry)
}

func (h *ApiHandler) DeleteEntriesEntryId(ctx echo.Context, entryId openapi_types.UUID) error {
	userID, err := GetUserIDFromContext(ctx.Request().Context())
	if err != nil {
		return err
	}

	// Call the use case method
	err = h.useCase.DeleteEntry(ctx.Request().Context(), userID, entryId)
	if err != nil {
		var httpErr *echo.HTTPError
		if errors.As(err, &httpErr) {
			return httpErr // Return the error directly from use case
		}
		// Check for specific domain errors if needed, otherwise generic
		// Note: Use case now returns echo.HTTPError directly for known cases like 404
		return newApiError(http.StatusInternalServerError, "Failed to delete entry", err)
	}

	return ctx.NoContent(http.StatusNoContent)
}

func (h *ApiHandler) GetEntriesEntryId(ctx echo.Context, entryId openapi_types.UUID) error {
	userID, err := GetUserIDFromContext(ctx.Request().Context())
	if err != nil {
		return err
	}

	// Call the use case method
	entry, err := h.useCase.GetEntryByID(ctx.Request().Context(), userID, entryId)
	if err != nil {
		var httpErr *echo.HTTPError
		if errors.As(err, &httpErr) {
			return httpErr // Return the error directly from use case
		}
		// Note: Use case now returns echo.HTTPError directly for known cases like 404
		return newApiError(http.StatusInternalServerError, "Failed to retrieve entry", err)
	}

	// Use case returns *api.Entry
	return ctx.JSON(http.StatusOK, *entry)
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

	// Call the use case method
	updatedEntry, err := h.useCase.UpdateEntry(ctx.Request().Context(), userID, entryId, req)
	if err != nil {
		var httpErr *echo.HTTPError
		if errors.As(err, &httpErr) {
			return httpErr // Return the error directly from use case
		}
		// Note: Use case now returns echo.HTTPError directly for known cases like 400, 404
		return newApiError(http.StatusInternalServerError, "Failed to update entry", err)
	}

	// Use case returns *api.Entry
	return ctx.JSON(http.StatusOK, *updatedEntry)
}

// --- Theme Handlers ---

func (h *ApiHandler) GetThemes(ctx echo.Context) error {
	userID, err := GetUserIDFromContext(ctx.Request().Context())
	if err != nil {
		return err
	}

	// Call the use case method
	themes, err := h.useCase.GetThemes(ctx.Request().Context(), userID)
	if err != nil {
		var httpErr *echo.HTTPError
		if errors.As(err, &httpErr) {
			return httpErr // Return the error directly from use case
		}
		return newApiError(http.StatusInternalServerError, "Failed to retrieve themes", err)
	}

	// Use case returns []api.Theme
	return ctx.JSON(http.StatusOK, themes)
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

	// Call the use case method
	createdTheme, err := h.useCase.CreateTheme(ctx.Request().Context(), userID, req)
	if err != nil {
		var httpErr *echo.HTTPError
		if errors.As(err, &httpErr) {
			return httpErr // Return the error directly from use case
		}
		// Note: Use case now returns echo.HTTPError directly for known cases like 400
		return newApiError(http.StatusInternalServerError, "Failed to create theme", err)
	}

	// Use case returns *api.Theme
	return ctx.JSON(http.StatusCreated, *createdTheme)
}

func (h *ApiHandler) DeleteThemesThemeId(ctx echo.Context, themeId openapi_types.UUID) error {
	userID, err := GetUserIDFromContext(ctx.Request().Context())
	if err != nil {
		return err
	}

	// Call the use case method
	err = h.useCase.DeleteTheme(ctx.Request().Context(), userID, themeId)
	if err != nil {
		var httpErr *echo.HTTPError
		if errors.As(err, &httpErr) {
			return httpErr // Return the error directly from use case (e.g., 403, 404)
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

	// Call the use case method
	theme, err := h.useCase.GetThemeByID(ctx.Request().Context(), userID, themeId)
	if err != nil {
		var httpErr *echo.HTTPError
		if errors.As(err, &httpErr) {
			return httpErr // Return the error directly from use case (e.g., 404)
		}
		return newApiError(http.StatusInternalServerError, "Failed to retrieve theme", err)
	}

	// Use case returns *api.Theme
	return ctx.JSON(http.StatusOK, *theme)
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

	// Call the use case method
	updatedTheme, err := h.useCase.UpdateTheme(ctx.Request().Context(), userID, themeId, req)
	if err != nil {
		var httpErr *echo.HTTPError
		if errors.As(err, &httpErr) {
			return httpErr // Return the error directly from use case (e.g., 400, 403, 404)
		}
		return newApiError(http.StatusInternalServerError, "Failed to update theme", err)
	}

	// Use case returns *api.Theme
	return ctx.JSON(http.StatusOK, *updatedTheme)
}

// GetThemesThemeIdFeaturesFeatureName retrieves details about a specific feature supported by a theme.
// Placeholder implementation.
func (h *ApiHandler) GetThemesThemeIdFeaturesFeatureName(ctx echo.Context, themeId openapi_types.UUID, featureName string) error {
	userID, err := GetUserIDFromContext(ctx.Request().Context())
	if err != nil {
		return err
	}

	log.Printf("GetThemesThemeIdFeaturesFeatureName called for ThemeID: %s, Feature: %s, UserID: %s (Not Implemented - Requires Feature Use Case)", themeId, featureName, userID)

	// TODO: Implement a specific use case method for getting feature details if needed.
	// featureDetails, err := h.useCase.GetThemeFeature(ctx.Request().Context(), userID, themeId, featureName)
	// Handle errors and return response...

	return newApiError(http.StatusNotImplemented, fmt.Sprintf("Feature '%s' details not implemented for theme '%s'", featureName, themeId), nil)
}
