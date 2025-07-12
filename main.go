package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cyderes/data-ingestion-service/internal/config"
	"github.com/cyderes/data-ingestion-service/internal/ingestion"
	"github.com/cyderes/data-ingestion-service/internal/server"
	"github.com/cyderes/data-ingestion-service/internal/storage"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	// Initialize storage
	store, err := storage.NewStorage(cfg.Storage)
	if err != nil {
		log.Fatal("Failed to initialize storage:", err)
	}
	defer store.Close()

	// Initialize ingestion service
	ingestor := ingestion.NewService(cfg.Ingestion, store)

	// Initialize HTTP server for API endpoints
	httpServer := server.NewServer(cfg.Server, store)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start HTTP server
	go func() {
		log.Printf("Starting HTTP server on port %d", cfg.Server.Port)
		if err := httpServer.Start(); err != nil {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	// Start ingestion service
	go func() {
		log.Println("Starting data ingestion service")
		if err := ingestor.Start(ctx); err != nil {
			log.Printf("Ingestion service error: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	log.Println("Shutdown signal received, gracefully shutting down...")

	// Create shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Shutdown services
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	cancel() // Cancel ingestion context
	log.Println("Shutdown complete")
}