package storage

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/cyderes/data-ingestion-service/internal/config"
	"github.com/cyderes/data-ingestion-service/internal/models"
)

// DynamoDBStorage implements Storage interface using AWS DynamoDB
type DynamoDBStorage struct {
	client    *dynamodb.DynamoDB
	tableName string
}

// NewDynamoDBStorage creates a new DynamoDB storage instance
func NewDynamoDBStorage(cfg config.StorageConfig) (*DynamoDBStorage, error) {
	awsConfig := &aws.Config{
		Region: aws.String(cfg.Region),
	}

	// For local testing with DynamoDB Local
	if cfg.Endpoint != "" {
		awsConfig.Endpoint = aws.String(cfg.Endpoint)
	}

	sess, err := session.NewSession(awsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %w", err)
	}

	client := dynamodb.New(sess)
	storage := &DynamoDBStorage{
		client:    client,
		tableName: cfg.TableName,
	}

	// Create table if it doesn't exist (for local testing)
	if err := storage.ensureTable(); err != nil {
		return nil, fmt.Errorf("failed to ensure table exists: %w", err)
	}

	return storage, nil
}

// ensureTable creates the DynamoDB table if it doesn't exist
func (d *DynamoDBStorage) ensureTable() error {
	// Check if table exists
	_, err := d.client.DescribeTable(&dynamodb.DescribeTableInput{
		TableName: aws.String(d.tableName),
	})

	if err == nil {
		return nil // Table already exists
	}

	// Create table
	input := &dynamodb.CreateTableInput{
		TableName: aws.String(d.tableName),
		KeySchema: []*dynamodb.KeySchemaElement{
			{
				AttributeName: aws.String("id"),
				KeyType:       aws.String("HASH"),
			},
		},
		AttributeDefinitions: []*dynamodb.AttributeDefinition{
			{
				AttributeName: aws.String("id"),
				AttributeType: aws.String("N"),
			},
		},
		BillingMode: aws.String("PAY_PER_REQUEST"),
	}

	_, err = d.client.CreateTable(input)
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	// Wait for table to be created
	return d.client.WaitUntilTableExists(&dynamodb.DescribeTableInput{
		TableName: aws.String(d.tableName),
	})
}

// StorePosts stores posts in DynamoDB
func (d *DynamoDBStorage) StorePosts(ctx context.Context, posts []models.TransformedPost) error {
	for _, post := range posts {
		item, err := dynamodbattribute.MarshalMap(post)
		if err != nil {
			return fmt.Errorf("failed to marshal post %d: %w", post.ID, err)
		}

		_, err = d.client.PutItemWithContext(ctx, &dynamodb.PutItemInput{
			TableName: aws.String(d.tableName),
			Item:      item,
		})
		if err != nil {
			return fmt.Errorf("failed to store post %d: %w", post.ID, err)
		}
	}

	return nil
}

// GetPosts retrieves posts from DynamoDB with pagination
func (d *DynamoDBStorage) GetPosts(ctx context.Context, limit int, offset int) ([]models.TransformedPost, error) {
	input := &dynamodb.ScanInput{
		TableName: aws.String(d.tableName),
		Limit:     aws.Int64(int64(limit)),
	}

	result, err := d.client.ScanWithContext(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to scan posts: %w", err)
	}

	var posts []models.TransformedPost
	err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &posts)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal posts: %w", err)
	}

	return posts, nil
}

// GetPostByID retrieves a specific post by ID
func (d *DynamoDBStorage) GetPostByID(ctx context.Context, id int) (*models.TransformedPost, error) {
	input := &dynamodb.GetItemInput{
		TableName: aws.String(d.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"id": {
				N: aws.String(strconv.Itoa(id)),
			},
		},
	}

	result, err := d.client.GetItemWithContext(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get post %d: %w", id, err)
	}

	if result.Item == nil {
		return nil, nil // Post not found
	}

	var post models.TransformedPost
	err = dynamodbattribute.UnmarshalMap(result.Item, &post)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal post: %w", err)
	}

	return &post, nil
}

// UpdateIngestionStatus updates the ingestion status
func (d *DynamoDBStorage) UpdateIngestionStatus(ctx context.Context, status models.IngestionStatus) error {
	// Store in a separate table or use a fixed key
	item, err := dynamodbattribute.MarshalMap(status)
	if err != nil {
		return fmt.Errorf("failed to marshal ingestion status: %w", err)
	}

	// Add a fixed key for the status record
	item["id"] = &dynamodb.AttributeValue{S: aws.String("ingestion_status")}

	_, err = d.client.PutItemWithContext(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(d.tableName + "_status"),
		Item:      item,
	})
	
	return err
}

// GetIngestionStatus retrieves the current ingestion status
func (d *DynamoDBStorage) GetIngestionStatus(ctx context.Context) (*models.IngestionStatus, error) {
	input := &dynamodb.GetItemInput{
		TableName: aws.String(d.tableName + "_status"),
		Key: map[string]*dynamodb.AttributeValue{
			"id": {
				S: aws.String("ingestion_status"),
			},
		},
	}

	result, err := d.client.GetItemWithContext(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get ingestion status: %w", err)
	}

	if result.Item == nil {
		// Return default status if not found
		return &models.IngestionStatus{
			Status: "never_run",
		}, nil
	}

	var status models.IngestionStatus
	err = dynamodbattribute.UnmarshalMap(result.Item, &status)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal ingestion status: %w", err)
	}

	return &status, nil
}

// Close closes the DynamoDB connection
func (d *DynamoDBStorage) Close() error {
	// DynamoDB client doesn't need explicit closing
	return nil
}