# PHP Test Processor (PTP)

## **PTP - So your tests fail faster! 🚀** *(or succeed more optimally, we don't judge)*

A high-performance parallel test processor for PHPUnit tests written in Go. Execute PHP unit and integration tests in parallel to significantly reduce test execution time. Because waiting for tests is so 2023.

## 🚀 Features

- **Parallel Execution**: Run multiple PHPUnit tests simultaneously across multiple workers
- **Test Discovery**: Automatically discover test files in your project
- **Flexible Filtering**: Filter tests by name patterns with wildcard support
- **Interactive Error Viewer**: Beautiful TUI for viewing and managing test failures
- **Database Migrations**: Run migrations in parallel for all test databases
- **Real-time Progress**: See test execution progress with progress bars
- **Colorized Output**: Beautiful terminal output with colors
- **Test Case Listing**: List all test files and their test cases in a tree view
- **Single Binary**: No dependencies, just one executable

## 📋 Requirements

- **Go**: 1.22 or higher (for building from source)
- **PHP**: 7.4 or higher
- **PHPUnit**: Installed in your PHP project (`vendor/bin/phpunit`)
- **Laravel**: Project should use Laravel framework (for migrations support)

## 🔧 Installation

### Option 1: Download Pre-built Binary (Recommended)

**Linux (64-bit):**
```bash
curl -sL https://github.com/giorgi-leladze/php-test-processor/releases/latest/download/ptp-linux-amd64.tar.gz | tar -xz && sudo mv ptp-linux-amd64 /usr/local/bin/ptp && chmod +x /usr/local/bin/ptp

```

**macOS Intel:**
```bash
curl -sL https://github.com/giorgi-leladze/php-test-processor/releases/latest/download/ptp-darwin-amd64.tar.gz | tar -xz && sudo mv ptp-darwin-amd64 /usr/local/bin/ptp && chmod +x /usr/local/bin/ptp
```

**macOS Apple Silicon (M1/M2/M3):**
```bash
curl -sL https://github.com/giorgi-leladze/php-test-processor/releases/latest/download/ptp-darwin-arm64.tar.gz | tar -xz && sudo mv ptp-darwin-arm64 /usr/local/bin/ptp && chmod +x /usr/local/bin/ptp
```

Or visit the [Releases](https://github.com/giorgi-leladze/php-test-processor/releases) page to download manually.

### Option 2: Build from Source

```bash
# Clone the repository
git clone https://github.com/giorgi-leladze/php-test-processor.git
cd php-test-processor

# Build the binary
go build -o ptp .

# Make it executable (Linux/macOS)
chmod +x ptp

# Move to a directory in your PATH (optional)
sudo mv ptp /usr/local/bin/
```

### Option 4: Install via Go Install

```bash
go install github.com/giorgi-leladze/php-test-processor@latest
```

This will install the `ptp` binary to `$GOPATH/bin` (or `$HOME/go/bin` if `GOPATH` is not set).

## ⚙️ Configuration

PTP uses sensible defaults and doesn't require any configuration file. The following directories are automatically ignored when searching for test files:

- `vendor`
- `node_modules`
- `public`
- `storage`
- `bootstrap`
- `config`
- `database`
- `resources`
- `routes`

You can override the default number of processors (4) using the `--processors` flag.

## 📖 Usage

### Run Tests

```bash
# Run all tests with default settings
ptp run

# Run with custom number of processors
ptp run --processors 8

# Run tests from a specific directory
ptp run --test-path tests/Unit

# Filter tests by name pattern
ptp run --filter "*UserTest.php"
ptp run --filter "*Payment*"

# Run migrations before tests
ptp run --migrate

# Run migrations without fresh (only pending migrations)
ptp run --migrate --no-fresh

# Combine options
ptp run --test-path tests/Integration --filter "*Payment*" --processors 8
```

### List Tests

```bash
# List all test files
ptp list

# List tests with test cases (tree view)
ptp list --test-cases

# Filter tests when listing
ptp list --filter "*UserTest.php"

# List tests from specific directory
ptp list --test-path tests/Unit --test-cases
```

### Run Migrations

```bash
# Run migrations for all test databases
ptp migrate

# Run with custom number of workers
ptp migrate --processors 8

# Run without fresh (only pending migrations)
ptp migrate --no-fresh
```

### View Test Failures

```bash
# Open interactive error viewer
ptp faills

# Navigate with arrow keys, mark tests as resolved with 'R', view details with right arrow
```

## 🔍 How It Works

1. **Test Discovery**: Scans your project directory for `*Test.php` files (recursively from the specified path)
2. **Filtering**: Applies name filters if provided (supports wildcard patterns)
3. **Worker Pool**: Creates multiple worker processes (one per processor)
4. **Parallel Execution**: Each worker runs PHPUnit tests in isolated environments with separate databases
5. **Result Aggregation**: Collects and parses results from all workers
6. **Output**: Displays formatted results and saves to JSON for later viewing

## 🗄️ Database Setup

PTP automatically creates separate test databases for each worker:
- `testing_1`, `testing_2`, etc.

Each worker uses its own database to avoid conflicts during parallel execution.

## 📊 Output

Test results are saved to `storage/test-results.json` for later viewing with `ptp faills`.

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## 📝 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🙏 Acknowledgments

- Built with [Cobra](https://github.com/spf13/cobra) for CLI
- Uses [tview](https://github.com/rivo/tview) for the interactive TUI
- Uses [color](https://github.com/fatih/color) for terminal colors
- Uses [phpunit](https://github.com/sebastianbergmann/phpunit) for test execution


## 📞 Support

If you encounter any issues or have questions, please open an issue on GitHub.

---

**Note**: This project is actively maintained. For detailed architecture information, see [ARCHITECTURE.md](./ARCHITECTURE.md).
# php-test-processor
