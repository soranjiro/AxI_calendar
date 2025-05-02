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

// DynamoDBClient encapsulates the DynamoDB client and table name.
type DynamoDBClient struct {
	Client    *dynamodb.Client
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
		Client:    client,
		TableName: tableName,
	}, nil
}
