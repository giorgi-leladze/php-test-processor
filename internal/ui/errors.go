package ui

import (
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/fatih/color"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"ptp/internal/config"
	"ptp/internal/domain"
	"ptp/internal/parser"
	"ptp/internal/storage"
)

// LaraMux-style colors (dark theme, orange accent)
var (
	faillsBgDark    = tcell.NewRGBColor(30, 35, 42)   // dark blue-grey
	faillsFg        = tcell.NewRGBColor(220, 220, 220) // off-white
	faillsAccent    = tcell.NewRGBColor(230, 126, 34)  // orange
	faillsGreen     = tcell.NewRGBColor(46, 204, 113)  // green (resolved)
	faillsBorder    = tcell.NewRGBColor(60, 68, 78)
	faillsTitleFg   = tcell.NewRGBColor(240, 240, 240)
	faillsSelectedBg = tcell.NewRGBColor(52, 73, 94)   // slate selection
)

// SingleTestRunner runs a single test case (file + filter). Used by ErrorViewer for rerun.
type SingleTestRunner interface {
	RunFiltered(testPath string, filter string, workerID int) domain.TestResult
}

// ErrorViewer displays test failures in an interactive TUI
type ErrorViewer struct {
	config  *config.Config
	storage storage.Storage
	runner  SingleTestRunner
	parser  *parser.PHPUnitParser
}

// NewErrorViewer creates a new ErrorViewer
func NewErrorViewer(cfg *config.Config, st storage.Storage, runner SingleTestRunner, phpUnitParser *parser.PHPUnitParser) *ErrorViewer {
	return &ErrorViewer{
		config:  cfg,
		storage: st,
		runner:  runner,
		parser:  phpUnitParser,
	}
}

func failureKey(f *domain.TestFailure) string {
	return f.FilePath + "\x00" + f.TestName
}

