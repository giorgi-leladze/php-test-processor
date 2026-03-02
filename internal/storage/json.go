package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"ptp/internal/debug"
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

	timings := s.mergeTimings(results)

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
		Timings: timings,
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		debug.Logf("storage: marshal error: %v", err)
		return fmt.Errorf("marshal results: %w", err)
	}

	path := s.cfg.GetOutputPath()
	debug.Logf("storage: saving results to %s (%d bytes)", path, len(data))
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		debug.Logf("storage: create dir error: %v", err)
		return fmt.Errorf("create output dir: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		debug.Logf("storage: write error: %v", err)
		return fmt.Errorf("write results: %w", err)
	}
	return nil
}

// Load reads the last test results from the configured JSON output file.
func (s *JSONStorage) Load() (*domain.TestResultsOutput, error) {
	path := s.cfg.GetOutputPath()
	debug.Logf("storage: loading results from %s", path)
	data, err := os.ReadFile(path)
	if err != nil {
		debug.Logf("storage: read error: %v", err)
		return nil, fmt.Errorf("read results file: %w", err)
	}
	var output domain.TestResultsOutput
	if err := json.Unmarshal(data, &output); err != nil {
		debug.Logf("storage: parse error: %v", err)
		return nil, fmt.Errorf("parse results: %w", err)
	}
	debug.Logf("storage: loaded %d failures, %d total test files", len(output.Details), output.Meta.TotalTestFiles)
	return &output, nil
}

// mergeTimings loads historical timings and merges in new results using a running average.
func (s *JSONStorage) mergeTimings(results []domain.TestResult) map[string]*domain.TestTiming {
	timings := make(map[string]*domain.TestTiming)

	prev, err := s.Load()
	if err == nil && prev != nil && prev.Timings != nil {
		for k, v := range prev.Timings {
			cp := *v
			timings[k] = &cp
		}
	}

	for _, r := range results {
		dur := r.Duration.Seconds()
		if t, ok := timings[r.TestPath]; ok {
			t.Avg = (float64(t.Count)*t.Avg + dur) / float64(t.Count+1)
			t.Count++
		} else {
			timings[r.TestPath] = &domain.TestTiming{Count: 1, Avg: dur}
		}
	}

	return timings
}

// LoadTimings returns historical per-test timing data, or nil if unavailable.
func (s *JSONStorage) LoadTimings() map[string]*domain.TestTiming {
	prev, err := s.Load()
	if err != nil || prev == nil {
		return nil
	}
	return prev.Timings
}

// SaveOutput writes the full output to the configured JSON file (e.g. after re-running selected tests).
func (s *JSONStorage) SaveOutput(output *domain.TestResultsOutput) error {
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		debug.Logf("storage: marshal error (SaveOutput): %v", err)
		return fmt.Errorf("marshal results: %w", err)
	}
	path := s.cfg.GetOutputPath()
	debug.Logf("storage: saving output to %s (%d bytes)", path, len(data))
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		debug.Logf("storage: create dir error: %v", err)
		return fmt.Errorf("create output dir: %w", err)
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		debug.Logf("storage: open file error: %v", err)
		return fmt.Errorf("create results file: %w", err)
	}
	_, err = f.Write(data)
	if err != nil {
		f.Close()
		debug.Logf("storage: write error: %v", err)
		return fmt.Errorf("write results: %w", err)
	}
	if err := f.Sync(); err != nil {
		f.Close()
		debug.Logf("storage: sync error: %v", err)
		return fmt.Errorf("sync results file: %w", err)
	}
	f.Close()
	return nil
}
