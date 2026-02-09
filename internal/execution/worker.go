package execution

import (
	"sync"
	"time"

	"ptp/internal/config"
	"ptp/internal/domain"
	"ptp/internal/ui"
)

// WorkerPool manages a pool of workers for parallel test execution
type WorkerPool struct {
	config   *config.Config
	runner   *Runner
	scheduler Scheduler
	progress *ui.ProgressBar
}

// NewWorkerPool creates a new WorkerPool
func NewWorkerPool(cfg *config.Config, runner *Runner, scheduler Scheduler) *WorkerPool {
	return &WorkerPool{
		config:    cfg,
		runner:    runner,
		scheduler: scheduler,
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

	// Track progress
	var mu sync.Mutex
	var completedCount int
	var successCount, failCount int

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
				completedCount++
				if result.Success {
					successCount++
				} else {
					failCount++
				}
				// Update progress bar if available
				if wp.progress != nil {
					wp.progress.Update(successCount, failCount)
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

