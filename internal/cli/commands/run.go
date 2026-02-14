package commands

import (
	"fmt"
	"path/filepath"
	"strings"

	"ptp/internal/config"
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
	// Run migrations if flag is set
	if rc.config.Flags.Migrate {
		if err := rc.migrator.Run(rc.config.Processors, rc.config.Flags.NoFresh); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
		fmt.Println()
	}

	projectPath := rc.config.ProjectPath
	testPath := rc.config.GetTestPath()
	failFast := rc.config.Flags.FailFast
	onlyFailed := rc.config.Flags.OnlyFailed
	rerunFailures := rc.config.Flags.RerunFailures

	var tests []string
	if onlyFailed {
		last, err := rc.storage.Load()
		if err != nil || last == nil {
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
				return err
			}
			discovered = rc.filter.FilterByName(discovered, rc.config.Flags.NameFilter)
			tests = filterTestsToFailed(projectPath, discovered, failedSet)
			if len(tests) == 0 {
				color.Yellow("No matching test files for last run's failures. Run all tests? Skipping.")
				return nil
			}
		}
	}
	if !onlyFailed {
		discovered, err := rc.scanner.Scan(testPath)
		if err != nil {
			return err
		}
		tests = rc.filter.FilterByName(discovered, rc.config.Flags.NameFilter)
	}

	if len(tests) == 0 {
		color.Yellow("No tests to execute")
		return nil
	}

	testCaseCount, _ := rc.formatter.CountTestCases(tests)
	progressBar := ui.NewProgressBar(len(tests), testCaseCount)
	rc.executor.SetProgress(progressBar)

	results, duration, err := rc.executor.ExecuteWithOptions(tests, failFast)
	if err != nil {
		return err
	}

	var failures []domain.TestFailure
	for _, result := range results {
		if !result.Success {
			failures = append(failures, rc.parser.ParseFailure(result)...)
		}
	}

	if rerunFailures && len(failures) > 0 {
		failedSet := failedPathsFromFailures(projectPath, failures)
		rerunTests := filterTestsToFailed(projectPath, tests, failedSet)
		if len(rerunTests) > 0 {
			progressBar2 := ui.NewProgressBar(len(rerunTests), 0)
			rc.executor.SetProgress(progressBar2)
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
			return rc.formatter.PrintMetaStats()
		}
	}

	if err := rc.storage.Save(results, failures, duration, rc.config.Processors); err != nil {
		return fmt.Errorf("failed to save test results: %w", err)
	}
	return rc.formatter.PrintMetaStats()
}
