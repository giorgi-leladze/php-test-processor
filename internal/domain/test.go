package domain

// Test represents a test file to be executed
type Test struct {
	Path     string // Full path to the test file
	FilePath string // Relative file path
	FileName string // Just the filename
}

// TestCase represents a single test case within a test file
type TestCase struct {
	Name     string // Test method name
	FilePath string // Path to the test file containing this case
}

