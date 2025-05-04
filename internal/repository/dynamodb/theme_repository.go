package dynamodbrepo

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/soranjiro/axicalendar/internal/domain"
	"github.com/soranjiro/axicalendar/internal/domain/theme"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
)

// dynamoDBThemeRepository implements the ThemeRepository interface using DynamoDB.
type dynamoDBThemeRepository struct {
	dbClient *DynamoDBClient
}

// NewThemeRepository creates a new DynamoDB-backed ThemeRepository.
func NewThemeRepository(dbClient *DynamoDBClient) ThemeRepository {
	return &dynamoDBThemeRepository{dbClient: dbClient}
}

// GetThemeByID retrieves theme definition by theme ID and ensures access (default or owner).
func (r *dynamoDBThemeRepository) GetThemeByID(ctx context.Context, userID uuid.UUID, themeID uuid.UUID) (*theme.Theme, error) {
	// Fetch metadata item
	pk := "THEME#" + themeID.String()
	sk := "METADATA"
	getInput := &dynamodb.GetItemInput{
		TableName: aws.String(r.dbClient.TableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: pk},
			"SK": &types.AttributeValueMemberS{Value: sk},
		},
	}
	result, err := r.dbClient.Client.GetItem(ctx, getInput)
	if err != nil {
		return nil, fmt.Errorf("failed to get theme metadata: %w", err)
	}
	if result.Item == nil {
		return nil, errors.New("theme not found")
	}
	var theme theme.Theme
	if err := attributevalue.UnmarshalMap(result.Item, &theme); err != nil {
		return nil, fmt.Errorf("failed to unmarshal theme metadata: %w", err)
	}
	// Access check: default or owned
	if !theme.IsDefault {
		if theme.OwnerUserID == nil || *theme.OwnerUserID != userID {
			return nil, errors.New("forbidden")
		}
	}
	return &theme, nil
}

// ListThemes retrieves all themes available to a user (default + custom).
func (r *dynamoDBThemeRepository) ListThemes(ctx context.Context, userID uuid.UUID) ([]theme.Theme, error) {
	// Scan metadata items
	scanInput := &dynamodb.ScanInput{
		TableName:        aws.String(r.dbClient.TableName),
		FilterExpression: aws.String("SK = :md"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":md": &types.AttributeValueMemberS{Value: "METADATA"},
		},
	}
	paginator := dynamodb.NewScanPaginator(r.dbClient.Client, scanInput)
	var themes []theme.Theme
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to scan themes: %w", err)
		}
		var pageThemes []theme.Theme
		if err := attributevalue.UnmarshalListOfMaps(page.Items, &pageThemes); err != nil {
			return nil, fmt.Errorf("failed to unmarshal themes: %w", err)
		}
		for _, t := range pageThemes {
			if t.IsDefault || (t.OwnerUserID != nil && *t.OwnerUserID == userID) {
				themes = append(themes, t)
			}
		}
	}
	return themes, nil
}

// CreateTheme creates a new custom theme (metadata + user link).
func (r *dynamoDBThemeRepository) CreateTheme(ctx context.Context, inputTheme *theme.Theme) error {
	if inputTheme.ThemeID == uuid.Nil {
		inputTheme.ThemeID = uuid.New()
	}
	if inputTheme.OwnerUserID == nil || *inputTheme.OwnerUserID == uuid.Nil {
		return errors.New("owner user ID is required to create a theme")
	}
	now := time.Now()
	inputTheme.CreatedAt = now
	inputTheme.UpdatedAt = now
	inputTheme.IsDefault = false
	// Ensure SupportedFeatures is not nil (initialize if needed)
	if inputTheme.SupportedFeatures == nil {
		inputTheme.SupportedFeatures = []string{}
	}
	// Metadata item
	meta := *inputTheme
	meta.PK = themePK(inputTheme.ThemeID.String())
	meta.SK = themeMetadataSK()
	metaAV, err := attributevalue.MarshalMap(meta)
	if err != nil {
		return fmt.Errorf("failed to marshal theme metadata: %w", err)
	}
	// User link item
	link := theme.UserThemeLink{
		PK:        userPK(inputTheme.OwnerUserID.String()),
		SK:        userThemeLinkSK(inputTheme.ThemeID.String()),
		UserID:    *inputTheme.OwnerUserID,
		ThemeID:   inputTheme.ThemeID,
		ThemeName: inputTheme.ThemeName,
		CreatedAt: now,
	}
	linkAV, err := attributevalue.MarshalMap(link)
	if err != nil {
		return fmt.Errorf("failed to marshal user-theme link: %w", err)
	}
	// Write metadata then link
	if _, err := r.dbClient.Client.PutItem(ctx, &dynamodb.PutItemInput{TableName: aws.String(r.dbClient.TableName), Item: metaAV}); err != nil {
		return fmt.Errorf("failed to create theme metadata: %w", err)
	}
	if _, err := r.dbClient.Client.PutItem(ctx, &dynamodb.PutItemInput{TableName: aws.String(r.dbClient.TableName), Item: linkAV}); err != nil {
		// Attempt to roll back metadata creation on link failure
		log.Printf("WARN: Failed to create user-theme link for theme %s, attempting metadata rollback: %v", inputTheme.ThemeID, err)
		rollbackInput := &dynamodb.DeleteItemInput{
			TableName: aws.String(r.dbClient.TableName),
			Key: map[string]types.AttributeValue{
				"PK": &types.AttributeValueMemberS{Value: meta.PK},
				"SK": &types.AttributeValueMemberS{Value: meta.SK},
			},
		}
		if _, rollbackErr := r.dbClient.Client.DeleteItem(ctx, rollbackInput); rollbackErr != nil {
			log.Printf("ERROR: Failed to rollback theme metadata for theme %s: %v", inputTheme.ThemeID, rollbackErr)
		}
		return fmt.Errorf("failed to create user-theme link: %w", err)
	}
	return nil
}

