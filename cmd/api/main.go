package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/soranjiro/axicalendar/internal/api"
	"github.com/soranjiro/axicalendar/internal/handlers"
	"github.com/soranjiro/axicalendar/internal/repository"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// --- Dependency Injection ---
	// Initialize DynamoDB Client
	dbClient, err := repository.NewDynamoDBClient(ctx)
	if err != nil {
		log.Fatalf("Failed to initialize DynamoDB client: %v", err)
	}

	// Initialize Repositories
	themeRepo := repository.NewThemeRepository(dbClient)
	entryRepo := repository.NewEntryRepository(dbClient)

	// Initialize Handlers
	apiHandler := handlers.NewApiHandler(entryRepo, themeRepo)

	// --- Echo Setup ---
	e := echo.New()

	// --- Middleware ---
	// Logger
	e.Use(middleware.Logger())
	// Recover
	e.Use(middleware.Recover())
	// CORS (Allow all for local development)
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"}, // Be more specific in production
		AllowMethods: []string{http.MethodGet, http.MethodHead, http.MethodPut, http.MethodPatch, http.MethodPost, http.MethodDelete},
	}))

	// --- Dummy Authentication Middleware (for local testing) ---
	// This should be replaced with actual Cognito verification in production/staging
	e.Use(handlers.DummyAuthMiddleware)
	log.Println("WARNING: Using DummyAuthMiddleware for local testing. DO NOT USE IN PRODUCTION.")

	// --- Register API Handlers ---
	// The base path is "/" because API Gateway will handle stage paths
	api.RegisterHandlers(e, apiHandler)

	// --- Start Server ---
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default port
	}
	serverAddr := ":" + port

	// Start server in a goroutine
	go func() {
		if err := e.Start(serverAddr); err != nil && err != http.ErrServerClosed {
			e.Logger.Fatal("shutting down the server")
		}
	}()

	log.Printf("Server started on %s", serverAddr)

	// --- Graceful Shutdown ---
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit // Block until signal is received

	log.Println("Shutting down server...")
	cancel() // Signal background tasks to cancel

	// Create a context with timeout for shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := e.Shutdown(shutdownCtx); err != nil {
		e.Logger.Fatal(err)
	}

	log.Println("Server gracefully stopped")
}
