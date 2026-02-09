package ui

import "ptp/internal/domain"

// Viewer displays test results in an interactive TUI
type Viewer interface {
	View(results *domain.TestResultsOutput) error
}

