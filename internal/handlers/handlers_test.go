package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/soranjiro/axicalendar/internal/api"
	"github.com/soranjiro/axicalendar/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- Mocks ---

type MockEntryRepository struct {
	mock.Mock
}

func (m *MockEntryRepository) GetEntryByID(ctx context.Context, userID uuid.UUID, entryID uuid.UUID) (*models.Entry, error) {
	args := m.Called(ctx, userID, entryID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Entry), args.Error(1)
}

func (m *MockEntryRepository) ListEntriesByDateRange(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time, themeIDs []uuid.UUID) ([]models.Entry, error) {
	args := m.Called(ctx, userID, startDate, endDate, themeIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Entry), args.Error(1)
}

func (m *MockEntryRepository) CreateEntry(ctx context.Context, entry *models.Entry) error {
	args := m.Called(ctx, entry)
	return args.Error(0)
}

func (m *MockEntryRepository) UpdateEntry(ctx context.Context, entry *models.Entry) error {
	args := m.Called(ctx, entry)
	return args.Error(0)
}

func (m *MockEntryRepository) DeleteEntry(ctx context.Context, userID uuid.UUID, entryID uuid.UUID, entryDate string) error {
	args := m.Called(ctx, userID, entryID, entryDate)
	return args.Error(0)
}

type MockThemeRepository struct {
	mock.Mock
}

func (m *MockThemeRepository) GetThemeByID(ctx context.Context, userID uuid.UUID, themeID uuid.UUID) (*models.Theme, error) {
	args := m.Called(ctx, userID, themeID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Theme), args.Error(1)
}

func (m *MockThemeRepository) ListThemes(ctx context.Context, userID uuid.UUID) ([]models.Theme, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Theme), args.Error(1)
}

func (m *MockThemeRepository) CreateTheme(ctx context.Context, theme *models.Theme) error {
	args := m.Called(ctx, theme)
	return args.Error(0)
}

func (m *MockThemeRepository) UpdateTheme(ctx context.Context, theme *models.Theme) error {
	args := m.Called(ctx, theme)
	return args.Error(0)
}

func (m *MockThemeRepository) DeleteTheme(ctx context.Context, userID uuid.UUID, themeID uuid.UUID) error {
	args := m.Called(ctx, userID, themeID)
	return args.Error(0)
}

// --- Helper Functions ---

func setupTestContext(userID uuid.UUID) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil) // Method and path don't matter much here
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Inject UserID into context
	ctxWithUser := context.WithValue(context.Background(), UserIDContextKey, userID)
	c.SetRequest(req.WithContext(ctxWithUser))

	return c, rec
}

// --- Tests ---

func TestApiHandler_GetEntries_Success(t *testing.T) {
	// Arrange
	mockEntryRepo := new(MockEntryRepository)
	mockThemeRepo := new(MockThemeRepository) // Not used in this specific handler, but needed for ApiHandler
	handler := NewApiHandler(mockEntryRepo, mockThemeRepo)

	testUserID := uuid.New()
	testEntryID1 := uuid.New()
	testEntryID2 := uuid.New()
	testThemeID := uuid.New()
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	expectedEntries := []models.Entry{
		{EntryID: testEntryID1, UserID: testUserID, ThemeID: testThemeID, EntryDate: "2024-01-10", Data: map[string]interface{}{"field": "value1"}, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{EntryID: testEntryID2, UserID: testUserID, ThemeID: testThemeID, EntryDate: "2024-01-20", Data: map[string]interface{}{"field": "value2"}, CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}

	// Mock expectations
	mockEntryRepo.On("ListEntriesByDateRange", mock.Anything, testUserID, startDate, endDate, mock.AnythingOfType("[]uuid.UUID")).Return(expectedEntries, nil)

	c, rec := setupTestContext(testUserID)
	// Set query parameters
	q := c.Request().URL.Query()
	q.Add("start_date", startDate.Format("2006-01-02"))
	q.Add("end_date", endDate.Format("2006-01-02"))
	c.Request().URL.RawQuery = q.Encode()

	params := api.GetEntriesParams{
		StartDate: openapi_types.Date{Time: startDate},
		EndDate:   openapi_types.Date{Time: endDate},
	}

	// Act
	err := handler.GetEntries(c, params)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var responseEntries []api.Entry
	err = json.Unmarshal(rec.Body.Bytes(), &responseEntries)
	assert.NoError(t, err)
	assert.Len(t, responseEntries, 2)
	assert.Equal(t, *responseEntries[0].EntryId, testEntryID1)
	assert.Equal(t, *responseEntries[1].EntryId, testEntryID2)

	mockEntryRepo.AssertExpectations(t)
}

func TestApiHandler_GetEntries_RepoError(t *testing.T) {
	// Arrange
	mockEntryRepo := new(MockEntryRepository)
	mockThemeRepo := new(MockThemeRepository)
	handler := NewApiHandler(mockEntryRepo, mockThemeRepo)

	testUserID := uuid.New()
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)
	repoError := errors.New("database error")

	// Mock expectations
	mockEntryRepo.On("ListEntriesByDateRange", mock.Anything, testUserID, startDate, endDate, mock.AnythingOfType("[]uuid.UUID")).Return(nil, repoError)

	c, _ := setupTestContext(testUserID)
	// Set query parameters
	q := c.Request().URL.Query()
	q.Add("start_date", startDate.Format("2006-01-02"))
	q.Add("end_date", endDate.Format("2006-01-02"))
	c.Request().URL.RawQuery = q.Encode()

	params := api.GetEntriesParams{
		StartDate: openapi_types.Date{Time: startDate},
		EndDate:   openapi_types.Date{Time: endDate},
	}

	// Act
	err := handler.GetEntries(c, params)

	// Assert
	assert.Error(t, err)
	httpErr, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusInternalServerError, httpErr.Code)
	// Check the internal message if needed, but the external message is more important for API contract
	apiErr, ok := httpErr.Message.(api.Error)
	assert.True(t, ok)
	assert.Equal(t, "Failed to retrieve entries", apiErr.Message)

	mockEntryRepo.AssertExpectations(t)
}

