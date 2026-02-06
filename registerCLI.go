package main

import (
	"fmt"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	version = "dev"
	rootCmd = &cobra.Command{
		Use:   "ptp",
		Short: "Parallel PHPUnit test processor",
		Long: `A high-performance parallel test processor for PHPUnit tests.
Execute PHP unit and integration tests in parallel to significantly reduce test execution time.`,
		Version: version,
	}
)

type Flags struct {
	Processors int
	Filter     string
	Migrate    bool
	NoFresh    bool
	TestPath   string
	NameFilter string
	TestCases  bool
}

var GlobalFlags Flags

func init() {
	// Root cmd
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(migrateCmd)
	rootCmd.AddCommand(errorsCmd)

	// Run cmd
	runCmd.Flags().IntVarP(&GlobalFlags.Processors, "processors", "p", 4, "Number of processors to use")
	runCmd.Flags().BoolVarP(&GlobalFlags.Migrate, "migrate", "m", false, "Run migrations before executing tests")
	runCmd.Flags().BoolVar(&GlobalFlags.NoFresh, "no-fresh", false, "Run migrations without fresh (only pending migrations)")
	runCmd.Flags().StringVarP(&GlobalFlags.TestPath, "test-path", "t", "", "Path to the folder where test detection should start")
	runCmd.Flags().StringVarP(&GlobalFlags.NameFilter, "filter", "f", "", "Filter tests by name pattern (supports wildcards, e.g., '*UserTest.php' or '*Payment*')")

	// Migrate cmd
	migrateCmd.Flags().IntVarP(&GlobalFlags.Processors, "processors", "p", 4, "Number of processors/workers to use")
	migrateCmd.Flags().BoolVar(&GlobalFlags.NoFresh, "no-fresh", false, "Run migrations without fresh (only pending migrations)")

	// List cmd
	listCmd.Flags().StringVarP(&GlobalFlags.NameFilter, "filter", "f", "", "Filter tests by name pattern (supports wildcards, e.g., '*UserTest.php' or '*Payment*')")
	listCmd.Flags().StringVarP(&GlobalFlags.TestPath, "test-path", "t", "", "Path to the folder where test detection should start")
	listCmd.Flags().BoolVarP(&GlobalFlags.TestCases, "test-cases", "c", false, "List test cases instead of test files")
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run PHPUnit tests in parallel",
	Long:  "Discover and execute PHPUnit tests using parallel workers",
	RunE:  runTests,
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List discovered tests",
	Long:  "Scan and list all PHPUnit tests without executing them",
	RunE:  listTests,
}

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run database migrations for all test databases",
	Long:  "Execute migrations in parallel for all test databases used by workers",
	RunE:  runMigrationsCommand,
}

// runTests executes the test run command
func runTests(cmd *cobra.Command, args []string) error {
	// Run migrations if flag is set
	if GlobalFlags.Migrate {
		if err := runMigrations(GlobalFlags.Processors, GlobalFlags.NoFresh); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
		fmt.Println()
	}

	testPath := getTestPath()
	tests, err := findTestFiles(testPath, GlobalFlags.NameFilter)
	if err != nil {
		return err
	}

	if len(tests) == 0 {
		color.Yellow("No tests to execute")
		return nil
	}

	results, duration, err := executeTests(tests)
	if err != nil {
		return err
	}

	fmt.Println("Duration: ", duration.Round(time.Millisecond))

	var failures []TestFailure
	for _, result := range results {
		if !result.Success {
			parseTestFailure(result, &failures)
		}
	}

	if err := saveTestResultsToJSON(results, failures, duration); err != nil {
		return fmt.Errorf("failed to save test failures to JSON: %w", err)
	}

	return printMetaStats()
	// return outputTextResults(results, len(tests), duration)
}

// listTests lists all discovered tests
func listTests(cmd *cobra.Command, args []string) error {
	testPath := getTestPath()
	tests, err := findTestFiles(testPath, GlobalFlags.NameFilter)
	if err != nil {
		return err
	}

	if len(tests) == 0 {
		color.Yellow("No tests found")
		return nil
	}

	printTestCasesFormater(tests)

	return nil
}

// runMigrationsCommand executes the migrate command
func runMigrationsCommand(cmd *cobra.Command, args []string) error {
	workerCount := GlobalFlags.Processors

	return runMigrations(workerCount, GlobalFlags.NoFresh)
}
