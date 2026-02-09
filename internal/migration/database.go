package migration

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"ptp/internal/config"
)

// DatabaseManager manages test databases
type DatabaseManager struct {
	config *config.Config
}

// NewDatabaseManager creates a new DatabaseManager
func NewDatabaseManager(cfg *config.Config) *DatabaseManager {
	return &DatabaseManager{config: cfg}
}

// CheckAndCreateDatabases checks if test databases exist and creates them if they don't
func (dm *DatabaseManager) CheckAndCreateDatabases(workerCount int) ([]int, error) {
	// Load .env file from project directory
	envPath := filepath.Join(dm.config.ProjectPath, ".env")
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

	for i := 1; i <= workerCount; i++ {
		dbName := dm.config.GetDatabaseName(i)

		// Check if database exists
		exists, err := dm.databaseExists(db, dbName)
		if err != nil {
			return nil, fmt.Errorf("failed to check database %s: %w", dbName, err)
		}

		if !exists {
			// Create database
			if err := dm.createDatabase(db, dbName); err != nil {
				return nil, fmt.Errorf("failed to create database %s: %w", dbName, err)
			}
			createdCount++
		}

		availableWorkers = append(availableWorkers, i)
	}

	return availableWorkers, nil
}

// databaseExists checks if a database exists
func (dm *DatabaseManager) databaseExists(db *sql.DB, dbName string) (bool, error) {
	var exists bool
	query := "SELECT EXISTS(SELECT SCHEMA_NAME FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME = ?)"
	err := db.QueryRow(query, dbName).Scan(&exists)
	return exists, err
}

// createDatabase creates a new database
func (dm *DatabaseManager) createDatabase(db *sql.DB, dbName string) error {
	// Sanitize database name to prevent SQL injection
	if !dm.isValidDatabaseName(dbName) {
		return fmt.Errorf("invalid database name: %s", dbName)
	}

	query := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`", dbName)
	_, err := db.Exec(query)
	return err
}

// isValidDatabaseName validates database name (basic check)
func (dm *DatabaseManager) isValidDatabaseName(name string) bool {
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

