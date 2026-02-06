# GitHub Setup Guide

This guide will help you set up the repository on GitHub and prepare it for distribution.

## Initial Setup

### 1. Create GitHub Repository

1. Go to [GitHub](https://github.com) and create a new repository
2. Name it: `php-test-processor` (or your preferred name)
3. Choose visibility (public/private)
4. **Do NOT** initialize with README, .gitignore, or license (we already have these)

### 2. Update README.md

Before pushing, update the GitHub URLs in `README.md`:

```bash
# Replace "yourusername" with your actual GitHub username
sed -i 's/yourusername/YOUR_GITHUB_USERNAME/g' README.md
```

Or manually edit:
- Line 30: `git clone https://github.com/yourusername/php-test-processor.git`
- Line 46: `go install github.com/yourusername/php-test-processor@latest`
- Line 30: `https://github.com/yourusername/php-test-processor/releases`

### 3. Initialize Git (if not already done)

```bash
# Initialize git repository
git init

# Add all files
git add .

# Create initial commit
git commit -m "Initial commit: PHP Test Processor v0.1.0"

# Add remote (replace with your repository URL)
git remote add origin https://github.com/YOUR_USERNAME/php-test-processor.git

# Push to GitHub
git branch -M main
git push -u origin main
```

## Creating Releases

### 1. Tag a Release

```bash
# Create a tag
git tag -a v0.1.0 -m "Release version 0.1.0"
git push origin v0.1.0
```

### 2. Build Binaries for Multiple Platforms

Create a script to build for different platforms:

```bash
#!/bin/bash
# build-release.sh

VERSION="0.1.0"
BUILD_DIR="dist"

mkdir -p $BUILD_DIR

# Linux 64-bit
GOOS=linux GOARCH=amd64 go build -o $BUILD_DIR/ptp-linux-amd64 .
tar -czf $BUILD_DIR/ptp-linux-amd64.tar.gz -C $BUILD_DIR ptp-linux-amd64

# macOS 64-bit
GOOS=darwin GOARCH=amd64 go build -o $BUILD_DIR/ptp-darwin-amd64 .
tar -czf $BUILD_DIR/ptp-darwin-amd64.tar.gz -C $BUILD_DIR ptp-darwin-amd64

# macOS ARM64 (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o $BUILD_DIR/ptp-darwin-arm64 .
tar -czf $BUILD_DIR/ptp-darwin-arm64.tar.gz -C $BUILD_DIR ptp-darwin-arm64

# Windows 64-bit
GOOS=windows GOARCH=amd64 go build -o $BUILD_DIR/ptp-windows-amd64.exe .
zip -j $BUILD_DIR/ptp-windows-amd64.zip $BUILD_DIR/ptp-windows-amd64.exe

echo "Builds complete! Files are in $BUILD_DIR/"
```

### 3. Create GitHub Release

1. Go to your repository on GitHub
2. Click "Releases" → "Create a new release"
3. Choose the tag you created (e.g., `v0.1.0`)
4. Add release title: "Release v0.1.0"
5. Add release notes describing changes
6. Upload the built binaries from the `dist/` directory
7. Publish the release

## GitHub Actions (Optional)

Create `.github/workflows/release.yml` for automated releases:

```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.22'
      
      - name: Build Linux
        run: GOOS=linux GOARCH=amd64 go build -o ptp-linux-amd64 .
      
      - name: Build macOS
        run: |
          GOOS=darwin GOARCH=amd64 go build -o ptp-darwin-amd64 .
          GOOS=darwin GOARCH=arm64 go build -o ptp-darwin-arm64 .
      
      - name: Build Windows
        run: GOOS=windows GOARCH=amd64 go build -o ptp-windows-amd64.exe .
      
      - name: Create Release
        uses: softprops/action-gh-release@v1
        with:
          files: |
            ptp-linux-amd64
            ptp-darwin-amd64
            ptp-darwin-arm64
            ptp-windows-amd64.exe
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

## Repository Settings

### Enable GitHub Pages (Optional)

If you want to host documentation:
1. Go to Settings → Pages
2. Select source branch (e.g., `main`)
3. Select `/docs` folder (if you have one)

### Add Topics

Add relevant topics to your repository:
- `php`
- `phpunit`
- `testing`
- `parallel-testing`
- `go`
- `cli`
- `laravel`

### Add Description

Add a short description: "High-performance parallel test processor for PHPUnit tests written in Go"

## Next Steps

1. ✅ Update README.md with your GitHub username
2. ✅ Push code to GitHub
3. ✅ Create first release
4. ✅ Add repository description and topics
5. ✅ Consider adding GitHub Actions for CI/CD
6. ✅ Add badges to README (build status, version, etc.)

## Badges (Optional)

Add to README.md after the title:

```markdown
![Go Version](https://img.shields.io/badge/go-1.22+-blue.svg)
![License](https://img.shields.io/badge/license-MIT-green.svg)
![Release](https://img.shields.io/github/v/release/YOUR_USERNAME/php-test-processor)
```




