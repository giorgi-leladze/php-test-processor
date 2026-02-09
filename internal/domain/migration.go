package domain

// MigrationResult represents the result of a migration execution
type MigrationResult struct {
	WorkerID int
	Success  bool
	Output   string
	Error    error
}

