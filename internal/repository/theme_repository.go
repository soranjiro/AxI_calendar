package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
	"github.com/soranjiro/axicalendar/internal/models"
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
func (r *dynamoDBThemeRepository) GetThemeByID(ctx context.Context, userID uuid.UUID, themeID uuid.UUID) (*models.Theme, error) {
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
	var theme models.Theme
	if err := attributevalue.UnmarshalMap(result.Item, &theme); err != nil {
		return nil, fmt.Errorf("failed to unmarshal theme metadata: %w", err)
	}
	// Access check: default or owned
	if !theme.IsDefault {
		if theme.UserID == nil || *theme.UserID != userID {
			return nil, errors.New("forbidden")
		}
	}
	return &theme, nil
}

// ListThemes retrieves all themes available to a user (default + custom).
func (r *dynamoDBThemeRepository) ListThemes(ctx context.Context, userID uuid.UUID) ([]models.Theme, error) {
	// Scan metadata items
	scanInput := &dynamodb.ScanInput{
		TableName:        aws.String(r.dbClient.TableName),
		FilterExpression: aws.String("SK = :md"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":md": &types.AttributeValueMemberS{Value: "METADATA"},
		},
	}
	paginator := dynamodb.NewScanPaginator(r.dbClient.Client, scanInput)
	var themes []models.Theme
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to scan themes: %w", err)
		}
		var pageThemes []models.Theme
		if err := attributevalue.UnmarshalListOfMaps(page.Items, &pageThemes); err != nil {
			return nil, fmt.Errorf("failed to unmarshal themes: %w", err)
		}
		for _, t := range pageThemes {
			if t.IsDefault || (t.UserID != nil && *t.UserID == userID) {
				themes = append(themes, t)
			}
		}
	}
	return themes, nil
}

// CreateTheme creates a new custom theme (metadata + user link).
func (r *dynamoDBThemeRepository) CreateTheme(ctx context.Context, theme *models.Theme) error {
	if theme.ThemeID == uuid.Nil {
		theme.ThemeID = uuid.New()
	}
	if theme.UserID == nil || *theme.UserID == uuid.Nil {
		return errors.New("user ID is required to create a theme")
	}
	now := time.Now()
	theme.CreatedAt = now
	theme.UpdatedAt = now
	theme.IsDefault = false
	// Metadata item
	meta := *theme
	meta.PK = "THEME#" + theme.ThemeID.String()
	meta.SK = "METADATA"
	// clear GSI fields on metadata
	meta.GSI1PK = nil
	meta.GSI1SK = nil
	metaAV, err := attributevalue.MarshalMap(meta)
	if err != nil {
		return fmt.Errorf("failed to marshal theme metadata: %w", err)
	}
	// User link item
	link := models.UserThemeLink{
		PK:      userPK(theme.UserID.String()),
		SK:      themeSK(theme.ThemeID.String()),
		UserID:  *theme.UserID,
		ThemeID: theme.ThemeID,
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
		return fmt.Errorf("failed to create user-theme link: %w", err)
	}
	return nil
}

// UpdateTheme updates an existing custom theme's metadata.
func (r *dynamoDBThemeRepository) UpdateTheme(ctx context.Context, theme *models.Theme) error {
	if theme.ThemeID == uuid.Nil || theme.UserID == nil || *theme.UserID == uuid.Nil {
		return errors.New("theme ID and user ID are required for update")
	}
	// Ensure access and metadata key
	pk := "THEME#" + theme.ThemeID.String()
	sk := "METADATA"
	now := time.Now()
	fieldsAV, err := attributevalue.Marshal(theme.Fields)
	if err != nil {
		return fmt.Errorf("failed to marshal fields for update: %w", err)
	}
	updateExpr := "SET ThemeName = :name, Fields = :fields, UpdatedAt = :updatedAt"
	exprAttrValues := map[string]types.AttributeValue{
		":name":      &types.AttributeValueMemberS{Value: theme.ThemeName},
		":fields":    fieldsAV,
		":updatedAt": &types.AttributeValueMemberS{Value: now.Format(time.RFC3339Nano)},
	}
	if _, err := r.dbClient.Client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:                 aws.String(r.dbClient.TableName),
		Key:                       map[string]types.AttributeValue{"PK": &types.AttributeValueMemberS{Value: pk}, "SK": &types.AttributeValueMemberS{Value: sk}},
		UpdateExpression:          aws.String(updateExpr),
		ExpressionAttributeValues: exprAttrValues,
		ConditionExpression:       aws.String("attribute_exists(PK) AND attribute_exists(SK)"),
	}); err != nil {
		return fmt.Errorf("failed to update theme metadata: %w", err)
	}
	return nil
}

// DeleteTheme deletes a custom theme (metadata and user link).
func (r *dynamoDBThemeRepository) DeleteTheme(ctx context.Context, userID uuid.UUID, themeID uuid.UUID) error {
	if userID == uuid.Nil || themeID == uuid.Nil {
		return errors.New("user ID and theme ID are required for delete")
	}
	// Ensure theme exists and owned
	theme, err := r.GetThemeByID(ctx, userID, themeID)
	if err != nil {
		return err
	}
	if theme.IsDefault {
		return errors.New("cannot delete default theme")
	}
	// Delete metadata
	metaKey := map[string]types.AttributeValue{"PK": &types.AttributeValueMemberS{Value: "THEME#" + themeID.String()}, "SK": &types.AttributeValueMemberS{Value: "METADATA"}}
	if _, err := r.dbClient.Client.DeleteItem(ctx, &dynamodb.DeleteItemInput{TableName: aws.String(r.dbClient.TableName), Key: metaKey}); err != nil {
		return fmt.Errorf("failed to delete theme metadata: %w", err)
	}
	// Delete user link
	linkKey := map[string]types.AttributeValue{"PK": &types.AttributeValueMemberS{Value: userPK(userID.String())}, "SK": &types.AttributeValueMemberS{Value: themeSK(themeID.String())}}
	if _, err := r.dbClient.Client.DeleteItem(ctx, &dynamodb.DeleteItemInput{TableName: aws.String(r.dbClient.TableName), Key: linkKey}); err != nil {
		return fmt.Errorf("failed to delete user-theme link: %w", err)
	}
	return nil
}
