package config

const (
	// DefaultProjectPath is the default project path
	DefaultProjectPath = "."
	// DefaultTestPath is the default test path
	DefaultTestPath = "."
	// DefaultOutputJSONFile is the default output JSON file name
	DefaultOutputJSONFile = "test-results.json"
	// DefaultOutputJSONDir is the default output directory
	DefaultOutputJSONDir = "storage"
	// DefaultProcessors is the default number of processors
	DefaultProcessors = 4
)

// DefaultPathsToIgnore are the default directories to ignore when scanning for tests
var DefaultPathsToIgnore = []string{
	"vendor",
	"node_modules",
	"public",
	"storage",
	"bootstrap",
	"config",
	"database",
	"resources",
	"routes",
}
