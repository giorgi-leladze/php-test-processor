package commands

import (
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"ptp/internal/config"
	"ptp/internal/discovery"
	"ptp/internal/storage"
	"ptp/internal/ui"
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
	tests, err := lc.scanner.Scan(testPath)
	if err != nil {
		return err
	}

	// Filter tests
	tests = lc.filter.FilterByName(tests, lc.config.Flags.NameFilter)

	if len(tests) == 0 {
		color.Yellow("No tests found")
		return nil
	}

	var failedPaths map[string]struct{}
	if last, err := lc.storage.Load(); err == nil && last != nil && len(last.Details) > 0 {
		failedPaths = failedPathsFromOutput(lc.config.ProjectPath, last)
	}

	return lc.formatter.PrintTestList(tests, lc.config.Flags.TestCases, failedPaths)
}

