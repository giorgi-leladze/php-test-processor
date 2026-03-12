package commands

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

const (
	releasesBaseURL = "https://github.com/giorgi-leladze/php-test-processor/releases/latest/download"
)

// UpgradeCommand handles the upgrade command
type UpgradeCommand struct{}

// NewUpgradeCommand creates a new UpgradeCommand
func NewUpgradeCommand() *UpgradeCommand {
	return &UpgradeCommand{}
}

// Execute runs the command
func (uc *UpgradeCommand) Execute(cmd *cobra.Command, args []string) error {
	platform := fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH)
	if !isSupportedPlatform(runtime.GOOS, runtime.GOARCH) {
		return fmt.Errorf("upgrade is not supported on %s; please download the appropriate binary from https://github.com/giorgi-leladze/php-test-processor", platform)
	}

	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not determine executable path: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("could not resolve executable path: %w", err)
	}

	assetName := fmt.Sprintf("ptp-%s.tar.gz", platform)
	downloadURL := releasesBaseURL + "/" + assetName

	color.Cyan("Checking for latest release...")
	resp, err := http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: %s (HTTP %d)", downloadURL, resp.StatusCode)
	}

	color.Cyan("Downloading %s...", assetName)
	tmpDir, err := os.MkdirTemp("", "ptp-upgrade-*")
	if err != nil {
		return fmt.Errorf("could not create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	tarPath := filepath.Join(tmpDir, assetName)
	out, err := os.Create(tarPath)
	if err != nil {
		return fmt.Errorf("could not create temp file: %w", err)
	}
	_, err = io.Copy(out, resp.Body)
	out.Close()
	if err != nil {
		return fmt.Errorf("could not write download: %w", err)
	}

	color.Cyan("Extracting...")
	newBinaryPath, err := extractBinaryFromTarGz(tarPath, tmpDir, platform)
	if err != nil {
		return fmt.Errorf("extract failed: %w", err)
	}

	if err := os.Chmod(newBinaryPath, 0755); err != nil {
		return fmt.Errorf("could not chmod new binary: %w", err)
	}

	replacePath := execPath + ".new"
	if err := os.Rename(newBinaryPath, replacePath); err != nil {
		return fmt.Errorf("could not prepare replacement binary: %w", err)
	}

	if err := os.Rename(replacePath, execPath); err != nil {
		_ = os.Remove(replacePath)
		return fmt.Errorf("could not replace binary (try running with sudo if ptp is in a system directory): %w", err)
	}

	color.Green("Upgrade complete. You are now on the latest release.")
	return nil
}

func isSupportedPlatform(goos, goarch string) bool {
	switch goos {
	case "linux":
		return goarch == "amd64"
	case "darwin":
		return goarch == "amd64" || goarch == "arm64"
	default:
		return false
	}
}

func extractBinaryFromTarGz(tarPath, destDir, platform string) (string, error) {
	f, err := os.Open(tarPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return "", err
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	expectedName := "ptp-" + platform
	var extractedPath string

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		base := filepath.Base(hdr.Name)
		if base != expectedName {
			continue
		}
		extractedPath = filepath.Join(destDir, base)
		out, err := os.OpenFile(extractedPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode)&0755)
		if err != nil {
			return "", err
		}
		_, err = io.Copy(out, tr)
		out.Close()
		if err != nil {
			return "", err
		}
		return extractedPath, nil
	}

	return "", fmt.Errorf("binary %s not found in archive", expectedName)
}
