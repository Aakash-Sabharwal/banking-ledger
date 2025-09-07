package usecase

import (
	"context"
	"testing"
	"time"

	"banking-ledger/internal/domain"
	"banking-ledger/internal/usecase"
)

// MockAccountRepository implements domain.AccountRepository for testing
type MockAccountRepository struct {
	accounts map[string]*domain.Account
	nextID   int
}

func NewMockAccountRepository() *MockAccountRepository {
	return &MockAccountRepository{
		accounts: make(map[string]*domain.Account),
		nextID:   1,
	}
}

func (m *MockAccountRepository) Create(ctx context.Context, account *domain.Account) error {
	if account.ID == "" {
		account.ID = "test-id"
	}

	// Check for existing account with same user_id and currency
	for _, existing := range m.accounts {
		if existing.UserID == account.UserID && existing.Currency == account.Currency {
			return domain.ErrAccountExists
		}
	}

	account.CreatedAt = time.Now()
	account.UpdatedAt = time.Now()
	account.Version = 1

	m.accounts[account.ID] = account
	return nil
}

func (m *MockAccountRepository) GetByID(ctx context.Context, id string) (*domain.Account, error) {
	account, exists := m.accounts[id]
	if !exists {
		return nil, domain.ErrAccountNotFound
	}
	return account, nil
}

func (m *MockAccountRepository) GetByUserID(ctx context.Context, userID string) ([]*domain.Account, error) {
	var accounts []*domain.Account
	for _, account := range m.accounts {
		if account.UserID == userID {
			accounts = append(accounts, account)
		}
	}
	return accounts, nil
}

func (m *MockAccountRepository) Update(ctx context.Context, account *domain.Account) error {
	existing, exists := m.accounts[account.ID]
	if !exists {
		return domain.ErrAccountNotFound
	}

	if existing.Version != account.Version {
		return domain.ErrConcurrentUpdate
	}

	account.UpdatedAt = time.Now()
	account.Version++
	m.accounts[account.ID] = account
	return nil
}

func (m *MockAccountRepository) UpdateBalance(ctx context.Context, id string, newBalance float64, version int64) error {
	account, exists := m.accounts[id]
	if !exists {
		return domain.ErrAccountNotFound
	}

	if account.Version != version {
		return domain.ErrConcurrentUpdate
	}

	account.Balance = newBalance
	account.UpdatedAt = time.Now()
	account.Version++
	return nil
}

func (m *MockAccountRepository) Delete(ctx context.Context, id string) error {
	_, exists := m.accounts[id]
	if !exists {
		return domain.ErrAccountNotFound
	}
	delete(m.accounts, id)
	return nil
}

func (m *MockAccountRepository) List(ctx context.Context, limit, offset int) ([]*domain.Account, error) {
	var accounts []*domain.Account
	i := 0
	for _, account := range m.accounts {
		if i >= offset && i < offset+limit {
			accounts = append(accounts, account)
		}
		i++
	}
	return accounts, nil
}

// MockTransactionRepository implements domain.TransactionRepository for testing
type MockTransactionRepository struct {
	transactions map[string]*domain.Transaction
}

func NewMockTransactionRepository() *MockTransactionRepository {
	return &MockTransactionRepository{
		transactions: make(map[string]*domain.Transaction),
	}
}

func (m *MockTransactionRepository) Create(ctx context.Context, transaction *domain.Transaction) error {
	if transaction.ID == "" {
		transaction.ID = "test-tx-id"
	}
	transaction.CreatedAt = time.Now()
	transaction.UpdatedAt = time.Now()
	m.transactions[transaction.ID] = transaction
	return nil
}

func (m *MockTransactionRepository) GetByID(ctx context.Context, id string) (*domain.Transaction, error) {
	transaction, exists := m.transactions[id]
	if !exists {
		return nil, domain.ErrTransactionNotFound
	}
	return transaction, nil
}

func (m *MockTransactionRepository) GetByAccountID(ctx context.Context, accountID string, filter *domain.TransactionFilter) ([]*domain.Transaction, error) {
	var transactions []*domain.Transaction
	for _, tx := range m.transactions {
		if (tx.FromAccountID != nil && *tx.FromAccountID == accountID) ||
			(tx.ToAccountID != nil && *tx.ToAccountID == accountID) {
			transactions = append(transactions, tx)
		}
	}
	return transactions, nil
}

func (m *MockTransactionRepository) GetByFilter(ctx context.Context, filter *domain.TransactionFilter) ([]*domain.Transaction, error) {
	var transactions []*domain.Transaction
	for _, tx := range m.transactions {
		transactions = append(transactions, tx)
	}
	return transactions, nil
}

