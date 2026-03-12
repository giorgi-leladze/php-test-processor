package commands

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"ptp/internal/config"
	"ptp/internal/debug"
	"ptp/internal/discovery"
	"ptp/internal/domain"
	"ptp/internal/execution"
	"ptp/internal/migration"
	"ptp/internal/parser"
	"ptp/internal/storage"
	"ptp/internal/ui"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// normalizedPathForKey returns a path key for matching (slash, no .php, relative to project when possible).
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

// RunCommand handles the run command
type RunCommand struct {
	config    *config.Config
	scanner   *discovery.Scanner
	filter    *discovery.Filter
	executor  *execution.WorkerPool
	parser    *parser.PHPUnitParser
	storage   storage.Storage
	formatter *ui.Formatter
	migrator  migration.Migrator
	viewer    *ui.ErrorViewer
}

// NewRunCommand creates a new RunCommand
func NewRunCommand(
	cfg *config.Config,
	scanner *discovery.Scanner,
	filter *discovery.Filter,
	executor *execution.WorkerPool,
	parser *parser.PHPUnitParser,
	st storage.Storage,
	formatter *ui.Formatter,
	migrator migration.Migrator,
	viewer *ui.ErrorViewer,
) *RunCommand {
	return &RunCommand{
		config:    cfg,
		scanner:   scanner,
		filter:    filter,
		executor:  executor,
		parser:    parser,
		storage:   st,
		formatter: formatter,
		migrator:  migrator,
		viewer:    viewer,
	}
}

// failedPathsFromOutput returns a set of normalized paths that had failures.
func failedPathsFromOutput(projectPath string, out *domain.TestResultsOutput) map[string]struct{} {
	set := make(map[string]struct{})
	for _, d := range out.Details {
		key := normalizedPathForKey(projectPath, d.FilePath)
		set[key] = struct{}{}
	}
	return set
}

// failedPathsFromFailures returns a set of normalized paths from a failure list.
func failedPathsFromFailures(projectPath string, failures []domain.TestFailure) map[string]struct{} {
	set := make(map[string]struct{})
	for _, f := range failures {
		key := normalizedPathForKey(projectPath, f.FilePath)
		set[key] = struct{}{}
	}
	return set
}

// sortTestsByTimings sorts tests with the slowest (highest avg) first so they get
// dispatched early and don't become a tail bottleneck in the worker pool.
func sortTestsByTimings(tests []string, timings map[string]*domain.TestTiming) {
	if len(timings) == 0 {
		return
	}
	sort.Slice(tests, func(i, j int) bool {
		ai, bi := timings[tests[i]], timings[tests[j]]
		var avgI, avgJ float64
		if ai != nil {
			avgI = ai.Avg
		}
		if bi != nil {
			avgJ = bi.Avg
		}
		return avgI > avgJ
	})
}

// filterTestsToFailed returns only tests whose normalized path is in the failed set.
func filterTestsToFailed(projectPath string, tests []string, failedSet map[string]struct{}) []string {
	var out []string
	for _, t := range tests {
		key := normalizedPathForKey(projectPath, t)
		if _, ok := failedSet[key]; ok {
			out = append(out, t)
		}
	}
	return out
}

