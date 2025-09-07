package domain

import (
	"context"
)

// AccountRepository defines the interface for account data operations
type AccountRepository interface {
	Create(ctx context.Context, account *Account) error
	GetByID(ctx context.Context, id string) (*Account, error)
	GetByUserID(ctx context.Context, userID string) ([]*Account, error)
	Update(ctx context.Context, account *Account) error
	UpdateBalance(ctx context.Context, id string, newBalance float64, version int64) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, limit, offset int) ([]*Account, error)
}

// TransactionRepository defines the interface for transaction data operations
type TransactionRepository interface {
	Create(ctx context.Context, transaction *Transaction) error
	GetByID(ctx context.Context, id string) (*Transaction, error)
	GetByAccountID(ctx context.Context, accountID string, filter *TransactionFilter) ([]*Transaction, error)
	GetByFilter(ctx context.Context, filter *TransactionFilter) ([]*Transaction, error)
	Update(ctx context.Context, transaction *Transaction) error
	UpdateStatus(ctx context.Context, id string, status TransactionStatus, errorMessage string) error
	Count(ctx context.Context, filter *TransactionFilter) (int64, error)
}

// MessageQueue defines the interface for message queue operations
type MessageQueue interface {
	Publish(ctx context.Context, queueName string, message []byte) error
	Subscribe(ctx context.Context, queueName string, handler func([]byte) error) error
	Close() error
}

// AccountService defines the interface for account business logic
type AccountService interface {
	CreateAccount(ctx context.Context, userID string, initialBalance float64, currency string) (*Account, error)
	GetAccount(ctx context.Context, id string) (*Account, error)
	GetAccountsByUser(ctx context.Context, userID string) ([]*Account, error)
	GetAccountSummary(ctx context.Context, id string) (*AccountSummary, error)
	ListAccounts(ctx context.Context, limit, offset int) ([]*Account, error)
	DeactivateAccount(ctx context.Context, id string) error
}

// TransactionService defines the interface for transaction business logic
type TransactionService interface {
	ProcessTransaction(ctx context.Context, request *TransactionRequest) (*Transaction, error)
	GetTransaction(ctx context.Context, id string) (*Transaction, error)
	GetTransactionHistory(ctx context.Context, accountID string, filter *TransactionFilter) ([]*Transaction, error)
	GetTransactionsByFilter(ctx context.Context, filter *TransactionFilter) ([]*Transaction, error)
	CancelTransaction(ctx context.Context, id string) error
}

// LedgerService defines the interface for ledger operations
type LedgerService interface {
	RecordTransaction(ctx context.Context, transaction *Transaction) error
	GetAccountBalance(ctx context.Context, accountID string) (float64, error)
	GetTransactionHistory(ctx context.Context, accountID string, filter *TransactionFilter) ([]*Transaction, error)
	GetAccountStatement(ctx context.Context, accountID string, fromDate, toDate string) ([]*Transaction, error)
}

// NotificationService defines the interface for notifications
type NotificationService interface {
	NotifyTransactionCompleted(ctx context.Context, transaction *Transaction) error
	NotifyTransactionFailed(ctx context.Context, transaction *Transaction, error error) error
	NotifyLowBalance(ctx context.Context, account *Account) error
}
