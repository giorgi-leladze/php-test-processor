package commands

import (
	"ptp/internal/config"
	"ptp/internal/debug"
	"ptp/internal/storage"
	"ptp/internal/ui"

	"github.com/spf13/cobra"
)

// FaillsCommand handles the faills command
type FaillsCommand struct {
	config  *config.Config
	storage storage.Storage
	viewer  *ui.ErrorViewer
}

// NewFaillsCommand creates a new FaillsCommand
func NewFaillsCommand(cfg *config.Config, st storage.Storage, viewer *ui.ErrorViewer) *FaillsCommand {
	return &FaillsCommand{
		config:  cfg,
		storage:  st,
		viewer:   viewer,
	}
}

// Execute runs the command
func (fc *FaillsCommand) Execute(cmd *cobra.Command, args []string) error {
	debug.Log("faills: loading results from storage")
	results, err := fc.storage.Load()
	if err != nil {
		debug.Logf("faills: failed to load results: %v", err)
		return err
	}
	debug.Logf("faills: loaded %d failure details", len(results.Details))
	return fc.viewer.View(results)
}

