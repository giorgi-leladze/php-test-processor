package discovery

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

// Parser parses test files to extract test cases
type Parser struct{}

// NewParser creates a new Parser
func NewParser() *Parser {
	return &Parser{}
}

// FindTestCases finds all test cases in a test file
func (p *Parser) FindTestCases(filePath string) ([]string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading file %s: %w", filePath, err)
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
