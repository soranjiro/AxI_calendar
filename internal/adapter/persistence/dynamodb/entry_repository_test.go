package dynamodbrepo

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/soranjiro/axicalendar/internal/domain/entry"
)

func setupEntryRepoTest() (*dynamoDBEntryRepository, *MockDynamoDBAPI) {
	mockDB := new(MockDynamoDBAPI)
	dbClient := &DynamoDBClient{Client: mockDB, TableName: "test-table"}
	repo := NewEntryRepository(dbClient).(*dynamoDBEntryRepository)
	return repo, mockDB
}

func TestDynamoDBEntryRepository_GetEntryByID_Success(t *testing.T) {
	repo, mockDB := setupEntryRepoTest()
	ctx := context.Background()
	testUserID := uuid.New()
	testEntryID := uuid.New()
	testThemeID := uuid.New()
	entryDate := "2024-01-15"

	expectedEntry := &entry.Entry{
		PK:        userPK(testUserID.String()),
		SK:        entrySK(entryDate, testEntryID.String()),
		EntryID:   testEntryID,
		ThemeID:   testThemeID,
		UserID:    testUserID,
		EntryDate: entryDate,
		Data:      map[string]interface{}{"field": "value"},
		CreatedAt: time.Now().Add(-time.Hour),
		UpdatedAt: time.Now(),
		GSI1PK:    userGSI1PK(testUserID.String()),
		GSI1SK:    entryGSI1SK(entryDate, testThemeID.String()),
	}
	item, _ := attributevalue.MarshalMap(expectedEntry)

	mockDB.On("Query", ctx, mock.MatchedBy(func(input *dynamodb.QueryInput) bool {
		return *input.TableName == repo.dbClient.TableName &&
			*input.IndexName == "GSI1" &&
			*input.KeyConditionExpression == "GSI1PK = :pkval" &&
			*input.FilterExpression == "EntryID = :entryId"
	})).Return(&dynamodb.QueryOutput{Items: []map[string]types.AttributeValue{item}, Count: 1}, nil)

	entry, err := repo.GetEntryByID(ctx, testUserID, testEntryID)

	assert.NoError(t, err)
	assert.NotNil(t, entry)
	assert.Equal(t, expectedEntry.EntryID, entry.EntryID)
	mockDB.AssertExpectations(t)
}

func TestDynamoDBEntryRepository_GetEntryByID_NotFound(t *testing.T) {
	repo, mockDB := setupEntryRepoTest()
	ctx := context.Background()
	testUserID := uuid.New()
	testEntryID := uuid.New()

	mockDB.On("Query", ctx, mock.AnythingOfType("*dynamodb.QueryInput")).Return(&dynamodb.QueryOutput{Items: []map[string]types.AttributeValue{}, Count: 0}, nil)

	entry, err := repo.GetEntryByID(ctx, testUserID, testEntryID)

	assert.Error(t, err)
	assert.Nil(t, entry)
	assert.EqualError(t, err, "entry not found")
	mockDB.AssertExpectations(t)
}

func TestDynamoDBEntryRepository_GetEntryByID_QueryError(t *testing.T) {
	repo, mockDB := setupEntryRepoTest()
	ctx := context.Background()
	testUserID := uuid.New()
	testEntryID := uuid.New()
	dbError := errors.New("dynamodb error")

	mockDB.On("Query", ctx, mock.AnythingOfType("*dynamodb.QueryInput")).Return(nil, dbError)

	entry, err := repo.GetEntryByID(ctx, testUserID, testEntryID)

	assert.Error(t, err)
	assert.Nil(t, entry)
	assert.Contains(t, err.Error(), "failed to query entries")
	mockDB.AssertExpectations(t)
}

