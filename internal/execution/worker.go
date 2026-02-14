package execution

import (
	"context"
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

// Execute executes tests in parallel using worker pool (no fail-fast).
func (wp *WorkerPool) Execute(tests []string) ([]domain.TestResult, time.Duration, error) {
	return wp.ExecuteWithOptions(tests, false)
}

// ExecuteWithOptions executes tests with optional fail-fast (stop on first failure).
func (wp *WorkerPool) ExecuteWithOptions(tests []string, failFast bool) ([]domain.TestResult, time.Duration, error) {
	if len(tests) == 0 {
		return nil, 0, nil
	}
	if !failFast {
		return wp.executeAll(tests)
	}
	return wp.executeFailFast(tests)
}

// executeAll runs all tests (original behavior).
func (wp *WorkerPool) executeAll(tests []string) ([]domain.TestResult, time.Duration, error) {
	testQueue := make(chan string, len(tests))
	results := make(chan domain.TestResult, len(tests))
	for _, test := range tests {
		testQueue <- test
	}
	close(testQueue)

	var mu sync.Mutex
	var completedFiles int
	var passedCases, failedCases int
	startTime := time.Now()
	workerCount := wp.config.Processors
	if workerCount <= 0 {
		workerCount = 1
	}

	var wg sync.WaitGroup
	for i := 1; i <= workerCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
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
				if wp.progress != nil {
					wp.progress.Update(completedFiles, passedCases, failedCases)
				}
				mu.Unlock()
			}
		}(i)
	}
	go func() {
		wg.Wait()
		close(results)
	}()

	var allResults []domain.TestResult
	for result := range results {
		allResults = append(allResults, result)
	}
	if wp.progress != nil {
		wp.progress.Finish()
	}
	return allResults, time.Since(startTime), nil
}

// executeFailFast runs tests and stops after the first failure.
func (wp *WorkerPool) executeFailFast(tests []string) ([]domain.TestResult, time.Duration, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	testQueue := make(chan string, 1)
	results := make(chan domain.TestResult, len(tests))

	go func() {
		defer close(testQueue)
		for _, test := range tests {
			select {
			case <-ctx.Done():
				return
			case testQueue <- test:
			}
		}
	}()

	var mu sync.Mutex
	var completedFiles int
	var passedCases, failedCases int
	var seenFailure bool
	startTime := time.Now()
	workerCount := wp.config.Processors
	if workerCount <= 0 {
		workerCount = 1
	}

	var wg sync.WaitGroup
	for i := 1; i <= workerCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for testPath := range testQueue {
				result := wp.runner.Run(testPath, workerID)
				mu.Lock()
				done := seenFailure
				mu.Unlock()
				if done {
					continue
				}
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
				if wp.progress != nil {
					wp.progress.Update(completedFiles, passedCases, failedCases)
				}
				if !result.Success {
					seenFailure = true
					cancel()
				}
				mu.Unlock()
			}
		}(i)
	}
	go func() {
		wg.Wait()
		close(results)
	}()

	var allResults []domain.TestResult
	for result := range results {
		allResults = append(allResults, result)
	}
	if wp.progress != nil {
		wp.progress.Finish()
	}
	return allResults, time.Since(startTime), nil
}

