package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"banking-ledger/internal/domain"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// PostgreSQLAccountRepository implements the AccountRepository interface
type PostgreSQLAccountRepository struct {
	db *sqlx.DB
}

// NewPostgreSQLAccountRepository creates a new PostgreSQL account repository
func NewPostgreSQLAccountRepository(db *sqlx.DB) domain.AccountRepository {
	return &PostgreSQLAccountRepository{db: db}
}

// Create creates a new account
func (r *PostgreSQLAccountRepository) Create(ctx context.Context, account *domain.Account) error {
	if account.ID == "" {
		account.ID = uuid.New().String()
	}

	account.CreatedAt = time.Now()
	account.UpdatedAt = time.Now()
	account.Version = 1

	query := `
		INSERT INTO accounts (id, user_id, balance, currency, status, created_at, updated_at, version)
		VALUES (:id, :user_id, :balance, :currency, :status, :created_at, :updated_at, :version)
	`

	_, err := r.db.NamedExecContext(ctx, query, account)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code {
			case "23505": // unique_violation
				return domain.ErrAccountExists
			}
		}
		return fmt.Errorf("failed to create account: %w", err)
	}

	return nil
}

// GetByID retrieves an account by ID
func (r *PostgreSQLAccountRepository) GetByID(ctx context.Context, id string) (*domain.Account, error) {
	var account domain.Account

	query := `
		SELECT id, user_id, balance, currency, status, created_at, updated_at, version
		FROM accounts
		WHERE id = $1
	`

	err := r.db.GetContext(ctx, &account, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrAccountNotFound
		}
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	return &account, nil
}

// GetByUserID retrieves accounts by user ID
func (r *PostgreSQLAccountRepository) GetByUserID(ctx context.Context, userID string) ([]*domain.Account, error) {
	var accounts []*domain.Account

	query := `
		SELECT id, user_id, balance, currency, status, created_at, updated_at, version
		FROM accounts
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	err := r.db.SelectContext(ctx, &accounts, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get accounts by user ID: %w", err)
	}

	return accounts, nil
}

// Update updates an account
func (r *PostgreSQLAccountRepository) Update(ctx context.Context, account *domain.Account) error {
	account.UpdatedAt = time.Now()

	query := `
		UPDATE accounts
		SET user_id = :user_id, balance = :balance, currency = :currency, 
		    status = :status, updated_at = :updated_at, version = version + 1
		WHERE id = :id AND version = :version
	`

	result, err := r.db.NamedExecContext(ctx, query, account)
	if err != nil {
		return fmt.Errorf("failed to update account: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return domain.ErrConcurrentUpdate
	}

	account.Version++
	return nil
}

// UpdateBalance updates account balance with optimistic locking
func (r *PostgreSQLAccountRepository) UpdateBalance(ctx context.Context, id string, newBalance float64, version int64) error {
	query := `
		UPDATE accounts
		SET balance = $1, updated_at = $2, version = version + 1
		WHERE id = $3 AND version = $4
	`

	result, err := r.db.ExecContext(ctx, query, newBalance, time.Now(), id, version)
	if err != nil {
		return fmt.Errorf("failed to update account balance: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return domain.ErrConcurrentUpdate
	}

	return nil
}

// Delete deletes an account
func (r *PostgreSQLAccountRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM accounts WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete account: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return domain.ErrAccountNotFound
	}

	return nil
}

// List retrieves accounts with pagination
func (r *PostgreSQLAccountRepository) List(ctx context.Context, limit, offset int) ([]*domain.Account, error) {
	var accounts []*domain.Account

	query := `
		SELECT id, user_id, balance, currency, status, created_at, updated_at, version
		FROM accounts
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	err := r.db.SelectContext(ctx, &accounts, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %w", err)
	}

	return accounts, nil
}
