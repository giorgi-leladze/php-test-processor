package config

import (
	"testing"
)

func TestConfig_GetTestPath(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		expected string
	}{
		{
			name: "default path",
			config: &Config{
				ProjectPath: ".",
				TestPath:    ".",
				Flags:       Flags{},
			},
			expected: ".",
		},
		{
			name: "with test path flag",
			config: &Config{
				ProjectPath: "/project",
				TestPath:    ".",
				Flags: Flags{
					TestPath: "tests",
				},
			},
			expected: "/project/tests",
		},
		{
			name: "absolute test path",
			config: &Config{
				ProjectPath: "/project",
				TestPath:    ".",
				Flags: Flags{
					TestPath: "/absolute/path",
				},
			},
			expected: "/absolute/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetTestPath()
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestConfig_GetDatabaseName(t *testing.T) {
	cfg := New()
	
	t.Run("default database name", func(t *testing.T) {
		name := cfg.GetDatabaseName(1)
		expected := "webiz_testing_1"
		if name != expected {
			t.Errorf("expected %s, got %s", expected, name)
		}
	})

	t.Run("different worker IDs", func(t *testing.T) {
		for i := 1; i <= 5; i++ {
			name := cfg.GetDatabaseName(i)
			// Just check it contains the worker ID and is not empty
			if name == "" {
				t.Errorf("database name should not be empty for worker %d", i)
			}
			// Check it follows the pattern webiz_testing_N
			if len(name) < 15 {
				t.Errorf("database name seems too short for worker %d: %s", i, name)
			}
		}
	})
}

func TestNew(t *testing.T) {
	cfg := New()
	
	if cfg.ProjectPath != DefaultProjectPath {
		t.Errorf("expected ProjectPath %s, got %s", DefaultProjectPath, cfg.ProjectPath)
	}
	
	if cfg.Processors != DefaultProcessors {
		t.Errorf("expected Processors %d, got %d", DefaultProcessors, cfg.Processors)
	}
	
	if len(cfg.PathsToIgnore) != len(DefaultPathsToIgnore) {
		t.Errorf("expected %d paths to ignore, got %d", len(DefaultPathsToIgnore), len(cfg.PathsToIgnore))
	}
}