// UpdateTheme updates an existing custom theme's metadata.
func (r *dynamoDBThemeRepository) UpdateTheme(ctx context.Context, theme *theme.Theme) error {
	if theme.ThemeID == uuid.Nil || theme.OwnerUserID == nil || *theme.OwnerUserID == uuid.Nil {
		return errors.New("theme ID and owner user ID are required for update")
	}
	// Ensure access and metadata key
	pk := themePK(theme.ThemeID.String())
	sk := themeMetadataSK()
	now := time.Now()
	fieldsAV, err := attributevalue.Marshal(theme.Fields)
	if err != nil {
		return fmt.Errorf("failed to marshal fields for update: %w", err)
	}
	// Ensure SupportedFeatures is not nil before marshalling
	if theme.SupportedFeatures == nil {
		theme.SupportedFeatures = []string{}
	}
	featuresAV, err := attributevalue.Marshal(theme.SupportedFeatures)
	if err != nil {
		// This should ideally not happen if we initialize to empty slice
		return fmt.Errorf("failed to marshal supported features for update: %w", err)
	}

	updateExpr := "SET ThemeName = :name, Fields = :fields, UpdatedAt = :updatedAt, SupportedFeatures = :features" // Add SupportedFeatures
	exprAttrValues := map[string]types.AttributeValue{
		":name":      &types.AttributeValueMemberS{Value: theme.ThemeName},
		":fields":    fieldsAV,
		":updatedAt": &types.AttributeValueMemberS{Value: now.Format(time.RFC3339Nano)},
		":features":  featuresAV, // Add features to update
	}
	// Condition: Must exist, not be default, and owned by the user
	conditionExpr := "attribute_exists(PK) AND attribute_exists(SK) AND IsDefault = :false AND OwnerUserID = :userId"
	condAttrValues := map[string]types.AttributeValue{
		":false":  &types.AttributeValueMemberBOOL{Value: false},
		":userId": &types.AttributeValueMemberS{Value: theme.OwnerUserID.String()},
	}
	// Merge expression attribute values, handling potential key collisions (though unlikely here)
	mergedExprAttrValues := make(map[string]types.AttributeValue)
	for k, v := range exprAttrValues {
		mergedExprAttrValues[k] = v
	}
	for k, v := range condAttrValues {
		if _, exists := mergedExprAttrValues[k]; !exists {
			mergedExprAttrValues[k] = v
		} else {
			// If collision occurs, log it. In this specific case, it shouldn't happen.
			log.Printf("WARN: Attribute key collision between update and condition: %s. Condition value will be used.", k)
			mergedExprAttrValues[k] = v // Keep condition value if collision
		}
	}

	if _, err := r.dbClient.Client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:                 aws.String(r.dbClient.TableName),
		Key:                       map[string]types.AttributeValue{"PK": &types.AttributeValueMemberS{Value: pk}, "SK": &types.AttributeValueMemberS{Value: sk}},
		UpdateExpression:          aws.String(updateExpr),
		ConditionExpression:       aws.String(conditionExpr),
		ExpressionAttributeValues: mergedExprAttrValues,
	}); err != nil {
		var condCheckFailed *types.ConditionalCheckFailedException
		if errors.As(err, &condCheckFailed) {
			// Check if the theme exists first to give a more specific error
			// Use GetThemeByID which includes the ownership check logic
			_, getErr := r.GetThemeByID(ctx, *theme.OwnerUserID, theme.ThemeID)
			if getErr != nil {
				if errors.Is(getErr, domain.ErrNotFound) { // Use ErrNotFound from repository errors
					return domain.ErrNotFound // Theme doesn't exist
				}
				if errors.Is(getErr, domain.ErrForbidden) {
					return domain.ErrForbidden // Theme exists but not owned or is default
				}
				// Other error during GetThemeByID check
				return fmt.Errorf("failed to update theme metadata (and failed to check existence/ownership): %w", err)
			}
			// If GetThemeByID succeeded without error, something unexpected happened with the condition check.
			// This path might be less likely if GetThemeByID covers the checks.
			return fmt.Errorf("failed to update theme metadata due to condition check failure, but ownership/existence check passed: %w", err)
		}
		// Other update error
		return fmt.Errorf("failed to update theme metadata: %w", err)
	}

	// Update ThemeName in the UserThemeLink item as well for consistency
	linkPK := userPK(theme.OwnerUserID.String())
	linkSK := userThemeLinkSK(theme.ThemeID.String()) // Use userThemeLinkSK helper
	linkUpdateExpr := "SET ThemeName = :name"
	linkExprAttrValues := map[string]types.AttributeValue{
		":name": &types.AttributeValueMemberS{Value: theme.ThemeName},
	}
	if _, err := r.dbClient.Client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:                 aws.String(r.dbClient.TableName),
		Key:                       map[string]types.AttributeValue{"PK": &types.AttributeValueMemberS{Value: linkPK}, "SK": &types.AttributeValueMemberS{Value: linkSK}},
		UpdateExpression:          aws.String(linkUpdateExpr),
		ExpressionAttributeValues: linkExprAttrValues,
		ConditionExpression:       aws.String("attribute_exists(PK) AND attribute_exists(SK)"), // Ensure link exists
	}); err != nil {
		// Log warning if link update fails, but don't fail the whole operation
		log.Printf("WARN: Failed to update ThemeName in UserThemeLink for theme %s: %v", theme.ThemeID, err)
	}

	return nil
}

