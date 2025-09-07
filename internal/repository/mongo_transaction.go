package repository

import (
	"context"
	"fmt"
	"time"

	"banking-ledger/internal/domain"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoTransactionRepository implements the TransactionRepository interface
type MongoTransactionRepository struct {
	collection *mongo.Collection
}

// NewMongoTransactionRepository creates a new MongoDB transaction repository
func NewMongoTransactionRepository(db *mongo.Database, collectionName string) domain.TransactionRepository {
	return &MongoTransactionRepository{
		collection: db.Collection(collectionName),
	}
}

// Create creates a new transaction
func (r *MongoTransactionRepository) Create(ctx context.Context, transaction *domain.Transaction) error {
	if transaction.ID == "" {
		transaction.ID = uuid.New().String()
	}

	transaction.CreatedAt = time.Now()
	transaction.UpdatedAt = time.Now()

	_, err := r.collection.InsertOne(ctx, transaction)
	if err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}

	return nil
}

// GetByID retrieves a transaction by ID
func (r *MongoTransactionRepository) GetByID(ctx context.Context, id string) (*domain.Transaction, error) {
	var transaction domain.Transaction

	filter := bson.M{"_id": id}
	err := r.collection.FindOne(ctx, filter).Decode(&transaction)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, domain.ErrTransactionNotFound
		}
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	return &transaction, nil
}

// GetByAccountID retrieves transactions by account ID
func (r *MongoTransactionRepository) GetByAccountID(ctx context.Context, accountID string, filter *domain.TransactionFilter) ([]*domain.Transaction, error) {
	if filter == nil {
		filter = &domain.TransactionFilter{}
	}
	filter.AccountID = &accountID

	return r.GetByFilter(ctx, filter)
}

// GetByFilter retrieves transactions by filter
func (r *MongoTransactionRepository) GetByFilter(ctx context.Context, filter *domain.TransactionFilter) ([]*domain.Transaction, error) {
	mongoFilter := r.buildMongoFilter(filter)

	opts := options.Find()
	opts.SetSort(bson.D{{Key: "created_at", Value: -1}})

	if filter.Limit > 0 {
		opts.SetLimit(int64(filter.Limit))
	}
	if filter.Offset > 0 {
		opts.SetSkip(int64(filter.Offset))
	}

	cursor, err := r.collection.Find(ctx, mongoFilter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find transactions: %w", err)
	}
	defer cursor.Close(ctx)

	var transactions []*domain.Transaction
	for cursor.Next(ctx) {
		var transaction domain.Transaction
		if err := cursor.Decode(&transaction); err != nil {
			return nil, fmt.Errorf("failed to decode transaction: %w", err)
		}
		transactions = append(transactions, &transaction)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return transactions, nil
}

// Update updates a transaction
func (r *MongoTransactionRepository) Update(ctx context.Context, transaction *domain.Transaction) error {
	transaction.UpdatedAt = time.Now()

	filter := bson.M{"_id": transaction.ID}
	update := bson.M{"$set": transaction}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update transaction: %w", err)
	}

	if result.MatchedCount == 0 {
		return domain.ErrTransactionNotFound
	}

	return nil
}

// UpdateStatus updates transaction status
func (r *MongoTransactionRepository) UpdateStatus(ctx context.Context, id string, status domain.TransactionStatus, errorMessage string) error {
	filter := bson.M{"_id": id}
	update := bson.M{
		"$set": bson.M{
			"status":        status,
			"error_message": errorMessage,
			"updated_at":    time.Now(),
		},
	}

	if status == domain.TransactionStatusCompleted {
		update["$set"].(bson.M)["processed_at"] = time.Now()
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update transaction status: %w", err)
	}

	if result.MatchedCount == 0 {
		return domain.ErrTransactionNotFound
	}

	return nil
}

// Count counts transactions by filter
func (r *MongoTransactionRepository) Count(ctx context.Context, filter *domain.TransactionFilter) (int64, error) {
	mongoFilter := r.buildMongoFilter(filter)

	count, err := r.collection.CountDocuments(ctx, mongoFilter)
	if err != nil {
		return 0, fmt.Errorf("failed to count transactions: %w", err)
	}

	return count, nil
}

func (r *MongoTransactionRepository) buildMongoFilter(filter *domain.TransactionFilter) bson.M {
	mongoFilter := bson.M{}

	if filter.AccountID != nil {
		mongoFilter["$or"] = []bson.M{
			{"from_account_id": *filter.AccountID},
			{"to_account_id": *filter.AccountID},
		}
	}

	if filter.Type != nil {
		mongoFilter["type"] = *filter.Type
	}

	if filter.Status != nil {
		mongoFilter["status"] = *filter.Status
	}

	if filter.FromDate != nil || filter.ToDate != nil {
		dateFilter := bson.M{}
		if filter.FromDate != nil {
			dateFilter["$gte"] = *filter.FromDate
		}
		if filter.ToDate != nil {
			dateFilter["$lte"] = *filter.ToDate
		}
		mongoFilter["created_at"] = dateFilter
	}

	if filter.MinAmount != nil || filter.MaxAmount != nil {
		amountFilter := bson.M{}
		if filter.MinAmount != nil {
			amountFilter["$gte"] = *filter.MinAmount
		}
		if filter.MaxAmount != nil {
			amountFilter["$lte"] = *filter.MaxAmount
		}
		mongoFilter["amount"] = amountFilter
	}

	return mongoFilter
}
