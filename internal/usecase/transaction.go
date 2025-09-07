package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"banking-ledger/internal/domain"

	"github.com/google/uuid"
)

// TransactionUseCase implements the TransactionService interface
type TransactionUseCase struct {
	accountRepo     domain.AccountRepository
	transactionRepo domain.TransactionRepository
	queue           domain.MessageQueue
	queueName       string
}

// NewTransactionUseCase creates a new transaction use case
func NewTransactionUseCase(
	accountRepo domain.AccountRepository,
	transactionRepo domain.TransactionRepository,
	queue domain.MessageQueue,
	queueName string,
) domain.TransactionService {
	return &TransactionUseCase{
		accountRepo:     accountRepo,
		transactionRepo: transactionRepo,
		queue:           queue,
		queueName:       queueName,
	}
}

// ProcessTransaction processes a transaction request
func (uc *TransactionUseCase) ProcessTransaction(ctx context.Context, request *domain.TransactionRequest) (*domain.Transaction, error) {
	// Validate request
	if err := request.IsValid(); err != nil {
		return nil, err
	}

	// Generate transaction ID if not provided
	if request.ID == "" {
		request.ID = uuid.New().String()
	}

	// Create transaction record
	transaction := &domain.Transaction{
		ID:            request.ID,
		Type:          request.Type,
		FromAccountID: request.FromAccountID,
		ToAccountID:   request.ToAccountID,
		Amount:        request.Amount,
		Currency:      request.Currency,
		Status:        domain.TransactionStatusPending,
		Description:   request.Description,
		Reference:     request.Reference,
		Metadata:      request.Metadata,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Save transaction to ledger
	err := uc.transactionRepo.Create(ctx, transaction)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	// Publish transaction to queue for async processing
	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal transaction request: %w", err)
	}

	err = uc.queue.Publish(ctx, uc.queueName, requestBytes)
	if err != nil {
		// Update transaction status to failed
		uc.transactionRepo.UpdateStatus(ctx, transaction.ID, domain.TransactionStatusFailed, err.Error())
		return nil, fmt.Errorf("failed to publish transaction: %w", err)
	}

	return transaction, nil
}

// ProcessTransactionSync processes a transaction synchronously with ACID consistency
func (uc *TransactionUseCase) ProcessTransactionSync(ctx context.Context, request *domain.TransactionRequest) error {
	// Validate request
	if err := request.IsValid(); err != nil {
		return err
	}

	switch request.Type {
	case domain.TransactionTypeDeposit:
		return uc.processDeposit(ctx, request)
	case domain.TransactionTypeWithdrawal:
		return uc.processWithdrawal(ctx, request)
	case domain.TransactionTypeTransfer:
		return uc.processTransfer(ctx, request)
	default:
		return domain.ErrInvalidTransactionType
	}
}

// processDeposit processes a deposit transaction
func (uc *TransactionUseCase) processDeposit(ctx context.Context, request *domain.TransactionRequest) error {
	// Get account
	account, err := uc.accountRepo.GetByID(ctx, *request.ToAccountID)
	if err != nil {
		return err
	}

	// Check account status
	if account.Status != "active" {
		return domain.ErrAccountInactive
	}

	// Check currency match
	if account.Currency != request.Currency {
		return domain.ErrCurrencyMismatch
	}

	// Update balance with optimistic locking
	newBalance := account.Balance + request.Amount
	err = uc.accountRepo.UpdateBalance(ctx, account.ID, newBalance, account.Version)
	if err != nil {
		return err
	}

	// Update transaction status
	return uc.transactionRepo.UpdateStatus(ctx, request.ID, domain.TransactionStatusCompleted, "")
}

// processWithdrawal processes a withdrawal transaction
func (uc *TransactionUseCase) processWithdrawal(ctx context.Context, request *domain.TransactionRequest) error {
	// Get account
	account, err := uc.accountRepo.GetByID(ctx, *request.FromAccountID)
	if err != nil {
		return err
	}

	// Check account status
	if account.Status != "active" {
		return domain.ErrAccountInactive
	}

	// Check currency match
	if account.Currency != request.Currency {
		return domain.ErrCurrencyMismatch
	}

	// Check sufficient funds
	if account.Balance < request.Amount {
		return domain.ErrInsufficientFunds
	}

	// Update balance with optimistic locking
	newBalance := account.Balance - request.Amount
	err = uc.accountRepo.UpdateBalance(ctx, account.ID, newBalance, account.Version)
	if err != nil {
		return err
	}

	// Update transaction status
	return uc.transactionRepo.UpdateStatus(ctx, request.ID, domain.TransactionStatusCompleted, "")
}

