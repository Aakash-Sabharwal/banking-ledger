package handlers

import (
	"net/http"
	"strconv"

	"banking-ledger/internal/domain"

	"github.com/labstack/echo/v4"
)

// AccountHandler handles account-related HTTP requests
type AccountHandler struct {
	accountService domain.AccountService
}

// NewAccountHandler creates a new account handler
func NewAccountHandler(accountService domain.AccountService) *AccountHandler {
	return &AccountHandler{
		accountService: accountService,
	}
}

// CreateAccountRequest represents the request body for creating an account
type CreateAccountRequest struct {
	UserID         string  `json:"user_id" validate:"required"`
	InitialBalance float64 `json:"initial_balance" validate:"min=0"`
	Currency       string  `json:"currency" validate:"required,len=3"`
}

// CreateAccount creates a new account
func (h *AccountHandler) CreateAccount(c echo.Context) error {
	var req CreateAccountRequest
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

	account, err := h.accountService.CreateAccount(
		c.Request().Context(),
		req.UserID,
		req.InitialBalance,
		req.Currency,
	)
	if err != nil {
		switch err {
		case domain.ErrAccountExists:
			return c.JSON(http.StatusConflict, map[string]string{
				"error": "Account already exists",
			})
		case domain.ErrInvalidAmount:
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid amount",
			})
		case domain.ErrMissingCurrency:
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Missing currency",
			})
		default:
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Internal server error",
			})
		}
	}

	return c.JSON(http.StatusCreated, account)
}

// GetAccount retrieves an account by ID
func (h *AccountHandler) GetAccount(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Account ID is required",
		})
	}

	account, err := h.accountService.GetAccount(c.Request().Context(), id)
	if err != nil {
		switch err {
		case domain.ErrAccountNotFound:
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Account not found",
			})
		default:
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Internal server error",
			})
		}
	}

	return c.JSON(http.StatusOK, account)
}

// GetAccountsByUser retrieves accounts by user ID
func (h *AccountHandler) GetAccountsByUser(c echo.Context) error {
	userID := c.QueryParam("user_id")
	if userID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "User ID is required",
		})
	}

	accounts, err := h.accountService.GetAccountsByUser(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Internal server error",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"accounts": accounts,
		"count":    len(accounts),
	})
}

// GetAccountSummary retrieves account summary
func (h *AccountHandler) GetAccountSummary(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Account ID is required",
		})
	}

	summary, err := h.accountService.GetAccountSummary(c.Request().Context(), id)
	if err != nil {
		switch err {
		case domain.ErrAccountNotFound:
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Account not found",
			})
		default:
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Internal server error",
			})
		}
	}

	return c.JSON(http.StatusOK, summary)
}

// ListAccounts retrieves accounts with pagination
func (h *AccountHandler) ListAccounts(c echo.Context) error {
	limit := 10
	offset := 0

	if l := c.QueryParam("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	if o := c.QueryParam("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil {
			offset = parsed
		}
	}

	accounts, err := h.accountService.ListAccounts(c.Request().Context(), limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Internal server error",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"accounts": accounts,
		"count":    len(accounts),
		"limit":    limit,
		"offset":   offset,
	})
}

// DeactivateAccount deactivates an account
func (h *AccountHandler) DeactivateAccount(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Account ID is required",
		})
	}

	err := h.accountService.DeactivateAccount(c.Request().Context(), id)
	if err != nil {
		switch err {
		case domain.ErrAccountNotFound:
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Account not found",
			})
		default:
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Internal server error",
			})
		}
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Account deactivated successfully",
	})
}

// GetAccountBalance retrieves the current balance of an account
func (h *AccountHandler) GetAccountBalance(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Account ID is required",
		})
	}

	account, err := h.accountService.GetAccount(c.Request().Context(), id)
	if err != nil {
		switch err {
		case domain.ErrAccountNotFound:
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Account not found",
			})
		default:
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Internal server error",
			})
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"account_id": account.ID,
		"balance":    account.Balance,
		"currency":   account.Currency,
		"status":     account.Status,
		"updated_at": account.UpdatedAt,
	})
}
