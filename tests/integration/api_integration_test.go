package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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

// TestConfig for integration tests
type TestConfig struct {
	PostgresURL string
	MongoURL    string
	RabbitMQURL string
}

func getTestConfig() *TestConfig {
	return &TestConfig{
		PostgresURL: "postgres://postgres:postgres@localhost:5432/banking_ledger_test?sslmode=disable",
		MongoURL:    "mongodb://localhost:27017/ledger_test",
		RabbitMQURL: "amqp://guest:guest@localhost:5672/",
	}
}

// setupTestServer sets up a test server with real database connections
func setupTestServer(t *testing.T) (*echo.Echo, func()) {
	testCfg := getTestConfig()

	// Setup PostgreSQL
	postgresDB, err := sqlx.Connect("postgres", testCfg.PostgresURL)
	if err != nil {
		t.Skipf("Skipping integration test: PostgreSQL not available: %v", err)
	}

	// Setup MongoDB
	cfg := config.MongoDBConfig{
		URL:        testCfg.MongoURL,
		Database:   "ledger_test",
		Collection: "transactions_test",
	}
	mongoDB, err := database.NewMongoDBConnection(cfg)
	if err != nil {
		t.Skipf("Skipping integration test: MongoDB not available: %v", err)
	}

	// Setup RabbitMQ
	messageQueue, err := queue.NewRabbitMQQueue(testCfg.RabbitMQURL)
	if err != nil {
		t.Skipf("Skipping integration test: RabbitMQ not available: %v", err)
	}

	// Run migrations
	if err := database.MigratePostgreSQL(postgresDB); err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	// Initialize repositories
	accountRepo := repository.NewPostgreSQLAccountRepository(postgresDB)
	transactionRepo := repository.NewMongoTransactionRepository(mongoDB, cfg.Collection)

	// Initialize use cases
	accountService := usecase.NewAccountUseCase(accountRepo, transactionRepo)
	transactionService := usecase.NewTransactionUseCase(
		accountRepo,
		transactionRepo,
		messageQueue,
		"test_transactions",
	)

	// Setup Echo server
	e := echo.New()
	routes.SetupRoutes(e, accountService, transactionService)

	// Cleanup function
	cleanup := func() {
		// Clean up test data
		postgresDB.Exec("DELETE FROM accounts")
		mongoDB.Collection(cfg.Collection).Drop(context.Background())

		// Close connections
		postgresDB.Close()
		messageQueue.Close()
	}

	return e, cleanup
}

func TestAccountIntegration(t *testing.T) {
	e, cleanup := setupTestServer(t)
	defer cleanup()

	t.Run("Create Account", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"user_id":         "test-user-1",
			"initial_balance": 1000.0,
			"currency":        "USD",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/accounts", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Errorf("Expected status %d, got %d", http.StatusCreated, rec.Code)
		}

		var account domain.Account
		if err := json.Unmarshal(rec.Body.Bytes(), &account); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if account.UserID != "test-user-1" {
			t.Errorf("Expected user_id 'test-user-1', got '%s'", account.UserID)
		}
		if account.Balance != 1000.0 {
			t.Errorf("Expected balance 1000.0, got %f", account.Balance)
		}
		if account.Currency != "USD" {
			t.Errorf("Expected currency 'USD', got '%s'", account.Currency)
		}
	})

	t.Run("Get Account", func(t *testing.T) {
		// First create an account
		reqBody := map[string]interface{}{
			"user_id":         "test-user-2",
			"initial_balance": 500.0,
			"currency":        "EUR",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/accounts", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		var createdAccount domain.Account
		json.Unmarshal(rec.Body.Bytes(), &createdAccount)

		// Now get the account
		req = httptest.NewRequest(http.MethodGet, "/api/v1/accounts/"+createdAccount.ID, nil)
		rec = httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
		}

		var account domain.Account
		if err := json.Unmarshal(rec.Body.Bytes(), &account); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if account.ID != createdAccount.ID {
			t.Errorf("Expected ID '%s', got '%s'", createdAccount.ID, account.ID)
		}
	})

	t.Run("Create Account - Validation Error", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"user_id":         "test-user-3",
			"initial_balance": -100.0, // Invalid negative balance
			"currency":        "USD",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/accounts", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})
}

func TestTransactionIntegration(t *testing.T) {
	e, cleanup := setupTestServer(t)
	defer cleanup()

	// Create test accounts first
	var account1, account2 domain.Account

	// Create account 1
	reqBody := map[string]interface{}{
		"user_id":         "user1",
		"initial_balance": 1000.0,
		"currency":        "USD",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/accounts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	json.Unmarshal(rec.Body.Bytes(), &account1)

	// Create account 2
	reqBody["user_id"] = "user2"
	reqBody["initial_balance"] = 500.0
	body, _ = json.Marshal(reqBody)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/accounts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	json.Unmarshal(rec.Body.Bytes(), &account2)

	t.Run("Process Deposit Transaction", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"type":          "deposit",
			"to_account_id": account1.ID,
			"amount":        200.0,
			"currency":      "USD",
			"description":   "Test deposit",
			"reference":     "DEP001",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusAccepted {
			t.Errorf("Expected status %d, got %d", http.StatusAccepted, rec.Code)
		}

		var transaction domain.Transaction
		if err := json.Unmarshal(rec.Body.Bytes(), &transaction); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if transaction.Type != domain.TransactionTypeDeposit {
			t.Errorf("Expected type 'deposit', got '%s'", transaction.Type)
		}
		if transaction.Amount != 200.0 {
			t.Errorf("Expected amount 200.0, got %f", transaction.Amount)
		}
		if transaction.Status != domain.TransactionStatusPending {
			t.Errorf("Expected status 'pending', got '%s'", transaction.Status)
		}
	})

	t.Run("Process Transfer Transaction", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"type":            "transfer",
			"from_account_id": account1.ID,
			"to_account_id":   account2.ID,
			"amount":          150.0,
			"currency":        "USD",
			"description":     "Test transfer",
			"reference":       "TRF001",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusAccepted {
			t.Errorf("Expected status %d, got %d", http.StatusAccepted, rec.Code)
		}

		var transaction domain.Transaction
		if err := json.Unmarshal(rec.Body.Bytes(), &transaction); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if transaction.Type != domain.TransactionTypeTransfer {
			t.Errorf("Expected type 'transfer', got '%s'", transaction.Type)
		}
	})

	t.Run("Process Transaction - Validation Error", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"type":          "deposit",
			"to_account_id": account1.ID,
			"amount":        -50.0, // Invalid negative amount
			"currency":      "USD",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})
}

func TestHealthCheck(t *testing.T) {
	e, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got '%v'", response["status"])
	}
}
