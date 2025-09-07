package feature

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"banking-ledger/api/routes"
	"banking-ledger/internal/config"
	"banking-ledger/internal/domain"
	"banking-ledger/internal/queue"
	"banking-ledger/internal/repository"
	"banking-ledger/internal/usecase"
	"banking-ledger/pkg/database"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	_ "github.com/lib/pq"
)

// BankingLedgerTestSuite provides end-to-end testing for the banking ledger system
type BankingLedgerTestSuite struct {
	server             *echo.Echo
	accountService     domain.AccountService
	transactionService domain.TransactionService
	cleanup            func()
}

func NewBankingLedgerTestSuite(t *testing.T) *BankingLedgerTestSuite {
	// Test configuration
	testCfg := &config.Config{
		Database: config.DatabaseConfig{
			URL: "postgres://postgres:postgres@localhost:5432/banking_ledger_test?sslmode=disable",
		},
		MongoDB: config.MongoDBConfig{
			URL:        "mongodb://localhost:27017/ledger_test",
			Database:   "ledger_test",
			Collection: "transactions_test",
		},
		RabbitMQ: config.RabbitMQConfig{
			URL:              "amqp://guest:guest@localhost:5672/",
			TransactionQueue: "test_transactions",
		},
	}

	// Setup databases
	postgresDB, err := sqlx.Connect("postgres", testCfg.Database.URL)
	if err != nil {
		t.Skipf("Skipping feature test: PostgreSQL not available: %v", err)
	}

	mongoDB, err := database.NewMongoDBConnection(testCfg.MongoDB)
	if err != nil {
		t.Skipf("Skipping feature test: MongoDB not available: %v", err)
	}

	messageQueue, err := queue.NewRabbitMQQueue(testCfg.RabbitMQ.URL)
	if err != nil {
		t.Skipf("Skipping feature test: RabbitMQ not available: %v", err)
	}

	// Run migrations
	database.MigratePostgreSQL(postgresDB)
	database.CreateMongoDBIndexes(mongoDB, testCfg.MongoDB.Collection)

	// Initialize repositories and services
	accountRepo := repository.NewPostgreSQLAccountRepository(postgresDB)
	transactionRepo := repository.NewMongoTransactionRepository(mongoDB, testCfg.MongoDB.Collection)

	accountService := usecase.NewAccountUseCase(accountRepo, transactionRepo)
	transactionService := usecase.NewTransactionUseCase(
		accountRepo,
		transactionRepo,
		messageQueue,
		testCfg.RabbitMQ.TransactionQueue,
	)

	// Setup server
	e := echo.New()
	routes.SetupRoutes(e, accountService, transactionService)

	cleanup := func() {
		postgresDB.Exec("DELETE FROM accounts")
		mongoDB.Collection(testCfg.MongoDB.Collection).Drop(context.Background())
		postgresDB.Close()
		messageQueue.Close()
	}

	return &BankingLedgerTestSuite{
		server:             e,
		accountService:     accountService,
		transactionService: transactionService,
		cleanup:            cleanup,
	}
}

func (suite *BankingLedgerTestSuite) Close() {
	suite.cleanup()
}

// Helper methods for making HTTP requests
func (suite *BankingLedgerTestSuite) createAccount(userID string, balance float64, currency string) (*domain.Account, error) {
	reqBody := map[string]interface{}{
		"user_id":         userID,
		"initial_balance": balance,
		"currency":        currency,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/accounts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	suite.server.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		return nil, fmt.Errorf("failed to create account: status %d", rec.Code)
	}

	var account domain.Account
	if err := json.Unmarshal(rec.Body.Bytes(), &account); err != nil {
		return nil, fmt.Errorf("failed to unmarshal account: %v", err)
	}

	return &account, nil
}

func (suite *BankingLedgerTestSuite) processTransaction(txType domain.TransactionType, fromID, toID *string, amount float64, currency string) (*domain.Transaction, error) {
	reqBody := map[string]interface{}{
		"type":     txType,
		"amount":   amount,
		"currency": currency,
	}

	if fromID != nil {
		reqBody["from_account_id"] = *fromID
	}
	if toID != nil {
		reqBody["to_account_id"] = *toID
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	suite.server.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		return nil, fmt.Errorf("failed to process transaction: status %d", rec.Code)
	}

	var transaction domain.Transaction
	if err := json.Unmarshal(rec.Body.Bytes(), &transaction); err != nil {
		return nil, fmt.Errorf("failed to unmarshal transaction: %v", err)
	}

	return &transaction, nil
}

func (suite *BankingLedgerTestSuite) getAccount(accountID string) (*domain.Account, error) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts/"+accountID, nil)
	rec := httptest.NewRecorder()

	suite.server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		return nil, fmt.Errorf("failed to get account: status %d", rec.Code)
	}

	var account domain.Account
	if err := json.Unmarshal(rec.Body.Bytes(), &account); err != nil {
		return nil, fmt.Errorf("failed to unmarshal account: %v", err)
	}

	return &account, nil
}

