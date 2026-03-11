package commands

import (
	"ptp/internal/config"
	"ptp/internal/debug"
	"ptp/internal/migration"

	"github.com/spf13/cobra"
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
	debug.Logf("migrate: starting (workers=%d, fresh=%v)", workerCount, mc.config.Flags.Fresh)
	if err := mc.migrator.Run(workerCount, mc.config.Flags.Fresh); err != nil {
		debug.Logf("migrate: failed: %v", err)
		return err
	}
	debug.Log("migrate: completed successfully")
	return nil
}

