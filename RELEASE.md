# Creating a Release

This guide explains how to create and publish a new release of PTP.

## Prerequisites

- Go 1.22 or higher installed
- Git configured
- GitHub repository access

## Manual Release Process

### 1. Build Release Binaries Locally

```bash
# Run the build script
./build-release.sh v0.1.0

# This will create binaries in the dist/ directory:
# - ptp-linux-amd64.tar.gz
# - ptp-darwin-amd64.tar.gz
# - ptp-darwin-arm64.tar.gz
# - checksums.txt
```

### 2. Create a Git Tag

```bash
# Create and push the tag
git tag -a v0.1.0 -m "Release version 0.1.0"
git push origin v0.1.0
```

### 3. Create GitHub Release

1. Go to your repository on GitHub
2. Click "Releases" → "Create a new release"
3. Choose the tag you just created (e.g., `v0.1.0`)
4. Add release title: "Release v0.1.0"
5. Add release notes describing changes
6. Upload the built binaries from the `dist/` directory:
   - `ptp-linux-amd64.tar.gz`
   - `ptp-darwin-amd64.tar.gz`
   - `ptp-darwin-arm64.tar.gz`
   - `checksums.txt`
7. Publish the release

## Automated Release Process (GitHub Actions)

If you've set up GitHub Actions (`.github/workflows/release.yml`), releases are automated:

1. Create and push a tag:
   ```bash
   git tag -a v0.1.0 -m "Release version 0.1.0"
   git push origin v0.1.0
   ```

2. GitHub Actions will automatically:
   - Build binaries for all platforms
   - Generate checksums
   - Create a GitHub release
   - Upload all artifacts

3. Check the "Actions" tab in your GitHub repository to monitor the build

## Release Checklist

- [ ] Update version number if needed
- [ ] Update CHANGELOG.md (if you have one)
- [ ] Test the build script locally
- [ ] Create git tag
- [ ] Push tag to GitHub
- [ ] Create GitHub release (or verify automated release)
- [ ] Verify download links work
- [ ] Update documentation if needed

## Version Numbering

Follow [Semantic Versioning](https://semver.org/):
- `v1.0.0` - Major release
- `v1.1.0` - Minor release (new features)
- `v1.1.1` - Patch release (bug fixes)

## Testing the Release

After creating a release, test the download:

```bash
# Linux
wget https://github.com/yourusername/php-test-processor/releases/download/v0.1.0/ptp-linux-amd64.tar.gz
tar -xzf ptp-linux-amd64.tar.gz
./ptp-linux-amd64 --version

# macOS
wget https://github.com/yourusername/php-test-processor/releases/download/v0.1.0/ptp-darwin-arm64.tar.gz
tar -xzf ptp-darwin-arm64.tar.gz
./ptp-darwin-arm64 --version
```

