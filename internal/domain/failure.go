package domain

// TestFailure represents a failed test case
type TestFailure struct {
	TestName     string   `json:"test_name"`
	FilePath     string   `json:"file_path"`
	ErrorDetails string   `json:"error_details"`
	StackTrace   []string `json:"stack_trace"`
	File         string   `json:"file"`
	Line         int      `json:"line"`
	Message      string   `json:"message"`
	Resolved     bool     `json:"resolved,omitempty"` // Track if test case is marked as resolved
}

