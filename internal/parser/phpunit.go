package parser

import (
	"fmt"
	"regexp"
	"strings"

	"ptp/internal/domain"
)

// PHPUnitParser parses PHPUnit test output
type PHPUnitParser struct{}

// NewPHPUnitParser creates a new PHPUnitParser
func NewPHPUnitParser() *PHPUnitParser {
	return &PHPUnitParser{}
}

// ParseTestCounts extracts passed and failed test case counts from PHPUnit output.
// Returns (passed, failed). If parsing fails, returns (1,0) for success or (0,1) for failure (file-level fallback).
func (p *PHPUnitParser) ParseTestCounts(result domain.TestResult) (passed, failed int) {
	output := result.Output

	// OK (N tests, ...) - all passed
	okMatch := regexp.MustCompile(`OK\s*\(\s*(\d+)\s+tests`).FindStringSubmatch(output)
	if len(okMatch) >= 2 {
		var total int
		fmt.Sscanf(okMatch[1], "%d", &total)
		return total, 0
	}

	// FAILURES! or ERRORS! - Tests: N, Assertions: ..., Failures: F, Errors: E
	testsMatch := regexp.MustCompile(`Tests:\s*(\d+)`).FindStringSubmatch(output)
	failMatch := regexp.MustCompile(`Failures:\s*(\d+)`).FindStringSubmatch(output)
	errMatch := regexp.MustCompile(`Errors:\s*(\d+)`).FindStringSubmatch(output)
	var total, failures, errors int
	if len(testsMatch) >= 2 {
		fmt.Sscanf(testsMatch[1], "%d", &total)
	}
	if len(failMatch) >= 2 {
		fmt.Sscanf(failMatch[1], "%d", &failures)
	}
	if len(errMatch) >= 2 {
		fmt.Sscanf(errMatch[1], "%d", &errors)
	}
	failed = failures + errors
	if total >= failed {
		passed = total - failed
	}
	if passed > 0 || failed > 0 {
		return passed, failed
	}

	// Fallback: one "test" per file
	if result.Success {
		return 1, 0
	}
	return 0, 1
}

// ParseFailure parses test failure from PHPUnit output
func (p *PHPUnitParser) ParseFailure(result domain.TestResult) []domain.TestFailure {
	var failures []domain.TestFailure
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
			testFailure := p.parseTestFailureCase(i, str, match)
			failures = append(failures, *testFailure)
			continue
		}
	}

	return failures
}

func (p *PHPUnitParser) parseTestFailureCase(i int, str []string, match *regexp.Regexp) *domain.TestFailure {
	filePath, name := p.parseTestFailureLine(str[i])
	testFailure := &domain.TestFailure{
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

func (p *PHPUnitParser) parseTestFailureLine(line string) (filepath string, name string) {
	split := strings.Split(line, "::")

	nm := split[0]
	nm = strings.Split(nm, ")")[1]
	nm = strings.TrimSpace(nm)
	// Replace backslashes with forward slashes for file path
	nm = strings.ReplaceAll(nm, "\\", "/")

	n := split[1]

	return nm, n
}

