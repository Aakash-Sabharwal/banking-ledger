package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"banking-ledger/api/routes"
	"banking-ledger/internal/config"
	"banking-ledger/internal/queue"
	"banking-ledger/internal/repository"
	"banking-ledger/internal/usecase"
	"banking-ledger/pkg/database"

	"github.com/labstack/echo/v4"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize logger
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Printf("Starting Banking Ledger API on port %s", cfg.Server.Port)

	// Initialize databases
	postgresDB, err := database.NewPostgreSQLConnection(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer postgresDB.Close()

	mongoDB, err := database.NewMongoDBConnection(cfg.MongoDB)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	// Run migrations
	if err := database.MigratePostgreSQL(postgresDB); err != nil {
		log.Fatalf("Failed to migrate PostgreSQL: %v", err)
	}

	if err := database.CreateMongoDBIndexes(mongoDB, cfg.MongoDB.Collection); err != nil {
		log.Fatalf("Failed to create MongoDB indexes: %v", err)
	}

	// Initialize message queue
	messageQueue, err := queue.NewRabbitMQQueue(cfg.RabbitMQ.URL)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer messageQueue.Close()

	// Initialize repositories
	accountRepo := repository.NewPostgreSQLAccountRepository(postgresDB)
	transactionRepo := repository.NewMongoTransactionRepository(mongoDB, cfg.MongoDB.Collection)

	// Initialize use cases
	accountService := usecase.NewAccountUseCase(accountRepo, transactionRepo)
	transactionService := usecase.NewTransactionUseCase(
		accountRepo,
		transactionRepo,
		messageQueue,
		cfg.RabbitMQ.TransactionQueue,
	)

	// Initialize Echo
	e := echo.New()

	// Setup routes
	routes.SetupRoutes(e, accountService, transactionService)

	// Start server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Server.Port),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start server in a goroutine
	go func() {
		if err := e.StartServer(server); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	log.Printf("Server started on port %s", cfg.Server.Port)

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Create a deadline for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	// Shutdown server
	if err := e.Shutdown(ctx); err != nil {
		log.Fatalf("Failed to shutdown server: %v", err)
	}

	log.Println("Server stopped")
}
