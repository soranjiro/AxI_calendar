package repository

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
	"github.com/soranjiro/axicalendar/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupThemeRepoTest() (*dynamoDBThemeRepository, *MockDynamoDBAPI) {
	mockDB := new(MockDynamoDBAPI)
	dbClient := &DynamoDBClient{Client: mockDB, TableName: "test-table"}
	repo := NewThemeRepository(dbClient).(*dynamoDBThemeRepository)
	return repo, mockDB
}

func TestDynamoDBThemeRepository_GetThemeByID_Success_Default(t *testing.T) {
	repo, mockDB := setupThemeRepoTest()
	ctx := context.Background()
	testUserID := uuid.New() // User doesn't matter for default
	testThemeID := uuid.New()

	expectedTheme := &models.Theme{
		PK:        "THEME#" + testThemeID.String(),
		SK:        "METADATA",
		ThemeID:   testThemeID,
		ThemeName: "Default Theme",
		Fields:    []models.ThemeField{{Name: "title", Type: models.FieldTypeText, Required: true}},
		IsDefault: true,
		UserID:    nil,
		CreatedAt: time.Now().Add(-time.Hour),
		UpdatedAt: time.Now(),
	}
	item, _ := attributevalue.MarshalMap(expectedTheme)

	mockDB.On("GetItem", ctx, mock.MatchedBy(func(input *dynamodb.GetItemInput) bool {
		expectedPK := "THEME#" + testThemeID.String()
		expectedSK := "METADATA"

		// Ensure input and TableName are not nil before dereferencing
		if input == nil || input.TableName == nil {
			return false
		}

		actualPKAttr, pkOk := input.Key["PK"].(*types.AttributeValueMemberS)
		actualSKAttr, skOk := input.Key["SK"].(*types.AttributeValueMemberS)

		return *input.TableName == repo.dbClient.TableName &&
			pkOk && actualPKAttr.Value == expectedPK &&
			skOk && actualSKAttr.Value == expectedSK
	})).Return(&dynamodb.GetItemOutput{Item: item}, nil)

	theme, err := repo.GetThemeByID(ctx, testUserID, testThemeID)

	assert.NoError(t, err)
	assert.NotNil(t, theme)
	assert.Equal(t, expectedTheme.ThemeID, theme.ThemeID)
	assert.True(t, theme.IsDefault)
	mockDB.AssertExpectations(t)
}

func TestDynamoDBThemeRepository_GetThemeByID_Success_CustomOwned(t *testing.T) {
	repo, mockDB := setupThemeRepoTest()
	ctx := context.Background()
	testUserID := uuid.New()
	testThemeID := uuid.New()

	expectedTheme := &models.Theme{
		PK:        "THEME#" + testThemeID.String(),
		SK:        "METADATA",
		ThemeID:   testThemeID,
		ThemeName: "My Custom Theme",
		Fields:    []models.ThemeField{{Name: "custom_field", Type: models.FieldTypeNumber}},
		IsDefault: false,
		UserID:    &testUserID, // Owned by the user
		CreatedAt: time.Now().Add(-time.Hour),
		UpdatedAt: time.Now(),
	}
	item, _ := attributevalue.MarshalMap(expectedTheme)

	mockDB.On("GetItem", ctx, mock.AnythingOfType("*dynamodb.GetItemInput")).Return(&dynamodb.GetItemOutput{Item: item}, nil)

	theme, err := repo.GetThemeByID(ctx, testUserID, testThemeID)

	assert.NoError(t, err)
	assert.NotNil(t, theme)
	assert.Equal(t, expectedTheme.ThemeID, theme.ThemeID)
	assert.False(t, theme.IsDefault)
	assert.Equal(t, testUserID, *theme.UserID)
	mockDB.AssertExpectations(t)
}

