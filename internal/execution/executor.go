package execution

import (
	"ptp/internal/domain"
	"time"
)

// Executor executes tests and returns results
type Executor interface {
	Execute(tests []string) ([]domain.TestResult, time.Duration, error)
}