// View displays test failures in an interactive TUI (LaraMux-inspired design)
func (ev *ErrorViewer) View(results *domain.TestResultsOutput) error {
	if len(results.Details) == 0 {
		color.Green("✓ No test failures found!")
		return nil
	}

	// Marked test cases (temporary selection for group rerun). Key = failureKey(failure).
	marked := make(map[string]bool)
	// Running: test cases currently being re-run (show loader). Key = failureKey(failure).
	runningKeys := make(map[string]bool)

	var updateFooter func() // set after footer is created; used by showFilter/hideFilter

	app := tview.NewApplication()

	// --- Theme: dark background and consistent colors ---
	app.SetBeforeDrawFunc(func(screen tcell.Screen) bool {
		screen.SetStyle(tcell.StyleDefault.Foreground(faillsFg).Background(faillsBgDark))
		return false
	})

	// --- Top header: title + counts ---
	headerLeft := tview.NewTextView().
		SetText(" Test Failures").
		SetTextColor(faillsTitleFg).
		SetDynamicColors(false)
	headerLeft.SetBackgroundColor(faillsBgDark)

	headerRight := tview.NewTextView().
		SetTextAlign(tview.AlignRight).
		SetDynamicColors(false)
	headerRight.SetBackgroundColor(faillsBgDark)

	// --- Filter: indices into results.Details that match current filter (by test name or path) ---
	var filterStr string
	filteredIndices := make([]int, 0, len(results.Details))
	matchesFilter := func(realIdx int) bool {
		if filterStr == "" {
			return true
		}
		f := results.Details[realIdx]
		lower := strings.ToLower(filterStr)
		return strings.Contains(strings.ToLower(f.TestName), lower) ||
			strings.Contains(strings.ToLower(f.FilePath), lower)
	}
	applyFilter := func() {
		filteredIndices = filteredIndices[:0]
		for i := range results.Details {
			if matchesFilter(i) {
				filteredIndices = append(filteredIndices, i)
			}
		}
	}
	applyFilter()

	// Filter input: shown inside list panel when "f" is pressed, hidden otherwise
	filterInput := tview.NewInputField().
		SetLabel(" Filter: ").
		SetPlaceholder("filter by test name or path...").
		SetFieldTextColor(faillsFg).
		SetFieldBackgroundColor(faillsSelectedBg).
		SetLabelColor(faillsAccent)
	filterInput.SetBackgroundColor(faillsBgDark)

	updateHeaderCounts := func() {
		markedCount := 0
		for _, f := range results.Details {
			if marked[failureKey(&f)] {
				markedCount++
			}
		}
		if filterStr == "" {
			if markedCount > 0 {
				headerRight.SetText(fmt.Sprintf("%d failures  ·  %d marked ", len(results.Details), markedCount))
			} else {
				headerRight.SetText(fmt.Sprintf("%d failures ", len(results.Details)))
			}
		} else {
			if markedCount > 0 {
				headerRight.SetText(fmt.Sprintf("%d shown  ·  %d marked ", len(filteredIndices), markedCount))
			} else {
				headerRight.SetText(fmt.Sprintf("%d shown ", len(filteredIndices)))
			}
		}
		headerRight.SetTextColor(faillsFg)
	}

	headerFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(
			tview.NewFlex().
				SetDirection(tview.FlexColumn).
				AddItem(headerLeft, 0, 1, false).
				AddItem(headerRight, 20, 0, false),
			1, 0, false,
		).
		AddItem(newSeparatorLine(faillsAccent), 1, 0, false)

	// --- List (left): failed tests with ► and status dot ---
	list := tview.NewList().
		ShowSecondaryText(false).
		SetHighlightFullLine(true)

	// realIdx = index into results.Details; listPos = 1-based position in filtered list (for display)
	getListItemText := func(realIdx int, listPos int, selected bool) string {
		failure := &results.Details[realIdx]
		testName := failure.TestName
		if testName == "" {
			testName = fmt.Sprintf("Test %d", realIdx+1)
		}
		prefix := "  "
		if selected {
			prefix = "[orange]►[white] "
		}
		if runningKeys[failureKey(failure)] {
			return fmt.Sprintf("%s[yellow]  ⟳  [white] %d. %s", prefix, listPos, testName)
		}
		if marked[failureKey(failure)] {
			return fmt.Sprintf("%s[yellow]▸[white] %d. %s", prefix, listPos, testName)
		}
		return fmt.Sprintf("%s[gray]•[white] %d. %s", prefix, listPos, testName)
	}

	updateListItem := func(listIdx int) {
		if listIdx < 0 || listIdx >= len(filteredIndices) {
			return
		}
		realIdx := filteredIndices[listIdx]
		list.SetItemText(listIdx, getListItemText(realIdx, listIdx+1, list.GetCurrentItem() == listIdx), "")
	}

	refreshListSelectionIndicator := func(prev, curr int) {
		// Bounds-check against actual list length (list may have been rebuilt with fewer items)
		listCount := list.GetItemCount()
		if prev >= 0 && prev < listCount && prev < len(filteredIndices) {
			realIdx := filteredIndices[prev]
			list.SetItemText(prev, getListItemText(realIdx, prev+1, false), "")
		}
		if curr >= 0 && curr < listCount && curr < len(filteredIndices) {
			realIdx := filteredIndices[curr]
			list.SetItemText(curr, getListItemText(realIdx, curr+1, true), "")
		}
	}

	refreshAllListItems := func() {
		for i := 0; i < list.GetItemCount() && i < len(filteredIndices); i++ {
			realIdx := filteredIndices[i]
			list.SetItemText(i, getListItemText(realIdx, i+1, list.GetCurrentItem() == i), "")
		}
	}

	var lastListIndex int
	var updateDetails func()
	rebuildList := func() {
		list.Clear()
		for i, realIdx := range filteredIndices {
			list.AddItem(getListItemText(realIdx, i+1, i == 0), "", 0, nil)
		}
		lastListIndex = 0
		updateHeaderCounts()
		if updateDetails != nil {
			updateDetails()
		}
	}
	rebuildList()

	list.SetMainTextColor(faillsFg).
		SetSelectedTextColor(tcell.ColorWhite).
		SetSelectedBackgroundColor(faillsSelectedBg).
		SetSecondaryTextColor(faillsFg)

	list.SetBorder(true).
		SetBorderColor(faillsBorder).
		SetTitle(" Failed tests ").
		SetTitleColor(faillsAccent).
		SetTitleAlign(tview.AlignLeft)
	list.SetBackgroundColor(faillsBgDark)

	// Left column: either list only, or filter row + list (when "f" opens filter)
	leftColWithFilter := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(filterInput, 1, 0, false).
		AddItem(list, 0, 1, true)
	leftContainer := tview.NewFrame(list)

	showFilter := func() {
		leftContainer.SetPrimitive(leftColWithFilter)
		app.SetFocus(filterInput)
		updateFooter()
	}
	hideFilter := func() {
		for key := range marked {
			delete(marked, key)
		}
		filterInput.SetText("")
		filterStr = ""
		applyFilter()
		rebuildList()
		leftContainer.SetPrimitive(list)
		app.SetFocus(list)
		updateHeaderCounts()
		updateFooter()
	}

	filterInput.SetChangedFunc(func(text string) {
		filterStr = strings.TrimSpace(text)
		applyFilter()
		if len(filteredIndices) == 0 && filterStr != "" {
			// Zero matches: get out of select/filter mode and show full list
			hideFilter()
			return
		}
		rebuildList()
	})

	// --- Right: details panel (stats + body) ---
	statsView := tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(false).
		SetTextColor(faillsFg)
	statsView.SetBackgroundColor(faillsBgDark)

	detailsView := tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(true).
		SetWordWrap(true).
		SetTextColor(faillsFg)
	detailsView.SetBackgroundColor(faillsBgDark)

	detailsContent := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(statsView, 2, 0, false).
		AddItem(detailsView, 0, 1, false)
	detailsContent.SetBorder(true).
		SetBorderColor(faillsBorder).
		SetTitle(" Details ").
		SetTitleColor(faillsAccent).
		SetTitleAlign(tview.AlignLeft)
	detailsContent.SetBackgroundColor(faillsBgDark)

	updateDetails = func() {
		listIdx := list.GetCurrentItem()
		if len(filteredIndices) == 0 || listIdx < 0 || listIdx >= len(filteredIndices) {
			statsView.SetText("")
			detailsView.SetText("[gray]No failures match the filter.[white]")
			detailsContent.SetTitle(" Details ")
			return
		}
		realIdx := filteredIndices[listIdx]
		failure := results.Details[realIdx]
		statsView.SetText(ev.formatFailureStats(failure, realIdx+1))
		detailsView.SetText(ev.formatFailureDetails(failure))
		detailsContent.SetTitle(fmt.Sprintf(" Details · %s ", failure.TestName))
	}
	updateDetails()

	list.SetChangedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		listCount := list.GetItemCount()
		prev := lastListIndex
		if prev >= listCount {
			prev = -1 // skip updating prev if it's stale (e.g. after filter rebuilt list)
		}
		refreshListSelectionIndicator(prev, index)
		lastListIndex = index
		updateDetails()
	})

	// --- Main content: left (list or filter+list) | details ---
	mainFlex := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(leftContainer, 0, 1, true).
		AddItem(detailsContent, 0, 2, false)

	// --- Bottom keybind bar: mode-dependent (LaraMux style) ---
	// Modes: test_cases_list (browsing list), test_cases_list_group_selection (has marked items),
	// test_case_view (focus on details panel), test_cases_filter (filter input visible).
	footer := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	footer.SetBackgroundColor(faillsBorder)
	footer.SetTextColor(faillsFg)

	getFooterForMode := func(mode string) string {
		switch mode {
		case "test_cases_filter":
			return "[orange]Enter[white] Apply  [orange]Esc[white] Cancel & clear marks  [orange]Ctrl+C[white] Quit"
		case "test_case_view":
			return "[orange]←[white]/[orange]Esc[white] Back to list  [orange]Ctrl+C[white] Quit"
		case "test_cases_list_group_selection":
			return "[orange]r[white] Rerun marked  [orange]Space[white] Toggle mark  [orange]Enter[white] View  [orange]f[white] Filter  [orange]Esc[white] Clear marks & exit  ↑↓ Navigate  [orange]Ctrl+C[white] Quit"
		}
		// test_cases_list and fallback
		return "[orange]r[white] Rerun  [orange]Space[white] Mark  [orange]Enter[white] View details  [orange]f[white] Filter  ↑↓ Navigate  [orange]Esc[white] Clear marks  [orange]Ctrl+C[white] Quit"
	}

	updateFooter = func() {
		focus := app.GetFocus()
		var mode string
		switch {
		case focus == filterInput:
			mode = "test_cases_filter"
		case focus == detailsView:
			mode = "test_case_view"
		case focus == list && len(marked) > 0:
			mode = "test_cases_list_group_selection"
		default:
			mode = "test_cases_list"
		}
		footer.SetText(getFooterForMode(mode))
	}
	updateFooter()

	// --- Root layout ---
	root := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(headerFlex, 2, 0, false).
		AddItem(mainFlex, 0, 1, true).
		AddItem(footer, 1, 0, false)

	filterInput.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			hideFilter()
			return nil
		case tcell.KeyEnter:
			app.SetFocus(list)
			updateFooter()
			return nil
		case tcell.KeyCtrlC:
			app.Stop()
			return nil
		}
		return event
	})

	// Rerun selected or current test(s) and update JSON (only re-run entries).
	runRerun := func() {
		var targets []*domain.TestFailure
		if len(marked) > 0 {
			for i := range results.Details {
				f := &results.Details[i]
				if marked[failureKey(f)] {
					targets = append(targets, f)
				}
			}
		} else {
			listIdx := list.GetCurrentItem()
			if listIdx < 0 || listIdx >= len(filteredIndices) {
				return
			}
			realIdx := filteredIndices[listIdx]
			targets = append(targets, &results.Details[realIdx])
		}
		if len(targets) == 0 {
			return
		}

		wasBulk := len(marked) > 0 // bulk = rerun on marked set; after done we unmark all and exit select mode
		// Capture selection to restore after rerun (avoid focus jump)
		selectionKey := failureKey(targets[0])
		targetKeys := make([]string, 0, len(targets))
		for _, f := range targets {
			key := failureKey(f)
			runningKeys[key] = true
			targetKeys = append(targetKeys, key)
		}
		refreshAllListItems()

		go func() {
			toRemove := make(map[string]bool)
			toUpdate := make(map[string]domain.TestFailure)
			for _, f := range targets {
				key := failureKey(f)
				result := ev.runner.RunFiltered(f.FilePath, f.TestName, 1)
				if result.Success {
					toRemove[key] = true
				} else {
					failures := ev.parser.ParseFailure(result)
					if len(failures) > 0 {
						toUpdate[key] = failures[0]
					}
				}
			}

			// Wake the event loop so the update runs even when the user hasn't pressed a key.
			// QueueUpdateDraw runs our callback and then forces a redraw.
			app.QueueEvent(tcell.NewEventKey(tcell.KeyRune, 0, tcell.ModNone))
			app.QueueUpdateDraw(func() {
				// Clear running state only on main thread so UI and state stay in sync
				for _, key := range targetKeys {
					delete(runningKeys, key)
				}
				for key := range toRemove {
					delete(marked, key)
				}

				var newDetails []domain.TestFailure
				for _, f := range results.Details {
					key := failureKey(&f)
					if toRemove[key] {
						continue
					}
					if upd, ok := toUpdate[key]; ok {
						newDetails = append(newDetails, upd)
					} else {
						newDetails = append(newDetails, f)
					}
				}
				results.Details = newDetails
				results.Meta.FailedTestCases = len(newDetails)

				if err := ev.storage.SaveOutput(results); err != nil {
					return
				}
				applyFilter()
				rebuildList()

				if wasBulk {
					// After bulk action: unmark all and get out of select/filter mode
					for key := range marked {
						delete(marked, key)
					}
					hideFilter()
				}

				// Restore selection to the same test (or first item) to avoid focus jump
				desiredIdx := 0
				for i, realIdx := range filteredIndices {
					if realIdx < len(results.Details) && failureKey(&results.Details[realIdx]) == selectionKey {
						desiredIdx = i
						break
					}
				}
				if desiredIdx >= list.GetItemCount() {
					desiredIdx = list.GetItemCount() - 1
				}
				if desiredIdx < 0 {
					desiredIdx = 0
				}
				list.SetCurrentItem(desiredIdx)
				lastListIndex = desiredIdx
				refreshAllListItems()
				updateHeaderCounts()
				updateDetails()

				if len(results.Details) == 0 {
					app.Stop()
				}
			})
		}()
	}

	// --- Input: list ---
	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Swallow synthetic wake-up event (used to process queued rerun updates when idle)
		if event.Key() == tcell.KeyRune && event.Rune() == 0 {
			return nil
		}
		switch event.Key() {
		case tcell.KeyUp, tcell.KeyDown:
			return event
		case tcell.KeyEsc:
			hideFilter() // clear all marks and exit select/filter mode
			return nil
		case tcell.KeyEnter, tcell.KeyRight:
			app.SetFocus(detailsView)
			updateFooter()
			return nil
		case tcell.KeyCtrlC:
			app.Stop()
			return nil
		case tcell.KeyRune:
			if event.Rune() == 'r' || event.Rune() == 'R' {
				runRerun()
				return nil
			}
			if event.Rune() == ' ' {
				listIdx := list.GetCurrentItem()
				if listIdx >= 0 && listIdx < len(filteredIndices) {
					realIdx := filteredIndices[listIdx]
					f := &results.Details[realIdx]
					key := failureKey(f)
					marked[key] = !marked[key]
					updateListItem(listIdx)
					refreshListSelectionIndicator(-1, listIdx)
					updateHeaderCounts()
					updateFooter()
				}
				return nil
			}
			if event.Rune() == 'f' {
				showFilter()
				return nil
			}
		}
		return event
	})

	detailsView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyLeft, tcell.KeyEsc:
			app.SetFocus(list)
			updateFooter()
			return nil
		case tcell.KeyCtrlC:
			app.Stop()
			return nil
		}
		return event
	})

	updateDetails()
	if err := app.SetRoot(root, true).SetFocus(list).Run(); err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}
	return nil
}

