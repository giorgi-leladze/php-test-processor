package commands

import (
	"github.com/spf13/cobra"
	"ptp/internal/config"
	"ptp/internal/storage"
	"ptp/internal/ui"
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
	results, err := fc.storage.Load()
	if err != nil {
		return err
	}

	return fc.viewer.View(results)
}

