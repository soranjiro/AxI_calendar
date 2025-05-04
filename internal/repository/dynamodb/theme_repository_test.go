package dynamodbrepo

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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/soranjiro/axicalendar/internal/domain"
	"github.com/soranjiro/axicalendar/internal/domain/theme"
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

	expectedTheme := &theme.Theme{
		PK:          "THEME#" + testThemeID.String(),
		SK:          "METADATA",
		ThemeID:     testThemeID,
		ThemeName:   "Default Theme",
		Fields:      []theme.ThemeField{{Name: "title", Type: theme.FieldTypeText, Required: true}},
		IsDefault:   true,
		OwnerUserID: nil,
		CreatedAt:   time.Now().Add(-time.Hour),
		UpdatedAt:   time.Now(),
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

	expectedTheme := &theme.Theme{
		PK:          "THEME#" + testThemeID.String(),
		SK:          "METADATA",
		ThemeID:     testThemeID,
		ThemeName:   "My Custom Theme",
		Fields:      []theme.ThemeField{{Name: "custom_field", Type: theme.FieldTypeNumber}},
		IsDefault:   false,
		OwnerUserID: &testUserID, // Owned by the user
		CreatedAt:   time.Now().Add(-time.Hour),
		UpdatedAt:   time.Now(),
	}
	item, _ := attributevalue.MarshalMap(expectedTheme)

	mockDB.On("GetItem", ctx, mock.AnythingOfType("*dynamodb.GetItemInput")).Return(&dynamodb.GetItemOutput{Item: item}, nil)

	theme, err := repo.GetThemeByID(ctx, testUserID, testThemeID)

	assert.NoError(t, err)
	assert.NotNil(t, theme)
	assert.Equal(t, expectedTheme.ThemeID, theme.ThemeID)
	assert.False(t, theme.IsDefault)
	assert.Equal(t, testUserID, *theme.OwnerUserID)
	mockDB.AssertExpectations(t)
}

func TestDynamoDBThemeRepository_GetThemeByID_Forbidden(t *testing.T) {
	repo, mockDB := setupThemeRepoTest()
	ctx := context.Background()
	testUserID := uuid.New()  // Requesting user
	otherUserID := uuid.New() // Owner user
	testThemeID := uuid.New()

	forbiddenTheme := &theme.Theme{
		PK:          "THEME#" + testThemeID.String(),
		SK:          "METADATA",
		ThemeID:     testThemeID,
		ThemeName:   "Someone Else's Theme",
		IsDefault:   false,
		OwnerUserID: &otherUserID, // Owned by someone else
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
	assert.ErrorIs(t, err, domain.ErrNotFound) // Use ErrorIs and domain.ErrNotFound
	mockDB.AssertExpectations(t)
}

func TestDynamoDBThemeRepository_ListThemes_Success(t *testing.T) {
	repo, mockDB := setupThemeRepoTest()
	ctx := context.Background()
	testUserID := uuid.New()
	otherUserID := uuid.New()

	defaultTheme := theme.Theme{ThemeID: uuid.New(), ThemeName: "Default", IsDefault: true}
	userTheme := theme.Theme{ThemeID: uuid.New(), ThemeName: "My Theme", IsDefault: false, OwnerUserID: &testUserID}
	otherTheme := theme.Theme{ThemeID: uuid.New(), ThemeName: "Other User Theme", IsDefault: false, OwnerUserID: &otherUserID}

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
	testTheme := &theme.Theme{
		ThemeName:         "New Custom Theme",
		Fields:            []theme.ThemeField{{Name: "field1", Type: theme.FieldTypeText}},
		OwnerUserID:       &testUserID,
		SupportedFeatures: []string{"summary"}, // Add features
	}

	// Expect PutItem for metadata
	mockDB.On("PutItem", ctx, mock.MatchedBy(func(input *dynamodb.PutItemInput) bool {
		var meta theme.Theme
		err := attributevalue.UnmarshalMap(input.Item, &meta)
		assert.NoError(t, err)
		// Check PK, SK, TableName, and that SupportedFeatures are marshalled
		return *input.TableName == repo.dbClient.TableName &&
			strings.HasPrefix(meta.PK, "THEME#") &&
			meta.SK == "METADATA" &&
			assert.ObjectsAreEqual(testTheme.SupportedFeatures, meta.SupportedFeatures) && // Check features
			meta.OwnerUserID != nil && *meta.OwnerUserID == testUserID && // Check owner
			!meta.IsDefault // Check IsDefault is false
	})).Return(&dynamodb.PutItemOutput{}, nil).Once() // Expect once for metadata

	// Expect PutItem for user link
	mockDB.On("PutItem", ctx, mock.MatchedBy(func(input *dynamodb.PutItemInput) bool {
		var link theme.UserThemeLink
		err := attributevalue.UnmarshalMap(input.Item, &link)
		assert.NoError(t, err)
		// Check PK, SK, TableName, UserID, ThemeID, ThemeName
		return *input.TableName == repo.dbClient.TableName &&
			link.PK == userPK(testUserID.String()) && // Use helper
			strings.HasPrefix(link.SK, "THEME#") && // ThemeID is generated, check prefix
			link.UserID == testUserID &&
			link.ThemeName == testTheme.ThemeName
	})).Return(&dynamodb.PutItemOutput{}, nil).Once() // Expect once for link

	err := repo.CreateTheme(ctx, testTheme)

	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, testTheme.ThemeID) // Ensure ThemeID was generated
	assert.False(t, testTheme.IsDefault)
	assert.Equal(t, []string{"summary"}, testTheme.SupportedFeatures) // Check features remain
	mockDB.AssertExpectations(t)
}

