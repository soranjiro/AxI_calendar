package main

import (
	"context"
	"log"
	"net/http"  // HTTPサーバー関連のパッケージをインポート
	"os"        // OSシグナル処理のためにインポート
	"os/signal" // OSシグナル処理のためにインポート
	"syscall"   // OSシグナル処理のためにインポート
	"time"      // タイムアウト処理のためにインポート

	repo "github.com/soranjiro/axicalendar/internal/adapter/persistence/dynamodb"
	"github.com/soranjiro/axicalendar/internal/domain/feature" // feature をインポート
	"github.com/soranjiro/axicalendar/internal/presentation/api"
	"github.com/soranjiro/axicalendar/internal/presentation/api/handler"
	"github.com/soranjiro/axicalendar/internal/usecase"
	"github.com/soranjiro/axicalendar/internal/usecase/services"                 // services をインポート

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware" // ミドルウェアパッケージをインポート
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

	// Initialize Feature Executor Registry
	featureRegistry := feature.NewInMemoryExecutorRegistry()

	// Initialize Services
	entryService := services.NewEntryService(entryRepo)
	themeService := services.NewThemeService(themeRepo)

	// Initialize Use Case (using the consolidated constructor)
	uc := usecase.NewUseCase(themeRepo, entryRepo, featureRegistry, entryService, themeService) // featureRegistry, entryService, themeService を渡す

	// Initialize Handlers
	// Pass the single use case interface
	apiHandler := handler.NewApiHandler(uc)

	// --- Echo Setup ---
	e := echo.New()

	// --- Middleware ---
	e.Use(middleware.Logger())  // リクエストログ
	e.Use(middleware.Recover()) // パニックからの回復

	// --- Dummy Authentication Middleware (for local testing) ---
	// Use the specific handler package
	e.Use(handler.DummyAuthMiddleware)
	log.Println("WARNING: Using DummyAuthMiddleware for local testing. DO NOT USE IN PRODUCTION.")

	// --- Register API Handlers ---
	// The base path is "/" because API Gateway will handle stage paths
	// Use the generated api package for RegisterHandlers
	api.RegisterHandlers(e, apiHandler)

	// --- Start Server ---
	go func() {
		// ポート8080でサーバーを起動
		if err := e.Start(":8080"); err != nil && err != http.ErrServerClosed {
			e.Logger.Fatal("shutting down the server")
		}
	}()

	// --- Graceful Shutdown ---
	quit := make(chan os.Signal, 1)
	// SIGINT (Ctrl+C) または SIGTERM を受け取ったら通知
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit // シグナルを受け取るまで待機
	log.Println("Shutting down server...")

	// 10秒のタイムアウト付きでシャットダウン
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := e.Shutdown(shutdownCtx); err != nil {
		e.Logger.Fatal(err)
	}
	log.Println("Server gracefully stopped")
}