func TestDynamoDBThemeRepository_GetThemeByID_Forbidden(t *testing.T) {
	repo, mockDB := setupThemeRepoTest()
	ctx := context.Background()
	testUserID := uuid.New()  // Requesting user
	otherUserID := uuid.New() // Owner user
	testThemeID := uuid.New()

	forbiddenTheme := &models.Theme{
		PK:        "THEME#" + testThemeID.String(),
		SK:        "METADATA",
		ThemeID:   testThemeID,
		ThemeName: "Someone Else's Theme",
		IsDefault: false,
		UserID:    &otherUserID, // Owned by someone else
	}
	item, _ := attributevalue.MarshalMap(forbiddenTheme)

	mockDB.On("GetItem", ctx, mock.AnythingOfType("*dynamodb.GetItemInput")).Return(&dynamodb.GetItemOutput{Item: item}, nil)

	theme, err := repo.GetThemeByID(ctx, testUserID, testThemeID)

	assert.Error(t, err)
	assert.Nil(t, theme)
	assert.EqualError(t, err, "forbidden")
	mockDB.AssertExpectations(t)
}

func TestDynamoDBThemeRepository_GetThemeByID_NotFound(t *testing.T) {
	repo, mockDB := setupThemeRepoTest()
	ctx := context.Background()
	testUserID := uuid.New()
	testThemeID := uuid.New()

	mockDB.On("GetItem", ctx, mock.AnythingOfType("*dynamodb.GetItemInput")).Return(&dynamodb.GetItemOutput{Item: nil}, nil) // Item not found

	theme, err := repo.GetThemeByID(ctx, testUserID, testThemeID)

	assert.Error(t, err)
	assert.Nil(t, theme)
	assert.EqualError(t, err, "theme not found")
	mockDB.AssertExpectations(t)
}

func TestDynamoDBThemeRepository_ListThemes_Success(t *testing.T) {
	repo, mockDB := setupThemeRepoTest()
	ctx := context.Background()
	testUserID := uuid.New()
	otherUserID := uuid.New()

	defaultTheme := models.Theme{ThemeID: uuid.New(), ThemeName: "Default", IsDefault: true}
	userTheme := models.Theme{ThemeID: uuid.New(), ThemeName: "My Theme", IsDefault: false, UserID: &testUserID}
	otherTheme := models.Theme{ThemeID: uuid.New(), ThemeName: "Other User Theme", IsDefault: false, UserID: &otherUserID}

	itemDefault, _ := attributevalue.MarshalMap(defaultTheme)
	itemUser, _ := attributevalue.MarshalMap(userTheme)
	itemOther, _ := attributevalue.MarshalMap(otherTheme)

	// Mock Scan to return all three themes
	mockDB.On("Scan", ctx, mock.MatchedBy(func(input *dynamodb.ScanInput) bool {
		return *input.TableName == repo.dbClient.TableName &&
			*input.FilterExpression == "SK = :md"
	})).Return(&dynamodb.ScanOutput{Items: []map[string]types.AttributeValue{itemDefault, itemUser, itemOther}, Count: 3}, nil)

	themes, err := repo.ListThemes(ctx, testUserID)

	assert.NoError(t, err)
	assert.Len(t, themes, 2) // Should include default and user's theme, exclude other user's theme

	foundDefault := false
	foundUser := false
	for _, th := range themes {
		if th.ThemeID == defaultTheme.ThemeID {
			foundDefault = true
		}
		if th.ThemeID == userTheme.ThemeID {
			foundUser = true
		}
	}
	assert.True(t, foundDefault, "Default theme not found in list")
	assert.True(t, foundUser, "User's theme not found in list")

	mockDB.AssertExpectations(t)
}

func TestDynamoDBThemeRepository_CreateTheme_Success(t *testing.T) {
	repo, mockDB := setupThemeRepoTest()
	ctx := context.Background()
	testUserID := uuid.New()
	testTheme := &models.Theme{
		ThemeName: "New Custom Theme",
		Fields:    []models.ThemeField{{Name: "field1", Type: models.FieldTypeText}},
		UserID:    &testUserID,
	}

	// Expect PutItem for metadata
	mockDB.On("PutItem", ctx, mock.MatchedBy(func(input *dynamodb.PutItemInput) bool {
		var meta models.Theme
		err := attributevalue.UnmarshalMap(input.Item, &meta)
		assert.NoError(t, err) // Add error check
		return *input.TableName == repo.dbClient.TableName && strings.HasPrefix(meta.PK, "THEME#") && meta.SK == "METADATA"
	})).Return(&dynamodb.PutItemOutput{}, nil).Once() // Expect once for metadata

	// Expect PutItem for user link
	mockDB.On("PutItem", ctx, mock.MatchedBy(func(input *dynamodb.PutItemInput) bool {
		var link models.UserThemeLink
		err := attributevalue.UnmarshalMap(input.Item, &link)
		assert.NoError(t, err) // Add error check
		return *input.TableName == repo.dbClient.TableName && strings.HasPrefix(link.PK, "USER#") && strings.HasPrefix(link.SK, "THEME#")
	})).Return(&dynamodb.PutItemOutput{}, nil).Once() // Expect once for link

	err := repo.CreateTheme(ctx, testTheme)

	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, testTheme.ThemeID)
	assert.False(t, testTheme.IsDefault)
	mockDB.AssertExpectations(t)
}

