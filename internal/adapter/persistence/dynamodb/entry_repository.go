package dynamodbrepo

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"

	"github.com/soranjiro/axicalendar/internal/domain/entry"
)

// dynamoDBEntryRepository implements the EntryRepository interface using DynamoDB.
type dynamoDBEntryRepository struct {
	dbClient *DynamoDBClient
}

// NewEntryRepository creates a new DynamoDB-backed EntryRepository.
func NewEntryRepository(dbClient *DynamoDBClient) entry.Repository { // Changed EntryRepository to entry.Repository
	return &dynamoDBEntryRepository{dbClient: dbClient}
}

// GetEntryByID retrieves a single entry by its ID and user ID.
// Requires entryDate to construct the full SK, which is inefficient.
// Consider adding a GSI by EntryID if direct lookup is frequent.
// Current approach: Query GSI1 by UserID and filter by EntryID (less efficient).
// Alternative: Store EntryDate with EntryID somewhere accessible or change PK/SK structure.
// For now, we'll assume a query approach on GSI1 or require EntryDate.
// Let's stick to the interface definition which only provides UserID and EntryID.
// We will query GSI1 (PK=USER#<user_id>, SK=ENTRY_DATE#<date>#<entry_id>) and filter.
func (r *dynamoDBEntryRepository) GetEntryByID(ctx context.Context, userID uuid.UUID, entryID uuid.UUID) (*entry.Entry, error) {
	gsi1pk := userGSI1PK(userID.String())
	entryIDStr := entryID.String()

	log.Printf("Getting entry by ID %s for user %s using GSI1 scan/filter", entryID, userID)

	queryInput := &dynamodb.QueryInput{
		TableName:              aws.String(r.dbClient.TableName),
		IndexName:              aws.String("GSI1"),
		KeyConditionExpression: aws.String("GSI1PK = :pkval"),
		FilterExpression:       aws.String("EntryID = :entryId"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pkval":   &types.AttributeValueMemberS{Value: gsi1pk},
			":entryId": &types.AttributeValueMemberS{Value: entryIDStr},
		},
	}

	paginator := dynamodb.NewQueryPaginator(r.dbClient.Client, queryInput)

	var foundEntry *entry.Entry

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			log.Printf("Error querying GSI1 for entry %s, user %s: %v", entryID, userID, err)
			return nil, fmt.Errorf("failed to query entries: %w", err)
		}

		var pageEntries []entry.Entry
		err = attributevalue.UnmarshalListOfMaps(page.Items, &pageEntries)
		if err != nil {
			log.Printf("Error unmarshalling entries for entry %s, user %s: %v", entryID, userID, err)
			return nil, fmt.Errorf("failed to unmarshal entry data: %w", err)
		}

		// Since the filter is on EntryID, double-check the exact EntryID
		for i := range pageEntries {
			if pageEntries[i].EntryID == entryID {
				foundEntry = &pageEntries[i]
				break // Found the specific entry
			}
		}
		if foundEntry != nil {
			break
		}
	}

	if foundEntry == nil {
		log.Printf("Entry %s not found for user %s", entryID, userID)
		return nil, errors.New("entry not found") // Consider specific error type
	}

	log.Printf("Successfully retrieved entry %s for user %s", entryID, userID)
	return foundEntry, nil
}

