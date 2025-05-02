package handlers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/soranjiro/axicalendar/internal/api"
	"github.com/soranjiro/axicalendar/internal/models"
	"github.com/soranjiro/axicalendar/internal/repository"
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
	return echo.NewHTTPError(statusCode, api.Error{Message: message})
}

// ApiHandler implements the api.ServerInterface
type ApiHandler struct {
	EntryRepo repository.EntryRepository
	ThemeRepo repository.ThemeRepository
}

// NewApiHandler creates a new ApiHandler
func NewApiHandler(entryRepo repository.EntryRepository, themeRepo repository.ThemeRepository) *ApiHandler {
	return &ApiHandler{EntryRepo: entryRepo, ThemeRepo: themeRepo}
}

// --- Auth Handlers (Placeholder Implementations - Require Cognito) ---

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
	for i, entry := range entries {
		apiEntries[i] = models.ToApiEntry(entry)
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
	theme, err := h.ThemeRepo.GetThemeByID(ctx.Request().Context(), userID, req.ThemeId)
	if err != nil {
		if errors.Is(err, errors.New("theme not found")) || errors.Is(err, errors.New("forbidden")) {
			return newApiError(http.StatusNotFound, "Theme not found or access denied", err)
		}
		return newApiError(http.StatusInternalServerError, "Failed to validate theme", err)
	}

	// Validate data against theme fields
	if err := validateEntryData(req.Data, theme.Fields); err != nil {
		return newApiError(http.StatusBadRequest, fmt.Sprintf("Entry data validation failed: %v", err), nil)
	}

	newEntry := models.Entry{
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
		return ctx.JSON(http.StatusCreated, models.ToApiEntry(newEntry))
	}

	return ctx.JSON(http.StatusCreated, models.ToApiEntry(*createdEntry))
}

