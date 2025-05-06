package handler

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/soranjiro/axicalendar/internal/interfaces/api"
	"github.com/soranjiro/axicalendar/internal/application"

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

	// Call use case which returns domain user
	domainUser, err := h.useCase.GetAuthMe(ctx.Request().Context(), userID)
	if err != nil {
		var httpErr *echo.HTTPError
		if errors.As(err, &httpErr) {
			return httpErr // Return the error directly
		}
		return newApiError(http.StatusInternalServerError, "Failed to get user details", err)
	}

	// Convert domain user to API user
	apiUser := ToApiUser(*domainUser) // Use converter

	log.Printf("GetAuthMe called for UserID: %s", userID.String())
	return ctx.JSON(http.StatusOK, apiUser)
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

	// Call the use case method, which returns domain entries
	domainEntries, err := h.useCase.GetEntries(ctx.Request().Context(), userID, params)
	if err != nil {
		var httpErr *echo.HTTPError
		if errors.As(err, &httpErr) {
			return httpErr // Return the error directly from use case
		}
		return newApiError(http.StatusInternalServerError, "Failed to retrieve entries", err)
	}

	// Convert domain entries to API entries
	apiEntries, err := ToApiEntries(domainEntries) // Use converter
	if err != nil {
		log.Printf("Error converting domain entries to API format: %v", err)
		return newApiError(http.StatusInternalServerError, "Failed to format entries response", err)
	}

	return ctx.JSON(http.StatusOK, apiEntries)
}

func (h *ApiHandler) PostEntries(ctx echo.Context) error {
	userID, err := GetUserIDFromContext(ctx.Request().Context())
	if err != nil {
		return err
	}

	var apiReq api.CreateEntryRequest
	if err := ctx.Bind(&apiReq); err != nil {
		return newApiError(http.StatusBadRequest, "Invalid request body", err)
	}

	// Convert API request to domain entry
	domainEntry, err := FromApiCreateEntryRequest(apiReq, userID) // Use converter
	if err != nil {
		return newApiError(http.StatusBadRequest, "Invalid entry data format", err)
	}

	// Call the use case method with domain entry
	createdDomainEntry, err := h.useCase.CreateEntry(ctx.Request().Context(), domainEntry)
	if err != nil {
		var httpErr *echo.HTTPError
		if errors.As(err, &httpErr) {
			return httpErr // Return the error directly from use case
		}
		return newApiError(http.StatusInternalServerError, "Failed to create entry", err)
	}

	// Convert created domain entry back to API entry
	apiEntry, err := ToApiEntry(*createdDomainEntry) // Use converter
	if err != nil {
		log.Printf("Error converting created domain entry to API format: %v", err)
		return newApiError(http.StatusInternalServerError, "Failed to format created entry response", err)
	}

	return ctx.JSON(http.StatusCreated, apiEntry)
}

