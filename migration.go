package main

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"github.com/schollz/progressbar/v3"
)

// MigrationResult represents the result of a migration execution
type MigrationResult struct {
	WorkerID int
	Success  bool
	Output   string
	Error    error
}

// checkTestDatabases checks if test databases exist and creates them if they don't
func checkTestDatabases(workerCount int) ([]int, error) {
	// Load .env file from project directory
	envPath := filepath.Join(PROJECT_PATH, ".env")
	if err := godotenv.Load(envPath); err != nil {
		// .env file might not exist, that's okay - use environment variables
		_ = err
	}

	// Get database connection info from environment or use defaults
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "127.0.0.1"
	}
	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		dbPort = "3306"
	}
	dbUser := os.Getenv("DB_USERNAME")
	if dbUser == "" {
		dbUser = "root"
	}
	dbPassword := os.Getenv("DB_PASSWORD")
	if dbPassword == "" {
		dbPassword = ""
	}

	// Connect to MySQL server (without specifying database)
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/", dbUser, dbPassword, dbHost, dbPort)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database server: %w", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database server: %w", err)
	}

	availableWorkers := make([]int, 0, workerCount)
	var createdCount int

	color.White("Checking test databases...\n")

	for i := 1; i <= workerCount; i++ {
		dbName := fmt.Sprintf("webiz_testing_%d", i)

		// Check if database exists
		exists, err := databaseExists(db, dbName)
		if err != nil {
			color.Yellow("Warning: Failed to check database %s: %v\n", dbName, err)
			continue
		}

		if !exists {
			// Create database
			color.Yellow("Creating database: %s\n", dbName)
			if err := createDatabase(db, dbName); err != nil {
				color.Red("Failed to create database %s: %v\n", dbName, err)
				continue
			}
			createdCount++
			color.Green("✓ Created database: %s\n", dbName)
		} else {
			color.Green("✓ Database exists: %s\n", dbName)
		}

		availableWorkers = append(availableWorkers, i)
	}

	if createdCount > 0 {
		fmt.Printf("\nCreated %d new database(s)\n\n", createdCount)
	}

	return availableWorkers, nil
}

// databaseExists checks if a database exists
func databaseExists(db *sql.DB, dbName string) (bool, error) {
	var exists bool
	query := "SELECT EXISTS(SELECT SCHEMA_NAME FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME = ?)"
	err := db.QueryRow(query, dbName).Scan(&exists)
	return exists, err
}

// createDatabase creates a new database
func createDatabase(db *sql.DB, dbName string) error {
	// Sanitize database name to prevent SQL injection
	if !isValidDatabaseName(dbName) {
		return fmt.Errorf("invalid database name: %s", dbName)
	}

	query := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`", dbName)
	_, err := db.Exec(query)
	return err
}

// isValidDatabaseName validates database name (basic check)
func isValidDatabaseName(name string) bool {
	// Only allow alphanumeric, underscore, and specific patterns
	if len(name) == 0 || len(name) > 64 {
		return false
	}
	// Check for SQL injection patterns
	invalidChars := []string{"'", "\"", ";", "--", "/*", "*/", "DROP", "DELETE", "TRUNCATE"}
	upperName := strings.ToUpper(name)
	for _, char := range invalidChars {
		if strings.Contains(upperName, char) {
			return false
		}
	}
	return true
}

// findMigrationFiles discovers all migration files in database/migrations
func findMigrationFiles() ([]string, error) {
	migrationsPath := filepath.Join(PROJECT_PATH, "database", "migrations")
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

// runMigrations executes migrations in parallel for all workers
// noFresh: if true, runs "migrate" instead of "migrate:fresh"
func runMigrations(workerCount int, noFresh bool) error {
	color.Cyan("\n╔════════════════════════════════════════════════════════════╗")
	color.Cyan("║               Running Database Migrations                  ║")
	color.Cyan("╚════════════════════════════════════════════════════════════╝\n")

	// Check available databases
	availableWorkers, err := checkTestDatabases(workerCount)
	if err != nil {
		return fmt.Errorf("failed to check databases: %w", err)
	}

	if len(availableWorkers) == 0 {
		return fmt.Errorf("no test databases available")
	}

	// Count migration files to determine total progress
	migrationFiles, err := findMigrationFiles()
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
		progressbar.OptionSetWriter(os.Stderr), // Write to stderr to avoid conflicts
		progressbar.OptionOnCompletion(func() {
			fmt.Fprint(os.Stderr, "\n")
		}),
		progressbar.OptionSetRenderBlankState(true),
	)

	// Start workers
	var wg sync.WaitGroup
	results := make(chan MigrationResult, len(availableWorkers))
	startTime := time.Now()

	for _, workerID := range availableWorkers {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			result := runMigrationForWorkerStreaming(id, bar, &completedCount, &progressMu, noFresh)
			results <- result
		}(workerID)
	}

	// Close results channel when all workers are done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var failedMigrations []MigrationResult
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
			color.Red("  Worker %d (DB: webiz_testing_%d): %v\n", result.WorkerID, result.WorkerID, result.Error)
		}
		return fmt.Errorf("migration failed for %d worker(s)", len(failedMigrations))
	}

	return nil
}

// runMigrationForWorkerStreaming executes migrate or migrate:fresh with streaming output and progress tracking
func runMigrationForWorkerStreaming(workerID int, bar *progressbar.ProgressBar, completedCount *int, progressMu *sync.Mutex, noFresh bool) MigrationResult {
	// Get absolute path of project directory
	projectAbsPath, err := filepath.Abs(PROJECT_PATH)
	if err != nil {
		return MigrationResult{
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
	cmd.Env = append(cmd.Env, fmt.Sprintf("DB_DATABASE=webiz_testing_%d", workerID))

	// Set working directory
	cmd.Dir = projectAbsPath

	// Get stdout and stderr pipes for streaming
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return MigrationResult{
			WorkerID: workerID,
			Success:  false,
			Output:   "",
			Error:    fmt.Errorf("failed to create stdout pipe: %w", err),
		}
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return MigrationResult{
			WorkerID: workerID,
			Success:  false,
			Output:   "",
			Error:    fmt.Errorf("failed to create stderr pipe: %w", err),
		}
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return MigrationResult{
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
		// Laravel outputs one line per migration file processed
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
		if err := scanner.Err(); err != nil {
			// Scanner error, but continue
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
		if err := scanner.Err(); err != nil {
			// Scanner error, but continue
		}
	}()

	// Wait for command to complete
	err = cmd.Wait()

	// Wait for all scanners to finish
	scanWg.Wait()

	output := outputBuilder.String()

	return MigrationResult{
		WorkerID: workerID,
		Success:  err == nil,
		Output:   output,
		Error:    err,
	}
}
