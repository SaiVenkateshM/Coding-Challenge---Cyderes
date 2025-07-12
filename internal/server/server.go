package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/cyderes/data-ingestion-service/internal/config"
	"github.com/cyderes/data-ingestion-service/internal/storage"
)

// Server handles HTTP requests
type Server struct {
	config  config.ServerConfig
	storage storage.Storage
	server  *http.Server
}

// NewServer creates a new HTTP server
func NewServer(cfg config.ServerConfig, store storage.Storage) *Server {
	s := &Server{
		config:  cfg,
		storage: store,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/posts", s.handlePosts)
	mux.HandleFunc("/posts/", s.handlePostByID)
	mux.HandleFunc("/status", s.handleStatus)

	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	return s
}

// Start starts the HTTP server
func (s *Server) Start() error {
	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

// handleHealth handles health check requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}

// handlePosts handles GET requests for posts
func (s *Server) handlePosts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 10 // default
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	offset := 0 // default
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Get posts from storage
	posts, err := s.storage.GetPosts(r.Context(), limit, offset)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to retrieve posts: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"posts":  posts,
		"count":  len(posts),
		"limit":  limit,
		"offset": offset,
	})
}

// handlePostByID handles GET requests for a specific post
func (s *Server) handlePostByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from path
	path := r.URL.Path
	if len(path) < 7 { // "/posts/"
		http.Error(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	idStr := path[7:] // Remove "/posts/"
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	// Get post from storage
	post, err := s.storage.GetPostByID(r.Context(), id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to retrieve post: %v", err), http.StatusInternalServerError)
		return
	}

	if post == nil {
		http.Error(w, "Post not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(post)
}

// handleStatus handles GET requests for ingestion status
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status, err := s.storage.GetIngestionStatus(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to retrieve status: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}