func (m *MockTransactionRepository) Update(ctx context.Context, transaction *domain.Transaction) error {
	_, exists := m.transactions[transaction.ID]
	if !exists {
		return domain.ErrTransactionNotFound
	}
	transaction.UpdatedAt = time.Now()
	m.transactions[transaction.ID] = transaction
	return nil
}

func (m *MockTransactionRepository) UpdateStatus(ctx context.Context, id string, status domain.TransactionStatus, errorMessage string) error {
	transaction, exists := m.transactions[id]
	if !exists {
		return domain.ErrTransactionNotFound
	}
	transaction.Status = status
	transaction.ErrorMessage = errorMessage
	transaction.UpdatedAt = time.Now()
	if status == domain.TransactionStatusCompleted {
		now := time.Now()
		transaction.ProcessedAt = &now
	}
	return nil
}

func (m *MockTransactionRepository) Count(ctx context.Context, filter *domain.TransactionFilter) (int64, error) {
	return int64(len(m.transactions)), nil
}

func TestAccountUseCase_CreateAccount(t *testing.T) {
	accountRepo := NewMockAccountRepository()
	transactionRepo := NewMockTransactionRepository()
	accountUseCase := usecase.NewAccountUseCase(accountRepo, transactionRepo)

	tests := []struct {
		name           string
		userID         string
		initialBalance float64
		currency       string
		expectError    bool
		expectedError  error
	}{
		{
			name:           "valid account creation",
			userID:         "user1",
			initialBalance: 1000.0,
			currency:       "USD",
			expectError:    false,
		},
		{
			name:           "negative balance",
			userID:         "user2",
			initialBalance: -100.0,
			currency:       "USD",
			expectError:    true,
			expectedError:  domain.ErrInvalidAmount,
		},
		{
			name:           "empty currency",
			userID:         "user3",
			initialBalance: 500.0,
			currency:       "",
			expectError:    true,
			expectedError:  domain.ErrMissingCurrency,
		},
		{
			name:           "duplicate account",
			userID:         "user1", // Same user as first test
			initialBalance: 500.0,
			currency:       "USD", // Same currency as first test
			expectError:    true,
			expectedError:  domain.ErrAccountExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			account, err := accountUseCase.CreateAccount(
				context.Background(),
				tt.userID,
				tt.initialBalance,
				tt.currency,
			)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tt.expectedError != nil && err != tt.expectedError {
					t.Errorf("Expected error %v, got %v", tt.expectedError, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got %v", err)
					return
				}
				if account == nil {
					t.Errorf("Expected account but got nil")
					return
				}
				if account.UserID != tt.userID {
					t.Errorf("Expected userID %s, got %s", tt.userID, account.UserID)
				}
				if account.Balance != tt.initialBalance {
					t.Errorf("Expected balance %f, got %f", tt.initialBalance, account.Balance)
				}
				if account.Currency != tt.currency {
					t.Errorf("Expected currency %s, got %s", tt.currency, account.Currency)
				}
				if account.Status != "active" {
					t.Errorf("Expected status 'active', got %s", account.Status)
				}
			}
		})
	}
}

func TestAccountUseCase_GetAccount(t *testing.T) {
	accountRepo := NewMockAccountRepository()
	transactionRepo := NewMockTransactionRepository()
	accountUseCase := usecase.NewAccountUseCase(accountRepo, transactionRepo)

	// Create a test account
	testAccount := &domain.Account{
		ID:       "test-account-1",
		UserID:   "user1",
		Balance:  1000.0,
		Currency: "USD",
		Status:   "active",
	}
	accountRepo.accounts[testAccount.ID] = testAccount

	tests := []struct {
		name          string
		accountID     string
		expectError   bool
		expectedError error
	}{
		{
			name:        "existing account",
			accountID:   "test-account-1",
			expectError: false,
		},
		{
			name:          "non-existing account",
			accountID:     "non-existing",
			expectError:   true,
			expectedError: domain.ErrAccountNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			account, err := accountUseCase.GetAccount(context.Background(), tt.accountID)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tt.expectedError != nil && err != tt.expectedError {
					t.Errorf("Expected error %v, got %v", tt.expectedError, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got %v", err)
					return
				}
				if account == nil {
					t.Errorf("Expected account but got nil")
					return
				}
				if account.ID != tt.accountID {
					t.Errorf("Expected account ID %s, got %s", tt.accountID, account.ID)
				}
			}
		})
	}
}
