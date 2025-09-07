package handlers

import (
	"net/http"
	"strconv"
	"time"

	"banking-ledger/internal/domain"

	"github.com/labstack/echo/v4"
)

// TransactionHandler handles transaction-related HTTP requests
type TransactionHandler struct {
	transactionService domain.TransactionService
}

// NewTransactionHandler creates a new transaction handler
func NewTransactionHandler(transactionService domain.TransactionService) *TransactionHandler {
	return &TransactionHandler{
		transactionService: transactionService,
	}
}

// ProcessTransactionRequest represents the request body for processing a transaction
type ProcessTransactionRequest struct {
	Type          domain.TransactionType `json:"type" validate:"required"`
	FromAccountID *string                `json:"from_account_id,omitempty"`
	ToAccountID   *string                `json:"to_account_id,omitempty"`
	Amount        float64                `json:"amount" validate:"required,gt=0"`
	Currency      string                 `json:"currency" validate:"required,len=3"`
	Description   string                 `json:"description"`
	Reference     string                 `json:"reference"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// ProcessTransaction processes a transaction
func (h *TransactionHandler) ProcessTransaction(c echo.Context) error {
	var req ProcessTransactionRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	transactionReq := &domain.TransactionRequest{
		Type:          req.Type,
		FromAccountID: req.FromAccountID,
		ToAccountID:   req.ToAccountID,
		Amount:        req.Amount,
		Currency:      req.Currency,
		Description:   req.Description,
		Reference:     req.Reference,
		Metadata:      req.Metadata,
	}

	transaction, err := h.transactionService.ProcessTransaction(c.Request().Context(), transactionReq)
	if err != nil {
		switch err {
		case domain.ErrInvalidAmount:
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid amount",
			})
		case domain.ErrInvalidTransactionType:
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid transaction type",
			})
		case domain.ErrMissingFromAccount:
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Missing from account",
			})
		case domain.ErrMissingToAccount:
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Missing to account",
			})
		case domain.ErrMissingAccounts:
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Missing from and to accounts",
			})
		case domain.ErrSameAccount:
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "From and to accounts cannot be the same",
			})
		case domain.ErrAccountNotFound:
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Account not found",
			})
		case domain.ErrInsufficientFunds:
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Insufficient funds",
			})
		case domain.ErrAccountInactive:
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Account is inactive",
			})
		case domain.ErrCurrencyMismatch:
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Currency mismatch",
			})
		default:
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Internal server error",
			})
		}
	}

	return c.JSON(http.StatusAccepted, transaction)
}

// GetTransaction retrieves a transaction by ID
func (h *TransactionHandler) GetTransaction(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Transaction ID is required",
		})
	}

	transaction, err := h.transactionService.GetTransaction(c.Request().Context(), id)
	if err != nil {
		switch err {
		case domain.ErrTransactionNotFound:
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Transaction not found",
			})
		default:
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Internal server error",
			})
		}
	}

	return c.JSON(http.StatusOK, transaction)
}

// GetTransactionHistory retrieves transaction history for an account
func (h *TransactionHandler) GetTransactionHistory(c echo.Context) error {
	accountID := c.Param("account_id")
	if accountID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Account ID is required",
		})
	}

	filter := h.parseTransactionFilter(c)
	transactions, err := h.transactionService.GetTransactionHistory(c.Request().Context(), accountID, filter)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Internal server error",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"transactions": transactions,
		"count":        len(transactions),
		"account_id":   accountID,
	})
}

// GetTransactionHistoryByQuery retrieves transaction history using query parameters
func (h *TransactionHandler) GetTransactionHistoryByQuery(c echo.Context) error {
	accountID := c.QueryParam("account_id")
	if accountID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Account ID is required",
		})
	}

	filter := h.parseTransactionFilter(c)
	transactions, err := h.transactionService.GetTransactionHistory(c.Request().Context(), accountID, filter)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Internal server error",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"transactions": transactions,
		"count":        len(transactions),
		"account_id":   accountID,
	})
}

// GetTransactions retrieves transactions by filter
func (h *TransactionHandler) GetTransactions(c echo.Context) error {
	filter := h.parseTransactionFilter(c)
	transactions, err := h.transactionService.GetTransactionsByFilter(c.Request().Context(), filter)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Internal server error",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"transactions": transactions,
		"count":        len(transactions),
	})
}

// CancelTransaction cancels a pending transaction
func (h *TransactionHandler) CancelTransaction(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Transaction ID is required",
		})
	}

	err := h.transactionService.CancelTransaction(c.Request().Context(), id)
	if err != nil {
		switch err {
		case domain.ErrTransactionNotFound:
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Transaction not found",
			})
		case domain.ErrTransactionAlreadyProcessed:
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Transaction already processed",
			})
		default:
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Internal server error",
			})
		}
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Transaction cancelled successfully",
	})
}

// parseTransactionFilter parses query parameters into a transaction filter
func (h *TransactionHandler) parseTransactionFilter(c echo.Context) *domain.TransactionFilter {
	filter := &domain.TransactionFilter{}

	if accountID := c.QueryParam("account_id"); accountID != "" {
		filter.AccountID = &accountID
	}

	if txType := c.QueryParam("type"); txType != "" {
		transactionType := domain.TransactionType(txType)
		filter.Type = &transactionType
	}

	if status := c.QueryParam("status"); status != "" {
		transactionStatus := domain.TransactionStatus(status)
		filter.Status = &transactionStatus
	}

	if fromDate := c.QueryParam("from_date"); fromDate != "" {
		if parsed, err := time.Parse(time.RFC3339, fromDate); err == nil {
			filter.FromDate = &parsed
		}
	}

	if toDate := c.QueryParam("to_date"); toDate != "" {
		if parsed, err := time.Parse(time.RFC3339, toDate); err == nil {
			filter.ToDate = &parsed
		}
	}

	if minAmount := c.QueryParam("min_amount"); minAmount != "" {
		if parsed, err := strconv.ParseFloat(minAmount, 64); err == nil {
			filter.MinAmount = &parsed
		}
	}

	if maxAmount := c.QueryParam("max_amount"); maxAmount != "" {
		if parsed, err := strconv.ParseFloat(maxAmount, 64); err == nil {
			filter.MaxAmount = &parsed
		}
	}

	if limit := c.QueryParam("limit"); limit != "" {
		if parsed, err := strconv.Atoi(limit); err == nil {
			filter.Limit = parsed
		}
	} else {
		filter.Limit = 10
	}

	if offset := c.QueryParam("offset"); offset != "" {
		if parsed, err := strconv.Atoi(offset); err == nil {
			filter.Offset = parsed
		}
	}

	return filter
}
