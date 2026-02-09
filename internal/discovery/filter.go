package discovery

import (
	"path/filepath"
	"strings"
)

// Filter filters test files by name pattern
type Filter struct{}

// NewFilter creates a new Filter
func NewFilter() *Filter {
	return &Filter{}
}

// FilterByName filters test files by name pattern using wildcard matching
// Supports patterns like "*UserTest.php" or "*Payment*"
func (f *Filter) FilterByName(tests []string, pattern string) []string {
	if pattern == "" {
		return tests
	}

	var filtered []string

	for _, test := range tests {
		// Get just the filename from the full path
		testName := filepath.Base(test)

		// Try to match using filepath.Match (supports * and ? wildcards)
		matched, err := filepath.Match(pattern, testName)
		if err == nil && matched {
			filtered = append(filtered, test)
			continue
		}

		// If pattern contains wildcards but filepath.Match didn't match,
		// try a more flexible substring match for patterns like "*Payment*"
		if strings.Contains(pattern, "*") {
			// Remove wildcards and check if the remaining pattern is in the test name
			patternParts := strings.Split(pattern, "*")
			allPartsMatch := true
			for _, part := range patternParts {
				if part != "" && !strings.Contains(testName, part) {
					allPartsMatch = false
					break
				}
			}
			if allPartsMatch && len(patternParts) > 0 {
				// Ensure at least one non-empty part exists
				hasNonEmptyPart := false
				for _, part := range patternParts {
					if part != "" {
						hasNonEmptyPart = true
						break
					}
				}
				if hasNonEmptyPart {
					filtered = append(filtered, test)
					continue
				}
			}
		}

		// If no wildcards, do a simple contains check
		if !strings.Contains(pattern, "*") && !strings.Contains(pattern, "?") {
			if strings.Contains(testName, pattern) {
				filtered = append(filtered, test)
			}
		}
	}

	return filtered
}