// DeleteTheme deletes a custom theme (metadata and user link).
func (r *dynamoDBThemeRepository) DeleteTheme(ctx context.Context, userID uuid.UUID, themeID uuid.UUID) error {
	if userID == uuid.Nil || themeID == uuid.Nil {
		return errors.New("user ID and theme ID are required for delete")
	}
	// 1. Ensure theme exists, is owned by the user, and is not default
	theme, err := r.GetThemeByID(ctx, userID, themeID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) { // Use ErrNotFound
			return domain.ErrNotFound
		}
		if errors.Is(err, domain.ErrForbidden) {
			return domain.ErrForbidden
		}
		return fmt.Errorf("failed to get theme for deletion check: %w", err) // Wrap original error
	}
	if theme.IsDefault {
		return errors.New("cannot delete default theme") // Specific error for default theme
	}
	// Owner check is implicitly done by GetThemeByID

	// 2. Delete metadata item
	metaPK := themePK(themeID.String())
	metaSK := themeMetadataSK()
	metaKey := map[string]types.AttributeValue{
		"PK": &types.AttributeValueMemberS{Value: metaPK},
		"SK": &types.AttributeValueMemberS{Value: metaSK},
	}
	if _, err := r.dbClient.Client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(r.dbClient.TableName),
		Key:       metaKey,
	}); err != nil {
		// Handle potential ConditionalCheckFailedException if condition added
		return fmt.Errorf("failed to delete theme metadata: %w", err)
	}

	// 3. Delete user link item
	linkPK := userPK(userID.String())
	linkSK := userThemeLinkSK(themeID.String()) // Use helper
	linkKey := map[string]types.AttributeValue{
		"PK": &types.AttributeValueMemberS{Value: linkPK},
		"SK": &types.AttributeValueMemberS{Value: linkSK},
	}
	if _, err := r.dbClient.Client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(r.dbClient.TableName),
		Key:       linkKey,
	}); err != nil {
		// Log warning if link deletion fails, as metadata is already gone.
		// This indicates potential data inconsistency.
		log.Printf("WARN: Failed to delete user-theme link for theme %s (PK=%s, SK=%s) after metadata deletion: %v", themeID, linkPK, linkSK, err)
	}

	return nil
}

// AddUserThemeLink creates a link item allowing a user to access a theme.
// TODO: Implement actual logic
func (r *dynamoDBThemeRepository) AddUserThemeLink(ctx context.Context, link *theme.UserThemeLink) error {
	log.Printf("WARN: AddUserThemeLink not implemented yet")
	// Placeholder implementation
	return errors.New("AddUserThemeLink not implemented")
}

// RemoveUserThemeLink removes the link item, revoking user access to a theme.
// TODO: Implement actual logic
func (r *dynamoDBThemeRepository) RemoveUserThemeLink(ctx context.Context, userID, themeID uuid.UUID) error {
	log.Printf("WARN: RemoveUserThemeLink not implemented yet")
	// Placeholder implementation
	return errors.New("RemoveUserThemeLink not implemented")
}

// ListUserThemes retrieves the UserThemeLink items for a user.
// TODO: Implement actual logic
func (r *dynamoDBThemeRepository) ListUserThemes(ctx context.Context, userID uuid.UUID) ([]theme.UserThemeLink, error) {
	log.Printf("WARN: ListUserThemes not implemented yet")
	// Placeholder implementation
	return nil, errors.New("ListUserThemes not implemented")
}

// Helper functions for PK/SK generation are defined in repository.go

// Define package-level errors for better checking are defined in repository.go
