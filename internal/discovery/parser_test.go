package discovery

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParser_FindTestCases(t *testing.T) {
	parser := NewParser()

	// Create a temporary PHP test file
	tmpDir, err := os.MkdirTemp("", "ptp-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "UserTest.php")
	phpContent := `<?php

class UserTest extends TestCase
{
    public function testCreateUser()
    {
        // test code
    }

    protected function testUpdateUser()
    {
        // test code
    }

    private function testDeleteUser()
    {
        // test code
    }

    /**
     * @test
     */
    public function testWithAnnotation()
    {
        // test code
    }

    public function helperMethod()
    {
        // not a test
    }
}
`
	if err := os.WriteFile(testFile, []byte(phpContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	t.Run("finds test methods", func(t *testing.T) {
		testCases, err := parser.FindTestCases(testFile)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should find at least testCreateUser, testUpdateUser, testDeleteUser, testWithAnnotation
		if len(testCases) < 4 {
			t.Errorf("expected at least 4 test cases, got %d: %v", len(testCases), testCases)
		}

		// Check for specific test methods
		found := make(map[string]bool)
		for _, tc := range testCases {
			found[tc] = true
		}

		expectedTests := []string{"testCreateUser", "testUpdateUser", "testDeleteUser", "testWithAnnotation"}
		for _, expected := range expectedTests {
			if !found[expected] {
				t.Errorf("expected to find test case %s", expected)
			}
		}

		// Should not find helperMethod
		if found["helperMethod"] {
			t.Error("should not find helperMethod as a test case")
		}
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		_, err := parser.FindTestCases("/non/existent/file.php")
		if err == nil {
			t.Error("expected error for non-existent file")
		}
	})
}

