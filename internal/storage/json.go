package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"ptp/internal/domain"
)

// Save writes test results and failures to the configured JSON output file.
func (s *JSONStorage) Save(results []domain.TestResult, failures []domain.TestFailure, duration time.Duration, workers int) error {
	passed := 0
	failed := 0
	for _, r := range results {
		if r.Success {
			passed++
		} else {
			failed++
		}
	}

	output := domain.TestResultsOutput{
		Meta: domain.TestResultsMeta{
			TotalTestFiles:  len(results),
			FailedTestFiles: failed,
			PassedTestFiles: passed,
			FailedTestCases: len(failures),
			Duration:        duration.String(),
			DurationSeconds: duration.Seconds(),
			Workers:         workers,
			Timestamp:       time.Now().Format(time.RFC3339),
		},
		Details: failures,
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal results: %w", err)
	}

	path := s.cfg.GetOutputPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write results: %w", err)
	}
	return nil
}

// Load reads the last test results from the configured JSON output file.
func (s *JSONStorage) Load() (*domain.TestResultsOutput, error) {
	path := s.cfg.GetOutputPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read results file: %w", err)
	}
	var output domain.TestResultsOutput
	if err := json.Unmarshal(data, &output); err != nil {
		return nil, fmt.Errorf("parse results: %w", err)
	}
	return &output, nil
}

// SaveOutput writes the full output to the configured JSON file (e.g. after re-running selected tests).
func (s *JSONStorage) SaveOutput(output *domain.TestResultsOutput) error {
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal results: %w", err)
	}
	path := s.cfg.GetOutputPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}
