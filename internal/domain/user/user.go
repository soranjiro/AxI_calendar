package user

import (
	"time"

	"github.com/soranjiro/axicalendar/internal/api" // Assuming api.gen.go is in internal/api

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// User represents a user in the system.
// Corresponds to api.User but includes DynamoDB keys.
type User struct {
	PK     string    `dynamodbav:"PK"` // Partition Key: USER#<user_id>
	SK     string    `dynamodbav:"SK"` // Sort Key: METADATA
	UserID uuid.UUID `dynamodbav:"UserID"`
	Email  string    `dynamodbav:"Email"` // Consider making this unique if needed
	// Store password hash, not plain text. Omitted here for simplicity.
	CreatedAt time.Time `dynamodbav:"CreatedAt"`
	UpdatedAt time.Time `dynamodbav:"UpdatedAt"`
	// Add other user profile fields as needed
}

// ToApiUser converts internal User to API User
func ToApiUser(mu User) api.User {
	userID := mu.UserID // Copy UUID
	email := mu.Email   // Copy string (assuming email is stored)
	// Convert string email to openapi_types.Email if necessary, depends on how it's stored/validated
	apiEmail := openapi_types.Email(email)

	return api.User{
		UserId: &userID,   // Assign pointer
		Email:  &apiEmail, // Assign pointer
	}
}
