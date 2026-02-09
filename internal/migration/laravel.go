package migration

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/schollz/progressbar/v3"
	"ptp/internal/config"
	"ptp/internal/domain"
)

// LaravelMigrator implements Migrator for Laravel migrations
type LaravelMigrator struct {
	config          *config.Config
	databaseManager *DatabaseManager
}

// NewLaravelMigrator creates a new LaravelMigrator
func NewLaravelMigrator(cfg *config.Config, dbManager *DatabaseManager) *LaravelMigrator {
	return &LaravelMigrator{
		config:          cfg,
		databaseManager: dbManager,
	}
}

// Run executes migrations in parallel for all workers
func (lm *LaravelMigrator) Run(workerCount int, noFresh bool) error {
	color.Cyan("\n╔════════════════════════════════════════════════════════════╗")
	color.Cyan("║               Running Database Migrations                  ║")
	color.Cyan("╚════════════════════════════════════════════════════════════╝\n")

	// Check available databases
	availableWorkers, err := lm.databaseManager.CheckAndCreateDatabases(workerCount)
	if err != nil {
		return fmt.Errorf("failed to check databases: %w", err)
	}

	if len(availableWorkers) == 0 {
		return fmt.Errorf("no test databases available")
	}

	// Count migration files to determine total progress
	migrationFiles, err := lm.findMigrationFiles()
	if err != nil {
		return fmt.Errorf("failed to find migration files: %w", err)
	}

	migrationCount := len(migrationFiles)
	totalProgress := len(availableWorkers) * migrationCount

	color.White("Workers: %d | Migration files: %d | Total progress: %d\n\n", len(availableWorkers), migrationCount, totalProgress)

	// Create progress bar
	var progressMu sync.Mutex
	completedCount := 0

	bar := progressbar.NewOptions(totalProgress,
		progressbar.OptionSetDescription(
			color.CyanString("Migrating: ")+
				color.GreenString("[completed: 0/%d]", totalProgress),
		),
		progressbar.OptionSetWidth(50),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        color.CyanString("█"),
			SaucerHead:    color.CyanString("█"),
			SaucerPadding: "░",
			BarStart:      "│",
			BarEnd:        "│",
		}),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionOnCompletion(func() {
			fmt.Fprint(os.Stderr, "\n")
		}),
		progressbar.OptionSetRenderBlankState(true),
	)

	// Start workers
	var wg sync.WaitGroup
	results := make(chan domain.MigrationResult, len(availableWorkers))
	startTime := time.Now()

	for _, workerID := range availableWorkers {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			result := lm.runMigrationForWorker(id, bar, &completedCount, &progressMu, noFresh)
			results <- result
		}(workerID)
	}

	// Close results channel when all workers are done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var failedMigrations []domain.MigrationResult
	for result := range results {
		if !result.Success {
			failedMigrations = append(failedMigrations, result)
		}
	}

	// Finish progress bar
	bar.Finish()

	duration := time.Since(startTime)

	// Print summary
	fmt.Print("\n")
	if len(failedMigrations) == 0 {
		color.Green("✓ Migrations completed successfully for all %d workers\n", len(availableWorkers))
		color.White("Duration: %s\n", duration.Round(time.Millisecond))
	} else {
		color.Red("✗ Migration failed for %d worker(s)\n", len(failedMigrations))
		for _, result := range failedMigrations {
			color.Red("  Worker %d (DB: %s): %v\n", result.WorkerID, lm.config.GetDatabaseName(result.WorkerID), result.Error)
		}
		return fmt.Errorf("migration failed for %d worker(s)", len(failedMigrations))
	}

	return nil
}

// findMigrationFiles discovers all migration files in database/migrations
func (lm *LaravelMigrator) findMigrationFiles() ([]string, error) {
	migrationsPath := filepath.Join(lm.config.ProjectPath, "database", "migrations")
	var migrationFiles []string

	err := filepath.WalkDir(migrationsPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		// Laravel migration files end with .php
		if strings.HasSuffix(d.Name(), ".php") {
			migrationFiles = append(migrationFiles, path)
		}

		return nil
	})

	return migrationFiles, err
}

// runMigrationForWorker executes migrate or migrate:fresh with streaming output and progress tracking
func (lm *LaravelMigrator) runMigrationForWorker(workerID int, bar *progressbar.ProgressBar, completedCount *int, progressMu *sync.Mutex, noFresh bool) domain.MigrationResult {
	// Get absolute path of project directory
	projectAbsPath, err := filepath.Abs(lm.config.ProjectPath)
	if err != nil {
		return domain.MigrationResult{
			WorkerID: workerID,
			Success:  false,
			Output:   "",
			Error:    fmt.Errorf("failed to get absolute project path: %w", err),
		}
	}

	artisanPath := filepath.Join(projectAbsPath, "artisan")
	ctx := context.Background()

	// Use migrate or migrate:fresh based on noFresh flag
	migrateCmd := "migrate:fresh"
	if noFresh {
		migrateCmd = "migrate"
	}

	cmd := exec.CommandContext(ctx, "php", artisanPath, migrateCmd, "--env=testing", "--force")

	// Set environment variables
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("DB_DATABASE=%s", lm.config.GetDatabaseName(workerID)))

	// Set working directory
	cmd.Dir = projectAbsPath

	// Get stdout and stderr pipes for streaming
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return domain.MigrationResult{
			WorkerID: workerID,
			Success:  false,
			Output:   "",
			Error:    fmt.Errorf("failed to create stdout pipe: %w", err),
		}
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return domain.MigrationResult{
			WorkerID: workerID,
			Success:  false,
			Output:   "",
			Error:    fmt.Errorf("failed to create stderr pipe: %w", err),
		}
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return domain.MigrationResult{
			WorkerID: workerID,
			Success:  false,
			Output:   "",
			Error:    fmt.Errorf("failed to start command: %w", err),
		}
	}

	var outputBuilder strings.Builder
	var scanWg sync.WaitGroup

	// Helper function to process a line and update progress
	processLine := func(line string) {
		line = strings.TrimSpace(line)
		if line == "" {
			return
		}

		// Skip common Laravel messages that aren't migration progress
		skipPatterns := []string{"Dropping all tables", "Dropped all tables", "Nothing to migrate", "Migration table created"}
		for _, skip := range skipPatterns {
			if strings.Contains(line, skip) {
				return
			}
		}

		// Count this line as a migration progress line
		progressMu.Lock()
		*completedCount++
		currentCount := *completedCount
		maxCount := bar.GetMax()
		progressMu.Unlock()

		// Update progress bar using Set() for absolute value
		bar.Set(currentCount)
		bar.Describe(color.CyanString("Migrating: ") +
			color.GreenString("[completed: %d/%d]", currentCount, maxCount))
	}

	// Stream stdout
	scanWg.Add(1)
	go func() {
		defer scanWg.Done()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			outputBuilder.WriteString(line)
			outputBuilder.WriteString("\n")
			processLine(line)
		}
	}()

	// Stream stderr
	scanWg.Add(1)
	go func() {
		defer scanWg.Done()
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			outputBuilder.WriteString(line)
			outputBuilder.WriteString("\n")
			processLine(line)
		}
	}()

	// Wait for command to complete
	err = cmd.Wait()

	// Wait for all scanners to finish
	scanWg.Wait()

	output := outputBuilder.String()

	return domain.MigrationResult{
		WorkerID: workerID,
		Success:  err == nil,
		Output:   output,
		Error:    err,
	}
}

