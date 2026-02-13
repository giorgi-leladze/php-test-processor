package ui

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/schollz/progressbar/v3"
)

// ProgressBar creates and manages progress bars
type ProgressBar struct {
	bar          *progressbar.ProgressBar
	totalCount   int
	testCaseCount int
}

// NewProgressBar creates a new progress bar. fileCount is the number of test files (bar total);
// testCaseCount is the number of test cases to show in the label (use 0 to show file count).
func NewProgressBar(fileCount, testCaseCount int) *ProgressBar {
	descCount := fileCount
	descLabel := "files"
	if testCaseCount > 0 {
		descCount = testCaseCount
		descLabel = "test cases"
	}
	bar := progressbar.NewOptions(fileCount,
		progressbar.OptionSetDescription(
			color.CyanString("Running tests")+
				color.WhiteString(" (%d %s): ", descCount, descLabel)+
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

	return &ProgressBar{
		bar:           bar,
		totalCount:   fileCount,
		testCaseCount: testCaseCount,
	}
}

// Update updates the progress bar. filesCompleted is the bar position (0..fileCount);
// passedCases and failedCases are test case counts shown in the label.
func (p *ProgressBar) Update(filesCompleted, passedCases, failedCases int) {
	p.bar.Set(filesCompleted)
	descCount := p.totalCount
	descLabel := "files"
	if p.testCaseCount > 0 {
		descCount = p.testCaseCount
		descLabel = "test cases"
	}
	p.bar.Describe(
		color.CyanString("Running tests") +
			color.WhiteString(" (%d %s): ", descCount, descLabel) +
			color.GreenString("[success: %d", passedCases) +
			" | " +
			color.RedString("failed: %d]", failedCases),
	)
}

// Finish completes the progress bar
func (p *ProgressBar) Finish() {
	p.bar.Finish()
}

