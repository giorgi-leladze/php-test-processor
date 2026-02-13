package ui

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/fatih/color"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"ptp/internal/config"
	"ptp/internal/domain"
	"ptp/internal/storage"
)

// ErrorViewer displays test failures in an interactive TUI
type ErrorViewer struct {
	config  *config.Config
	storage storage.Storage
}

// NewErrorViewer creates a new ErrorViewer
func NewErrorViewer(cfg *config.Config, st storage.Storage) *ErrorViewer {
	return &ErrorViewer{
		config:  cfg,
		storage: st,
	}
}

// View displays test failures in an interactive TUI
func (ev *ErrorViewer) View(results *domain.TestResultsOutput) error {
	if len(results.Details) == 0 {
		color.Green("✓ No test failures found!")
		return nil
	}

	// Track resolved test cases (by index) - load from JSON
	resolved := make(map[int]bool)
	for i, failure := range results.Details {
		if failure.Resolved {
			resolved[i] = true
		}
	}

	// Function to save resolved status to JSON file
	saveResolvedStatus := func() error {
		// Update the results with resolved status
		for i := range results.Details {
			results.Details[i].Resolved = resolved[i]
		}

		// Save to JSON file
		jsonData, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}

		outputPath := ev.config.GetOutputPath()
		return os.WriteFile(outputPath, jsonData, 0644)
	}

	// Create the application
	app := tview.NewApplication()

	// Create list for failed tests (left side)
	list := tview.NewList().
		ShowSecondaryText(false).
		SetHighlightFullLine(true).
		SetSelectedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
			// When Enter is pressed, we'll show details (handled by key handler)
		})

	// Function to get formatted text for a list item
	getListItemText := func(index int) string {
		failure := results.Details[index]
		testName := failure.TestName
		if testName == "" {
			testName = fmt.Sprintf("Test %d", index+1)
		}

		// Check if resolved
		isResolved := resolved[index]
		if isResolved {
			return fmt.Sprintf("[gray]✓ [yellow]%d.[gray] %s[white]", index+1, testName)
		} else {
			return fmt.Sprintf("[yellow]%d.[white] %s", index+1, testName)
		}
	}

	// Function to update list item display with resolved status
	updateListItem := func(index int) {
		if index < 0 || index >= list.GetItemCount() {
			return
		}
		mainText := getListItemText(index)
		list.SetItemText(index, mainText, "")
	}

	// Add failed tests to the list with numbers and colors
	for i := range results.Details {
		mainText := getListItemText(i)
		list.AddItem(mainText, "", 0, nil)
	}

	// Set list colors for better visibility
	list.SetMainTextColor(tview.Styles.PrimaryTextColor).
		SetSelectedTextColor(tcell.ColorWhite).
		SetSelectedBackgroundColor(tcell.ColorDarkCyan).
		SetSecondaryTextColor(tview.Styles.SecondaryTextColor)

	// Create stats header view (shows path and test case info)
	statsView := tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(false).
		SetWordWrap(false)

	// Create text view for error details (right side)
	detailsView := tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(true).
		SetWordWrap(true)

	// Create a container with right padding for the details view
	detailsContainer := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(detailsView, 0, 1, false).
		AddItem(tview.NewBox(), 2, 0, false)

	// Create right side layout: stats on top, details below
	rightSide := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(statsView, 3, 0, false).
		AddItem(detailsContainer, 0, 1, false)

	// Create simple flex layout: list on left (1/3), details on right (2/3)
	flex := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(list, 0, 1, true).
		AddItem(rightSide, 0, 2, false)

	// Count unresolved tests
	countUnresolved := func() int {
		count := 0
		for i := range results.Details {
			if !resolved[i] {
				count++
			}
		}
		return count
	}

	// Create header text view (so we can update it)
	headerView := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)

	// Function to update header
	updateHeader := func() {
		unresolved := countUnresolved()
		headerText := fmt.Sprintf(" Test Failures (%d total, %d unresolved) | Use ↑↓ to navigate, [yellow]R[white] to mark resolved, → to view details, ← to go back, Ctrl+C to exit ", len(results.Details), unresolved)
		headerView.SetText(headerText)
	}

	// Set initial header
	updateHeader()

	// Update details when selection changes
	updateDetails := func() {
		index := list.GetCurrentItem()
		if index >= 0 && index < len(results.Details) {
			failure := results.Details[index]

			// Update stats header
			statsText := ev.formatFailureStats(failure, index+1)
			statsView.SetText(statsText)

			// Update error details
			detailsView.SetText(ev.formatFailureDetails(failure))
		}
	}

	// Set up keyboard handlers for list
	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyUp, tcell.KeyDown:
			return event
		case tcell.KeyEnter, tcell.KeyRight:
			app.SetFocus(detailsView)
			return nil
		case tcell.KeyCtrlC:
			app.Stop()
			return nil
		case tcell.KeyRune:
			if event.Rune() == 'r' || event.Rune() == 'R' {
				index := list.GetCurrentItem()
				if index >= 0 && index < len(results.Details) {
					resolved[index] = !resolved[index]
					updateListItem(index)
					updateHeader()
					updateDetails()
					if err := saveResolvedStatus(); err != nil {
						_ = err
					}
				}
				return nil
			}
		}
		return event
	})

	// Set up keyboard handlers for details view
	detailsView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyLeft, tcell.KeyEsc:
			app.SetFocus(list)
			return nil
		case tcell.KeyCtrlC:
			app.Stop()
			return nil
		}
		return event
	})

	// Update details when list selection changes
	list.SetChangedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		updateDetails()
	})

	// Set initial details
	updateDetails()

	// Create main layout with title
	mainLayout := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(headerView, 1, 0, false).
		AddItem(
			tview.NewBox().SetDrawFunc(func(screen tcell.Screen, x, y, width, height int) (int, int, int, int) {
				return x, y, width, height
			}),
			1, 0, false,
		).
		AddItem(flex, 0, 1, true)

	// Run the application
	if err := app.SetRoot(mainLayout, true).SetFocus(list).Run(); err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}

	return nil
}