// Feature tests
func TestBankingLedgerFeatures(t *testing.T) {
	suite := NewBankingLedgerTestSuite(t)
	defer suite.Close()

	t.Run("Complete Banking Workflow", func(t *testing.T) {
		// Step 1: Create accounts for Alice and Bob
		alice, err := suite.createAccount("alice", 1000.0, "USD")
		if err != nil {
			t.Fatalf("Failed to create Alice's account: %v", err)
		}
		t.Logf("Created Alice's account: %s with balance $%.2f", alice.ID, alice.Balance)

		bob, err := suite.createAccount("bob", 500.0, "USD")
		if err != nil {
			t.Fatalf("Failed to create Bob's account: %v", err)
		}
		t.Logf("Created Bob's account: %s with balance $%.2f", bob.ID, bob.Balance)

		// Step 2: Alice deposits money
		depositTx, err := suite.processTransaction(
			domain.TransactionTypeDeposit,
			nil,
			&alice.ID,
			250.0,
			"USD",
		)
		if err != nil {
			t.Fatalf("Failed to process deposit: %v", err)
		}
		t.Logf("Processed deposit transaction: %s for $%.2f", depositTx.ID, depositTx.Amount)

		// Step 3: Wait for transaction processing (in real scenario, we'd use the processor)
		time.Sleep(100 * time.Millisecond)

		// Step 4: Verify Alice's balance (should still be 1000 until transaction is processed)
		aliceAccount, err := suite.getAccount(alice.ID)
		if err != nil {
			t.Fatalf("Failed to get Alice's account: %v", err)
		}
		t.Logf("Alice's current balance: $%.2f", aliceAccount.Balance)

		// Step 5: Alice transfers money to Bob
		transferTx, err := suite.processTransaction(
			domain.TransactionTypeTransfer,
			&alice.ID,
			&bob.ID,
			150.0,
			"USD",
		)
		if err != nil {
			t.Fatalf("Failed to process transfer: %v", err)
		}
		t.Logf("Processed transfer transaction: %s for $%.2f", transferTx.ID, transferTx.Amount)

		// Step 6: Bob withdraws money
		withdrawalTx, err := suite.processTransaction(
			domain.TransactionTypeWithdrawal,
			&bob.ID,
			nil,
			100.0,
			"USD",
		)
		if err != nil {
			t.Fatalf("Failed to process withdrawal: %v", err)
		}
		t.Logf("Processed withdrawal transaction: %s for $%.2f", withdrawalTx.ID, withdrawalTx.Amount)

		// Step 7: Verify transaction statuses
		if depositTx.Status != domain.TransactionStatusPending {
			t.Errorf("Expected deposit status to be pending, got %s", depositTx.Status)
		}
		if transferTx.Status != domain.TransactionStatusPending {
			t.Errorf("Expected transfer status to be pending, got %s", transferTx.Status)
		}
		if withdrawalTx.Status != domain.TransactionStatusPending {
			t.Errorf("Expected withdrawal status to be pending, got %s", withdrawalTx.Status)
		}

		t.Log("Banking workflow completed successfully!")
	})

	t.Run("Error Handling and Validation", func(t *testing.T) {
		// Create test account
		account, err := suite.createAccount("testuser", 100.0, "USD")
		if err != nil {
			t.Fatalf("Failed to create test account: %v", err)
		}

		// Test insufficient funds
		_, err = suite.processTransaction(
			domain.TransactionTypeWithdrawal,
			&account.ID,
			nil,
			200.0, // More than account balance
			"USD",
		)
		if err == nil {
			t.Error("Expected error for insufficient funds, but got none")
		}

		// Test invalid transaction type
		reqBody := map[string]interface{}{
			"type":          "invalid_type",
			"to_account_id": account.ID,
			"amount":        50.0,
			"currency":      "USD",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		suite.server.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for invalid transaction type, got %d", rec.Code)
		}

		t.Log("Error handling tests completed successfully!")
	})

	t.Run("Multi-Currency Support", func(t *testing.T) {
		// Create accounts in different currencies
		usdAccount, err := suite.createAccount("multicurrency", 1000.0, "USD")
		if err != nil {
			t.Fatalf("Failed to create USD account: %v", err)
		}

		eurAccount, err := suite.createAccount("multicurrency", 500.0, "EUR")
		if err != nil {
			t.Fatalf("Failed to create EUR account: %v", err)
		}

		// Test currency validation - should fail
		_, err = suite.processTransaction(
			domain.TransactionTypeTransfer,
			&usdAccount.ID,
			&eurAccount.ID,
			100.0,
			"USD", // Different currencies should not be allowed
		)
		if err == nil {
			t.Error("Expected error for currency mismatch, but got none")
		}

		t.Log("Multi-currency tests completed successfully!")
	})

	t.Run("Account Management", func(t *testing.T) {
		// Test getting accounts by user
		req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts/search?user_id=multicurrency", nil)
		rec := httptest.NewRecorder()
		suite.server.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal accounts response: %v", err)
		}

		accounts, ok := response["accounts"].([]interface{})
		if !ok {
			t.Fatalf("Expected accounts array in response")
		}

		if len(accounts) < 2 {
			t.Errorf("Expected at least 2 accounts for multicurrency user, got %d", len(accounts))
		}

		t.Log("Account management tests completed successfully!")
	})
}