func TestDynamoDBThemeRepository_UpdateTheme_Success(t *testing.T) {
	repo, mockDB := setupThemeRepoTest()
	ctx := context.Background()
	testUserID := uuid.New()
	testThemeID := uuid.New()
	themeToUpdate := &theme.Theme{
		ThemeID:           testThemeID,
		ThemeName:         "Updated Name",
		Fields:            []theme.ThemeField{{Name: "new_field", Type: theme.FieldTypeBoolean}},
		OwnerUserID:       &testUserID,
		SupportedFeatures: []string{"feature1", "feature2"}, // Add supported features
		IsDefault:         false,                            // Ensure it's not default for the update condition
		// CreatedAt should not be changed by UpdateTheme
	}

	// Mock the GetThemeByID check that happens inside UpdateTheme on conditional failure (not expected here, but good practice)
	// We don't mock GetThemeByID directly here because the success path doesn't call it.

	// Mock UpdateItem for metadata
	mockDB.On("UpdateItem", ctx, mock.MatchedBy(func(input *dynamodb.UpdateItemInput) bool {
		expectedPK := "THEME#" + testThemeID.String()
		expectedSK := "METADATA"

		if input == nil || input.TableName == nil || input.UpdateExpression == nil || input.ConditionExpression == nil {
			return false
		}

		actualPKAttr, pkOk := input.Key["PK"].(*types.AttributeValueMemberS)
		actualSKAttr, skOk := input.Key["SK"].(*types.AttributeValueMemberS)

		// Check PK, SK, TableName
		if !(pkOk && actualPKAttr.Value == expectedPK && skOk && actualSKAttr.Value == expectedSK && *input.TableName == repo.dbClient.TableName) {
			return false
		}

		// Check UpdateExpression includes SupportedFeatures
		if *input.UpdateExpression != "SET ThemeName = :name, Fields = :fields, UpdatedAt = :updatedAt, SupportedFeatures = :features" {
			t.Logf("UpdateExpression mismatch: expected %q, got %q", "SET ThemeName = :name, Fields = :fields, UpdatedAt = :updatedAt, SupportedFeatures = :features", *input.UpdateExpression)
			return false
		}

		// Check ConditionExpression
		if *input.ConditionExpression != "attribute_exists(PK) AND attribute_exists(SK) AND IsDefault = :false AND OwnerUserID = :userId" {
			t.Logf("ConditionExpression mismatch: expected %q, got %q", "attribute_exists(PK) AND attribute_exists(SK) AND IsDefault = :false AND OwnerUserID = :userId", *input.ConditionExpression)
			return false
		}

		// Check ExpressionAttributeValues contains all expected keys
		expectedKeys := []string{":name", ":fields", ":updatedAt", ":features", ":false", ":userId"}
		if len(input.ExpressionAttributeValues) != len(expectedKeys) {
			t.Logf("ExpressionAttributeValues length mismatch: expected %d, got %d", len(expectedKeys), len(input.ExpressionAttributeValues))
			return false
		}
		for _, key := range expectedKeys {
			if _, ok := input.ExpressionAttributeValues[key]; !ok {
				t.Logf("ExpressionAttributeValues missing key: %s", key)
				return false
			}
		}
		// Optionally, check specific values like :userId
		userIdAttr, userIdOk := input.ExpressionAttributeValues[":userId"].(*types.AttributeValueMemberS)
		if !userIdOk || userIdAttr.Value != testUserID.String() {
			t.Logf("ExpressionAttributeValues[:userId] mismatch or wrong type")
			return false
		}
		// Check :features (unmarshal to verify)
		featuresAttr, featuresOk := input.ExpressionAttributeValues[":features"].(*types.AttributeValueMemberL)
		if !featuresOk {
			t.Logf("ExpressionAttributeValues[:features] wrong type")
			return false
		}
		var actualFeatures []string
		err := attributevalue.Unmarshal(featuresAttr, &actualFeatures)
		if err != nil || !assert.ObjectsAreEqual(themeToUpdate.SupportedFeatures, actualFeatures) {
			t.Logf("ExpressionAttributeValues[:features] mismatch: expected %v, got %v (err: %v)", themeToUpdate.SupportedFeatures, actualFeatures, err)
			return false
		}

		return true
	})).Return(&dynamodb.UpdateItemOutput{}, nil).Once() // Expect metadata update once

	// Mock UpdateItem for user link
	mockDB.On("UpdateItem", ctx, mock.MatchedBy(func(input *dynamodb.UpdateItemInput) bool {
		expectedLinkPK := "USER#" + testUserID.String()
		expectedLinkSK := "THEME#" + testThemeID.String() // Use helper logic

		if input == nil || input.TableName == nil || input.UpdateExpression == nil || input.ConditionExpression == nil {
			return false
		}

		actualPKAttr, pkOk := input.Key["PK"].(*types.AttributeValueMemberS)
		actualSKAttr, skOk := input.Key["SK"].(*types.AttributeValueMemberS)

		// Check PK, SK, TableName
		if !(pkOk && actualPKAttr.Value == expectedLinkPK && skOk && actualSKAttr.Value == expectedLinkSK && *input.TableName == repo.dbClient.TableName) {
			return false
		}

		// Check UpdateExpression
		if *input.UpdateExpression != "SET ThemeName = :name" {
			return false
		}

		// Check ConditionExpression
		if *input.ConditionExpression != "attribute_exists(PK) AND attribute_exists(SK)" {
			return false
		}

		// Check ExpressionAttributeValues
		nameAttr, nameOk := input.ExpressionAttributeValues[":name"].(*types.AttributeValueMemberS)
		return nameOk && nameAttr.Value == themeToUpdate.ThemeName

	})).Return(&dynamodb.UpdateItemOutput{}, nil).Once() // Expect link update once

	err := repo.UpdateTheme(ctx, themeToUpdate)

	assert.NoError(t, err)
	mockDB.AssertExpectations(t)
	mockDB.AssertNumberOfCalls(t, "UpdateItem", 2) // Ensure both updates were called
}

