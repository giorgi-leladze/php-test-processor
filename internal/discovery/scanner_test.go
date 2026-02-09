package discovery

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanner_Scan(t *testing.T) {
	// Create a temporary directory structure for testing
	tmpDir, err := os.MkdirTemp("", "ptp-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test directory structure
	testDirs := []string{
		"tests/unit",
		"tests/integration",
		"vendor",
		"node_modules",
	}
	for _, dir := range testDirs {
		if err := os.MkdirAll(filepath.Join(tmpDir, dir), 0755); err != nil {
			t.Fatalf("failed to create dir %s: %v", dir, err)
		}
	}

	// Create test files
	testFiles := []string{
		"tests/unit/UserTest.php",
		"tests/unit/PaymentTest.php",
		"tests/integration/OrderTest.php",
		"vendor/some/lib.php",
		"node_modules/some/file.js",
		"not_a_test.php",
	}
	for _, file := range testFiles {
		fullPath := filepath.Join(tmpDir, file)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("failed to create dir for %s: %v", file, err)
		}
		if err := os.WriteFile(fullPath, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create file %s: %v", file, err)
		}
	}

	scanner := NewScanner([]string{"vendor", "node_modules"})

	t.Run("scans test files correctly", func(t *testing.T) {
		results, err := scanner.Scan(tmpDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should find 3 test files, not the ones in vendor/node_modules
		if len(results) != 3 {
			t.Errorf("expected 3 test files, got %d", len(results))
		}
	})

	t.Run("returns error for non-existent directory", func(t *testing.T) {
		_, err := scanner.Scan("/non/existent/path")
		if err == nil {
			t.Error("expected error for non-existent directory")
		}
	})

	t.Run("returns error for file instead of directory", func(t *testing.T) {
		testFile := filepath.Join(tmpDir, "testfile.txt")
		os.WriteFile(testFile, []byte("test"), 0644)
		_, err := scanner.Scan(testFile)
		if err == nil {
			t.Error("expected error for file path")
		}
	})
}

