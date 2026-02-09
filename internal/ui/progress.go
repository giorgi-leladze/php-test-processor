package ui

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/schollz/progressbar/v3"
)

// ProgressBar creates and manages progress bars
type ProgressBar struct {
	bar *progressbar.ProgressBar
}

// NewProgressBar creates a new progress bar
func NewProgressBar(count int) *ProgressBar {
	bar := progressbar.NewOptions(count,
		progressbar.OptionSetDescription(
			color.CyanString("Running tests: ")+
				color.GreenString("[success: 0")+
				" | "+
				color.RedString("failed: 0]"),
		),
		progressbar.OptionSetWidth(50),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        color.CyanString("█"),
			SaucerHead:    color.CyanString("█"),
			SaucerPadding: "░",
			BarStart:      "│",
			BarEnd:        "│",
		}),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionOnCompletion(func() {
			fmt.Fprint(os.Stderr, "\n")
		}),
		progressbar.OptionSetRenderBlankState(true),
	)

	return &ProgressBar{bar: bar}
}

// Update updates the progress bar with success and failure counts
func (p *ProgressBar) Update(successCount, failCount int) {
	p.bar.Set(successCount + failCount)
	p.bar.Describe(
		color.CyanString("Running tests: ") +
			color.GreenString("[success: %d", successCount) +
			" | " +
			color.RedString("failed: %d]", failCount),
	)
}

// Finish completes the progress bar
func (p *ProgressBar) Finish() {
	p.bar.Finish()
}

