package database

import (
	"context"
	"fmt"
	"time"

	"banking-ledger/internal/config"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// NewPostgreSQLConnection creates a new PostgreSQL connection
func NewPostgreSQLConnection(cfg config.DatabaseConfig) (*sqlx.DB, error) {
	db, err := sqlx.Connect("postgres", cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}

	return db, nil
}

// NewMongoDBConnection creates a new MongoDB connection
func NewMongoDBConnection(cfg config.MongoDBConfig) (*mongo.Database, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI(cfg.URL)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Test connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	database := client.Database(cfg.Database)
	return database, nil
}

// MigratePostgreSQL runs PostgreSQL migrations
func MigratePostgreSQL(db *sqlx.DB) error {
	// Create accounts table
	createAccountsTable := `
		CREATE TABLE IF NOT EXISTS accounts (
			id VARCHAR(36) PRIMARY KEY,
			user_id VARCHAR(255) NOT NULL,
			balance DECIMAL(20,8) NOT NULL DEFAULT 0,
			currency VARCHAR(3) NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'active',
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			version BIGINT NOT NULL DEFAULT 1,
			UNIQUE(user_id, currency)
		);
	`

	if _, err := db.Exec(createAccountsTable); err != nil {
		return fmt.Errorf("failed to create accounts table: %w", err)
	}

	// Create indexes
	createIndexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_accounts_user_id ON accounts(user_id);",
		"CREATE INDEX IF NOT EXISTS idx_accounts_status ON accounts(status);",
		"CREATE INDEX IF NOT EXISTS idx_accounts_created_at ON accounts(created_at);",
	}

	for _, index := range createIndexes {
		if _, err := db.Exec(index); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}

// CreateMongoDBIndexes creates MongoDB indexes
func CreateMongoDBIndexes(db *mongo.Database, collectionName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := db.Collection(collectionName)

	// Create indexes
	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{{"from_account_id", 1}},
		},
		{
			Keys: bson.D{{"to_account_id", 1}},
		},
		{
			Keys: bson.D{{"type", 1}},
		},
		{
			Keys: bson.D{{"status", 1}},
		},
		{
			Keys: bson.D{{"created_at", -1}},
		},
		{
			Keys: bson.D{{"from_account_id", 1}, {"created_at", -1}},
		},
		{
			Keys: bson.D{{"to_account_id", 1}, {"created_at", -1}},
		},
	}

	_, err := collection.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return fmt.Errorf("failed to create MongoDB indexes: %w", err)
	}

	return nil
}
