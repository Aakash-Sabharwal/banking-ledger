package usecase

import (
	"context"
	"time"

	"banking-ledger/internal/domain"

	"github.com/google/uuid"
)

// AccountUseCase implements the AccountService interface
type AccountUseCase struct {
	accountRepo     domain.AccountRepository
	transactionRepo domain.TransactionRepository
}

// NewAccountUseCase creates a new account use case
func NewAccountUseCase(
	accountRepo domain.AccountRepository,
	transactionRepo domain.TransactionRepository,
) domain.AccountService {
	return &AccountUseCase{
		accountRepo:     accountRepo,
		transactionRepo: transactionRepo,
	}
}

// CreateAccount creates a new account
func (uc *AccountUseCase) CreateAccount(ctx context.Context, userID string, initialBalance float64, currency string) (*domain.Account, error) {
	if initialBalance < 0 {
		return nil, domain.ErrInvalidAmount
	}

	if currency == "" {
		return nil, domain.ErrMissingCurrency
	}

	account := &domain.Account{
		ID:        uuid.New().String(),
		UserID:    userID,
		Balance:   initialBalance,
		Currency:  currency,
		Status:    "active",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Version:   1,
	}

	err := uc.accountRepo.Create(ctx, account)
	if err != nil {
		return nil, err
	}

	return account, nil
}

// GetAccount retrieves an account by ID
func (uc *AccountUseCase) GetAccount(ctx context.Context, id string) (*domain.Account, error) {
	return uc.accountRepo.GetByID(ctx, id)
}

// GetAccountsByUser retrieves accounts by user ID
func (uc *AccountUseCase) GetAccountsByUser(ctx context.Context, userID string) ([]*domain.Account, error) {
	return uc.accountRepo.GetByUserID(ctx, userID)
}

// GetAccountSummary retrieves account summary with transaction statistics
func (uc *AccountUseCase) GetAccountSummary(ctx context.Context, id string) (*domain.AccountSummary, error) {
	// Get account
	account, err := uc.accountRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Get transaction count
	filter := &domain.TransactionFilter{
		AccountID: &id,
	}

	count, err := uc.transactionRepo.Count(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Get last transaction
	filter.Limit = 1
	transactions, err := uc.transactionRepo.GetByAccountID(ctx, id, filter)
	if err != nil {
		return nil, err
	}

	var lastTransactionAt *time.Time
	if len(transactions) > 0 {
		lastTransactionAt = &transactions[0].CreatedAt
	}

	return &domain.AccountSummary{
		Account:           account,
		TransactionCount:  count,
		LastTransactionAt: lastTransactionAt,
	}, nil
}

// ListAccounts retrieves accounts with pagination
func (uc *AccountUseCase) ListAccounts(ctx context.Context, limit, offset int) ([]*domain.Account, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	return uc.accountRepo.List(ctx, limit, offset)
}

// DeactivateAccount deactivates an account
func (uc *AccountUseCase) DeactivateAccount(ctx context.Context, id string) error {
	account, err := uc.accountRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	account.Status = "inactive"
	account.UpdatedAt = time.Now()

	return uc.accountRepo.Update(ctx, account)
}