func TestDynamoDBEntryRepository_ListEntriesByDateRange_Success(t *testing.T) {
	repo, mockDB := setupEntryRepoTest()
	ctx := context.Background()
	testUserID := uuid.New()
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	entry1 := entry.Entry{EntryID: uuid.New(), UserID: testUserID, EntryDate: "2024-01-10", ThemeID: uuid.New()}
	entry2 := entry.Entry{EntryID: uuid.New(), UserID: testUserID, EntryDate: "2024-01-20", ThemeID: uuid.New()}
	item1, _ := attributevalue.MarshalMap(entry1)
	item2, _ := attributevalue.MarshalMap(entry2)

	mockDB.On("Query", ctx, mock.MatchedBy(func(input *dynamodb.QueryInput) bool {
		return *input.TableName == repo.dbClient.TableName &&
			*input.IndexName == "GSI1" &&
			*input.KeyConditionExpression == "GSI1PK = :pkval AND GSI1SK BETWEEN :startsk AND :endsk" &&
			input.FilterExpression == nil // No theme filter
	})).Return(&dynamodb.QueryOutput{Items: []map[string]types.AttributeValue{item1, item2}, Count: 2}, nil)

	entries, err := repo.ListEntriesByDateRange(ctx, testUserID, startDate, endDate, nil)

	assert.NoError(t, err)
	assert.Len(t, entries, 2)
	mockDB.AssertExpectations(t)
}

func TestDynamoDBEntryRepository_ListEntriesByDateRange_WithThemeFilter(t *testing.T) {
	repo, mockDB := setupEntryRepoTest()
	ctx := context.Background()
	testUserID := uuid.New()
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)
	themeID1 := uuid.New()

	entry1 := entry.Entry{EntryID: uuid.New(), UserID: testUserID, EntryDate: "2024-01-10", ThemeID: themeID1}
	// entry2 has themeID2, should be filtered out by mock setup if filter works
	item1, _ := attributevalue.MarshalMap(entry1)

	mockDB.On("Query", ctx, mock.MatchedBy(func(input *dynamodb.QueryInput) bool {
		return input.FilterExpression != nil &&
			*input.FilterExpression == "ThemeID IN (:theme0)" && // Check filter expression
			len(input.ExpressionAttributeValues) == 4 // :pkval, :startsk, :endsk, :theme0
	})).Return(&dynamodb.QueryOutput{Items: []map[string]types.AttributeValue{item1}, Count: 1}, nil)

	entries, err := repo.ListEntriesByDateRange(ctx, testUserID, startDate, endDate, []uuid.UUID{themeID1})

	assert.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, themeID1, entries[0].ThemeID)
	mockDB.AssertExpectations(t)
}

func TestDynamoDBEntryRepository_CreateEntry_Success(t *testing.T) {
	repo, mockDB := setupEntryRepoTest()
	ctx := context.Background()
	testEntry := &entry.Entry{
		UserID:    uuid.New(),
		ThemeID:   uuid.New(),
		EntryDate: "2024-01-15",
		Data:      map[string]interface{}{"field": "value"},
	}

	mockDB.On("PutItem", ctx, mock.MatchedBy(func(input *dynamodb.PutItemInput) bool {
		return *input.TableName == repo.dbClient.TableName &&
			*input.ConditionExpression == "attribute_not_exists(PK) AND attribute_not_exists(SK)"
	})).Return(&dynamodb.PutItemOutput{}, nil)

	err := repo.CreateEntry(ctx, testEntry)

	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, testEntry.EntryID) // Should be populated
	mockDB.AssertExpectations(t)
}

func TestDynamoDBEntryRepository_CreateEntry_AlreadyExists(t *testing.T) {
	repo, mockDB := setupEntryRepoTest()
	ctx := context.Background()
	testEntry := &entry.Entry{
		UserID:    uuid.New(),
		ThemeID:   uuid.New(),
		EntryDate: "2024-01-15",
		Data:      map[string]interface{}{"field": "value"},
	}

	mockDB.On("PutItem", ctx, mock.AnythingOfType("*dynamodb.PutItemInput")).Return(nil, &types.ConditionalCheckFailedException{})

	err := repo.CreateEntry(ctx, testEntry)

	assert.Error(t, err)
	assert.EqualError(t, err, "entry already exists")
	mockDB.AssertExpectations(t)
}

// --- Add tests for UpdateEntry (date change and no date change), DeleteEntry ---