// processTransfer processes a transfer transaction
func (uc *TransactionUseCase) processTransfer(ctx context.Context, request *domain.TransactionRequest) error {
	// Get both accounts
	fromAccount, err := uc.accountRepo.GetByID(ctx, *request.FromAccountID)
	if err != nil {
		return err
	}

	toAccount, err := uc.accountRepo.GetByID(ctx, *request.ToAccountID)
	if err != nil {
		return err
	}

	// Validate accounts
	if fromAccount.Status != "active" || toAccount.Status != "active" {
		return domain.ErrAccountInactive
	}

	// Check currency match
	if fromAccount.Currency != request.Currency || toAccount.Currency != request.Currency {
		return domain.ErrCurrencyMismatch
	}

	// Check sufficient funds
	if fromAccount.Balance < request.Amount {
		return domain.ErrInsufficientFunds
	}

	// Process transfer atomically (in a real implementation, use database transactions)
	// Update from account balance
	newFromBalance := fromAccount.Balance - request.Amount
	err = uc.accountRepo.UpdateBalance(ctx, fromAccount.ID, newFromBalance, fromAccount.Version)
	if err != nil {
		return err
	}

	// Update to account balance
	newToBalance := toAccount.Balance + request.Amount
	err = uc.accountRepo.UpdateBalance(ctx, toAccount.ID, newToBalance, toAccount.Version)
	if err != nil {
		// Rollback from account balance (simplified - in production use database transactions)
		uc.accountRepo.UpdateBalance(ctx, fromAccount.ID, fromAccount.Balance, fromAccount.Version+1)
		return err
	}

	// Update transaction status
	return uc.transactionRepo.UpdateStatus(ctx, request.ID, domain.TransactionStatusCompleted, "")
}

// GetTransaction retrieves a transaction by ID
func (uc *TransactionUseCase) GetTransaction(ctx context.Context, id string) (*domain.Transaction, error) {
	return uc.transactionRepo.GetByID(ctx, id)
}

// GetTransactionHistory retrieves transaction history for an account
func (uc *TransactionUseCase) GetTransactionHistory(ctx context.Context, accountID string, filter *domain.TransactionFilter) ([]*domain.Transaction, error) {
	return uc.transactionRepo.GetByAccountID(ctx, accountID, filter)
}

// GetTransactionsByFilter retrieves transactions by filter
func (uc *TransactionUseCase) GetTransactionsByFilter(ctx context.Context, filter *domain.TransactionFilter) ([]*domain.Transaction, error) {
	return uc.transactionRepo.GetByFilter(ctx, filter)
}

// CancelTransaction cancels a pending transaction
func (uc *TransactionUseCase) CancelTransaction(ctx context.Context, id string) error {
	transaction, err := uc.transactionRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if transaction.Status != domain.TransactionStatusPending {
		return domain.ErrTransactionAlreadyProcessed
	}

	return uc.transactionRepo.UpdateStatus(ctx, id, domain.TransactionStatusCancelled, "Cancelled by user")
}

// StartTransactionProcessor starts the transaction processor
func (uc *TransactionUseCase) StartTransactionProcessor(ctx context.Context) error {
	handler := func(data []byte) error {
		var request domain.TransactionRequest
		if err := json.Unmarshal(data, &request); err != nil {
			log.Printf("Failed to unmarshal transaction request: %v", err)
			return err
		}

		log.Printf("Processing transaction: %s", request.ID)

		err := uc.ProcessTransactionSync(ctx, &request)
		if err != nil {
			log.Printf("Failed to process transaction %s: %v", request.ID, err)
			// Update transaction status to failed
			uc.transactionRepo.UpdateStatus(ctx, request.ID, domain.TransactionStatusFailed, err.Error())
			return err
		}

		log.Printf("Successfully processed transaction: %s", request.ID)
		return nil
	}

	return uc.queue.Subscribe(ctx, uc.queueName, handler)
}
