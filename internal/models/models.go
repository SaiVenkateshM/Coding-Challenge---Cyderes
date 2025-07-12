package models

import "time"

// Post represents the original post structure from the API
type Post struct {
	UserID int    `json:"userId"`
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Body   string `json:"body"`
}

// TransformedPost represents the post after transformation
type TransformedPost struct {
	Post       `json:",inline"`
	IngestedAt time.Time `json:"ingested_at"`
	Source     string    `json:"source"`
}

// IngestionStatus tracks the status of ingestion runs
type IngestionStatus struct {
	LastSuccessfulRun time.Time `json:"last_successful_run"`
	LastAttempt       time.Time `json:"last_attempt"`
	Status            string    `json:"status"` // "success", "failure", "running"
	ErrorMessage      string    `json:"error_message,omitempty"`
	RecordsIngested   int       `json:"records_ingested"`
}