# PHP Test Processor - Architecture & Design

## High-Level Approach

### 1. Core Components

#### A. Test Discovery Service
- **Purpose**: Scan PHP project directories to discover test files
- **Functionality**:
  - Recursively find `*Test.php` files (PHPUnit convention)
  - Parse test classes to identify individual test methods
  - Build a test dependency graph (if needed for integration tests)
  - Categorize tests: unit vs integration

#### B. Test Queue Manager
- **Purpose**: Manage test execution queue with priority and dependency handling
- **Functionality**:
  - Maintain a queue of test tasks
  - Support priority levels (unit tests first, then integration)
  - Handle test dependencies (some integration tests may depend on others)
  - Distribute tests across available workers

#### C. Worker Pool
- **Purpose**: Execute PHP tests in parallel using multiple processes/containers
- **Functionality**:
  - Spawn and manage multiple PHP processes
  - Each worker runs PHPUnit for assigned test(s)
  - Isolate test execution (separate processes/containers)
  - Collect test results and output

#### D. Result Aggregator
- **Purpose**: Collect, parse, and aggregate test results
- **Functionality**:
  - Parse PHPUnit XML/JUnit output
  - Aggregate results from all workers
  - Generate unified test report
  - Track execution time per test

#### E. CLI Interface
- **Purpose**: Command-line interface for executing tests from terminal
- **Commands**:
  - `ptp run [options]` - Execute all tests in parallel
  - `ptp list [options]` - List discovered tests and test cases
  - `ptp migrate [options]` - Run database migrations for test databases
  - `ptp faills` - Interactive error viewer for test failures
- **Features**:
  - Real-time progress output with progress bars
  - Colorized terminal output
  - Interactive TUI for viewing test failures
  - Test case discovery and tree view
  - Wildcard filtering support
  - Parallel database migrations

### 2. Execution Flow

```
1. User executes CLI command (e.g., `ptp run`)
   ↓
2. CLI parses arguments and flags
   ↓
3. Test Discovery scans project for tests (or uses provided filters)
   ↓
4. Tests are queued and categorized
   ↓
5. Worker Pool distributes tests across workers
   ↓
6. Each worker executes PHPUnit in isolated environment
   ↓
7. Real-time progress updates displayed to terminal
   ↓
8. Results are collected and aggregated
   ↓
9. Final report is displayed and exit code is set
```

### 3. Parallelization Strategies

#### Option A: Process-Based (Recommended for simplicity)
- Spawn multiple `phpunit` processes
- Each process runs a subset of tests
- Use Go's `exec.Command` with proper isolation
- Pros: Simple, fast startup, good for unit tests
- Cons: Shared filesystem, potential conflicts

#### Option B: Container-Based (Recommended for integration tests)
- Use Docker containers for each worker
- Each container has isolated environment
- Mount project code as volume
- Pros: Complete isolation, reproducible
- Cons: Higher overhead, slower startup

#### Option C: Hybrid Approach
- Unit tests: Process-based (faster)
- Integration tests: Container-based (isolated)
- Best of both worlds

### 4. Test Distribution Algorithms

#### Round-Robin
- Distribute tests evenly across workers
- Simple and fair

#### Load-Based
- Monitor worker load and assign to least busy worker
- Better resource utilization

#### Dependency-Aware
- Group dependent tests together
- Execute in correct order
- Parallelize independent groups

### 5. Configuration

PTP uses sensible defaults and doesn't require any configuration file. The following directories are automatically ignored when searching for test files:
- `vendor`, `node_modules`, `public`, `storage`, `bootstrap`, `config`, `database`, `resources`, `routes`

#### Command-Line Flags

**`ptp run`** - Execute tests in parallel:
```bash
# Basic usage
ptp run

# With options
ptp run --processors 8
ptp run --test-path tests/Unit
ptp run --filter "*UserTest.php"
ptp run --migrate
ptp run --migrate --no-fresh
ptp run --test-path tests/Integration --filter "*Payment*" --processors 8
```

**`ptp list`** - List discovered tests:
```bash
# List all test files
ptp list

# List with test cases (tree view)
ptp list --test-cases

# Filter tests
ptp list --filter "*UserTest.php"
ptp list --test-path tests/Unit --test-cases
```

**`ptp migrate`** - Run database migrations:
```bash
# Run migrations for all test databases
ptp migrate

# With options
ptp migrate --processors 8
ptp migrate --no-fresh
```

**`ptp faills`** - Interactive error viewer:
```bash
# Open interactive TUI to view test failures
ptp faills
```

### 6. Data Structures

```go
type TestJob struct {
    ID          string
    Status      JobStatus
    Tests       []Test
    Results     TestResults
    StartedAt   time.Time
    CompletedAt *time.Time
}

type Test struct {
    File        string
    Class       string
    Method      string
    Type        TestType // unit or integration
    Dependencies []string
}

type TestResult struct {
    Test        Test
    Status      TestStatus // passed, failed, skipped, error
    Duration    time.Duration
    Output      string
    Error       string
}
```

### 7. Error Handling & Resilience

- Worker failure recovery: Re-queue failed tests
- Timeout handling: Kill hanging tests
- Resource limits: Prevent worker exhaustion
- Graceful shutdown: Complete running tests before exit

### 8. Monitoring & Observability

