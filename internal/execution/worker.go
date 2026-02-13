package execution

import (
	"sync"
	"time"

	"ptp/internal/config"
	"ptp/internal/domain"
	"ptp/internal/parser"
	"ptp/internal/ui"
)

// WorkerPool manages a pool of workers for parallel test execution
type WorkerPool struct {
	config   *config.Config
	runner   *Runner
	scheduler Scheduler
	progress *ui.ProgressBar
	parser   *parser.PHPUnitParser
}

// NewWorkerPool creates a new WorkerPool
func NewWorkerPool(cfg *config.Config, runner *Runner, scheduler Scheduler, phpUnitParser *parser.PHPUnitParser) *WorkerPool {
	return &WorkerPool{
		config:    cfg,
		runner:    runner,
		scheduler: scheduler,
		parser:    phpUnitParser,
	}
}

// SetProgress sets the progress bar for the worker pool
func (wp *WorkerPool) SetProgress(progress *ui.ProgressBar) {
	wp.progress = progress
}

// Execute executes tests in parallel using worker pool
func (wp *WorkerPool) Execute(tests []string) ([]domain.TestResult, time.Duration, error) {
	if len(tests) == 0 {
		return nil, 0, nil
	}

	// Create channels for test distribution and results
	testQueue := make(chan string, len(tests))
	results := make(chan domain.TestResult, len(tests))

	// Send all tests to the queue
	for _, test := range tests {
		testQueue <- test
	}
	close(testQueue)

	// Track progress: file count for bar position, test case counts for label
	var mu sync.Mutex
	var completedFiles int
	var passedCases, failedCases int

	startTime := time.Now()

	// Create worker pool
	var wg sync.WaitGroup
	workerCount := wp.config.Processors
	if workerCount <= 0 {
		workerCount = 1
	}

	for i := 1; i <= workerCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			// Worker loop: keep processing tests until channel is closed
			for testPath := range testQueue {
				result := wp.runner.Run(testPath, workerID)
				results <- result

				mu.Lock()
				completedFiles++
				if wp.parser != nil {
					p, f := wp.parser.ParseTestCounts(result)
					passedCases += p
					failedCases += f
				} else {
					if result.Success {
						passedCases++
					} else {
						failedCases++
					}
				}
				// Update progress bar if available (bar position = files, label = test cases)
				if wp.progress != nil {
					wp.progress.Update(completedFiles, passedCases, failedCases)
				}
				mu.Unlock()
			}
		}(i)
	}

	// Close results channel when all workers are done
	go func() {
		wg.Wait()
		close(results)
	}()

	var allResults []domain.TestResult
	for result := range results {
		allResults = append(allResults, result)
	}

	// Finish progress bar if available
	if wp.progress != nil {
		wp.progress.Finish()
	}

	duration := time.Since(startTime)

	return allResults, duration, nil
}

