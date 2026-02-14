package ui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fatih/color"
	"ptp/internal/config"
	"ptp/internal/domain"
	"ptp/internal/discovery"
)

// Formatter formats and displays output
type Formatter struct {
	config  *config.Config
	parser  *discovery.Parser
}

// NewFormatter creates a new Formatter
func NewFormatter(cfg *config.Config, parser *discovery.Parser) *Formatter {
	return &Formatter{
		config: cfg,
		parser: parser,
	}
}

// PrintMetaStats reads and displays meta statistics from the JSON results file
func (f *Formatter) PrintMetaStats() error {
	// Clear terminal screen
	fmt.Print("\033[2J\033[H")

	outputPath := f.config.GetOutputPath()

	// Read JSON file
	data, err := os.ReadFile(outputPath)
	if err != nil {
		return fmt.Errorf("failed to read JSON file: %w", err)
	}

	// Parse JSON
	var output domain.TestResultsOutput
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
		f.printFailedTestsTree(output.Details)
	}

	return nil
}

// TreeNode represents a node in the file tree structure
type TreeNode struct {
	Name     string
	Children map[string]*TreeNode
	Failures []domain.TestFailure
	IsFile   bool
}

// printFailedTestsTree prints a tree structure of failed tests
func (f *Formatter) printFailedTestsTree(failures []domain.TestFailure) {
	if len(failures) == 0 {
		return
	}

	// Group failures by file path
	fileMap := make(map[string][]domain.TestFailure)
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
	f.printTreeNode(root, "", true, true)
}

func (f *Formatter) printTreeNode(node *TreeNode, prefix string, isLast bool, isRoot bool) {
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
		f.printTreeNode(child, newPrefix, isLastChild, false)
	}
}

// CountTestCases returns the total number of test cases across the given test files.
func (f *Formatter) CountTestCases(tests []string) (int, error) {
	var total int
	for _, test := range tests {
		cases, err := f.parser.FindTestCases(test)
		if err != nil {
			return 0, err
		}
		total += len(cases)
	}
	return total, nil
}

// normalizedPathForKey returns a path key for matching (same logic as commands package).
func normalizedPathForKey(projectPath, path string) string {
	p := path
	if projectPath != "" {
		if rel, err := filepath.Rel(projectPath, path); err == nil && rel != ".." && !strings.HasPrefix(rel, "..") {
			p = rel
		}
	}
	p = filepath.ToSlash(p)
	p = strings.TrimSuffix(p, ".php")
	return strings.ToLower(p)
}

// PrintTestList prints a list of test files, optionally with test cases.
// failedPaths is optional; if set, files in this set are marked with [F] in red (from last run).
func (f *Formatter) PrintTestList(tests []string, showTestCases bool, failedPaths map[string]struct{}) error {
	if showTestCases {
		// Display tree view with test cases
		color.Green("Found %d test file(s) with test cases:\n", len(tests))

		for i, test := range tests {
			testCases, err := f.parser.FindTestCases(test)
			if err != nil {
				color.Red("Error reading test file %s: %v", test, err)
				continue
			}

			// Get relative path for cleaner display
			relPath, err := filepath.Rel(f.config.ProjectPath, test)
			if err != nil {
				relPath = test
			}

			failMarker := ""
			if len(failedPaths) > 0 {
				key := normalizedPathForKey(f.config.ProjectPath, test)
				if _, ok := failedPaths[key]; ok {
					failMarker = " " + color.RedString("[F]")
				}
			}

			// Print test file as root node
			isLastFile := i == len(tests)-1
			if isLastFile {
				color.Cyan("└── %s%s", relPath, failMarker)
			} else {
				color.Cyan("├── %s%s", relPath, failMarker)
			}

			// Print test cases as children
			if len(testCases) == 0 {
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
			relPath, err := filepath.Rel(f.config.ProjectPath, test)
			if err != nil {
				relPath = test
			}

			failMarker := ""
			if len(failedPaths) > 0 {
				key := normalizedPathForKey(f.config.ProjectPath, test)
				if _, ok := failedPaths[key]; ok {
					failMarker = " " + color.RedString("[F]")
				}
			}

			if i == len(tests)-1 {
				color.Cyan("└── %s%s", relPath, failMarker)
			} else {
				color.Cyan("├── %s%s", relPath, failMarker)
			}
		}
	}

	return nil
}

