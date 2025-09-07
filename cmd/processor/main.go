package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"banking-ledger/internal/config"
	"banking-ledger/internal/queue"
	"banking-ledger/internal/repository"
	"banking-ledger/internal/usecase"
	"banking-ledger/pkg/database"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize logger
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starting Banking Ledger Transaction Processor")

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

	// Initialize message queue
	messageQueue, err := queue.NewRabbitMQQueue(cfg.RabbitMQ.URL)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer messageQueue.Close()

	// Initialize repositories
	accountRepo := repository.NewPostgreSQLAccountRepository(postgresDB)
	transactionRepo := repository.NewMongoTransactionRepository(mongoDB, cfg.MongoDB.Collection)

	// Initialize transaction service
	transactionService := usecase.NewTransactionUseCase(
		accountRepo,
		transactionRepo,
		messageQueue,
		cfg.RabbitMQ.TransactionQueue,
	)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start transaction processor
	if err := transactionService.(*usecase.TransactionUseCase).StartTransactionProcessor(ctx); err != nil {
		log.Fatalf("Failed to start transaction processor: %v", err)
	}

	log.Println("Transaction processor started and listening for messages...")

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down transaction processor...")
	cancel()

	log.Println("Transaction processor stopped")
}
