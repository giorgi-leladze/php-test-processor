package commands

import (
	"ptp/internal/config"
	"ptp/internal/debug"
	"ptp/internal/discovery"
	"ptp/internal/storage"
	"ptp/internal/ui"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// ListCommand handles the list command
type ListCommand struct {
	config    *config.Config
	scanner   *discovery.Scanner
	filter    *discovery.Filter
	formatter *ui.Formatter
	storage   storage.Storage
}

// NewListCommand creates a new ListCommand
func NewListCommand(
	cfg *config.Config,
	scanner *discovery.Scanner,
	filter *discovery.Filter,
	formatter *ui.Formatter,
	st storage.Storage,
) *ListCommand {
	return &ListCommand{
		config:    cfg,
		scanner:   scanner,
		filter:    filter,
		formatter: formatter,
		storage:   st,
	}
}

// Execute runs the command
func (lc *ListCommand) Execute(cmd *cobra.Command, args []string) error {
	testPath := lc.config.GetTestPath()
	debug.Logf("list: scanning for tests in %q", testPath)
	tests, err := lc.scanner.Scan(testPath)
	if err != nil {
		debug.Logf("list: scan failed: %v", err)
		return err
	}

	tests = lc.filter.FilterByName(tests, lc.config.Flags.NameFilter)
	debug.Logf("list: found %d tests after filtering", len(tests))

	if len(tests) == 0 {
		color.Yellow("No tests found")
		return nil
	}

	var failedPaths map[string]struct{}
	if last, err := lc.storage.Load(); err == nil && last != nil && len(last.Details) > 0 {
		failedPaths = failedPathsFromOutput(lc.config.ProjectPath, last)
	} else if err != nil {
		debug.Logf("list: could not load previous results: %v", err)
	}

	return lc.formatter.PrintTestList(tests, lc.config.Flags.TestCases, failedPaths)
}

