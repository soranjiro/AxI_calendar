package repository

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

const (
	TableNameEnvVar = "DYNAMODB_TABLE_NAME"
)

// DynamoDBAPI defines the interface for DynamoDB operations used by repositories.
// This allows for mocking in tests.
type DynamoDBAPI interface {
	GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	DeleteItem(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error)
	UpdateItem(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error)
	Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
	Scan(ctx context.Context, params *dynamodb.ScanInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error)
	TransactWriteItems(ctx context.Context, params *dynamodb.TransactWriteItemsInput, optFns ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error)
}

// DynamoDBClient encapsulates the DynamoDB client and table name.
type DynamoDBClient struct {
	Client    DynamoDBAPI // Use the interface type
	TableName string
}

// NewDynamoDBClient creates a new DynamoDB client wrapper.
// It reads the table name from the DYNAMODB_TABLE_NAME environment variable.
func NewDynamoDBClient(ctx context.Context) (*DynamoDBClient, error) {
	tableName := os.Getenv(TableNameEnvVar)
	if tableName == "" {
		log.Fatalf("Environment variable %s must be set", TableNameEnvVar)
		// In a real app, return an error instead of Fatalf
		// return nil, fmt.Errorf("environment variable %s must be set", TableNameEnvVar)
	}

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
		// return nil, fmt.Errorf("unable to load SDK config: %w", err)
	}

	client := dynamodb.NewFromConfig(cfg)

	log.Printf("DynamoDB client initialized for table: %s", tableName)
	return &DynamoDBClient{
		Client:    client, // *dynamodb.Client satisfies the DynamoDBAPI interface
		TableName: tableName,
	}, nil
}
