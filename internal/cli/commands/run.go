package commands

import (
	"fmt"

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

// Execute runs the command
func (rc *RunCommand) Execute(cmd *cobra.Command, args []string) error {
	// Run migrations if flag is set
	if rc.config.Flags.Migrate {
		if err := rc.migrator.Run(rc.config.Processors, rc.config.Flags.NoFresh); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
		fmt.Println()
	}

	// Discover tests
	testPath := rc.config.GetTestPath()
	tests, err := rc.scanner.Scan(testPath)
	if err != nil {
		return err
	}

	// Filter tests
	tests = rc.filter.FilterByName(tests, rc.config.Flags.NameFilter)

	if len(tests) == 0 {
		color.Yellow("No tests to execute")
		return nil
	}

	// Create and set progress bar
	progressBar := ui.NewProgressBar(len(tests))
	rc.executor.SetProgress(progressBar)

	// Execute tests
	results, duration, err := rc.executor.Execute(tests)
	if err != nil {
		return err
	}

	// Parse failures
	var failures []domain.TestFailure
	for _, result := range results {
		if !result.Success {
			failures = append(failures, rc.parser.ParseFailure(result)...)
		}
	}

	// Save results
	if err := rc.storage.Save(results, failures, duration, rc.config.Processors); err != nil {
		return fmt.Errorf("failed to save test results: %w", err)
	}

	// Print stats
	return rc.formatter.PrintMetaStats()
}
