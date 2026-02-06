package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/fatih/color"
)

// TestResult represents the result of executing a test
type TestResult struct {
	TestPath string
	Success  bool
	Output   string
	Error    error
}

func executeTests(tests []string) ([]TestResult, time.Duration, error) {
	totalTests := len(tests)
	if totalTests == 0 {
		return nil, 0, nil
	}

	// Print header
	color.Cyan("\n╔════════════════════════════════════════════════════════════╗")
	color.Cyan("║          PHP Test Processor - Parallel Execution           ║")
	color.Cyan("╚════════════════════════════════════════════════════════════╝\n")

	color.White("Total tests: %d | Workers: %d\n", totalTests, GlobalFlags.Processors)

	// Create a channel to send test paths to workers
	testQueue := make(chan string, len(tests))
	results := make(chan TestResult, len(tests))

	// Send all tests to the queue
	for _, test := range tests {
		testQueue <- test
	}
	close(testQueue) // Close channel so workers know when to stop

	// Track progress
	var mu sync.Mutex
	var completedCount int
	var successCount, failCount int

	startTime := time.Now()

	bar := progressBar(totalTests)

	// Create worker pool
	var wg sync.WaitGroup
	for i := 1; i <= GlobalFlags.Processors; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			// Worker loop: keep processing tests until channel is closed
			for testPath := range testQueue {
				result := runPHPUnitTest(testPath, workerID)
				results <- result

				mu.Lock()
				completedCount++
				if result.Success {
					successCount++
				} else {
					failCount++
				}
				// Update progress bar with real-time counts
				updateProgressBar(bar, successCount, failCount)
				mu.Unlock()
			}
		}(i)
	}

	// Close results channel when all workers are done
	go func() {
		wg.Wait()
		close(results)
	}()

	var allResults []TestResult
	for result := range results {
		allResults = append(allResults, result)
	}

	duration := time.Since(startTime)

	return allResults, duration, nil
}

// runPHPUnitTest executes PHPUnit for a single test file
func runPHPUnitTest(testPath string, id int) TestResult {
	// Path to phpunit (adjust based on your project structure)
	phpunitPath := fmt.Sprintf("%s/vendor/bin/phpunit", PROJECT_PATH)
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, phpunitPath, testPath)

	// Set environment variables
	cmd.Env = os.Environ() // Start with current environment
	cmd.Env = append(cmd.Env, fmt.Sprintf("DB_DATABASE=webiz_testing_%d", id))

	// Optionally set working directory
	cmd.Dir = PROJECT_PATH

	output, err := cmd.CombinedOutput()

	return TestResult{
		TestPath: testPath,
		Success:  err == nil,
		Output:   string(output), // XML output
		Error:    err,
	}
}
