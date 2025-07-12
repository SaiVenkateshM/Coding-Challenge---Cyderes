package storage

import (
	"context"
	"fmt"

	"github.com/cyderes/data-ingestion-service/internal/config"
	"github.com/cyderes/data-ingestion-service/internal/models"
)

// Storage interface defines the contract for data storage
type Storage interface {
	StorePosts(ctx context.Context, posts []models.TransformedPost) error
	GetPosts(ctx context.Context, limit int, offset int) ([]models.TransformedPost, error)
	GetPostByID(ctx context.Context, id int) (*models.TransformedPost, error)
	UpdateIngestionStatus(ctx context.Context, status models.IngestionStatus) error
	GetIngestionStatus(ctx context.Context) (*models.IngestionStatus, error)
	Close() error
}

// NewStorage creates a new storage instance based on configuration
func NewStorage(cfg config.StorageConfig) (Storage, error) {
	switch cfg.Type {
	case "dynamodb":
		return NewDynamoDBStorage(cfg)
	case "mongodb":
		return NewMongoDBStorage(cfg)
	case "postgresql":
		return NewPostgreSQLStorage(cfg)
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", cfg.Type)
	}
}