package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds the application configuration
type Config struct {
	Server   ServerConfig   `json:"server"`
	Database DatabaseConfig `json:"database"`
	MongoDB  MongoDBConfig  `json:"mongodb"`
	RabbitMQ RabbitMQConfig `json:"rabbitmq"`
	Logger   LoggerConfig   `json:"logger"`
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Port            string        `json:"port"`
	ReadTimeout     time.Duration `json:"read_timeout"`
	WriteTimeout    time.Duration `json:"write_timeout"`
	IdleTimeout     time.Duration `json:"idle_timeout"`
	ShutdownTimeout time.Duration `json:"shutdown_timeout"`
}

// DatabaseConfig holds PostgreSQL database configuration
type DatabaseConfig struct {
	URL             string        `json:"url"`
	MaxOpenConns    int           `json:"max_open_conns"`
	MaxIdleConns    int           `json:"max_idle_conns"`
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `json:"conn_max_idle_time"`
}

// MongoDBConfig holds MongoDB configuration
type MongoDBConfig struct {
	URL        string `json:"url"`
	Database   string `json:"database"`
	Collection string `json:"collection"`
}

// RabbitMQConfig holds RabbitMQ configuration
type RabbitMQConfig struct {
	URL               string        `json:"url"`
	TransactionQueue  string        `json:"transaction_queue"`
	NotificationQueue string        `json:"notification_queue"`
	MaxRetries        int           `json:"max_retries"`
	RetryDelay        time.Duration `json:"retry_delay"`
}

// LoggerConfig holds logger configuration
type LoggerConfig struct {
	Level      string `json:"level"`
	Format     string `json:"format"`
	OutputPath string `json:"output_path"`
}

// Load loads configuration from environment variables
func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port:            getEnvOrDefault("SERVER_PORT", "8080"),
			ReadTimeout:     getDurationOrDefault("SERVER_READ_TIMEOUT", 30*time.Second),
			WriteTimeout:    getDurationOrDefault("SERVER_WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:     getDurationOrDefault("SERVER_IDLE_TIMEOUT", 60*time.Second),
			ShutdownTimeout: getDurationOrDefault("SERVER_SHUTDOWN_TIMEOUT", 10*time.Second),
		},
		Database: DatabaseConfig{
			URL:             getEnvOrDefault("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/banking_ledger?sslmode=disable"),
			MaxOpenConns:    getIntOrDefault("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getIntOrDefault("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getDurationOrDefault("DB_CONN_MAX_LIFETIME", 300*time.Second),
			ConnMaxIdleTime: getDurationOrDefault("DB_CONN_MAX_IDLE_TIME", 300*time.Second),
		},
		MongoDB: MongoDBConfig{
			URL:        getEnvOrDefault("MONGODB_URL", "mongodb://mongo:mongo@localhost:27017/ledger"),
			Database:   getEnvOrDefault("MONGODB_DATABASE", "ledger"),
			Collection: getEnvOrDefault("MONGODB_COLLECTION", "transactions"),
		},
		RabbitMQ: RabbitMQConfig{
			URL:               getEnvOrDefault("RABBITMQ_URL", "amqp://rabbitmq:rabbitmq@localhost:5672/"),
			TransactionQueue:  getEnvOrDefault("RABBITMQ_TRANSACTION_QUEUE", "transactions"),
			NotificationQueue: getEnvOrDefault("RABBITMQ_NOTIFICATION_QUEUE", "notifications"),
			MaxRetries:        getIntOrDefault("RABBITMQ_MAX_RETRIES", 3),
			RetryDelay:        getDurationOrDefault("RABBITMQ_RETRY_DELAY", 5*time.Second),
		},
		Logger: LoggerConfig{
			Level:      getEnvOrDefault("LOG_LEVEL", "info"),
			Format:     getEnvOrDefault("LOG_FORMAT", "json"),
			OutputPath: getEnvOrDefault("LOG_OUTPUT_PATH", "stdout"),
		},
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getDurationOrDefault(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