// formatFailureDetails formats a test failure for display using tview color tags ([red], [cyan], etc.)
func (ev *ErrorViewer) formatFailureDetails(failure domain.TestFailure) string {
	var builder strings.Builder
	w := tabwriter.NewWriter(&builder, 0, 0, 2, ' ', 0)

	// Test name
	fmt.Fprintf(w, "[red]✗ Test: %s[white]\n\n", failure.TestName)

	// File path
	fmt.Fprintf(w, "[cyan]File: %s[white]\n", failure.FilePath)
	if failure.File != "" && failure.Line > 0 {
		fmt.Fprintf(w, "[yellow]Location: %s:%d[white]\n", failure.File, failure.Line)
	}
	fmt.Fprintf(w, "\n")

	// Error message
	if failure.Message != "" {
		fmt.Fprintf(w, "[yellow]Message:[white]\n%s\n\n", failure.Message)
	}

	// Error details (JSON)
	if failure.ErrorDetails != "" {
		fmt.Fprintf(w, "[yellow]Error Details:[white]\n%s\n\n", failure.ErrorDetails)
	}

	// Stack trace
	if len(failure.StackTrace) > 0 {
		fmt.Fprintf(w, "[yellow]Stack Trace:[white]\n")
		for i, trace := range failure.StackTrace {
			if i < 10 {
				fmt.Fprintf(w, "  %s\n", trace)
			}
		}
		if len(failure.StackTrace) > 10 {
			fmt.Fprintf(w, "  [gray]... and %d more lines[white]\n", len(failure.StackTrace)-10)
		}
	}

	w.Flush()
	return builder.String()
}

// formatFailureStats formats the stats header for a test failure
func (ev *ErrorViewer) formatFailureStats(failure domain.TestFailure, number int) string {
	var builder strings.Builder

	path := failure.FilePath
	if path == "" {
		path = "Unknown path"
	}

	testCase := failure.TestName
	if testCase == "" {
		testCase = fmt.Sprintf("Test %d", number)
	}

	statsLine := fmt.Sprintf("[cyan]path:[white] [yellow]%s[white]::[yellow]%s[white]", path, testCase)
	builder.WriteString(statsLine)
	builder.WriteString("\n")

	return builder.String()
}

