package commands

import (
	"github.com/spf13/cobra"
	"ptp/internal/config"
	"ptp/internal/migration"
)

// MigrateCommand handles the migrate command
type MigrateCommand struct {
	config   *config.Config
	migrator migration.Migrator
}

// NewMigrateCommand creates a new MigrateCommand
func NewMigrateCommand(cfg *config.Config, migrator migration.Migrator) *MigrateCommand {
	return &MigrateCommand{
		config:   cfg,
		migrator: migrator,
	}
}

// Execute runs the command
func (mc *MigrateCommand) Execute(cmd *cobra.Command, args []string) error {
	workerCount := mc.config.Processors
	return mc.migrator.Run(workerCount, mc.config.Flags.NoFresh)
}

