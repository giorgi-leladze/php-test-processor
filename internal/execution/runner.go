package execution

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"ptp/internal/config"
	"ptp/internal/domain"
)

// Runner executes a single PHPUnit test
type Runner struct {
	config *config.Config
}

// NewRunner creates a new Runner
func NewRunner(cfg *config.Config) *Runner {
	return &Runner{config: cfg}
}

// Run executes PHPUnit for a single test file
func (r *Runner) Run(testPath string, workerID int) domain.TestResult {
	phpunitPath := r.config.GetPHPUnitPath()
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, phpunitPath, testPath)

	// Set environment variables
	cmd.Env = os.Environ() // Start with current environment
	cmd.Env = append(cmd.Env, fmt.Sprintf("DB_DATABASE=%s", r.config.GetDatabaseName(workerID)))

	// Set working directory
	cmd.Dir = r.config.ProjectPath

	output, err := cmd.CombinedOutput()

	return domain.TestResult{
		TestPath: testPath,
		Success:  err == nil,
		Output:   string(output),
		Error:    err,
	}
}

