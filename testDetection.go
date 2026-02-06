package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

func findTestFiles(root string, nameFilter string) ([]string, error) {
	var testfiles []string
	skipDirs, err := getSkipDirs()
	if err != nil {
		return nil, err
	}

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

			if skipDirs[name] {
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

	return filterTestsByName(testfiles, nameFilter), err
}

// filterTestsByName filters test files by name pattern using wildcard matching
// Supports patterns like "*UserTest.php" or "*Payment*"
func filterTestsByName(tests []string, pattern string) []string {
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

func getSkipDirs() (map[string]bool, error) {
	skipDir := make(map[string]bool)
	for _, dir := range PATH_TO_IGNORE {
		skipDir[dir] = true
	}
	return skipDir, nil
}

func findTestCases(filePath string) ([]string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Println("error reading file", err)
		return nil, err
	}

	fileContent := string(content)
	testCasesMap := make(map[string]bool) // Use map to avoid duplicates

	// Pattern 1: Methods starting with "test" (more comprehensive)
	// Matches:
	// - public function testCreateUser()
	// - function test_user_login()
	// - protected static function testSomething()
	// - private function test_it_does_something()
	// - final public function testSomething()
	testMethodPattern := regexp.MustCompile(`(?m)^\s*(?:(?:public|protected|private|static|final)\s+)*(?:public|protected|private)?\s*function\s+(test\w+|test_\w+)\s*\(`)
	matches := testMethodPattern.FindAllStringSubmatch(fileContent, -1)

	for _, match := range matches {
		if len(match) > 1 {
			methodName := match[1]
			testCasesMap[methodName] = true
		}
	}

	// Pattern 2: Methods with @test annotation (more flexible)
	// Matches various formats:
	// - @test
	// - /** @test */
	// - @test on same line or next line
	// Handles different spacing and line breaks
	annotatedPatterns := []*regexp.Regexp{
		// @test on previous line(s) followed by function
		regexp.MustCompile(`(?m)@test\s*\n\s*(?:/\*\*.*?\*/)?\s*(?:(?:public|protected|private|static|final)\s+)*(?:public|protected|private)?\s*function\s+(\w+)\s*\(`),
		// @test in docblock (handles multi-line docblocks)
		regexp.MustCompile(`(?m)/\*\*[\s\S]*?@test[\s\S]*?\*/\s*(?:(?:public|protected|private|static|final)\s+)*(?:public|protected|private)?\s*function\s+(\w+)\s*\(`),
		// @test on same line as function (less common but possible)
		regexp.MustCompile(`(?m)@test.*?function\s+(\w+)\s*\(`),
	}

	for _, pattern := range annotatedPatterns {
		annotatedMatches := pattern.FindAllStringSubmatch(fileContent, -1)
		for _, match := range annotatedMatches {
			if len(match) > 1 {
				methodName := match[1]
				// Skip if it's already a test method (starts with "test")
				if !strings.HasPrefix(methodName, "test") {
					testCasesMap[methodName] = true
				}
			}
		}
	}

	// Convert map to sorted slice for consistent output
	var testCases []string
	for testCase := range testCasesMap {
		testCases = append(testCases, testCase)
	}

	// Sort for consistent output
	sort.Strings(testCases)

	return testCases, nil
}