// ListEntriesByDateRange retrieves entries for a user within a specific date range.
// Uses GSI1 (PK=USER#<user_id>, SK between ENTRY_DATE#<start_date> and ENTRY_DATE#<end_date>)
// Filters by a mandatory theme ID (uses the first from the slice).
func (r *dynamoDBEntryRepository) ListEntriesByDateRange(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time, themeID uuid.UUID) ([]entry.Entry, error) {
	gsi1pk := userGSI1PK(userID.String())
	startSK := entryDateSKPrefix(startDate.Format("2006-01-02")) // ENTRY_DATE#YYYY-MM-DD
	endSK := entryDateSKPrefix(endDate.Format("2006-01-02"))     // ENTRY_DATE#YYYY-MM-DD
	if startDate.After(endDate) {
		return nil, errors.New("start date cannot be after end date")
	}
	if themeID == uuid.Nil {
		return nil, errors.New("theme ID is required to filter entries")
	}
	log.Printf("Listing entries for user %s from %s to %s, theme %s", userID, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"), themeID)

	keyCondExpr := "GSI1PK = :pkval AND GSI1SK BETWEEN :startsk AND :endsk"
	filterExprStr := "ThemeID = :themeId"
	exprAttrValues := map[string]types.AttributeValue{
		":pkval":   &types.AttributeValueMemberS{Value: gsi1pk},
		":startsk": &types.AttributeValueMemberS{Value: startSK},
		":endsk":   &types.AttributeValueMemberS{Value: endSK + "\uffff"}, // Use high-codepoint char for inclusive end range
		":themeId": &types.AttributeValueMemberB{Value: themeID[:]},
	}

	queryInput := &dynamodb.QueryInput{
		TableName:                 aws.String(r.dbClient.TableName),
		IndexName:                 aws.String("GSI1"),
		KeyConditionExpression:    aws.String(keyCondExpr),
		FilterExpression:          aws.String(filterExprStr),
		ExpressionAttributeValues: exprAttrValues,
	}

	paginator := dynamodb.NewQueryPaginator(r.dbClient.Client, queryInput)

	var entries []entry.Entry
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			log.Printf("Error querying entries for user %s in date range: %v", userID, err)
			return nil, fmt.Errorf("failed to query entries: %w", err)
		}

		var pageEntries []entry.Entry
		err = attributevalue.UnmarshalListOfMaps(page.Items, &pageEntries)
		if err != nil {
			log.Printf("Error unmarshalling entries for user %s in date range: %v", userID, err)
			return nil, fmt.Errorf("failed to unmarshal entry data: %w", err)
		}
		entries = append(entries, pageEntries...)
	}

	log.Printf("Successfully listed %d entries for user %s in date range", len(entries), userID)
	return entries, nil
}

// GetEntriesForSummary retrieves entries for a specific user, theme, and year-month (YYYY-MM).
// Uses GSI1 (PK=USER#<user_id>, SK starts with ENTRY_DATE#<year_month>) and filters by ThemeID.
func (r *dynamoDBEntryRepository) GetEntriesForSummary(ctx context.Context, userID uuid.UUID, themeID uuid.UUID, yearMonth string) ([]entry.Entry, error) {
	if themeID == uuid.Nil {
		return nil, errors.New("theme ID is required to filter entries")
	}
	// Validate yearMonth format (YYYY-MM)
	if len(yearMonth) != 7 || yearMonth[4] != '-' {
		return nil, errors.New("invalid yearMonth format, expected YYYY-MM")
	}

	gsi1pk := userGSI1PK(userID.String())
	skPrefix := entryDateSKPrefix(yearMonth) // ENTRY_DATE#YYYY-MM

	log.Printf("Listing entries for summary: user %s, theme %s, yearMonth %s", userID, themeID, yearMonth)

	keyCondExpr := "GSI1PK = :pkval AND begins_with(GSI1SK, :skprefix)"
	filterExpr := "ThemeID = :themeId"
	exprAttrValues := map[string]types.AttributeValue{
		":pkval":    &types.AttributeValueMemberS{Value: gsi1pk},
		":skprefix": &types.AttributeValueMemberS{Value: skPrefix},
		":themeId":  &types.AttributeValueMemberB{Value: themeID[:]},
	}

	queryInput := &dynamodb.QueryInput{
		TableName:                 aws.String(r.dbClient.TableName),
		IndexName:                 aws.String("GSI1"),
		KeyConditionExpression:    aws.String(keyCondExpr),
		FilterExpression:          aws.String(filterExpr),
		ExpressionAttributeValues: exprAttrValues,
	}

	paginator := dynamodb.NewQueryPaginator(r.dbClient.Client, queryInput)

	var entries []entry.Entry
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			log.Printf("Error querying entries for summary (user %s, theme %s, month %s): %v", userID, themeID, yearMonth, err)
			return nil, fmt.Errorf("failed to query entries for summary: %w", err)
		}

		var pageEntries []entry.Entry
		err = attributevalue.UnmarshalListOfMaps(page.Items, &pageEntries)
		if err != nil {
			log.Printf("Error unmarshalling entries for summary (user %s, theme %s, month %s): %v", userID, themeID, yearMonth, err)
			return nil, fmt.Errorf("failed to unmarshal entry data for summary: %w", err)
		}
		entries = append(entries, pageEntries...)
	}

	log.Printf("Successfully listed %d entries for summary (user %s, theme %s, month %s)", len(entries), userID, themeID, yearMonth)
	return entries, nil
}

