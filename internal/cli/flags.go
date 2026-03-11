package cli

import "ptp/internal/config"

// Flags holds command-line flags
type Flags struct {
	Processors    int
	Filter        string
	SkipMigrate   bool
	Fresh         bool
	TestPath      string
	NameFilter    string
	TestCases     bool
	FailFast      bool
	OnlyFailed    bool
	RerunFailures bool
	OpenFaills    bool
}

// ToConfigFlags converts CLI flags to config flags
func (f *Flags) ToConfigFlags() config.Flags {
	return config.Flags{
		Processors:    f.Processors,
		Filter:        f.Filter,
		SkipMigrate:   f.SkipMigrate,
		Fresh:         f.Fresh,
		TestPath:      f.TestPath,
		NameFilter:    f.NameFilter,
		TestCases:     f.TestCases,
		FailFast:      f.FailFast,
		OnlyFailed:    f.OnlyFailed,
		RerunFailures: f.RerunFailures,
		OpenFaills:    f.OpenFaills,
	}
}