func TestDynamoDBThemeRepository_DeleteTheme_Success(t *testing.T) {
	repo, mockDB := setupThemeRepoTest()
	ctx := context.Background()
	testUserID := uuid.New()
	testThemeID := uuid.New()

	// Mock GetThemeByID to confirm ownership and non-default status
	ownedTheme := &theme.Theme{
		ThemeID:     testThemeID,
		IsDefault:   false,
		OwnerUserID: &testUserID,
	}
	item, _ := attributevalue.MarshalMap(ownedTheme)
	// Use the actual key structure GetThemeByID uses
	getPK := "THEME#" + testThemeID.String()
	getSK := "METADATA"
	mockDB.On("GetItem", ctx, mock.MatchedBy(func(input *dynamodb.GetItemInput) bool {
		pkAttr, pkOk := input.Key["PK"].(*types.AttributeValueMemberS)
		skAttr, skOk := input.Key["SK"].(*types.AttributeValueMemberS)
		return pkOk && pkAttr.Value == getPK && skOk && skAttr.Value == getSK
	})).Return(&dynamodb.GetItemOutput{Item: item}, nil).Once()

	// Expect DeleteItem for metadata
	expectedMetaKey := map[string]types.AttributeValue{
		"PK": &types.AttributeValueMemberS{Value: themePK(testThemeID.String())}, // Use helper
		"SK": &types.AttributeValueMemberS{Value: themeMetadataSK()},             // Use helper
	}
	mockDB.On("DeleteItem", ctx, mock.MatchedBy(func(input *dynamodb.DeleteItemInput) bool {
		return *input.TableName == repo.dbClient.TableName &&
			assert.ObjectsAreEqual(expectedMetaKey, input.Key)
	})).Return(&dynamodb.DeleteItemOutput{}, nil).Once()

	// Expect DeleteItem for user link
	expectedLinkKey := map[string]types.AttributeValue{
		"PK": &types.AttributeValueMemberS{Value: userPK(testUserID.String())},           // Use helper
		"SK": &types.AttributeValueMemberS{Value: userThemeLinkSK(testThemeID.String())}, // Use helper
	}
	mockDB.On("DeleteItem", ctx, mock.MatchedBy(func(input *dynamodb.DeleteItemInput) bool {
		return *input.TableName == repo.dbClient.TableName &&
			assert.ObjectsAreEqual(expectedLinkKey, input.Key)
	})).Return(&dynamodb.DeleteItemOutput{}, nil).Once()

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
	defaultTheme := &theme.Theme{
		ThemeID:   testThemeID,
		IsDefault: true,
		// OwnerUserID is nil for default
	}
	item, _ := attributevalue.MarshalMap(defaultTheme)
	getPK := "THEME#" + testThemeID.String()
	getSK := "METADATA"
	mockDB.On("GetItem", ctx, mock.MatchedBy(func(input *dynamodb.GetItemInput) bool {
		pkAttr, pkOk := input.Key["PK"].(*types.AttributeValueMemberS)
		skAttr, skOk := input.Key["SK"].(*types.AttributeValueMemberS)
		return pkOk && pkAttr.Value == getPK && skOk && skAttr.Value == getSK
	})).Return(&dynamodb.GetItemOutput{Item: item}, nil).Once()

	err := repo.DeleteTheme(ctx, testUserID, testThemeID)

	assert.Error(t, err)
	assert.EqualError(t, err, "cannot delete default theme") // Check specific error message
	mockDB.AssertExpectations(t)
	// Ensure DeleteItem was not called
	mockDB.AssertNotCalled(t, "DeleteItem", mock.Anything, mock.Anything)
}

