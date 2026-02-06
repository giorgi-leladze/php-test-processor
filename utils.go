package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type TestResultsOutput struct {
	Meta    TestResultsMeta `json:"meta"`
	Details []TestFailure   `json:"details"`
}

type TestResultsMeta struct {
	TotalTestFiles  int     `json:"total_test_files"`
	FailedTestFiles int     `json:"failed_test_files"`
	PassedTestFiles int     `json:"passed_test_files"`
	FailedTestCases int     `json:"failed_test_cases"`
	Duration        string  `json:"duration"`
	DurationSeconds float64 `json:"duration_seconds"`
	Workers         int     `json:"workers"`
	Timestamp       string  `json:"timestamp"`
}

func saveTestResultsToJSON(results []TestResult, failures []TestFailure, duration time.Duration) error {
	// Calculate statistics
	totalTestFiles := len(results)
	var failedTestFiles, passedTestFiles int
	for _, result := range results {
		if result.Success {
			passedTestFiles++
		} else {
			failedTestFiles++
		}
	}
	failedTestCases := len(failures)

	// Build meta information
	meta := TestResultsMeta{
		TotalTestFiles:  totalTestFiles,
		FailedTestFiles: failedTestFiles,
		PassedTestFiles: passedTestFiles,
		FailedTestCases: failedTestCases,
		Duration:        duration.String(),
		DurationSeconds: duration.Seconds(),
		Workers:         GlobalFlags.Processors,
		Timestamp:       time.Now().Format(time.RFC3339),
	}

	// Create output structure
	output := TestResultsOutput{
		Meta:    meta,
		Details: failures,
	}

	outputPath := filepath.Join(OUTPUT_JSON_DIR, OUTPUT_JSON_FILE)

	// Create storage directory if it doesn't exist
	if err := os.MkdirAll(OUTPUT_JSON_DIR, 0755); err != nil {
		return fmt.Errorf("failed to create storage directory: %w", err)
	}

	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(outputPath, jsonData, 0644)
}

func getTestPath() string {
	// Determine test path: use flag if provided, otherwise use default
	testPath := PROJECT_PATH + "/" + TEST_PATH
	if GlobalFlags.TestPath != "" {
		// If TestPath is provided, make it relative to PROJECT_PATH if it's not absolute
		if filepath.IsAbs(GlobalFlags.TestPath) {
			testPath = GlobalFlags.TestPath
		} else {
			testPath = filepath.Join(PROJECT_PATH, GlobalFlags.TestPath)
		}
	}

	return testPath
}
