package main

import (
	"fmt"
	"os"

	"ptp/internal/cli"
	"ptp/internal/cli/commands"
	"ptp/internal/config"

	"github.com/spf13/cobra"
)

var version = "dev"

func main() {
	// Create root command
	rootCmd := &cobra.Command{
		Use:     "ptp",
		Short:   "Parallel PHPUnit test processor",
		Long:    `A high-performance parallel test processor for PHPUnit tests. Execute PHP unit and integration tests in parallel to significantly reduce test execution time.`,
		Version: version,
	}

	// Create initial config with defaults
	cfg := config.New()

	// Create flags struct (will be populated by command flags)
	var flags cli.Flags

	// Create commands with dependencies
	cmds := commands.NewCommands(cfg)

	// Register all commands
	cmds.Register(rootCmd, &flags, cfg)

	// Execute root command
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
