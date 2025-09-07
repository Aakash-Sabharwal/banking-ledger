package domain

import (
	"time"
)

// TransactionType represents the type of transaction
type TransactionType string

const (
	TransactionTypeDeposit    TransactionType = "deposit"
	TransactionTypeWithdrawal TransactionType = "withdrawal"
	TransactionTypeTransfer   TransactionType = "transfer"
)

// TransactionStatus represents the status of a transaction
type TransactionStatus string

const (
	TransactionStatusPending   TransactionStatus = "pending"
	TransactionStatusCompleted TransactionStatus = "completed"
	TransactionStatusFailed    TransactionStatus = "failed"
	TransactionStatusCancelled TransactionStatus = "cancelled"
)

// Account represents a bank account
type Account struct {
	ID        string    `json:"id" db:"id"`
	UserID    string    `json:"user_id" db:"user_id"`
	Balance   float64   `json:"balance" db:"balance"`
	Currency  string    `json:"currency" db:"currency"`
	Status    string    `json:"status" db:"status"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
	Version   int64     `json:"version" db:"version"` // For optimistic locking
}

// Transaction represents a transaction in the system
type Transaction struct {
	ID            string                 `json:"id" bson:"_id"`
	Type          TransactionType        `json:"type" bson:"type"`
	FromAccountID *string                `json:"from_account_id,omitempty" bson:"from_account_id,omitempty"`
	ToAccountID   *string                `json:"to_account_id,omitempty" bson:"to_account_id,omitempty"`
	Amount        float64                `json:"amount" bson:"amount"`
	Currency      string                 `json:"currency" bson:"currency"`
	Status        TransactionStatus      `json:"status" bson:"status"`
	Description   string                 `json:"description" bson:"description"`
	Reference     string                 `json:"reference" bson:"reference"`
	Metadata      map[string]interface{} `json:"metadata,omitempty" bson:"metadata,omitempty"`
	CreatedAt     time.Time              `json:"created_at" bson:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at" bson:"updated_at"`
	ProcessedAt   *time.Time             `json:"processed_at,omitempty" bson:"processed_at,omitempty"`
	ErrorMessage  string                 `json:"error_message,omitempty" bson:"error_message,omitempty"`
}

// TransactionRequest represents a request to process a transaction
type TransactionRequest struct {
	ID            string                 `json:"id"`
	Type          TransactionType        `json:"type"`
	FromAccountID *string                `json:"from_account_id,omitempty"`
	ToAccountID   *string                `json:"to_account_id,omitempty"`
	Amount        float64                `json:"amount"`
	Currency      string                 `json:"currency"`
	Description   string                 `json:"description"`
	Reference     string                 `json:"reference"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// IsValid validates the transaction request
func (tr *TransactionRequest) IsValid() error {
	if tr.Amount <= 0 {
		return ErrInvalidAmount
	}

	if tr.Currency == "" {
		return ErrMissingCurrency
	}

	switch tr.Type {
	case TransactionTypeDeposit:
		if tr.ToAccountID == nil {
			return ErrMissingToAccount
		}
	case TransactionTypeWithdrawal:
		if tr.FromAccountID == nil {
			return ErrMissingFromAccount
		}
	case TransactionTypeTransfer:
		if tr.FromAccountID == nil || tr.ToAccountID == nil {
			return ErrMissingAccounts
		}
		if *tr.FromAccountID == *tr.ToAccountID {
			return ErrSameAccount
		}
	default:
		return ErrInvalidTransactionType
	}

	return nil
}

// AccountSummary represents account summary information
type AccountSummary struct {
	Account           *Account   `json:"account"`
	TransactionCount  int64      `json:"transaction_count"`
	LastTransactionAt *time.Time `json:"last_transaction_at"`
}

// TransactionFilter represents filters for transaction queries
type TransactionFilter struct {
	AccountID *string            `json:"account_id,omitempty"`
	Type      *TransactionType   `json:"type,omitempty"`
	Status    *TransactionStatus `json:"status,omitempty"`
	FromDate  *time.Time         `json:"from_date,omitempty"`
	ToDate    *time.Time         `json:"to_date,omitempty"`
	MinAmount *float64           `json:"min_amount,omitempty"`
	MaxAmount *float64           `json:"max_amount,omitempty"`
	Limit     int                `json:"limit,omitempty"`
	Offset    int                `json:"offset,omitempty"`
}
