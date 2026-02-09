package parser

import "ptp/internal/domain"

// Parser parses test results and extracts failures
type Parser interface {
	ParseFailure(result domain.TestResult) []domain.TestFailure
}

