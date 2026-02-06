package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/schollz/progressbar/v3"
)

func formatTestExecutionOutput(output []string) {
}

func formatTestListOutput(output []string) {
}

func formatTestVersionOutput(output []string) {
}

func progressBar(count int) *progressbar.ProgressBar {
	// Create progress bar with dynamic description
	bar := progressbar.NewOptions(count,
		progressbar.OptionSetDescription(
			color.CyanString("Running tests: ")+
				color.GreenString("[success: 0")+
				" | "+
				color.RedString("failed: 0]"),
		),
		progressbar.OptionSetWidth(50),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        color.CyanString("█"),
			SaucerHead:    color.CyanString("█"),
			SaucerPadding: "░",
			BarStart:      "│",
			BarEnd:        "│",
		}),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionOnCompletion(func() {
			fmt.Print("\n")
		}),
		progressbar.OptionSetRenderBlankState(true),
	)

	return bar
}

func updateProgressBar(bar *progressbar.ProgressBar, successCount int, failCount int) {
	bar.Set(successCount + failCount)
	bar.Describe(
		color.CyanString("Running tests: ") +
			color.GreenString("[success: %d", successCount) +
			" | " +
			color.RedString("failed: %d]", failCount),
	)
}

// outputTextResults formats and outputs test results as text
func outputTextResults(results []TestResult, totalTests int, duration time.Duration) error {
	var successCount, failCount int
	var failedTests []TestResult

	for _, result := range results {
		if result.Success {
			successCount++
		} else {
			failCount++
			failedTests = append(failedTests, result)
		}
	}

	// Print summary
	fmt.Print("\n")
	color.Cyan("╔════════════════════════════════════════════════════════════╗")
	color.Cyan("║                      Test Summary                          ║")
	color.Cyan("╚════════════════════════════════════════════════════════════╝\n")

	// Print statistics
	if successCount > 0 {
		color.Green("✓ Passed: %d", successCount)
		fmt.Println()
	}
	if failCount > 0 {
		color.Red("✗ Failed: %d", failCount)
		fmt.Println()
	}

	color.White("Total: %d | Duration: %s\n", totalTests, duration.Round(time.Millisecond))

	// Print failed tests if any
	if failCount > 0 {
		fmt.Println()
		color.Red("╔════════════════════════════════════════════════════════════╗")
		color.Red("║                      Failed Tests                          ║")
		color.Red("╚════════════════════════════════════════════════════════════╝\n")

		for i, result := range failedTests {
			color.Red("%d. %s", i+1, result.TestPath)
			if result.Error != nil {
				color.Yellow("   Error: %v", result.Error)
			}
			fmt.Println()
		}
	}

	return nil
}

// printMetaStats reads and displays meta statistics from the JSON results file
func printMetaStats() error {
	// Clear terminal screen
	fmt.Print("\033[2J\033[H") // ANSI escape codes: clear screen and move cursor to top

	outputPath := filepath.Join(OUTPUT_JSON_DIR, OUTPUT_JSON_FILE)

	// Read JSON file
	data, err := os.ReadFile(outputPath)
	if err != nil {
		return fmt.Errorf("failed to read JSON file: %w", err)
	}

	// Parse JSON
	var output TestResultsOutput
	if err := json.Unmarshal(data, &output); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	meta := output.Meta

	// Print header
	fmt.Print("\n")
	color.Cyan("╔═══════════════════════════════════════════════════════════════╗")
	color.Cyan("║                    Test Execution Statistics                  ║")
	color.Cyan("╚═══════════════════════════════════════════════════════════════╝\n")

	// Print table
	fmt.Println("┌─────────────────────────────────┬─────────────────────────────┐")

	// Total Test Files
	fmt.Printf("│ %-31s │ ", "Total Test Files")
	color.White("%-27d │\n", meta.TotalTestFiles)
	fmt.Println("├─────────────────────────────────┼─────────────────────────────┤")

	// Passed Test Files
	fmt.Printf("│ %-31s │ ", "Passed Test Files")
	color.Green("%-27d │\n", meta.PassedTestFiles)
	fmt.Println("├─────────────────────────────────┼─────────────────────────────┤")

	// Failed Test Files
	fmt.Printf("│ %-31s │ ", "Failed Test Files")
	color.Red("%-27d │\n", meta.FailedTestFiles)
	fmt.Println("├─────────────────────────────────┼─────────────────────────────┤")

	// Failed Test Cases
	fmt.Printf("│ %-31s │ ", "Failed Test Cases")
	color.Red("%-27d │\n", meta.FailedTestCases)
	fmt.Println("├─────────────────────────────────┼─────────────────────────────┤")

	// Duration
	fmt.Printf("│ %-31s │ ", "Duration")
	durationStr := fmt.Sprintf("%.2fs", meta.DurationSeconds)
	color.White("%-27s │\n", durationStr)
	fmt.Println("├─────────────────────────────────┼─────────────────────────────┤")

	// Workers
	fmt.Printf("│ %-31s │ ", "Workers")
	color.White("%-27d │\n", meta.Workers)
	fmt.Println("├─────────────────────────────────┼─────────────────────────────┤")

	// Timestamp
	fmt.Printf("│ %-31s │ ", "Timestamp")
	color.White("%-27s │\n", meta.Timestamp)

	fmt.Println("└─────────────────────────────────┴─────────────────────────────┘")

	// Print summary line
	fmt.Println()
	if meta.FailedTestFiles == 0 {
		color.Green("✓ All tests passed!")
	} else {
		color.Red("✗ %d test file(s) failed with %d test case failure(s)", meta.FailedTestFiles, meta.FailedTestCases)
		fmt.Println()
		printFailedTestsTree(output.Details)
	}

	return nil
}

