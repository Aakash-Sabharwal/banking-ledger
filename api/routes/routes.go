package routes

import (
	"banking-ledger/api/handlers"
	"banking-ledger/api/middleware"
	"banking-ledger/internal/domain"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

// CustomValidator provides custom validation for request bodies
type CustomValidator struct {
	validator *validator.Validate
}

// Validate validates the request body
func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}

// SetupRoutes sets up all application routes
func SetupRoutes(
	e *echo.Echo,
	accountService domain.AccountService,
	transactionService domain.TransactionService,
) {
	// Set custom validator
	e.Validator = &CustomValidator{validator: validator.New()}

	// Global middleware
	e.Use(middleware.RequestID())
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())
	e.Use(middleware.RateLimiter())
	e.Use(middleware.Timeout(30 * time.Second))
	e.Use(middleware.HealthCheck())

	// Initialize handlers
	accountHandler := handlers.NewAccountHandler(accountService)
	transactionHandler := handlers.NewTransactionHandler(transactionService)

	// API version 1
	v1 := e.Group("/api/v1")

	// Account routes
	accounts := v1.Group("/accounts")
	{
		accounts.POST("", accountHandler.CreateAccount)
		accounts.GET("", accountHandler.ListAccounts)
		accounts.GET("/search", accountHandler.GetAccountsByUser)
		accounts.GET("/:id", accountHandler.GetAccount)
		accounts.GET("/:id/balance", accountHandler.GetAccountBalance)
		accounts.GET("/:id/summary", accountHandler.GetAccountSummary)
		accounts.PATCH("/:id/deactivate", accountHandler.DeactivateAccount)
	}

	// Transaction routes
	transactions := v1.Group("/transactions")
	{
		transactions.POST("", transactionHandler.ProcessTransaction)
		transactions.GET("", transactionHandler.GetTransactions)
		transactions.GET("/history", transactionHandler.GetTransactionHistoryByQuery)
		transactions.GET("/:id", transactionHandler.GetTransaction)
		transactions.PATCH("/:id/cancel", transactionHandler.CancelTransaction)
	}

	// Account transaction routes
	v1.GET("/accounts/:account_id/transactions", transactionHandler.GetTransactionHistory)

	// API documentation endpoint
	v1.GET("/docs", func(c echo.Context) error {
		return c.JSON(200, map[string]interface{}{
			"service": "Banking Ledger API",
			"version": "1.0.0",
			"endpoints": map[string]interface{}{
				"accounts": map[string]interface{}{
					"POST /api/v1/accounts":                          "Create account",
					"GET /api/v1/accounts":                           "List accounts",
					"GET /api/v1/accounts/search?user_id={}":         "Get accounts by user",
					"GET /api/v1/accounts/{id}":                      "Get account",
					"GET /api/v1/accounts/{id}/balance":              "Get account balance",
					"GET /api/v1/accounts/{id}/summary":              "Get account summary",
					"PATCH /api/v1/accounts/{id}/deactivate":         "Deactivate account",
					"GET /api/v1/accounts/{account_id}/transactions": "Get account transactions",
				},
				"transactions": map[string]interface{}{
					"POST /api/v1/transactions":                      "Process transaction",
					"GET /api/v1/transactions":                       "Get transactions",
					"GET /api/v1/transactions/history?account_id={}": "Get transaction history by query",
					"GET /api/v1/transactions/{id}":                  "Get transaction",
					"PATCH /api/v1/transactions/{id}/cancel":         "Cancel transaction",
				},
			},
		})
	})
}
