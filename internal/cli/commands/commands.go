package commands

import (
	"ptp/internal/cli"
	"ptp/internal/config"
	"ptp/internal/discovery"
	"ptp/internal/execution"
	"ptp/internal/migration"
	"ptp/internal/parser"
	"ptp/internal/storage"
	"ptp/internal/ui"

	"github.com/spf13/cobra"
)

// Commands holds all CLI commands
type Commands struct {
	Run     *RunCommand
	List    *ListCommand
	Migrate *MigrateCommand
	Faills  *FaillsCommand
}

// NewCommands creates all commands with dependencies
func NewCommands(cfg *config.Config) *Commands {
	// Initialize dependencies
	scanner := discovery.NewScanner(cfg.PathsToIgnore)
	filter := discovery.NewFilter()
	testCaseParser := discovery.NewParser()
	runner := execution.NewRunner(cfg)
	scheduler := execution.NewRoundRobinScheduler()
	phpunitParser := parser.NewPHPUnitParser()
	executor := execution.NewWorkerPool(cfg, runner, scheduler, phpunitParser)
	jsonStorage := storage.NewJSONStorage(cfg)
	formatter := ui.NewFormatter(cfg, testCaseParser)
	dbManager := migration.NewDatabaseManager(cfg)
	migrator := migration.NewLaravelMigrator(cfg, dbManager)
	errorViewer := ui.NewErrorViewer(cfg, jsonStorage, runner, phpunitParser)

	return &Commands{
		Run:     NewRunCommand(cfg, scanner, filter, executor, phpunitParser, jsonStorage, formatter, migrator, errorViewer),
		List:    NewListCommand(cfg, scanner, filter, formatter, jsonStorage),
		Migrate: NewMigrateCommand(cfg, migrator),
		Faills:  NewFaillsCommand(cfg, jsonStorage, errorViewer),
	}
}

// Register registers all commands with cobra
func (c *Commands) Register(rootCmd *cobra.Command, flags *cli.Flags, cfg *config.Config) {
	// Run command
	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run PHPUnit tests in parallel",
		Long:  "Discover and execute PHPUnit tests using parallel workers",
		RunE:  c.Run.Execute,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// Update config with flags after parsing
			cfg.Flags = flags.ToConfigFlags()
			if flags.Processors > 0 {
				cfg.Processors = flags.Processors
			}
			return nil
		},
	}
	runCmd.Flags().IntVarP(&flags.Processors, "processors", "p", 4, "Number of processors to use")
	runCmd.Flags().BoolVarP(&flags.Migrate, "migrate", "m", false, "Run migrations before executing tests")
	runCmd.Flags().BoolVar(&flags.NoFresh, "no-fresh", false, "Run migrations without fresh (only pending migrations)")
	runCmd.Flags().StringVarP(&flags.TestPath, "test-path", "t", "", "Path to the folder where test detection should start")
	runCmd.Flags().StringVarP(&flags.NameFilter, "filter", "f", "", "Filter tests by name pattern (supports wildcards, e.g., '*UserTest.php' or '*Payment*')")
	runCmd.Flags().BoolVar(&flags.FailFast, "fail-fast", false, "Stop on first test failure")
	runCmd.Flags().BoolVar(&flags.OnlyFailed, "failed", false, "Run only tests that failed in the last run (from storage/test-results.json)")
	runCmd.Flags().BoolVar(&flags.RerunFailures, "rerun-failures", false, "After running all tests, rerun only failed ones once and save that result")
	runCmd.Flags().BoolVar(&flags.OpenFaills, "open-faills", false, "Open the faills viewer when the run finishes with failures")
	rootCmd.AddCommand(runCmd)

	// List command
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List discovered tests",
		Long:  "Scan and list all PHPUnit tests without executing them",
		RunE:  c.List.Execute,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			cfg.Flags = flags.ToConfigFlags()
			return nil
		},
	}
	listCmd.Flags().StringVarP(&flags.NameFilter, "filter", "f", "", "Filter tests by name pattern (supports wildcards, e.g., '*UserTest.php' or '*Payment*')")
	listCmd.Flags().StringVarP(&flags.TestPath, "test-path", "t", "", "Path to the folder where test detection should start")
	listCmd.Flags().BoolVarP(&flags.TestCases, "test-cases", "c", false, "List test cases instead of test files")
	rootCmd.AddCommand(listCmd)

	// Migrate command
	migrateCmd := &cobra.Command{
		Use:   "migrate",
		Short: "Run database migrations for all test databases",
		Long:  "Execute migrations in parallel for all test databases used by workers",
		RunE:  c.Migrate.Execute,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			cfg.Flags = flags.ToConfigFlags()
			if flags.Processors > 0 {
				cfg.Processors = flags.Processors
			}
			return nil
		},
	}
	migrateCmd.Flags().IntVarP(&flags.Processors, "processors", "p", 4, "Number of processors/workers to use")
	migrateCmd.Flags().BoolVar(&flags.NoFresh, "no-fresh", false, "Run migrations without fresh (only pending migrations)")
	rootCmd.AddCommand(migrateCmd)

	// Faills command
	faillsCmd := &cobra.Command{
		Use:   "faills",
		Short: "View test failures interactively",
		Long:  "Display test failures from the last test run in an interactive viewer",
		RunE:  c.Faills.Execute,
	}
	rootCmd.AddCommand(faillsCmd)
}
