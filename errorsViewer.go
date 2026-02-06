package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/fatih/color"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/spf13/cobra"
)

var errorsCmd = &cobra.Command{
	Use:   "faills",
	Short: "View test failures interactively",
	Long:  "Display test failures from the last test run in an interactive viewer",
	RunE:  viewErrors,
}

// viewErrors displays test failures in an interactive TUI
func viewErrors(cmd *cobra.Command, args []string) error {
	// Load test results from JSON file
	outputPath := filepath.Join(OUTPUT_JSON_DIR, OUTPUT_JSON_FILE)
	data, err := os.ReadFile(outputPath)
	if err != nil {
		return fmt.Errorf("failed to read test results file: %w\nRun tests first to generate results", err)
	}

	var results TestResultsOutput
	if err := json.Unmarshal(data, &results); err != nil {
		return fmt.Errorf("failed to parse test results: %w", err)
	}

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

		return os.WriteFile(outputPath, jsonData, 0644)
	}

	// Create the application
	app := tview.NewApplication()

	// Create list for failed tests (left side)
	list := tview.NewList().
		ShowSecondaryText(false). // Don't show secondary text
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
			// Show with checkmark and strikethrough effect (grayed out)
			return fmt.Sprintf("[gray]✓ [yellow]%d.[gray] %s[white]", index+1, testName)
		} else {
			// Normal display
			return fmt.Sprintf("[yellow]%d.[white] %s", index+1, testName)
		}
	}

	// Function to update list item display with resolved status
	updateListItem := func(index int) {
		if index < 0 || index >= list.GetItemCount() {
			return
		}
		mainText := getListItemText(index)
		// Update the item
		list.SetItemText(index, mainText, "")
	}

	// Add failed tests to the list with numbers and colors
	for i := range results.Details {
		mainText := getListItemText(i)
		list.AddItem(mainText, "", 0, nil)
	}

	// Set list colors for better visibility
	// Use bright colors for selected items
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
		AddItem(tview.NewBox(), 2, 0, false) // 2 columns of padding on the right

	// Create right side layout: stats on top, details below
	rightSide := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(statsView, 3, 0, false). // 3 lines for stats
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
			statsText := formatFailureStats(failure, index+1)
			statsView.SetText(statsText)

			// Update error details
			detailsView.SetText(formatFailureDetails(failure))
		}
	}

	// Set up keyboard handlers for list
	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyUp, tcell.KeyDown:
			// Let the list handle navigation
			return event
		case tcell.KeyEnter, tcell.KeyRight:
			// Focus details view (though it's read-only, this shows it's selected)
			app.SetFocus(detailsView)
			return nil
		case tcell.KeyCtrlC:
			// Exit
			app.Stop()
			return nil
		case tcell.KeyRune:
			// Check for 'r' or 'R' to mark as resolved
			if event.Rune() == 'r' || event.Rune() == 'R' {
				index := list.GetCurrentItem()
				if index >= 0 && index < len(results.Details) {
					// Toggle resolved status
					resolved[index] = !resolved[index]
					updateListItem(index)
					// Update header with new count
					updateHeader()
					// Update details to reflect the change
					updateDetails()
					// Save resolved status to JSON file
					if err := saveResolvedStatus(); err != nil {
						// Log error but don't stop the app
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
			// Go back to list
			app.SetFocus(list)
			return nil
		case tcell.KeyCtrlC:
			// Exit
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
				// Empty box for spacing
				return x, y, width, height
			}),
			1, 0, false, // 1 line of padding
		).
		AddItem(flex, 0, 1, true)

	// Run the application
	if err := app.SetRoot(mainLayout, true).SetFocus(list).Run(); err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}

	return nil
}

// formatFailureDetails formats a test failure for display
func formatFailureDetails(failure TestFailure) string {
	var builder strings.Builder
	w := tabwriter.NewWriter(&builder, 0, 0, 2, ' ', 0)

	// Test name
	fmt.Fprintf(w, "%s\n", color.RedString("✗ Test: %s", failure.TestName))
	fmt.Fprintf(w, "\n")

	// File path
	fmt.Fprintf(w, "%s\n", color.CyanString("File: %s", failure.FilePath))
	if failure.File != "" && failure.Line > 0 {
		fmt.Fprintf(w, "%s\n", color.YellowString("Location: %s:%d", failure.File, failure.Line))
	}
	fmt.Fprintf(w, "\n")

	// Error message
	if failure.Message != "" {
		fmt.Fprintf(w, "%s\n", color.YellowString("Message:"))
		fmt.Fprintf(w, "%s\n", failure.Message)
		fmt.Fprintf(w, "\n")
	}

	// Error details (JSON)
	if failure.ErrorDetails != "" {
		fmt.Fprintf(w, "%s\n", color.YellowString("Error Details:"))
		fmt.Fprintf(w, "%s\n", failure.ErrorDetails)
		fmt.Fprintf(w, "\n")
	}

	// Stack trace
	if len(failure.StackTrace) > 0 {
		fmt.Fprintf(w, "%s\n", color.YellowString("Stack Trace:"))
		for i, trace := range failure.StackTrace {
			if i < 10 { // Limit to first 10 lines
				fmt.Fprintf(w, "  %s\n", trace)
			}
		}
		if len(failure.StackTrace) > 10 {
			fmt.Fprintf(w, "  ... and %d more lines\n", len(failure.StackTrace)-10)
		}
	}

	w.Flush()
	return builder.String()
}

// formatFailureStats formats the stats header for a test failure
func formatFailureStats(failure TestFailure, number int) string {
	var builder strings.Builder

	// Format: path: path::testcase
	path := failure.FilePath
	if path == "" {
		path = "Unknown path"
	}

	testCase := failure.TestName
	if testCase == "" {
		testCase = fmt.Sprintf("Test %d", number)
	}

	// Format as: path: path::testcase with tview colors
	// Use tview color tags: [cyan] for cyan, [yellow] for yellow
	statsLine := fmt.Sprintf("[cyan]path:[white] [yellow]%s[white]::[yellow]%s[white]", path, testCase)
	builder.WriteString(statsLine)
	builder.WriteString("\n")

	return builder.String()
}

// colorizePath adds color to file paths
func colorizePath(path string) string {
	return color.CyanString(path)
}
