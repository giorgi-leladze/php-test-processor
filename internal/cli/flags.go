package cli

import "ptp/internal/config"

// Flags holds command-line flags
type Flags struct {
	Processors    int
	Filter        string
	Migrate       bool
	NoFresh       bool
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
		Migrate:       f.Migrate,
		NoFresh:       f.NoFresh,
		TestPath:      f.TestPath,
		NameFilter:    f.NameFilter,
		TestCases:     f.TestCases,
		FailFast:      f.FailFast,
		OnlyFailed:    f.OnlyFailed,
		RerunFailures: f.RerunFailures,
		OpenFaills:    f.OpenFaills,
	}
}

