package domain_test

import (
	"banking-ledger/internal/domain"
	"testing"
)

func TestTransactionRequest_IsValid(t *testing.T) {
	tests := []struct {
		name        string
		request     domain.TransactionRequest
		expectError bool
		expectedErr error
	}{
		{
			name: "valid deposit request",
			request: domain.TransactionRequest{
				Type:        domain.TransactionTypeDeposit,
				ToAccountID: stringPtr("account1"),
				Amount:      100.0,
				Currency:    "USD",
			},
			expectError: false,
		},
		{
			name: "valid withdrawal request",
			request: domain.TransactionRequest{
				Type:          domain.TransactionTypeWithdrawal,
				FromAccountID: stringPtr("account1"),
				Amount:        50.0,
				Currency:      "USD",
			},
			expectError: false,
		},
		{
			name: "valid transfer request",
			request: domain.TransactionRequest{
				Type:          domain.TransactionTypeTransfer,
				FromAccountID: stringPtr("account1"),
				ToAccountID:   stringPtr("account2"),
				Amount:        75.0,
				Currency:      "USD",
			},
			expectError: false,
		},
		{
			name: "invalid amount - zero",
			request: domain.TransactionRequest{
				Type:        domain.TransactionTypeDeposit,
				ToAccountID: stringPtr("account1"),
				Amount:      0,
				Currency:    "USD",
			},
			expectError: true,
			expectedErr: domain.ErrInvalidAmount,
		},
		{
			name: "invalid amount - negative",
			request: domain.TransactionRequest{
				Type:        domain.TransactionTypeDeposit,
				ToAccountID: stringPtr("account1"),
				Amount:      -10.0,
				Currency:    "USD",
			},
			expectError: true,
			expectedErr: domain.ErrInvalidAmount,
		},
		{
			name: "missing currency",
			request: domain.TransactionRequest{
				Type:        domain.TransactionTypeDeposit,
				ToAccountID: stringPtr("account1"),
				Amount:      100.0,
			},
			expectError: true,
			expectedErr: domain.ErrMissingCurrency,
		},
		{
			name: "deposit missing to account",
			request: domain.TransactionRequest{
				Type:     domain.TransactionTypeDeposit,
				Amount:   100.0,
				Currency: "USD",
			},
			expectError: true,
			expectedErr: domain.ErrMissingToAccount,
		},
		{
			name: "withdrawal missing from account",
			request: domain.TransactionRequest{
				Type:     domain.TransactionTypeWithdrawal,
				Amount:   50.0,
				Currency: "USD",
			},
			expectError: true,
			expectedErr: domain.ErrMissingFromAccount,
		},
		{
			name: "transfer missing from account",
			request: domain.TransactionRequest{
				Type:        domain.TransactionTypeTransfer,
				ToAccountID: stringPtr("account2"),
				Amount:      75.0,
				Currency:    "USD",
			},
			expectError: true,
			expectedErr: domain.ErrMissingAccounts,
		},
		{
			name: "transfer missing to account",
			request: domain.TransactionRequest{
				Type:          domain.TransactionTypeTransfer,
				FromAccountID: stringPtr("account1"),
				Amount:        75.0,
				Currency:      "USD",
			},
			expectError: true,
			expectedErr: domain.ErrMissingAccounts,
		},
		{
			name: "transfer same account",
			request: domain.TransactionRequest{
				Type:          domain.TransactionTypeTransfer,
				FromAccountID: stringPtr("account1"),
				ToAccountID:   stringPtr("account1"),
				Amount:        75.0,
				Currency:      "USD",
			},
			expectError: true,
			expectedErr: domain.ErrSameAccount,
		},
		{
			name: "invalid transaction type",
			request: domain.TransactionRequest{
				Type:        domain.TransactionType("invalid"),
				ToAccountID: stringPtr("account1"),
				Amount:      100.0,
				Currency:    "USD",
			},
			expectError: true,
			expectedErr: domain.ErrInvalidTransactionType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.IsValid()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tt.expectedErr != nil && err != tt.expectedErr {
					t.Errorf("Expected error %v, got %v", tt.expectedErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got %v", err)
				}
			}
		})
	}
}

func stringPtr(s string) *string {
	return &s
}
