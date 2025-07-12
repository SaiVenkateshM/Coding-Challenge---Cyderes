package ingestion

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/cyderes/data-ingestion-service/internal/config"
	"github.com/cyderes/data-ingestion-service/internal/models"
	"github.com/cyderes/data-ingestion-service/internal/storage"
)

// Service handles data ingestion from external APIs
type Service struct {
	config     config.IngestionConfig
	storage    storage.Storage
	httpClient *http.Client
}

// NewService creates a new ingestion service
func NewService(cfg config.IngestionConfig, store storage.Storage) *Service {
	return &Service{
		config:  cfg,
		storage: store,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

// Start begins the ingestion process
func (s *Service) Start(ctx context.Context) error {
	// Perform initial ingestion
	if err := s.IngestData(ctx); err != nil {
		return fmt.Errorf("initial ingestion failed: %w", err)
	}

	// Set up periodic ingestion
	ticker := time.NewTicker(s.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := s.IngestData(ctx); err != nil {
				// Log error but don't stop the service
				fmt.Printf("Ingestion error: %v\n", err)
			}
		}
	}
}

// IngestData fetches data from the API and stores it
func (s *Service) IngestData(ctx context.Context) error {
	// Fetch data from API
	posts, err := s.fetchPosts(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch posts: %w", err)
	}

	// Transform data
	transformedPosts := s.transformPosts(posts)

	// Store data
	if err := s.storage.StorePosts(ctx, transformedPosts); err != nil {
		return fmt.Errorf("failed to store posts: %w", err)
	}

	fmt.Printf("Successfully ingested %d posts\n", len(transformedPosts))
	return nil
}

// fetchPosts fetches posts from the API with retry logic
func (s *Service) fetchPosts(ctx context.Context) ([]models.Post, error) {
	var lastErr error
	
	for attempt := 0; attempt < s.config.RetryCount; attempt++ {
		posts, err := s.fetchPostsOnce(ctx)
		if err == nil {
			return posts, nil
		}
		
		lastErr = err
		if attempt < s.config.RetryCount-1 {
			// Wait before retrying (exponential backoff)
			waitTime := time.Duration(attempt+1) * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(waitTime):
			}
		}
	}
	
	return nil, fmt.Errorf("failed after %d attempts: %w", s.config.RetryCount, lastErr)
}

// fetchPostsOnce performs a single fetch attempt
func (s *Service) fetchPostsOnce(ctx context.Context) ([]models.Post, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", s.config.APIEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var posts []models.Post
	if err := json.Unmarshal(body, &posts); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return posts, nil
}

// transformPosts adds ingestion metadata to posts
func (s *Service) transformPosts(posts []models.Post) []models.TransformedPost {
	now := time.Now().UTC()
	transformed := make([]models.TransformedPost, len(posts))

	for i, post := range posts {
		transformed[i] = models.TransformedPost{
			Post:       post,
			IngestedAt: now,
			Source:     "placeholder_api",
		}
	}

	return transformed
}