// CreateEntry saves a new calendar entry.
func (r *dynamoDBEntryRepository) CreateEntry(ctx context.Context, entry *entry.Entry) error {
	if entry.EntryID == uuid.Nil {
		entry.EntryID = uuid.New()
	}
	if entry.UserID == uuid.Nil {
		return errors.New("user ID is required to create an entry")
	}
	if entry.EntryDate == "" {
		return errors.New("entry date is required to create an entry") // Ensure YYYY-MM-DD format
	}

	now := time.Now()
	entry.CreatedAt = now
	entry.UpdatedAt = now

	// Set PK, SK, and GSI keys
	entry.PK = userPK(entry.UserID.String())
	entry.SK = entrySK(entry.EntryDate, entry.EntryID.String())
	entry.GSI1PK = entry.PK // GSI1 uses UserID as PK
	entry.GSI1SK = entryGSI1SK(entry.EntryDate, entry.ThemeID.String())

	entryAV, err := attributevalue.MarshalMap(entry)
	if err != nil {
		log.Printf("Error marshalling entry for create (ID: %s): %v", entry.EntryID, err)
		return fmt.Errorf("failed to marshal entry: %w", err)
	}

	log.Printf("Creating entry: PK=%s, SK=%s, GSI1PK=%s, GSI1SK=%s", entry.PK, entry.SK, entry.GSI1PK, entry.GSI1SK)

	putInput := &dynamodb.PutItemInput{
		TableName:           aws.String(r.dbClient.TableName),
		Item:                entryAV,
		ConditionExpression: aws.String("attribute_not_exists(PK) AND attribute_not_exists(SK)"), // Ensure it's new
	}

	_, err = r.dbClient.Client.PutItem(ctx, putInput)
	if err != nil {
		var condCheckFailed *types.ConditionalCheckFailedException
		if errors.As(err, &condCheckFailed) {
			log.Printf("Conditional check failed creating entry %s: %v", entry.EntryID, err)
			return errors.New("entry already exists")
		}
		log.Printf("Error creating entry %s: %v", entry.EntryID, err)
		return fmt.Errorf("failed to create entry: %w", err)
	}

	log.Printf("Successfully created entry %s for user %s", entry.EntryID, entry.UserID)
	return nil
}

// UpdateEntry updates an existing calendar entry.
// If EntryDate changes, this involves deleting the old item and putting a new one
// because EntryDate is part of the Sort Key.
func (r *dynamoDBEntryRepository) UpdateEntry(ctx context.Context, updatedEntry *entry.Entry) error {
	if updatedEntry.EntryID == uuid.Nil || updatedEntry.UserID == uuid.Nil || updatedEntry.EntryDate == "" {
		return errors.New("entry ID, user ID, and entry date are required for update")
	}

	// 1. Get the existing entry to check if the date has changed
	existingEntry, err := r.GetEntryByID(ctx, updatedEntry.UserID, updatedEntry.EntryID)
	if err != nil {
		// GetEntryByID already logs and returns "entry not found" or other errors
		return err
	}

	// 2. Check if EntryDate has changed
	if existingEntry.EntryDate == updatedEntry.EntryDate {
		// Date hasn't changed, perform a standard UpdateItem
		return r.updateItem(ctx, updatedEntry, existingEntry.EntryDate)
	} else {
		// Date has changed, perform Delete + Put within a transaction
		log.Printf("EntryDate changed for entry %s (from %s to %s). Performing Delete+Put transaction.", updatedEntry.EntryID, existingEntry.EntryDate, updatedEntry.EntryDate)
		return r.deleteAndPutItemTransaction(ctx, updatedEntry, existingEntry.EntryDate)
	}
}

