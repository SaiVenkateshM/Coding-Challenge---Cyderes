package ingestion

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/cyderes/data-ingestion-service/internal/config"
	"github.com/cyderes/data-ingestion-service/internal/models"
)

// MockStorage is a mock implementation of the Storage interface
type MockStorage struct {
	mock.Mock
}

func (m *MockStorage) StorePosts(ctx context.Context, posts []models.TransformedPost) error {
	args := m.Called(ctx, posts)
	return args.Error(0)
}

func (m *MockStorage) GetPosts(ctx context.Context, limit int, offset int) ([]models.TransformedPost, error) {
	args := m.Called(ctx, limit, offset)
	return args.Get(0).([]models.TransformedPost), args.Error(1)
}

func (m *MockStorage) GetPostByID(ctx context.Context, id int) (*models.TransformedPost, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.TransformedPost), args.Error(1)
}

func (m *MockStorage) UpdateIngestionStatus(ctx context.Context, status models.IngestionStatus) error {
	args := m.Called(ctx, status)
	return args.Error(0)
}

func (m *MockStorage) GetIngestionStatus(ctx context.Context) (*models.IngestionStatus, error) {
	args := m.Called(ctx)
	return args.Get(0).(*models.IngestionStatus), args.Error(1)
}

func (m *MockStorage) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestService_fetchPostsOnce(t *testing.T) {
	// Create test data
	testPosts := []models.Post{
		{UserID: 1, ID: 1, Title: "Test Post 1", Body: "Test body 1"},
		{UserID: 1, ID: 2, Title: "Test Post 2", Body: "Test body 2"},
	}

	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(testPosts)
	}))
	defer server.Close()

	// Create service with mock storage
	mockStorage := new(MockStorage)
	cfg := config.IngestionConfig{
		APIEndpoint: server.URL,
		Timeout:     30 * time.Second,
		RetryCount:  3,
	}
	
	service := NewService(cfg, mockStorage)

	// Test fetchPostsOnce
	ctx := context.Background()
	posts, err := service.fetchPostsOnce(ctx)

	assert.NoError(t, err)
	assert.Len(t, posts, 2)
	assert.Equal(t, testPosts[0].ID, posts[0].ID)
	assert.Equal(t, testPosts[0].Title, posts[0].Title)
}

func TestService_fetchPostsOnce_APIError(t *testing.T) {
	// Create mock server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Create service
	mockStorage := new(MockStorage)
	cfg := config.IngestionConfig{
		APIEndpoint: server.URL,
		Timeout:     30 * time.Second,
		RetryCount:  3,
	}
	
	service := NewService(cfg, mockStorage)

	// Test fetchPostsOnce
	ctx := context.Background()
	posts, err := service.fetchPostsOnce(ctx)

	assert.Error(t, err)
	assert.Nil(t, posts)
	assert.Contains(t, err.Error(), "API returned status 500")
}

func TestService_fetchPostsOnce_InvalidJSON(t *testing.T) {
	// Create mock server that returns invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	// Create service
	mockStorage := new(MockStorage)
	cfg := config.IngestionConfig{
		APIEndpoint: server.URL,
		Timeout:     30 * time.Second,
		RetryCount:  3,
	}
	
	service := NewService(cfg, mockStorage)

	// Test fetchPostsOnce
	ctx := context.Background()
	posts, err := service.fetchPostsOnce(ctx)

	assert.Error(t, err)
	assert.Nil(t, posts)
	assert.Contains(t, err.Error(), "failed to unmarshal response")
}

func TestService_transformPosts(t *testing.T) {
	// Create test data
	originalPosts := []models.Post{
		{UserID: 1, ID: 1, Title: "Test Post 1", Body: "Test body 1"},
		{UserID: 1, ID: 2, Title: "Test Post 2", Body: "Test body 2"},
	}

	// Create service
	mockStorage := new(MockStorage)
	cfg := config.IngestionConfig{}
	service := NewService(cfg, mockStorage)

	// Test transformPosts
	transformedPosts := service.transformPosts(originalPosts)

	assert.Len(t, transformedPosts, 2)
	
	for i, post := range transformedPosts {
		assert.Equal(t, originalPosts[i].ID, post.ID)
		assert.Equal(t, originalPosts[i].Title, post.Title)
		assert.Equal(t, originalPosts[i].Body, post.Body)
		assert.Equal(t, originalPosts[i].UserID, post.UserID)
		assert.Equal(t, "placeholder_api", post.Source)
		assert.WithinDuration(t, time.Now().UTC(), post.IngestedAt, time.Second)
	}
}

func TestService_IngestData(t *testing.T) {
	// Create test data
	testPosts := []models.Post{
		{UserID: 1, ID: 1, Title: "Test Post 1", Body: "Test body 1"},
	}

	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(testPosts)
	}))
	defer server.Close()

	// Create service with mock storage
	mockStorage := new(MockStorage)
	mockStorage.On("StorePosts", mock.Anything, mock.AnythingOfType("[]models.TransformedPost")).Return(nil)
	
	cfg := config.IngestionConfig{
		APIEndpoint: server.URL,
		Timeout:     30 * time.Second,
		RetryCount:  3,
	}
	
	service := NewService(cfg, mockStorage)

	// Test IngestData
	ctx := context.Background()
	err := service.IngestData(ctx)

	assert.NoError(t, err)
	mockStorage.AssertExpectations(t)
}

func TestService_IngestData_StorageError(t *testing.T) {
	// Create test data
	testPosts := []models.Post{
		{UserID: 1, ID: 1, Title: "Test Post 1", Body: "Test body 1"},
	}

	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(testPosts)
	}))
	defer server.Close()

	// Create service with mock storage that returns error
	mockStorage := new(MockStorage)
	mockStorage.On("StorePosts", mock.Anything, mock.AnythingOfType("[]models.TransformedPost")).Return(assert.AnError)
	
	cfg := config.IngestionConfig{
		APIEndpoint: server.URL,
		Timeout:     30 * time.Second,
		RetryCount:  3,
	}
	
	service := NewService(cfg, mockStorage)

	// Test IngestData
	ctx := context.Background()
	err := service.IngestData(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to store posts")
	mockStorage.AssertExpectations(t)
}

func TestService_fetchPosts_WithRetry(t *testing.T) {
	callCount := 0
	
	// Create mock server that fails twice then succeeds
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		
		testPosts := []models.Post{
			{UserID: 1, ID: 1, Title: "Test Post 1", Body: "Test body 1"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(testPosts)
	}))
	defer server.Close()

	// Create service
	mockStorage := new(MockStorage)
	cfg := config.IngestionConfig{
		APIEndpoint: server.URL,
		Timeout:     30 * time.Second,
		RetryCount:  3,
	}
	
	service := NewService(cfg, mockStorage)

	// Test fetchPosts with retry
	ctx := context.Background()
	posts, err := service.fetchPosts(ctx)

	assert.NoError(t, err)
	assert.Len(t, posts, 1)
	assert.Equal(t, 3, callCount) // Should have retried twice
}

func TestService_fetchPosts_ExceedsRetryLimit(t *testing.T) {
	// Create mock server that always fails
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Create service
	mockStorage := new(MockStorage)
	cfg := config.IngestionConfig{
		APIEndpoint: server.URL,
		Timeout:     30 * time.Second,
		RetryCount:  3,
	}
	
	service := NewService(cfg, mockStorage)

	// Test fetchPosts with exceeded retry limit
	ctx := context.Background()
	posts, err := service.fetchPosts(ctx)

	assert.Error(t, err)
	assert.Nil(t, posts)
	assert.Contains(t, err.Error(), "failed after 3 attempts")
}