func TestApiHandler_GetEntries_InvalidDateRange(t *testing.T) {
	// Arrange
	mockEntryRepo := new(MockEntryRepository) // No repo call expected
	mockThemeRepo := new(MockThemeRepository)
	handler := NewApiHandler(mockEntryRepo, mockThemeRepo)

	testUserID := uuid.New()
	startDate := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC) // End date is before start date
	endDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	c, _ := setupTestContext(testUserID)
	// Set query parameters
	q := c.Request().URL.Query()
	q.Add("start_date", startDate.Format("2006-01-02"))
	q.Add("end_date", endDate.Format("2006-01-02"))
	c.Request().URL.RawQuery = q.Encode()

	params := api.GetEntriesParams{
		StartDate: openapi_types.Date{Time: startDate},
		EndDate:   openapi_types.Date{Time: endDate},
	}

	// Act
	err := handler.GetEntries(c, params)

	// Assert
	assert.Error(t, err)
	httpErr, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusBadRequest, httpErr.Code)
	apiErr, ok := httpErr.Message.(api.Error)
	assert.True(t, ok)
	assert.Equal(t, "end_date cannot be before start_date", apiErr.Message)

	mockEntryRepo.AssertNotCalled(t, "ListEntriesByDateRange")
}

// --- Add more tests for other handlers (PostEntries, GetThemes, etc.) ---
// Example for PostEntries validation failure
func TestApiHandler_PostEntries_ValidationError(t *testing.T) {
	// Arrange
	mockEntryRepo := new(MockEntryRepository)
	mockThemeRepo := new(MockThemeRepository)
	handler := NewApiHandler(mockEntryRepo, mockThemeRepo)

	testUserID := uuid.New()
	testThemeID := uuid.New()
	entryDate := time.Now()

	// Define a theme with a required field "name" of type "text"
	theme := &models.Theme{
		ThemeID:   testThemeID,
		ThemeName: "Test Theme",
		Fields: []models.ThemeField{
			{Name: "name", Label: "Name", Type: models.FieldTypeText, Required: true},
			{Name: "optional_num", Label: "Optional Number", Type: models.FieldTypeNumber, Required: false},
		},
		IsDefault: false,
		UserID:    &testUserID,
	}

	// Mock GetThemeByID to return the theme
	mockThemeRepo.On("GetThemeByID", mock.Anything, testUserID, testThemeID).Return(theme, nil)

	c, _ := setupTestContext(testUserID)
	// Request body *missing* the required "name" field
	reqBody := api.CreateEntryRequest{
		ThemeId:   testThemeID,
		EntryDate: openapi_types.Date{Time: entryDate},
		Data: map[string]interface{}{
			"optional_num": 123,
		},
	}
	jsonBody, _ := json.Marshal(reqBody)
	c.Request().Body = io.NopCloser(strings.NewReader(string(jsonBody)))
	c.Request().Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	// Act
	err := handler.PostEntries(c)

	// Assert
	assert.Error(t, err)
	httpErr, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusBadRequest, httpErr.Code)
	apiErr, ok := httpErr.Message.(api.Error)
	assert.True(t, ok)
	assert.Contains(t, apiErr.Message, "missing required field") // Check for part of the expected validation error message

	mockThemeRepo.AssertExpectations(t)
	mockEntryRepo.AssertNotCalled(t, "CreateEntry") // CreateEntry should not be called if validation fails
}