// newSeparatorLine draws a thin horizontal line in the given color
func newSeparatorLine(c tcell.Color) *tview.Box {
	return tview.NewBox().
		SetDrawFunc(func(screen tcell.Screen, x, y, width, height int) (int, int, int, int) {
			style := tcell.StyleDefault.Foreground(c)
			for i := 0; i < width; i++ {
				screen.SetContent(x+i, y, '─', nil, style)
			}
			return 0, 0, 0, 0
		})
}

// formatFailureDetails formats a test failure for display using tview color tags
func (ev *ErrorViewer) formatFailureDetails(failure domain.TestFailure) string {
	var builder strings.Builder
	w := tabwriter.NewWriter(&builder, 0, 0, 2, ' ', 0)

	fmt.Fprintf(w, "[red]✗ Test: %s[white]\n\n", failure.TestName)
	fmt.Fprintf(w, "[cyan]File: %s[white]\n", failure.FilePath)
	if failure.File != "" && failure.Line > 0 {
		fmt.Fprintf(w, "[yellow]Location: %s:%d[white]\n", failure.File, failure.Line)
	}
	fmt.Fprintf(w, "\n")

	if failure.Message != "" {
		fmt.Fprintf(w, "[yellow]Message:[white]\n%s\n\n", failure.Message)
	}
	if failure.ErrorDetails != "" {
		fmt.Fprintf(w, "[yellow]Error Details:[white]\n%s\n\n", failure.ErrorDetails)
	}
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

// formatFailureStats formats the one-line path::test for the details header
func (ev *ErrorViewer) formatFailureStats(failure domain.TestFailure, number int) string {
	path := failure.FilePath
	if path == "" {
		path = "Unknown path"
	}
	testCase := failure.TestName
	if testCase == "" {
		testCase = fmt.Sprintf("Test %d", number)
	}
	return fmt.Sprintf("[cyan]path:[white] [yellow]%s[white] :: [yellow]%s[white]", path, testCase)
}