- Metrics: Tests/second, success rate, average duration
- Logging: Structured logs for debugging
- Tracing: Track test execution through system

## Why Go is Excellent for This Task

### Advantages:

1. **Concurrency Model**
   - Goroutines are perfect for managing multiple test workers
   - Channels for communication between components
   - Built-in `sync` package for coordination
   - Much simpler than thread management in other languages

2. **Performance**
   - Fast startup time (important for spawning workers)
   - Low memory overhead per goroutine
   - Excellent for I/O-bound tasks (executing external processes)

3. **Standard Library**
   - `os/exec` for running PHPUnit processes
   - `context` for timeout and cancellation
   - `sync` for worker pools and coordination
   - `encoding/json` and `encoding/xml` for parsing results

4. **Cross-Platform**
   - Single binary deployment
   - Works on Linux, macOS, Windows
   - No runtime dependencies

5. **Ecosystem**
   - Excellent CLI libraries (cobra, urfave/cli)
   - Docker client libraries available
   - Great terminal UI libraries for progress and colored output
   - Excellent tooling (go test, gofmt, etc.)

6. **Process Management**
   - Easy to spawn and manage child processes
   - Good signal handling for graceful shutdown
   - Process isolation capabilities

### Potential Considerations:

1. **PHP Integration**
   - You'll be executing PHP, not writing PHP
   - Need to parse PHPUnit XML output (standard format)
   - May need PHP project structure understanding

2. **Learning Curve**
   - If team is PHP-focused, Go might be new
   - But Go is relatively simple to learn

3. **Alternative Considerations**
   - PHP itself could work (but less efficient for parallelization)
   - Node.js could work (but Go's concurrency is superior)
   - Python could work (but slower and more complex threading)

## Recommended Tech Stack

- **Language**: Go 1.21+
- **CLI Framework**: `cobra` or `urfave/cli` (recommended: `cobra` for better structure)
- **Process Management**: `os/exec` with `context` for timeouts
- **Container Management**: `docker/docker/client` (if using containers)
- **Configuration**: `viper` (works great with cobra) or `envconfig`
- **Logging**: `log/slog` (Go 1.21+) or `zerolog`
- **Terminal UI**: `charmbracelet/bubbletea` or `fatih/color` for colored output
- **Progress Bars**: `schollz/progressbar` or `cheggaaa/pb`
- **Testing**: `go test` with `testify` for assertions

## CLI-Specific Advantages

### Why CLI Over HTTP Server?

1. **Simplicity**
   - No need to manage server lifecycle
   - Direct execution from terminal
   - Easier to integrate into existing workflows

2. **CI/CD Integration**
   - Works seamlessly with CI/CD pipelines
   - Standard exit codes for automation
   - JSON output for parsing results

3. **Developer Experience**
   - Familiar command-line interface
   - Can be used as drop-in replacement for `phpunit`
   - Real-time feedback during execution

4. **Resource Efficiency**
   - No persistent server process
   - Lower memory footprint
   - Spawns only when needed

5. **Portability**
   - Single binary can be distributed easily
   - No need to configure ports or network
   - Works in any environment with PHP

## CLI Command Structure

### Available Commands

**`ptp run`** - Run PHPUnit tests in parallel
- Flags: `--processors` (default: 4), `--test-path`, `--filter`, `--migrate`, `--no-fresh`

**`ptp list`** - List discovered test files and test cases
- Flags: `--test-path`, `--filter`, `--test-cases` (tree view)

**`ptp migrate`** - Run database migrations for test databases
- Flags: `--processors`, `--no-fresh`

**`ptp faills`** - Interactive TUI for viewing test failures
- Navigation: Arrow keys, `R` to mark resolved, `Ctrl+C` to exit

### Exit Codes
- `0`: All tests passed or command completed successfully
- `1`: One or more tests failed or error occurred

### Output Formats

#### Terminal Output
- Real-time progress bars showing test execution
- Colorized output (green for success, red for failures)
- Summary statistics after completion
- Test results saved to `storage/test-results.json`

#### Interactive Error Viewer (`ptp faills`)
- Tree-like view of test failures
- Navigate with arrow keys
- Mark tests as resolved/unresolved
- View detailed error information
- Persistent state saved to JSON

#### Test Listing (`ptp list`)
- Simple list of test files
- Tree view with test cases (`--test-cases` flag)
- Colorized output for better readability

## Current Implementation Status

### ✅ Implemented Features
- **CLI Structure**: Full cobra-based CLI with multiple commands
- **Test Discovery**: Recursive file scanning for `*Test.php` files
- **Parallel Execution**: Worker pool with configurable processor count
- **Test Filtering**: Wildcard pattern matching for test names
- **Path Selection**: Custom test path specification
- **Database Migrations**: Parallel migration execution for test databases
- **Interactive Error Viewer**: TUI for viewing and managing test failures
- **Test Case Discovery**: Parse and display test methods from test files
- **Tree View**: Beautiful tree-structured output for test listing
- **Progress Display**: Real-time progress bars during execution
- **Result Persistence**: JSON output for test results and failures

### 🔄 Future Enhancements (Not Yet Implemented)
- Container-based execution (Docker support)
- Test dependency resolution
- Advanced scheduling algorithms
- JUnit XML output format
- Test timeout handling
- Worker failure recovery with re-queueing
- Comprehensive structured logging
- Performance metrics and observability

