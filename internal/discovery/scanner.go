package discovery

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Scanner scans for test files in a directory
type Scanner struct {
	skipDirs map[string]bool
}

// NewScanner creates a new Scanner with the given directories to skip
func NewScanner(skipDirs []string) *Scanner {
	skipMap := make(map[string]bool)
	for _, dir := range skipDirs {
		skipMap[dir] = true
	}
	return &Scanner{skipDirs: skipMap}
}

// Scan finds all test files in the given root directory
func (s *Scanner) Scan(root string) ([]string, error) {
	var testfiles []string

	// Clean and validate the root path
	root = filepath.Clean(root)
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("test path does not exist: %s", root)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("test path is not a directory: %s", root)
	}

	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			name := d.Name()
			// Skip hidden directories (starting with .)
			if strings.HasPrefix(name, ".") && name != "." && name != ".." {
				return filepath.SkipDir
			}

			if s.skipDirs[name] {
				return filepath.SkipDir
			}

			return nil
		}

		// Check if file ends with Test.php
		if strings.HasSuffix(d.Name(), "Test.php") {
			testfiles = append(testfiles, path)
			return nil
		}

		return nil
	})

	return testfiles, err
}