// RunOnlyFailedAndSave runs only the test files that failed in the last run, saves results, and returns the new output.
// Used by the faills TUI "R" key. Returns (nil, nil) if no previous run or no failures to run.
func (rc *RunCommand) RunOnlyFailedAndSave() (*domain.TestResultsOutput, error) {
	last, err := rc.storage.Load()
	if err != nil || last == nil || len(last.Details) == 0 {
		return nil, nil
	}
	failedSet := failedPathsFromOutput(rc.config.ProjectPath, last)
	testPath := rc.config.GetTestPath()
	discovered, err := rc.scanner.Scan(testPath)
	if err != nil {
		return nil, err
	}
	discovered = rc.filter.FilterByName(discovered, rc.config.Flags.NameFilter)
	tests := filterTestsToFailed(rc.config.ProjectPath, discovered, failedSet)
	if len(tests) == 0 {
		return nil, nil
	}
	sortTestsByTimings(tests, rc.storage.LoadTimings())
	rc.executor.SetProgress(nil)
	results, duration, err := rc.executor.ExecuteWithOptions(tests, rc.config.Flags.FailFast)
	if err != nil {
		return nil, err
	}
	var failures []domain.TestFailure
	for _, result := range results {
		if !result.Success {
			failures = append(failures, rc.parser.ParseFailure(result)...)
		}
	}
	if err := rc.storage.Save(results, failures, duration, rc.config.Processors); err != nil {
		return nil, err
	}
	return rc.storage.Load()
}

