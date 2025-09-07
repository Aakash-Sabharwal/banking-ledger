package domain

import "errors"

var (
	// Account errors
	ErrAccountNotFound   = errors.New("account not found")
	ErrAccountExists     = errors.New("account already exists")
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrAccountInactive   = errors.New("account is inactive")
	ErrInvalidAccountID  = errors.New("invalid account ID")
	ErrConcurrentUpdate  = errors.New("concurrent update detected")

	// Transaction errors
	ErrTransactionNotFound         = errors.New("transaction not found")
	ErrInvalidAmount               = errors.New("invalid amount")
	ErrInvalidTransactionType      = errors.New("invalid transaction type")
	ErrMissingCurrency             = errors.New("missing currency")
	ErrMissingFromAccount          = errors.New("missing from account")
	ErrMissingToAccount            = errors.New("missing to account")
	ErrMissingAccounts             = errors.New("missing from and to accounts")
	ErrSameAccount                 = errors.New("from and to accounts cannot be the same")
	ErrTransactionAlreadyProcessed = errors.New("transaction already processed")
	ErrCurrencyMismatch            = errors.New("currency mismatch")

	// General errors
	ErrInvalidInput       = errors.New("invalid input")
	ErrDatabaseError      = errors.New("database error")
	ErrQueueError         = errors.New("queue error")
	ErrInternalError      = errors.New("internal error")
	ErrServiceUnavailable = errors.New("service unavailable")
)