// Add test for DeleteTheme when GetThemeByID returns Forbidden
func TestDynamoDBThemeRepository_DeleteTheme_Forbidden(t *testing.T) {
	repo, mockDB := setupThemeRepoTest()
	ctx := context.Background()
	testUserID := uuid.New()  // Requesting user
	otherUserID := uuid.New() // Owner user
	testThemeID := uuid.New()

	// Mock GetThemeByID to return forbidden error
	forbiddenTheme := &theme.Theme{
		PK:          "THEME#" + testThemeID.String(),
		SK:          "METADATA",
		ThemeID:     testThemeID,
		ThemeName:   "Someone Else's Theme",
		IsDefault:   false,
		OwnerUserID: &otherUserID, // Owned by someone else
	}
	item, _ := attributevalue.MarshalMap(forbiddenTheme)
	getPK := "THEME#" + testThemeID.String()
	getSK := "METADATA"
	mockDB.On("GetItem", ctx, mock.MatchedBy(func(input *dynamodb.GetItemInput) bool {
		pkAttr, pkOk := input.Key["PK"].(*types.AttributeValueMemberS)
		skAttr, skOk := input.Key["SK"].(*types.AttributeValueMemberS)
		return pkOk && pkAttr.Value == getPK && skOk && skAttr.Value == getSK
	})).Return(&dynamodb.GetItemOutput{Item: item}, nil).Once() // GetItem succeeds, but logic inside GetThemeByID returns forbidden

	err := repo.DeleteTheme(ctx, testUserID, testThemeID)

	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrForbidden) // Use errors.Is for wrapped errors
	mockDB.AssertExpectations(t)
	mockDB.AssertNotCalled(t, "DeleteItem", mock.Anything, mock.Anything)
}

