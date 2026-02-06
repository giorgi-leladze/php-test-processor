package main

import (
	"fmt"
	"regexp"
	"strings"
)

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

func parseTestFailure(result TestResult, failures *[]TestFailure) {
	str := strings.Split(result.Output, "\n")

	testFileName := result.TestPath
	testFileName = strings.TrimSuffix(testFileName, ".php")
	testFileName = strings.ReplaceAll(testFileName, "/", "\\")
	testFileName = testFileName + "::"

	pattern := "(?i)" + regexp.QuoteMeta(testFileName) // case insensitive
	match := regexp.MustCompile(pattern)

	for i := range len(str) {
		line := str[i]

		if match.MatchString(line) {
			testFailure := parseTestFailureCase(i, str, match)
			*failures = append(*failures, *testFailure)
			continue
		}
	}
}

func parseTestFailureCase(i int, str []string, match *regexp.Regexp) *TestFailure {
	filePath, name := parseTestFailureLine(str[i])
	testFailure := &TestFailure{
		TestName:     name,
		FilePath:     filePath,
		ErrorDetails: "",
		StackTrace:   []string{},
		File:         "",
		Line:         0,
		Message:      "",
	}

	var messageLines []string
	var jsonLines []string
	var stackTrace []string
	inJsonBlock := false
	jsonBraceCount := 0
	jsonBlockComplete := false

	// Parse from line after test name until next test or end
	for j := i + 1; j < len(str); j++ {
		line := str[j]
		trimmedLine := strings.TrimSpace(line)

		// Check if we hit the next test case
		if match.MatchString(line) {
			break
		}

		// Detect start of JSON block
		if trimmedLine == "{" && !inJsonBlock {
			inJsonBlock = true
			jsonBraceCount = 1
			jsonLines = append(jsonLines, line)
			continue
		}

		// If we're in JSON block, collect JSON lines
		if inJsonBlock {
			jsonLines = append(jsonLines, line)
			// Count braces to detect end of JSON
			jsonBraceCount += strings.Count(line, "{") - strings.Count(line, "}")
			if jsonBraceCount == 0 {
				// End of JSON block
				testFailure.ErrorDetails = strings.Join(jsonLines, "\n")
				inJsonBlock = false
				jsonBlockComplete = true
			}
			continue
		}

		// After JSON block, collect stack trace (file paths with line numbers)
		if jsonBlockComplete {
			// Stack trace lines are file paths with line numbers: /path/to/file.php:123
			if strings.Contains(line, ".php:") && (strings.HasPrefix(line, "/") || strings.Contains(line, "tests/")) {
				stackTrace = append(stackTrace, line)
				// Extract file and line from test file (not vendor files)
				if strings.Contains(line, "tests/") && testFailure.File == "" {
					parts := strings.Split(line, ":")
					if len(parts) >= 2 {
						testFailure.File = parts[0]
						fmt.Sscanf(parts[len(parts)-1], "%d", &testFailure.Line)
					}
				}
			}
			continue
		}

		// Before JSON block, collect message lines
		// Skip empty lines at the very start
		if len(messageLines) == 0 && trimmedLine == "" {
			continue
		}
		messageLines = append(messageLines, line)
	}

	// Join message lines (trim trailing empty lines)
	for len(messageLines) > 0 && strings.TrimSpace(messageLines[len(messageLines)-1]) == "" {
		messageLines = messageLines[:len(messageLines)-1]
	}
	testFailure.Message = strings.Join(messageLines, "\n")
	testFailure.StackTrace = stackTrace

	return testFailure
}

func parseTestFailureLine(line string) (filepath string, name string) {
	split := strings.Split(line, "::")

	nm := split[0]
	nm = strings.Split(nm, ")")[1]
	nm = strings.TrimSpace(nm)
	// Replace backslashes with forward slashes for file path
	nm = strings.ReplaceAll(nm, "\\", "/")

	n := split[1]

	return nm, n
}