// TreeNode represents a node in the file tree structure
type TreeNode struct {
	Name     string
	Children map[string]*TreeNode
	Failures []TestFailure
	IsFile   bool
}

// printFailedTestsTree prints a tree structure of failed tests
func printFailedTestsTree(failures []TestFailure) {
	if len(failures) == 0 {
		return
	}

	// Group failures by file path
	fileMap := make(map[string][]TestFailure)
	for _, failure := range failures {
		fileMap[failure.FilePath] = append(fileMap[failure.FilePath], failure)
	}

	root := &TreeNode{
		Name:     "",
		Children: make(map[string]*TreeNode),
		IsFile:   false,
	}

	// Process each file
	for filePath, fileFailures := range fileMap {
		parts := strings.Split(strings.TrimPrefix(filePath, "./"), "/")
		current := root

		// Navigate/create tree nodes for each path part
		for i, part := range parts {
			if part == "" {
				continue
			}

			if current.Children[part] == nil {
				current.Children[part] = &TreeNode{
					Name:     part,
					Children: make(map[string]*TreeNode),
					IsFile:   i == len(parts)-1,
				}
			}

			current = current.Children[part]

			// If this is the file (last part), add failures
			if i == len(parts)-1 {
				current.Failures = fileFailures
			}
		}
	}

	// Print tree recursively
	printTreeNode(root, "", true, true)
}

func printTreeNode(node *TreeNode, prefix string, isLast bool, isRoot bool) {
	// Sort children for consistent output
	var keys []string
	for key := range node.Children {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// Print children
	for i, key := range keys {
		child := node.Children[key]
		isLastChild := i == len(keys)-1

		// Determine connector
		var connector string
		if isRoot {
			connector = ""
		} else if isLastChild {
			connector = prefix + "   |_"
		} else {
			connector = prefix + "  |_"
		}

		// Print child node
		if child.IsFile {
			color.Yellow("%s%s", connector, child.Name)
		} else {
			color.Cyan("%s%s", connector, child.Name)
		}

		// Print test cases if this is a file
		if child.IsFile && len(child.Failures) > 0 {
			for j, failure := range child.Failures {
				isLastCase := j == len(child.Failures)-1
				var casePrefix string
				if isLastChild {
					if isLastCase {
						casePrefix = strings.ReplaceAll(prefix, "|", " ") + "        |_"
					} else {
						casePrefix = prefix + "  |        |_"
					}
				} else {
					if isLastCase {
						casePrefix = prefix + "  |        |_"
					} else {
						casePrefix = prefix + "  |  |     |_"
					}
				}
				color.Red("%s%s", casePrefix, failure.TestName)
			}
		}

		// Recursively print children
		var newPrefix string
		if isRoot {
			newPrefix = "  "
		} else if isLastChild {
			newPrefix = strings.ReplaceAll(prefix, "|", " ") + "  "
		} else {
			newPrefix = prefix + "  |"
		}
		printTreeNode(child, newPrefix, isLastChild, false)
	}
}

func printTestCasesFormater(tests []string) {
	if GlobalFlags.TestCases {
		// Display tree view with test cases
		color.Green("Found %d test file(s) with test cases:\n", len(tests))

		for i, test := range tests {
			testCases, err := findTestCases(test)
			if err != nil {
				color.Red("Error reading test file %s: %v", test, err)
				continue
			}

			// Get relative path for cleaner display
			relPath, err := filepath.Rel(PROJECT_PATH, test)
			if err != nil {
				relPath = test
			}

			// Print test file as root node
			isLastFile := i == len(tests)-1
			if isLastFile {
				// Last item
				color.Cyan("└── %s", relPath)
			} else {
				// Not last item
				color.Cyan("├── %s", relPath)
			}

			// Print test cases as children
			if len(testCases) == 0 {
				// No test cases found
				var prefix string
				if isLastFile {
					prefix = "    └── "
				} else {
					prefix = "│   └── "
				}
				fmt.Printf("%s%s\n", prefix, color.RedString("(no test cases found)"))
			} else {
				for j, testCase := range testCases {
					isLastCase := j == len(testCases)-1

					var prefix string
					if isLastFile {
						if isLastCase {
							prefix = "    └── "
						} else {
							prefix = "    ├── "
						}
					} else {
						if isLastCase {
							prefix = "│   └── "
						} else {
							prefix = "│   ├── "
						}
					}

					fmt.Printf("%s%s\n", prefix, color.YellowString(testCase))
				}
			}

			// Add spacing between files (except for the last one)
			if i < len(tests)-1 {
				fmt.Println()
			}
		}
	} else {
		// Display simple list of test files
		color.Green("Found %d test file(s):\n", len(tests))

		for i, test := range tests {
			// Get relative path for cleaner display
			relPath, err := filepath.Rel(PROJECT_PATH, test)
			if err != nil {
				relPath = test
			}

			if i == len(tests)-1 {
				color.Cyan("└── %s", relPath)
			} else {
				color.Cyan("├── %s", relPath)
			}
		}
	}
}