// Execute runs the command
func (rc *RunCommand) Execute(cmd *cobra.Command, args []string) error {
	debug.Logf("run: starting (processors=%d, testPath=%q, failFast=%v, onlyFailed=%v, rerunFailures=%v, skipMigrate=%v, fresh=%v)",
		rc.config.Processors, rc.config.GetTestPath(), rc.config.Flags.FailFast, rc.config.Flags.OnlyFailed,
		rc.config.Flags.RerunFailures, rc.config.Flags.SkipMigrate, rc.config.Flags.Fresh)

	if !rc.config.Flags.SkipMigrate {
		debug.Log("run: starting pre-test migrations")
		if err := rc.migrator.Run(rc.config.Processors, rc.config.Flags.Fresh); err != nil {
			debug.Logf("run: migration failed: %v", err)
			return fmt.Errorf("migration failed: %w", err)
		}
		debug.Log("run: migrations completed")
		if !debug.IsEnabled() {
			fmt.Println()
		}
	}

	projectPath := rc.config.ProjectPath
	testPath := rc.config.GetTestPath()
	failFast := rc.config.Flags.FailFast
	onlyFailed := rc.config.Flags.OnlyFailed
	rerunFailures := rc.config.Flags.RerunFailures

	var tests []string
	if onlyFailed {
		debug.Log("run: loading previous results for --failed mode")
		last, err := rc.storage.Load()
		if err != nil || last == nil {
			debug.Logf("run: no previous results (err=%v), falling back to all tests", err)
			color.Yellow("No previous run found (or no storage). Running all tests.")
			onlyFailed = false
		} else {
			failedSet := failedPathsFromOutput(projectPath, last)
			if len(failedSet) == 0 {
				color.Green("No failed tests in last run. Nothing to run.")
				return nil
			}
			discovered, err := rc.scanner.Scan(testPath)
			if err != nil {
				debug.Logf("run: scan failed for --failed mode: %v", err)
				return err
			}
			discovered = rc.filter.FilterByName(discovered, rc.config.Flags.NameFilter)
			tests = filterTestsToFailed(projectPath, discovered, failedSet)
			if len(tests) == 0 {
				debug.Log("run: no matching test files for previous failures")
				color.Yellow("No matching test files for last run's failures. Run all tests? Skipping.")
				return nil
			}
			debug.Logf("run: filtered to %d previously failed tests", len(tests))
		}
	}
	if !onlyFailed {
		debug.Logf("run: scanning for tests in %q", testPath)
		discovered, err := rc.scanner.Scan(testPath)
		if err != nil {
			debug.Logf("run: scan failed: %v", err)
			return err
		}
		tests = rc.filter.FilterByName(discovered, rc.config.Flags.NameFilter)
		debug.Logf("run: discovered %d test files", len(tests))
	}

	if len(tests) == 0 {
		color.Yellow("No tests to execute")
		return nil
	}

	timings := rc.storage.LoadTimings()
	sortTestsByTimings(tests, timings)
	debug.Logf("run: sorted %d tests by historical duration (slowest first)", len(tests))

	if !debug.IsEnabled() {
		testCaseCount, _ := rc.formatter.CountTestCases(tests)
		progressBar := ui.NewProgressBar(len(tests), testCaseCount)
		rc.executor.SetProgress(progressBar)
	}

	debug.Logf("run: executing %d tests (failFast=%v, workers=%d)", len(tests), failFast, rc.config.Processors)
	results, duration, err := rc.executor.ExecuteWithOptions(tests, failFast)
	if err != nil {
		debug.Logf("run: execution error: %v", err)
		return err
	}
	debug.Logf("run: execution finished in %s (%d results)", duration, len(results))

	var failures []domain.TestFailure
	for _, result := range results {
		if !result.Success {
			failures = append(failures, rc.parser.ParseFailure(result)...)
		}
	}

	debug.Logf("run: %d failures found across %d results", len(failures), len(results))

	if rerunFailures && len(failures) > 0 {
		failedSet := failedPathsFromFailures(projectPath, failures)
		rerunTests := filterTestsToFailed(projectPath, tests, failedSet)
		if len(rerunTests) > 0 {
			if !debug.IsEnabled() {
				progressBar2 := ui.NewProgressBar(len(rerunTests), 0)
				rc.executor.SetProgress(progressBar2)
			}
			results2, duration2, err2 := rc.executor.ExecuteWithOptions(rerunTests, failFast)
			if err2 != nil {
				return err2
			}
			var failures2 []domain.TestFailure
			for _, result := range results2 {
				if !result.Success {
					failures2 = append(failures2, rc.parser.ParseFailure(result)...)
				}
			}
			results = results2
			failures = failures2
			duration = duration2
			if err := rc.storage.Save(results, failures, duration, rc.config.Processors); err != nil {
				return fmt.Errorf("failed to save test results: %w", err)
			}
			if err := rc.formatter.PrintMetaStats(); err != nil {
				return err
			}
			if rc.config.Flags.OpenFaills && len(failures) > 0 && rc.viewer != nil {
				passed, failed := 0, 0
				for _, r := range results {
					if r.Success {
						passed++
					} else {
						failed++
					}
				}
				output := &domain.TestResultsOutput{
					Meta: domain.TestResultsMeta{
						TotalTestFiles:  len(results),
						FailedTestFiles: failed,
						PassedTestFiles: passed,
						FailedTestCases: len(failures),
						Duration:        duration.String(),
						DurationSeconds: duration.Seconds(),
						Workers:         rc.config.Processors,
						Timestamp:       time.Now().Format(time.RFC3339),
					},
					Details: failures,
				}
				if err := rc.viewer.View(output); err != nil {
					return err
				}
			}
			if len(failures) > 0 {
				return fmt.Errorf("%d test case(s) failed", len(failures))
			}
			return nil
		}
	}

	if err := rc.storage.Save(results, failures, duration, rc.config.Processors); err != nil {
		debug.Logf("run: failed to save results: %v", err)
		return fmt.Errorf("failed to save test results: %w", err)
	}
	debug.Log("run: results saved")
	if err := rc.formatter.PrintMetaStats(); err != nil {
		debug.Logf("run: failed to print stats: %v", err)
		return err
	}
	if rc.config.Flags.OpenFaills && len(failures) > 0 && rc.viewer != nil {
		passed, failed := 0, 0
		for _, r := range results {
			if r.Success {
				passed++
			} else {
				failed++
			}
		}
		output := &domain.TestResultsOutput{
			Meta: domain.TestResultsMeta{
				TotalTestFiles:  len(results),
				FailedTestFiles: failed,
				PassedTestFiles: passed,
				FailedTestCases: len(failures),
				Duration:        duration.String(),
				DurationSeconds: duration.Seconds(),
				Workers:         rc.config.Processors,
				Timestamp:       time.Now().Format(time.RFC3339),
			},
			Details: failures,
		}
		if err := rc.viewer.View(output); err != nil {
			return err
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("%d test case(s) failed", len(failures))
	}
	return nil
}