// updateItem performs a standard DynamoDB UpdateItem operation.
// Assumes EntryDate (part of SK) has NOT changed.
func (r *dynamoDBEntryRepository) updateItem(ctx context.Context, entry *entry.Entry, originalDate string) error {
	pk := userPK(entry.UserID.String())
	sk := entrySK(originalDate, entry.EntryID.String()) // Use original date for SK
	now := time.Now()

	// Construct UpdateExpression
	// Update Data, UpdatedAt, and potentially GSI1SK if ThemeID changed (though API prevents this)
	// Also update EntryDate attribute itself if it changed (even though SK uses original)
	updateExpr := "SET #data = :data, UpdatedAt = :updatedAt, GSI1SK = :gsi1sk, EntryDate = :entryDate"
	exprAttrNames := map[string]string{
		"#data": "Data", // "Data" is not a reserved word, but good practice
	}
	exprAttrValues := map[string]types.AttributeValue{
		":updatedAt": &types.AttributeValueMemberS{Value: now.Format(time.RFC3339Nano)},
		":gsi1sk":    &types.AttributeValueMemberS{Value: entryGSI1SK(entry.EntryDate, entry.ThemeID.String())},
		":entryDate": &types.AttributeValueMemberS{Value: entry.EntryDate},
	}

	dataAV, err := attributevalue.MarshalMap(entry.Data)
	if err != nil {
		log.Printf("Error marshalling entry data for update %s: %v", entry.EntryID, err)
		return fmt.Errorf("failed to marshal entry data: %w", err)
	}
	exprAttrValues[":data"] = &types.AttributeValueMemberM{Value: dataAV}

	log.Printf("Updating item: PK=%s, SK=%s", pk, sk)

	updateInput := &dynamodb.UpdateItemInput{
		TableName: aws.String(r.dbClient.TableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: pk},
			"SK": &types.AttributeValueMemberS{Value: sk},
		},
		UpdateExpression:          aws.String(updateExpr),
		ExpressionAttributeNames:  exprAttrNames,
		ExpressionAttributeValues: exprAttrValues,
		ConditionExpression:       aws.String("attribute_exists(PK) AND attribute_exists(SK)"), // Ensure item exists
		ReturnValues:              types.ReturnValueNone,
	}

	_, err = r.dbClient.Client.UpdateItem(ctx, updateInput)
	if err != nil {
		var condCheckFailed *types.ConditionalCheckFailedException
		if errors.As(err, &condCheckFailed) {
			log.Printf("Conditional check failed updating item %s: %v", entry.EntryID, err)
			return errors.New("entry not found")
		}
		log.Printf("Error updating item %s: %v", entry.EntryID, err)
		return fmt.Errorf("failed to update entry item: %w", err)
	}

	log.Printf("Successfully updated item %s", entry.EntryID)
	return nil
}

