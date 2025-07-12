package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the application
type Config struct {
	Storage   StorageConfig
	Ingestion IngestionConfig
	Server    ServerConfig
}

// StorageConfig holds storage-related configuration
type StorageConfig struct {
	Type        string // "dynamodb", "mongodb", "postgresql"
	Region      string // For AWS DynamoDB
	TableName   string
	Endpoint    string // Custom endpoint for local testing
	MongoDBURI  string
	PostgresURI string
}

// IngestionConfig holds ingestion-related configuration
type IngestionConfig struct {
	APIEndpoint string
	Interval    time.Duration
	Timeout     time.Duration
	RetryCount  int
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Port int
}

// Load loads configuration from environment variables with defaults
func Load() (*Config, error) {
	cfg := &Config{
		Storage: StorageConfig{
			Type:        getEnv("STORAGE_TYPE", "dynamodb"),
			Region:      getEnv("AWS_REGION", "us-west-2"),
			TableName:   getEnv("TABLE_NAME", "ingested_data"),
			Endpoint:    getEnv("DYNAMODB_ENDPOINT", ""), // For local DynamoDB
			MongoDBURI:  getEnv("MONGODB_URI", ""),
			PostgresURI: getEnv("POSTGRES_URI", ""),
		},
		Ingestion: IngestionConfig{
			APIEndpoint: getEnv("API_ENDPOINT", "https://jsonplaceholder.typicode.com/posts"),
			Interval:    getEnvDuration("INGESTION_INTERVAL", 5*time.Minute),
			Timeout:     getEnvDuration("API_TIMEOUT", 30*time.Second),
			RetryCount:  getEnvInt("RETRY_COUNT", 3),
		},
		Server: ServerConfig{
			Port: getEnvInt("SERVER_PORT", 8080),
		},
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}