func (h *ApiHandler) DeleteEntriesEntryId(ctx echo.Context, entryId openapi_types.UUID) error {
	userID, err := GetUserIDFromContext(ctx.Request().Context())
	if err != nil {
		return err
	}

	// Call the use case method (no conversion needed for IDs)
	err = h.useCase.DeleteEntry(ctx.Request().Context(), userID, entryId)
	if err != nil {
		var httpErr *echo.HTTPError
		if errors.As(err, &httpErr) {
			return httpErr // Return the error directly from use case
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

	// Call the use case method, returns domain entry
	domainEntry, err := h.useCase.GetEntryByID(ctx.Request().Context(), userID, entryId)
	if err != nil {
		var httpErr *echo.HTTPError
		if errors.As(err, &httpErr) {
			return httpErr // Return the error directly from use case
		}
		return newApiError(http.StatusInternalServerError, "Failed to retrieve entry", err)
	}

	// Convert domain entry to API entry
	apiEntry, err := ToApiEntry(*domainEntry) // Use converter
	if err != nil {
		log.Printf("Error converting domain entry to API format: %v", err)
		return newApiError(http.StatusInternalServerError, "Failed to format entry response", err)
	}

	return ctx.JSON(http.StatusOK, apiEntry)
}

func (h *ApiHandler) PutEntriesEntryId(ctx echo.Context, entryId openapi_types.UUID) error {
	userID, err := GetUserIDFromContext(ctx.Request().Context())
	if err != nil {
		return err
	}

	var apiReq api.UpdateEntryRequest
	if err := ctx.Bind(&apiReq); err != nil {
		return newApiError(http.StatusBadRequest, "Invalid request body", err)
	}

	updatedDomainEntry, err := h.useCase.UpdateEntry(ctx.Request().Context(), userID, entryId, apiReq)
	if err != nil {
		var httpErr *echo.HTTPError
		if errors.As(err, &httpErr) {
			return httpErr // Return the error directly from use case
		}
		return newApiError(http.StatusInternalServerError, "Failed to update entry", err)
	}

	// Convert updated domain entry back to API entry
	apiEntry, err := ToApiEntry(*updatedDomainEntry) // Use converter
	if err != nil {
		log.Printf("Error converting updated domain entry to API format: %v", err)
		return newApiError(http.StatusInternalServerError, "Failed to format updated entry response", err)
	}

	return ctx.JSON(http.StatusOK, apiEntry)
}

// --- Theme Handlers ---

func (h *ApiHandler) GetThemes(ctx echo.Context) error {
	userID, err := GetUserIDFromContext(ctx.Request().Context())
	if err != nil {
		return err
	}

	// Call the use case method, returns domain themes
	domainThemes, err := h.useCase.GetThemes(ctx.Request().Context(), userID)
	if err != nil {
		var httpErr *echo.HTTPError
		if errors.As(err, &httpErr) {
			return httpErr // Return the error directly from use case
		}
		return newApiError(http.StatusInternalServerError, "Failed to retrieve themes", err)
	}

	// Convert domain themes to API themes
	apiThemes, err := ToApiThemes(domainThemes) // Use converter
	if err != nil {
		log.Printf("Error converting domain themes to API format: %v", err)
		return newApiError(http.StatusInternalServerError, "Failed to format themes response", err)
	}

	return ctx.JSON(http.StatusOK, apiThemes)
}

func (h *ApiHandler) PostThemes(ctx echo.Context) error {
	userID, err := GetUserIDFromContext(ctx.Request().Context())
	if err != nil {
		return err
	}

	var apiReq api.CreateThemeRequest
	if err := ctx.Bind(&apiReq); err != nil {
		return newApiError(http.StatusBadRequest, "Invalid request body", err)
	}

	// Convert API request to domain theme
	domainTheme, err := FromApiCreateThemeRequest(apiReq, userID) // Use converter
	if err != nil {
		return newApiError(http.StatusBadRequest, "Invalid theme data format", err)
	}

	// Call the use case method with domain theme
	createdDomainTheme, err := h.useCase.CreateTheme(ctx.Request().Context(), domainTheme)
	if err != nil {
		var httpErr *echo.HTTPError
		if errors.As(err, &httpErr) {
			return httpErr // Return the error directly from use case
		}
		return newApiError(http.StatusInternalServerError, "Failed to create theme", err)
	}

	// Convert created domain theme back to API theme
	apiTheme, err := ToApiTheme(*createdDomainTheme) // Use converter
	if err != nil {
		log.Printf("Error converting created domain theme to API format: %v", err)
		return newApiError(http.StatusInternalServerError, "Failed to format created theme response", err)
	}

	return ctx.JSON(http.StatusCreated, apiTheme)
}

func (h *ApiHandler) DeleteThemesThemeId(ctx echo.Context, themeId openapi_types.UUID) error {
	userID, err := GetUserIDFromContext(ctx.Request().Context())
	if err != nil {
		return err
	}

	// Call the use case method (no conversion needed for IDs)
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

	// Call the use case method, returns domain theme
	domainTheme, err := h.useCase.GetThemeByID(ctx.Request().Context(), userID, themeId)
	if err != nil {
		var httpErr *echo.HTTPError
		if errors.As(err, &httpErr) {
			return httpErr // Return the error directly from use case (e.g., 404)
		}
		return newApiError(http.StatusInternalServerError, "Failed to retrieve theme", err)
	}

	// Convert domain theme to API theme
	apiTheme, err := ToApiTheme(*domainTheme) // Use converter
	if err != nil {
		log.Printf("Error converting domain theme to API format: %v", err)
		return newApiError(http.StatusInternalServerError, "Failed to format theme response", err)
	}

	return ctx.JSON(http.StatusOK, apiTheme)
}

func (h *ApiHandler) PutThemesThemeId(ctx echo.Context, themeId openapi_types.UUID) error {
	userID, err := GetUserIDFromContext(ctx.Request().Context())
	if err != nil {
		return err
	}

	var apiReq api.UpdateThemeRequest
	if err := ctx.Bind(&apiReq); err != nil {
		return newApiError(http.StatusBadRequest, "Invalid request body", err)
	}

	// 1. Get existing theme (needed for conversion and validation)
	existingDomainTheme, err := h.useCase.GetThemeByID(ctx.Request().Context(), userID, themeId)
	if err != nil {
		var httpErr *echo.HTTPError
		if errors.As(err, &httpErr) {
			// If GetThemeByID returns 404, the theme doesn't exist or isn't accessible
			if httpErr.Code == http.StatusNotFound {
				return newApiError(http.StatusNotFound, "Theme not found or access denied", nil)
			}
			return httpErr // Return other errors from GetThemeByID
		}
		return newApiError(http.StatusInternalServerError, "Failed to retrieve theme before update", err)
	}
	// Check if it's a default theme (cannot be modified)
	if existingDomainTheme.IsDefault {
		return newApiError(http.StatusForbidden, "Cannot modify a default theme", nil)
	}

	// 2. Convert API request to domain theme using the existing theme data
	domainThemeUpdate, err := FromApiUpdateThemeRequest(apiReq, themeId, userID, *existingDomainTheme)
	if err != nil {
		return newApiError(http.StatusBadRequest, "Invalid theme data format", err)
	}

	// 3. Call use case with the converted domain theme object
	updatedDomainTheme, err := h.useCase.UpdateTheme(ctx.Request().Context(), userID, themeId, domainThemeUpdate) // Pass domain object
	if err != nil {
		var httpErr *echo.HTTPError
		if errors.As(err, &httpErr) {
			return httpErr // Return the error directly from use case (e.g., 400, 403, 404)
		}
		return newApiError(http.StatusInternalServerError, "Failed to update theme", err)
	}

	// Convert updated domain theme back to API theme
	apiTheme, err := ToApiTheme(*updatedDomainTheme) // Use converter
	if err != nil {
		log.Printf("Error converting updated domain theme to API format: %v", err)
		return newApiError(http.StatusInternalServerError, "Failed to format updated theme response", err)
	}

	return ctx.JSON(http.StatusOK, apiTheme)
}

// GetThemesThemeIdFeaturesFeatureName retrieves details about a specific feature supported by a theme.
// Placeholder implementation.
func (h *ApiHandler) GetThemesThemeIdFeaturesFeatureName(ctx echo.Context, themeId openapi_types.UUID, featureName string) error {
	userID, err := GetUserIDFromContext(ctx.Request().Context())
	if err != nil {
		return err
	}

	log.Printf("GetThemesThemeIdFeaturesFeatureName called for ThemeID: %s, Feature: %s, UserID: %s (Not Implemented - Requires Feature Use Case)", themeId, featureName, userID)

	return newApiError(http.StatusNotImplemented, fmt.Sprintf("Feature '%s' details not implemented for theme '%s'", featureName, themeId), nil)
}