func TestDynamoDBThemeRepository_UpdateTheme_Success(t *testing.T) {
	repo, mockDB := setupThemeRepoTest()
	ctx := context.Background()
	testUserID := uuid.New()
	testThemeID := uuid.New()
	themeToUpdate := &models.Theme{
		ThemeID:   testThemeID,
		ThemeName: "Updated Name",
		Fields:    []models.ThemeField{{Name: "new_field", Type: models.FieldTypeBoolean}},
		UserID:    &testUserID,
		// IsDefault, CreatedAt should not be changed by UpdateTheme
	}

	mockDB.On("UpdateItem", ctx, mock.MatchedBy(func(input *dynamodb.UpdateItemInput) bool {
		expectedPK := "THEME#" + testThemeID.String()
		expectedSK := "METADATA"

		// Ensure input and TableName are not nil before dereferencing
		if input == nil || input.TableName == nil || input.UpdateExpression == nil || input.ConditionExpression == nil {
			return false
		}

		actualPKAttr, pkOk := input.Key["PK"].(*types.AttributeValueMemberS)
		actualSKAttr, skOk := input.Key["SK"].(*types.AttributeValueMemberS)

		return *input.TableName == repo.dbClient.TableName &&
			pkOk && actualPKAttr.Value == expectedPK && // Compare string values
			skOk && actualSKAttr.Value == expectedSK && // Compare string values
			*input.UpdateExpression == "SET ThemeName = :name, Fields = :fields, UpdatedAt = :updatedAt" &&
			*input.ConditionExpression == "attribute_exists(PK) AND attribute_exists(SK)"
	})).Return(&dynamodb.UpdateItemOutput{}, nil)

	err := repo.UpdateTheme(ctx, themeToUpdate)

	assert.NoError(t, err)
	mockDB.AssertExpectations(t)
}

func TestDynamoDBThemeRepository_DeleteTheme_Success(t *testing.T) {
	repo, mockDB := setupThemeRepoTest()
	ctx := context.Background()
	testUserID := uuid.New()
	testThemeID := uuid.New()

	// Mock GetThemeByID to confirm ownership and non-default status
	ownedTheme := &models.Theme{
		ThemeID:   testThemeID,
		IsDefault: false,
		UserID:    &testUserID,
	}
	item, _ := attributevalue.MarshalMap(ownedTheme)
	mockDB.On("GetItem", ctx, mock.AnythingOfType("*dynamodb.GetItemInput")).Return(&dynamodb.GetItemOutput{Item: item}, nil).Once()

	// Expect DeleteItem for metadata
	expectedMetaKey := map[string]types.AttributeValue{
		"PK": &types.AttributeValueMemberS{Value: "THEME#" + testThemeID.String()},
		"SK": &types.AttributeValueMemberS{Value: "METADATA"},
	}
	expectedMetaInput := &dynamodb.DeleteItemInput{
		TableName: &repo.dbClient.TableName,
		Key:       expectedMetaKey,
	}
	mockDB.On("DeleteItem", ctx, expectedMetaInput).Return(&dynamodb.DeleteItemOutput{}, nil).Once()

	// Expect DeleteItem for user link
	expectedLinkKey := map[string]types.AttributeValue{
		"PK": &types.AttributeValueMemberS{Value: userPK(testUserID.String())},
		"SK": &types.AttributeValueMemberS{Value: themeSK(testThemeID.String())},
	}
	expectedLinkInput := &dynamodb.DeleteItemInput{
		TableName: &repo.dbClient.TableName,
		Key:       expectedLinkKey,
	}
	mockDB.On("DeleteItem", ctx, expectedLinkInput).Return(&dynamodb.DeleteItemOutput{}, nil).Once()

	err := repo.DeleteTheme(ctx, testUserID, testThemeID)

	assert.NoError(t, err)
	mockDB.AssertExpectations(t)
}

