package main

import (
	"fmt"
	"os"
)

const (
	// Project path, this will be used to find the test files - will be changed later on with "./"
	PROJECT_PATH = "."
	TEST_PATH    = "."
	// Output JSON file path, this will be used to store the test results
	OUTPUT_JSON_FILE = "test-results.json"
	OUTPUT_JSON_DIR  = "storage" // Store results in storage folder
)

var PATH_TO_IGNORE = []string{"vendor", "node_modules", "public", "storage", "bootstrap", "config", "database", "resources", "routes"}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// color.Cyan("\n╔════════════════════════════════════════════════════════════╗")
	// color.Cyan("║              PHP Test Processor - Discovery              ║")
	// color.Cyan("╚════════════════════════════════════════════════════════════╝\n")

	// totalStart := time.Now()
	// start := time.Now()
	// color.White("Scanning for test files...")

	// test, err := findTestFiles("../../work/hrms-backend")

	// if err != nil {
	// 	color.Red("Error: %v", err)
	// 	return
	// }

	// duration := time.Since(start)

	// if len(test) == 0 {
	// 	color.Yellow("No test files found")
	// 	return
	// }

	// color.Green("✓ Found %d test file(s)", len(test))
	// color.White("  Discovery time: %s\n", duration.Round(time.Millisecond))

	// err = executeTests(test)

	// if err != nil {
	// 	fmt.Println("Error: ", err)
	// 	return
	// }

	// totalDuration := time.Since(totalStart)
	// color.Cyan("\n╔════════════════════════════════════════════════════════════╗")
	// color.Cyan("║                    All Tests Completed                   ║")
	// color.Cyan("╚════════════════════════════════════════════════════════════╝")
	// color.White("Total execution time: %s\n", totalDuration.Round(time.Millisecond))
}
