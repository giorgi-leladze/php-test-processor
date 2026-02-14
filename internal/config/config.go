package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// Config holds all configuration for the application
type Config struct {
	// Project settings
	ProjectPath string
	TestPath    string

	// Output settings
	OutputJSONFile string
	OutputJSONDir  string

	// Execution settings
	Processors int

	// Paths to ignore when scanning
	PathsToIgnore []string

	// Command flags
	Flags Flags
}

// Flags holds command-line flags
type Flags struct {
	Processors int
	Filter     string
	Migrate    bool
	NoFresh    bool
	TestPath   string
	NameFilter string
	TestCases  bool
}

// New creates a new Config with defaults
func New() *Config {
	cfg := &Config{
		ProjectPath:    DefaultProjectPath,
		TestPath:       DefaultTestPath,
		OutputJSONFile: DefaultOutputJSONFile,
		OutputJSONDir:  DefaultOutputJSONDir,
		Processors:     DefaultProcessors,
		Flags:          Flags{Processors: DefaultProcessors},
	}
	// Copy default paths to ignore
	cfg.PathsToIgnore = make([]string, len(DefaultPathsToIgnore))
	copy(cfg.PathsToIgnore, DefaultPathsToIgnore)
	return cfg
}

// Load creates a config and applies flags
func Load(flags Flags) *Config {
	cfg := New()
	cfg.Flags = flags

	// Apply flag overrides
	if flags.Processors > 0 {
		cfg.Processors = flags.Processors
	}

	return cfg
}

// GetTestPath returns the test path, using flag if provided
func (c *Config) GetTestPath() string {
	if c.Flags.TestPath != "" {
		// If TestPath is provided, make it relative to PROJECT_PATH if it's not absolute
		if filepath.IsAbs(c.Flags.TestPath) {
			return c.Flags.TestPath
		}
		return filepath.Join(c.ProjectPath, c.Flags.TestPath)
	}

	// Default: combine project path and test path
	return filepath.Join(c.ProjectPath, c.TestPath)
}

// GetOutputPath returns the full path to the output JSON file (under project so run and faills use the same file).
// Resolves to an absolute path so run and faills always read/write the same file regardless of cwd.
func (c *Config) GetOutputPath() string {
	p := filepath.Join(c.ProjectPath, c.OutputJSONDir, c.OutputJSONFile)
	if abs, err := filepath.Abs(p); err == nil {
		return abs
	}
	return p
}

// GetPHPUnitPath returns the path to PHPUnit binary
func (c *Config) GetPHPUnitPath() string {
	return filepath.Join(c.ProjectPath, "vendor", "bin", "phpunit")
}

// GetDatabaseName returns the database name for a worker
func (c *Config) GetDatabaseName(workerID int) string {
	prefix := os.Getenv("DB_DATABASE_PREFIX")
	if prefix == "" {
		prefix = "testing"
	}
	return fmt.Sprintf("%s_%d", prefix, workerID)
}