func TestDynamoDBThemeRepository_DeleteTheme_CannotDeleteDefault(t *testing.T) {
	repo, mockDB := setupThemeRepoTest()
	ctx := context.Background()
	testUserID := uuid.New()
	testThemeID := uuid.New()

	// Mock GetThemeByID to return a default theme
	defaultTheme := &models.Theme{
		ThemeID:   testThemeID,
		IsDefault: true,
	}
	item, _ := attributevalue.MarshalMap(defaultTheme)
	mockDB.On("GetItem", ctx, mock.AnythingOfType("*dynamodb.GetItemInput")).Return(&dynamodb.GetItemOutput{Item: item}, nil).Once()

	err := repo.DeleteTheme(ctx, testUserID, testThemeID)

	assert.Error(t, err)
	assert.EqualError(t, err, "cannot delete default theme")
	mockDB.AssertExpectations(t)
	// Ensure DeleteItem was not called
	mockDB.AssertNotCalled(t, "DeleteItem", mock.Anything, mock.Anything)
}

func TestDynamoDBThemeRepository_CreateTheme_DBError_Metadata(t *testing.T) {
	repo, mockDB := setupThemeRepoTest()
	ctx := context.Background()
	testUserID := uuid.New()
	testTheme := &models.Theme{
		ThemeName: "New Custom Theme",
		Fields:    []models.ThemeField{{Name: "field1", Type: models.FieldTypeText}},
		UserID:    &testUserID,
	}
	dbError := errors.New("dynamodb put error")

	// Mock PutItem for metadata to return an error
	mockDB.On("PutItem", ctx, mock.MatchedBy(func(input *dynamodb.PutItemInput) bool {
		var meta models.Theme
		err := attributevalue.UnmarshalMap(input.Item, &meta)
		assert.NoError(t, err) // Add error check
		return *input.TableName == repo.dbClient.TableName && strings.HasPrefix(meta.PK, "THEME#") && meta.SK == "METADATA"
	})).Return(nil, dbError).Once()

	err := repo.CreateTheme(ctx, testTheme)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create theme metadata")
	mockDB.AssertExpectations(t)
	// Ensure the second PutItem (for the link) was not called
	mockDB.AssertNumberOfCalls(t, "PutItem", 1)
}

func TestDynamoDBThemeRepository_CreateTheme_DBError_Link(t *testing.T) {
	repo, mockDB := setupThemeRepoTest()
	ctx := context.Background()
	testUserID := uuid.New()
	testTheme := &models.Theme{
		ThemeName: "New Custom Theme",
		Fields:    []models.ThemeField{{Name: "field1", Type: models.FieldTypeText}},
		UserID:    &testUserID,
	}
	dbError := errors.New("dynamodb put error")

	// Mock PutItem for metadata to succeed
	mockDB.On("PutItem", ctx, mock.MatchedBy(func(input *dynamodb.PutItemInput) bool {
		var meta models.Theme
		err := attributevalue.UnmarshalMap(input.Item, &meta)
		assert.NoError(t, err) // Add error check
		return *input.TableName == repo.dbClient.TableName && strings.HasPrefix(meta.PK, "THEME#") && meta.SK == "METADATA"
	})).Return(&dynamodb.PutItemOutput{}, nil).Once()

	// Mock PutItem for user link to return an error
	mockDB.On("PutItem", ctx, mock.MatchedBy(func(input *dynamodb.PutItemInput) bool {
		var link models.UserThemeLink
		err := attributevalue.UnmarshalMap(input.Item, &link)
		assert.NoError(t, err) // Add error check
		return *input.TableName == repo.dbClient.TableName && strings.HasPrefix(link.PK, "USER#") && strings.HasPrefix(link.SK, "THEME#")
	})).Return(nil, dbError).Once()

	err := repo.CreateTheme(ctx, testTheme)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create user-theme link")
	mockDB.AssertExpectations(t)
	mockDB.AssertNumberOfCalls(t, "PutItem", 2) // Both PutItem calls were attempted
}
