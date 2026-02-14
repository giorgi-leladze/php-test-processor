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
	return r.run(testPath, "", workerID)
}

// RunFiltered runs PHPUnit for a single test file with --filter to run one test case (e.g. method name).
func (r *Runner) RunFiltered(testPath string, filter string, workerID int) domain.TestResult {
	return r.run(testPath, filter, workerID)
}

func (r *Runner) run(testPath string, filter string, workerID int) domain.TestResult {
	phpunitPath := r.config.GetPHPUnitPath()
	ctx := context.Background()
	args := []string{testPath}
	if filter != "" {
		args = append(args, "--filter", filter)
	}
	cmd := exec.CommandContext(ctx, phpunitPath, args...)

	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("DB_DATABASE=%s", r.config.GetDatabaseName(workerID)))
	cmd.Dir = r.config.ProjectPath

	output, err := cmd.CombinedOutput()

	return domain.TestResult{
		TestPath: testPath,
		Success:  err == nil,
		Output:   string(output),
		Error:    err,
	}
}

