package execution

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"ptp/internal/config"
	"ptp/internal/debug"
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
	cmd.Env = append(cmd.Env, fmt.Sprintf("DB_DATABASE=%s", r.config.GetDatabaseName(0)))
	cmd.Env = append(cmd.Env, fmt.Sprintf("TEST_TOKEN=%d", workerID))

	cmd.Dir = r.config.ProjectPath

	debug.Logf("runner[w%d]: exec %s %v (dir=%s, db=%s)", workerID, phpunitPath, args, r.config.ProjectPath, r.config.GetDatabaseName(workerID))

	st := time.Now()
	output, err := cmd.CombinedOutput()
	dur := time.Since(st)

	if err != nil {
		debug.Logf("runner[w%d]: FAILED %s in %.2fs: %v", workerID, testPath, dur.Seconds(), err)
	} else {
		debug.Logf("runner[w%d]: PASSED %s in %.2fs", workerID, testPath, dur.Seconds())
	}

	return domain.TestResult{
		TestPath: testPath,
		Success:  err == nil,
		Output:   string(output),
		Error:    err,
		Duration: dur,
	}
}
