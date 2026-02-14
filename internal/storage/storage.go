package storage

import (
	"time"

	"ptp/internal/config"
	"ptp/internal/domain"
)

// Storage persists and loads test run results (e.g. for the faills viewer).
type Storage interface {
	Save(results []domain.TestResult, failures []domain.TestFailure, duration time.Duration, workers int) error
	Load() (*domain.TestResultsOutput, error)
	// SaveOutput writes the full output (e.g. after partial re-run updates).
	SaveOutput(output *domain.TestResultsOutput) error
}

// JSONStorage stores results in a JSON file under the configured output path.
type JSONStorage struct {
	cfg *config.Config
}

// NewJSONStorage returns a Storage that reads/writes the config's output JSON path.
func NewJSONStorage(cfg *config.Config) *JSONStorage {
	return &JSONStorage{cfg: cfg}
}
