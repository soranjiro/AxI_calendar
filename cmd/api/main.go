package main

import (
	"context"
	"log"

	"github.com/soranjiro/axicalendar/internal/api"
	"github.com/soranjiro/axicalendar/internal/api/handler"
	repo "github.com/soranjiro/axicalendar/internal/repository/dynamodb" // Alias to avoid conflict

	"github.com/labstack/echo/v4"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// --- Dependency Injection ---
	// Initialize DynamoDB Client
	// Use the specific dynamodb package for NewDynamoDBClient
	dbClient, err := repo.NewDynamoDBClient(ctx)
	if err != nil {
		log.Fatalf("Failed to initialize DynamoDB client: %v", err)
	}

	// Initialize Repositories
	// Use the specific dynamodb package for New...Repository functions
	themeRepo := repo.NewThemeRepository(dbClient)
	entryRepo := repo.NewEntryRepository(dbClient)

	// Initialize Handlers
	// Use the specific handler package
	apiHandler := handler.NewApiHandler(entryRepo, themeRepo)

	// --- Echo Setup ---
	e := echo.New()

	// ... existing middleware setup ...

	// --- Dummy Authentication Middleware (for local testing) ---
	// Use the specific handler package
	e.Use(handler.DummyAuthMiddleware)
	log.Println("WARNING: Using DummyAuthMiddleware for local testing. DO NOT USE IN PRODUCTION.")

	// --- Register API Handlers ---
	// The base path is "/" because API Gateway will handle stage paths
	// Use the generated api package for RegisterHandlers
	api.RegisterHandlers(e, apiHandler)

	// ... existing server start and shutdown logic ...
}