// Add test for DeleteTheme when GetThemeByID returns NotFound
func TestDynamoDBThemeRepository_DeleteTheme_NotFound(t *testing.T) {
	repo, mockDB := setupThemeRepoTest()
	ctx := context.Background()
	testUserID := uuid.New()
	testThemeID := uuid.New()

	// Mock GetThemeByID to return not found error
	getPK := "THEME#" + testThemeID.String()
	getSK := "METADATA"
	mockDB.On("GetItem", ctx, mock.MatchedBy(func(input *dynamodb.GetItemInput) bool {
		pkAttr, pkOk := input.Key["PK"].(*types.AttributeValueMemberS)
		skAttr, skOk := input.Key["SK"].(*types.AttributeValueMemberS)
		return pkOk && pkAttr.Value == getPK && skOk && skAttr.Value == getSK
	})).Return(&dynamodb.GetItemOutput{Item: nil}, nil).Once() // Item not found

	err := repo.DeleteTheme(ctx, testUserID, testThemeID)

	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrNotFound) // Use errors.Is for wrapped errors
	mockDB.AssertExpectations(t)
	mockDB.AssertNotCalled(t, "DeleteItem", mock.Anything, mock.Anything)
}

func TestDynamoDBThemeRepository_CreateTheme_DBError_Metadata(t *testing.T) {
	repo, mockDB := setupThemeRepoTest()
	ctx := context.Background()
	testUserID := uuid.New()
	testTheme := &theme.Theme{
		ThemeName:   "New Custom Theme",
		Fields:      []theme.ThemeField{{Name: "field1", Type: theme.FieldTypeText}},
		OwnerUserID: &testUserID,
	}
	dbError := errors.New("dynamodb put error")

	// Mock PutItem for metadata to return an error
	mockDB.On("PutItem", ctx, mock.MatchedBy(func(input *dynamodb.PutItemInput) bool {
		var meta theme.Theme
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
	testTheme := &theme.Theme{
		ThemeName:   "New Custom Theme",
		Fields:      []theme.ThemeField{{Name: "field1", Type: theme.FieldTypeText}},
		OwnerUserID: &testUserID,
	}
	dbError := errors.New("dynamodb put error")

	// Mock PutItem for metadata to succeed
	mockDB.On("PutItem", ctx, mock.MatchedBy(func(input *dynamodb.PutItemInput) bool {
		var meta theme.Theme
		err := attributevalue.UnmarshalMap(input.Item, &meta)
		assert.NoError(t, err) // Add error check
		// Capture the generated ThemeID for the rollback mock
		testTheme.ThemeID = meta.ThemeID // Assuming ThemeID is set before this mock is called
		return *input.TableName == repo.dbClient.TableName && strings.HasPrefix(meta.PK, "THEME#") && meta.SK == "METADATA"
	})).Return(&dynamodb.PutItemOutput{}, nil).Once()

	// Mock PutItem for user link to return an error
	mockDB.On("PutItem", ctx, mock.MatchedBy(func(input *dynamodb.PutItemInput) bool {
		var link theme.UserThemeLink
		err := attributevalue.UnmarshalMap(input.Item, &link)
		assert.NoError(t, err) // Add error check
		return *input.TableName == repo.dbClient.TableName && strings.HasPrefix(link.PK, "USER#") && strings.HasPrefix(link.SK, "THEME#")
	})).Return(nil, dbError).Once()

	// Mock DeleteItem for metadata rollback
	mockDB.On("DeleteItem", ctx, mock.MatchedBy(func(input *dynamodb.DeleteItemInput) bool {
		// Need to ensure testTheme.ThemeID is captured from the first PutItem mock
		expectedPK := themePK(testTheme.ThemeID.String()) // Use captured/generated ThemeID
		expectedSK := themeMetadataSK()
		pkAttr, pkOk := input.Key["PK"].(*types.AttributeValueMemberS)
		skAttr, skOk := input.Key["SK"].(*types.AttributeValueMemberS)
		return *input.TableName == repo.dbClient.TableName &&
			pkOk && pkAttr.Value == expectedPK &&
			skOk && skAttr.Value == expectedSK
	})).Return(&dynamodb.DeleteItemOutput{}, nil).Once() // Expect rollback DeleteItem call

	err := repo.CreateTheme(ctx, testTheme)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create user-theme link")
	mockDB.AssertExpectations(t)
	mockDB.AssertNumberOfCalls(t, "PutItem", 2)    // Both PutItem calls were attempted
	mockDB.AssertNumberOfCalls(t, "DeleteItem", 1) // Rollback DeleteItem was called
}