// deleteAndPutItemTransaction deletes the old entry and puts the new entry within a transaction.
// Used when EntryDate (part of SK) changes during an update.
func (r *dynamoDBEntryRepository) deleteAndPutItemTransaction(ctx context.Context, newEntryData *entry.Entry, oldDate string) error {
	now := time.Now()
	newEntryData.UpdatedAt = now
	// Preserve CreatedAt if possible, or set it if missing (shouldn't be)
	if newEntryData.CreatedAt.IsZero() {
		// Attempt to fetch original CreatedAt - this adds complexity, maybe just set to UpdatedAt?
		// For simplicity, let's assume CreatedAt was populated correctly before calling UpdateEntry.
		// If not, setting it to 'now' might be acceptable depending on requirements.
		log.Printf("WARN: CreatedAt is zero for entry %s during date change update. Setting to UpdatedAt.", newEntryData.EntryID)
		newEntryData.CreatedAt = now
	}

	// Prepare Delete operation for the old item
	oldPK := userPK(newEntryData.UserID.String())
	oldSK := entrySK(oldDate, newEntryData.EntryID.String())
	deleteItem := types.TransactWriteItem{
		Delete: &types.Delete{
			TableName: aws.String(r.dbClient.TableName),
			Key: map[string]types.AttributeValue{
				"PK": &types.AttributeValueMemberS{Value: oldPK},
				"SK": &types.AttributeValueMemberS{Value: oldSK},
			},
			ConditionExpression: aws.String("attribute_exists(PK) AND attribute_exists(SK)"), // Ensure old item exists
		},
	}

	// Prepare Put operation for the new item
	newEntryData.PK = userPK(newEntryData.UserID.String())
	newEntryData.SK = entrySK(newEntryData.EntryDate, newEntryData.EntryID.String())
	newEntryData.GSI1PK = newEntryData.PK
	newEntryData.GSI1SK = entryGSI1SK(newEntryData.EntryDate, newEntryData.ThemeID.String())

	newItemAV, err := attributevalue.MarshalMap(newEntryData)
	if err != nil {
		log.Printf("Error marshalling new entry data for transaction %s: %v", newEntryData.EntryID, err)
		return fmt.Errorf("failed to marshal new entry data for transaction: %w", err)
	}

	putItem := types.TransactWriteItem{
		Put: &types.Put{
			TableName:           aws.String(r.dbClient.TableName),
			Item:                newItemAV,
			ConditionExpression: aws.String("attribute_not_exists(PK) AND attribute_not_exists(SK)"), // Ensure new item doesn't exist
		},
	}

	// Execute Transaction
	transactInput := &dynamodb.TransactWriteItemsInput{
		TransactItems: []types.TransactWriteItem{deleteItem, putItem},
	}

	log.Printf("Executing transaction for entry %s: Delete(PK=%s, SK=%s), Put(PK=%s, SK=%s)", newEntryData.EntryID, oldPK, oldSK, newEntryData.PK, newEntryData.SK)

	_, err = r.dbClient.Client.TransactWriteItems(ctx, transactInput)
	if err != nil {
		// Check for transaction cancellation reasons
		var txc *types.TransactionCanceledException
		if errors.As(err, &txc) {
			log.Printf("Transaction cancelled for entry %s update: %v", newEntryData.EntryID, txc.CancellationReasons)
			// Analyze cancellation reasons - could be condition check failed on delete (old item not found) or put (new item already exists)
			for _, reason := range txc.CancellationReasons {
				if reason.Code != nil && *reason.Code == "ConditionalCheckFailed" {
					// Determine if it was the delete or put that failed based on the item structure if possible, or return a generic error
					return errors.New("entry update failed due to condition check (original not found or target date conflict)")
				}
			}
			return fmt.Errorf("entry update transaction cancelled: %w", err)
		}
		log.Printf("Error executing transaction for entry %s update: %v", newEntryData.EntryID, err)
		return fmt.Errorf("failed to execute entry update transaction: %w", err)
	}

	log.Printf("Successfully updated entry %s via transaction (date changed)", newEntryData.EntryID)
	return nil
}

// DeleteEntry deletes a calendar entry.
func (r *dynamoDBEntryRepository) DeleteEntry(ctx context.Context, userID uuid.UUID, entryID uuid.UUID, entryDate string) error {
	if userID == uuid.Nil || entryID == uuid.Nil || entryDate == "" {
		return errors.New("user ID, entry ID, and entry date are required for delete")
	}

	pk := userPK(userID.String())
	sk := entrySK(entryDate, entryID.String())

	log.Printf("Deleting entry: PK=%s, SK=%s", pk, sk)

	deleteInput := &dynamodb.DeleteItemInput{
		TableName: aws.String(r.dbClient.TableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: pk},
			"SK": &types.AttributeValueMemberS{Value: sk},
		},
		ConditionExpression: aws.String("attribute_exists(PK) AND attribute_exists(SK)"), // Ensure item exists
	}

	_, err := r.dbClient.Client.DeleteItem(ctx, deleteInput)
	if err != nil {
		var condCheckFailed *types.ConditionalCheckFailedException
		if errors.As(err, &condCheckFailed) {
			log.Printf("Conditional check failed deleting entry %s: %v", entryID, err)
			return errors.New("entry not found")
		}
		log.Printf("Error deleting entry %s: %v", entryID, err)
		return fmt.Errorf("failed to delete entry: %w", err)
	}

	log.Printf("Successfully deleted entry %s for user %s", entryID, userID)
	return nil
}
