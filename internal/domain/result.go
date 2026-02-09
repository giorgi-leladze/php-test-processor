package domain

import "time"

// TestResult represents the result of executing a test file
type TestResult struct {
	TestPath string    // Path to the test file that was executed
	Success  bool      // Whether the test passed
	Output   string    // Raw output from PHPUnit
	Error    error     // Error if execution failed
	Duration time.Duration // Time taken to execute
}

// TestResultsMeta contains metadata about a test run
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

// TestResultsOutput is the complete output structure for test results
type TestResultsOutput struct {
	Meta    TestResultsMeta `json:"meta"`
	Details []TestFailure   `json:"details"`
}