func (h *ApiHandler) DeleteEntriesEntryId(ctx echo.Context, entryId openapi_types.UUID) error {
	userID, err := GetUserIDFromContext(ctx.Request().Context())
	if err != nil {
		return err
	}

	// Need EntryDate to delete. Get the entry first.
	entry, err := h.EntryRepo.GetEntryByID(ctx.Request().Context(), userID, entryId)
	if err != nil {
		if errors.Is(err, errors.New("entry not found")) {
			return newApiError(http.StatusNotFound, "Entry not found", err)
		}
		return newApiError(http.StatusInternalServerError, "Failed to retrieve entry before delete", err)
	}

	// Now delete using the retrieved date
	err = h.EntryRepo.DeleteEntry(ctx.Request().Context(), userID, entryId, entry.EntryDate)
	if err != nil {
		if errors.Is(err, errors.New("entry not found")) { // Should not happen if GetEntryByID succeeded, but check anyway
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

	entry, err := h.EntryRepo.GetEntryByID(ctx.Request().Context(), userID, entryId)
	if err != nil {
		if errors.Is(err, errors.New("entry not found")) {
			return newApiError(http.StatusNotFound, "Entry not found", err)
		}
		return newApiError(http.StatusInternalServerError, "Failed to retrieve entry", err)
	}

	return ctx.JSON(http.StatusOK, models.ToApiEntry(*entry))
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
		if errors.Is(err, errors.New("entry not found")) {
			return newApiError(http.StatusNotFound, "Entry not found", err)
		}
		return newApiError(http.StatusInternalServerError, "Failed to retrieve existing entry", err)
	}

	// Validate theme exists (it should, but check anyway)
	theme, err := h.ThemeRepo.GetThemeByID(ctx.Request().Context(), userID, existingEntry.ThemeID)
	if err != nil {
		// This indicates data inconsistency if the entry existed but the theme doesn't
		log.Printf("ERROR: Entry %s references non-existent/inaccessible theme %s", entryId, existingEntry.ThemeID)
		return newApiError(http.StatusNotFound, "Associated theme not found or access denied", err)
	}

	// Validate new data against theme fields
	if err := validateEntryData(req.Data, theme.Fields); err != nil {
		return newApiError(http.StatusBadRequest, fmt.Sprintf("Entry data validation failed: %v", err), nil)
	}

	// Prepare updated entry model
	updatedEntry := models.Entry{
		EntryID:   entryId,
		ThemeID:   existingEntry.ThemeID, // Theme cannot be changed
		UserID:    userID,
		EntryDate: req.EntryDate.Format("2006-01-02"), // Store as YYYY-MM-DD string
		Data:      req.Data,
		// CreatedAt, UpdatedAt, PK, SK, GSI keys handled by repository
	}

	// Pass the *original* date to UpdateEntry to find the item if the date might change
	// Assuming repository.UpdateEntry handles potential date changes (PK/SK modification)
	err = h.EntryRepo.UpdateEntry(ctx.Request().Context(), &updatedEntry)
	if err != nil {
		if errors.Is(err, errors.New("entry not found")) {
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
		return ctx.JSON(http.StatusOK, models.ToApiEntry(updatedEntry))
	}

	return ctx.JSON(http.StatusOK, models.ToApiEntry(*finalEntry))
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
	for i, theme := range themes {
		apiThemes[i] = models.ToApiTheme(theme)
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

	modelFields := make([]models.ThemeField, len(req.Fields))
	for i, f := range req.Fields {
		modelFields[i] = models.FromApiThemeField(f)
	}

	newTheme := models.Theme{
		ThemeID:   uuid.New(), // Generate new ID
		ThemeName: req.ThemeName,
		Fields:    modelFields,
		IsDefault: false, // User-created themes are not default
		UserID:    &userID,
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
		return ctx.JSON(http.StatusCreated, models.ToApiTheme(newTheme))
	}

	return ctx.JSON(http.StatusCreated, models.ToApiTheme(*createdTheme))
}

func (h *ApiHandler) DeleteThemesThemeId(ctx echo.Context, themeId openapi_types.UUID) error {
	userID, err := GetUserIDFromContext(ctx.Request().Context())
	if err != nil {
		return err
	}

	// Repository's DeleteTheme already checks ownership and if it's default
	err = h.ThemeRepo.DeleteTheme(ctx.Request().Context(), userID, themeId)
	if err != nil {
		if errors.Is(err, errors.New("theme not found")) {
			return newApiError(http.StatusNotFound, "Theme not found", err)
		}
		if errors.Is(err, errors.New("forbidden")) || errors.Is(err, errors.New("cannot delete default theme")) {
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

	theme, err := h.ThemeRepo.GetThemeByID(ctx.Request().Context(), userID, themeId)
	if err != nil {
		if errors.Is(err, errors.New("theme not found")) || errors.Is(err, errors.New("forbidden")) {
			// Treat forbidden as not found from the user's perspective
			return newApiError(http.StatusNotFound, "Theme not found or access denied", err)
		}
		return newApiError(http.StatusInternalServerError, "Failed to retrieve theme", err)
	}

	return ctx.JSON(http.StatusOK, models.ToApiTheme(*theme))
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

	// Check if theme exists, is owned by user, and is not default *before* updating
	existingTheme, err := h.ThemeRepo.GetThemeByID(ctx.Request().Context(), userID, themeId)
	if err != nil {
		if errors.Is(err, errors.New("theme not found")) || errors.Is(err, errors.New("forbidden")) {
			return newApiError(http.StatusNotFound, "Theme not found or access denied", err)
		}
		return newApiError(http.StatusInternalServerError, "Failed to retrieve theme before update", err)
	}
	if existingTheme.IsDefault {
		return newApiError(http.StatusForbidden, "Cannot modify a default theme", nil)
	}
	// GetThemeByID already checks ownership for non-default themes

	modelFields := make([]models.ThemeField, len(req.Fields))
	for i, f := range req.Fields {
		modelFields[i] = models.FromApiThemeField(f)
	}

	updatedTheme := models.Theme{
		ThemeID:   themeId,
		ThemeName: req.ThemeName,
		Fields:    modelFields,
		IsDefault: false, // Should already be false
		UserID:    &userID,
		// CreatedAt, UpdatedAt, PK, SK handled by repository
	}

	if err := h.ThemeRepo.UpdateTheme(ctx.Request().Context(), &updatedTheme); err != nil {
		// Handle potential conditional check failure (not found)
		if strings.Contains(err.Error(), "ConditionalCheckFailed") {
			return newApiError(http.StatusNotFound, "Theme not found during update (conditional check failed)", err)
		}
		return newApiError(http.StatusInternalServerError, "Failed to update theme", err)
	}

	// Fetch the updated theme to return the full object
	finalTheme, err := h.ThemeRepo.GetThemeByID(ctx.Request().Context(), userID, themeId)
	if err != nil {
		log.Printf("WARN: Failed to fetch updated theme %s: %v", themeId, err)
		// Return the data we sent for update as approximation
		updatedTheme.CreatedAt = existingTheme.CreatedAt // Keep original
		updatedTheme.UpdatedAt = time.Now()              // Approximate
		return ctx.JSON(http.StatusOK, models.ToApiTheme(updatedTheme))
	}

	return ctx.JSON(http.StatusOK, models.ToApiTheme(*finalTheme))
}

// --- Validation Helpers ---

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
		// Basic check for valid characters (adjust regex as needed)
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

		// Validate type
		isValidType := false
		validTypes := []models.ThemeFieldType{
			models.FieldTypeText, models.FieldTypeDate, models.FieldTypeDateTime,
			models.FieldTypeNumber, models.FieldTypeBoolean, models.FieldTypeTextarea,
			models.FieldTypeSelect,
		}
		for _, vt := range validTypes {
			if models.ThemeFieldType(field.Type) == vt {
				isValidType = true
				break
			}
		}
		if !isValidType {
			return fmt.Errorf("field '%s': invalid type '%s'", field.Name, field.Type)
		}
		// Add more specific validation if needed (e.g., options for select)
	}
	return nil
}

// isValidFieldName checks if a field name is valid (e.g., snake_case).
func isValidFieldName(name string) bool {
	// Simple check: allow lowercase letters, numbers, and underscore. Must start with letter.
	if name == "" || (name[0] < 'a' || name[0] > 'z') {
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
func validateEntryData(data map[string]interface{}, fields []models.ThemeField) error {
	definedFields := make(map[string]models.ThemeField)
	for _, f := range fields {
		definedFields[f.Name] = f
	}

	// Check required fields are present
	for _, field := range fields {
		if field.Required {
			val, exists := data[field.Name]
			if !exists {
				return fmt.Errorf("required field '%s' is missing", field.Name)
			}
			// Also check if the value is considered "empty" (e.g., empty string for text)
			if val == nil || (field.Type == models.FieldTypeText && val == "") { // Add checks for other types if needed
				return fmt.Errorf("required field '%s' cannot be empty", field.Name)
			}
		}
	}

	// Check provided data fields exist in theme and have correct type (basic check)
	for key, value := range data {
		fieldDef, exists := definedFields[key]
		if !exists {
			return fmt.Errorf("field '%s' is not defined in the theme", key)
		}

		// Skip validation for nil values unless the field is required (handled above)
		if value == nil {
			continue
		}

		// Basic type validation (can be expanded)
		switch fieldDef.Type {
		case models.FieldTypeText, models.FieldTypeTextarea, models.FieldTypeSelect:
			if _, ok := value.(string); !ok {
				return fmt.Errorf("field '%s' expects a string, got %T", key, value)
			}
		case models.FieldTypeNumber:
			// DynamoDB might store numbers as float64 when unmarshalled into interface{}
			if _, ok := value.(float64); !ok {
				// Allow int as well? For simplicity, expect float64 from JSON unmarshal
				if _, okInt := value.(int); !okInt {
					return fmt.Errorf("field '%s' expects a number, got %T", key, value)
				}
			}
		case models.FieldTypeBoolean:
			if _, ok := value.(bool); !ok {
				return fmt.Errorf("field '%s' expects a boolean, got %T", key, value)
			}
		case models.FieldTypeDate:
			// Expect YYYY-MM-DD string format from JSON
			valStr, ok := value.(string)
			if !ok {
				return fmt.Errorf("field '%s' expects a date string (YYYY-MM-DD), got %T", key, value)
			}
			if _, err := time.Parse("2006-01-02", valStr); err != nil {
				return fmt.Errorf("field '%s' has invalid date format: %v", key, err)
			}
		case models.FieldTypeDateTime:
			// Expect RFC3339 string format from JSON
			valStr, ok := value.(string)
			if !ok {
				return fmt.Errorf("field '%s' expects a datetime string (RFC3339), got %T", key, value)
			}
			if _, err := time.Parse(time.RFC3339Nano, valStr); err != nil { // Use RFC3339Nano for flexibility
				if _, err2 := time.Parse(time.RFC3339, valStr); err2 != nil {
					return fmt.Errorf("field '%s' has invalid datetime format (expected RFC3339/RFC3339Nano): %v", key, err)
				}
			}
		default:
			return fmt.Errorf("internal error: unknown field type '%s' for field '%s'", fieldDef.Type, key)
		}
	}

	return nil
}
