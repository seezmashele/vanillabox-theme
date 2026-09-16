package ui

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"vanillabox/internal/theme"
)

// testTheme is a small stand-in with one optioned component and one unavailable
// one, so the tests cover both the preferences step and what a partial asset
// tree produces. Installs are real, so it ships real files and redirects the
// data directory into the test's temporary space.
func testTheme(t *testing.T) *theme.Theme {
	t.Helper()

	assets := t.TempDir()
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	writeFile(t, filepath.Join(assets, "colors.colors"), "[General]\n")
	for _, tint := range []string{"neutral", "slate", "rose"} {
		writeFile(t, filepath.Join(assets, "variants", tint, "c.colors"), "["+tint+"]\n")
	}
	writeFile(t, filepath.Join(assets, "style", "widgets", "panel.svg"), "translucent\n")
	writeFile(t, filepath.Join(assets, "style", "opaque", "widgets", "panel.svg"), "opaque\n")

	return &theme.Theme{
		Name:     "Vanilla Box",
		Version:  "0.1.0",
		AssetDir: assets,
		Stamp:    "test",
		Components: []theme.Component{
			{
				ID: "colors", Name: "Color scheme", Source: "colors.colors",
				Target: "color-schemes", Required: true, Available: true,
				// Declares no options of its own, but reads the tint declared on
				// the Plasma style.
				Resolved: []theme.Resolved{{Source: "variants/{tint}/c.colors", Target: ""}},
			},
			{
				ID: "style", Name: "Plasma style", Source: "style",
				Target: "plasma/desktoptheme", Required: true, Available: true,
				Options: []theme.Option{
					{
						ID: "transparency", Name: "Transparency", Kind: theme.KindToggle,
						Group: "Options", Default: true,
						OverlayWhenOff: theme.Overlay{
							From:  "style/opaque",
							Files: []string{"widgets/panel.svg"},
						},
					},
					{
						ID: "tint", Name: "Colour", Kind: theme.KindSelect,
						Group: "Options", DefaultValue: "neutral",
						Values: []theme.OptionValue{
							{ID: "neutral", Name: "Neutral"},
							{ID: "slate", Name: "Slate"},
							{ID: "rose", Name: "Rose"},
						},
					},
				},
			},
			{
				ID: "sounds", Name: "Sounds", Source: "missing",
				Target: "sounds", Required: true, Available: false,
			},
			{
				ID: "extras", Name: "Extras", Source: "style",
				Target: "extras", Available: true,
				InstalledWhen: theme.Condition{Option: "tint", Value: "rose"},
			},
		},
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// send pushes a key through the model the way the runtime would.
func send(m Model, keys ...string) (Model, tea.Cmd) {
	var cmd tea.Cmd

	for _, k := range keys {
		var next tea.Model
		next, cmd = m.Update(keyPress(k))
		m = next.(Model)
	}

	return m, cmd
}

func keyPress(k string) tea.KeyPressMsg {
	if len(k) == 1 {
		return tea.KeyPressMsg{Code: rune(k[0]), Text: k}
	}

	switch k {
	case "enter":
		return tea.KeyPressMsg{Code: '\r'}
	case "esc":
		return tea.KeyPressMsg{Code: 27}
	case "down":
		return tea.KeyPressMsg{Code: 'j', Text: "j"}
	case "up":
		return tea.KeyPressMsg{Code: 'k', Text: "k"}
	case "left":
		return tea.KeyPressMsg{Code: 'h', Text: "h"}
	case "right":
		return tea.KeyPressMsg{Code: 'l', Text: "l"}
	}

	panic("unhandled key " + k)
}

// optionAt puts the cursor on the named switch, so tests do not depend on the
// order the rows happen to come out in.
func optionAt(t *testing.T, m Model, id string) Model {
	t.Helper()

	for i, r := range m.optionRows() {
		if r.kind == rowToggle && r.option.ID == id {
			m.optionCursor = i

			return m
		}
	}

	t.Fatalf("no %q switch among the visible preferences", id)

	return m
}

// TestPreferencesIsTheFirstScreen covers the removal of the checklist: there is
// nothing to choose between components, so the run opens on the preferences.
func TestPreferencesIsTheFirstScreen(t *testing.T) {
	m := New(testTheme(t))

	if m.screen != screenOptions {
		t.Fatalf("screen = %v, want screenOptions", m.screen)
	}
	view := renderView(m)
	if strings.Contains(view, "Choose what to install") {
		t.Error("the component checklist should be gone")
	}
	if !strings.Contains(view, "Step 1 of") {
		t.Error("the first screen should be the first page of preferences")
	}
}

// TestUnavailableComponentIsSkipped is what replaces the greyed-out checklist
// row: a gap in the asset tree installs the rest and says what it left out.
func TestUnavailableComponentIsSkipped(t *testing.T) {
	m := New(testTheme(t))

	if m.items[2].selected {
		t.Error("a component whose files are missing should not be queued")
	}

	m, _ = send(m, "enter")
	if view := renderView(m); !strings.Contains(view, "Sounds — unavailable, will be skipped") {
		t.Errorf("the review should name what it is skipping, got:\n%s", view)
	}
}

// TestEveryValueIsVisible is the point of the layout change: the alternatives
// are on screen rather than behind a control you have to operate to find them.
func TestEveryValueIsVisible(t *testing.T) {
	view := renderView(New(testTheme(t)))

	for _, name := range []string{"Neutral", "Slate", "Rose"} {
		if !strings.Contains(view, name) {
			t.Errorf("preferences should list %q without the user hunting for it", name)
		}
	}
}

// TestChoosingAValueReplacesTheCurrentOne covers the radio behaviour: picking
// one value of a choice unsets the other, rather than accumulating.
func TestChoosingAValueReplacesTheCurrentOne(t *testing.T) {
	m := New(testTheme(t))

	if got := m.choices.Values["tint"]; got != "neutral" {
		t.Fatalf("tint = %q, want the declared default", got)
	}

	m = rowFor(t, m, "tint", "rose")
	m, _ = send(m, " ")

	if got := m.choices.Values["tint"]; got != "rose" {
		t.Errorf("tint = %q, want rose", got)
	}

	m = rowFor(t, m, "tint", "slate")
	m, _ = send(m, " ")

	if got := m.choices.Values["tint"]; got != "slate" {
		t.Errorf("tint = %q, want slate — a choice holds one value", got)
	}
}

// TestCursorSkipsHeaders keeps the cursor on rows that do something. A header
// names a choice and cannot be selected.
func TestCursorSkipsHeaders(t *testing.T) {
	m := New(testTheme(t))
	rows := m.optionRows()

	if rows[m.optionCursor].kind == rowHeader {
		t.Fatal("the cursor should not start on a header")
	}

	for range len(rows) + 2 {
		m, _ = send(m, "down")
		if rows[m.optionCursor].kind == rowHeader {
			t.Fatalf("cursor landed on the header at row %d", m.optionCursor)
		}
	}

	for range len(rows) + 2 {
		m, _ = send(m, "up")
		if rows[m.optionCursor].kind == rowHeader {
			t.Fatalf("cursor landed on the header at row %d going back", m.optionCursor)
		}
	}
}

// TestShortTerminalScrolls checks the window follows the cursor rather than
// letting it walk off the bottom of a list taller than the screen.
func TestShortTerminalScrolls(t *testing.T) {
	m := New(testTheme(t))

	next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 13})
	m = next.(Model)

	visible := m.visibleRows(len(m.optionRows()))
	if visible >= len(m.optionRows()) {
		t.Fatalf("test needs a window shorter than the list: %d of %d", visible, len(m.optionRows()))
	}

	for range len(m.optionRows()) {
		m, _ = send(m, "down")

		if m.optionCursor < m.optionScroll || m.optionCursor >= m.optionScroll+visible {
			t.Fatalf("cursor %d outside the window [%d,%d)",
				m.optionCursor, m.optionScroll, m.optionScroll+visible)
		}
	}

	if !strings.Contains(renderView(m), "below") && !strings.Contains(renderView(m), "above") {
		t.Error("a clipped list should say how much is off screen")
	}
}

// pagedTheme asks its two preferences on two pages, the way the real manifest
// does, so the walk between them is covered without every other test having to
// navigate.
func pagedTheme(t *testing.T) *theme.Theme {
	th := testTheme(t)
	th.Components[1].Options[0].Group = "Switches" // transparency
	th.Components[1].Options[1].Group = "Colour"   // tint

	return th
}

// TestCursorStartsOnTheSelection is what lets a choice list its values in
// whatever order reads best. The cursor rests on the answer rather than on the
// first line, so accepting a page installs what it was already showing. Every
// shipped choice now lists its default first, so the test builds the divergence
// itself rather than borrowing one from the manifest: without this, a choice
// that ever listed another value first would open with the cursor one enter
// away from silently changing an answer nobody looked at.
func TestCursorStartsOnTheSelection(t *testing.T) {
	th := testTheme(t)

	// Default to the last value, so resting on the selection and resting on the
	// first selectable row cannot coincide.
	tint := &th.Components[1].Options[1]
	tint.DefaultValue = "rose"

	m := New(th)

	rows := m.optionRows()
	if m.optionCursor >= len(rows) {
		t.Fatalf("cursor %d past the end of %d rows", m.optionCursor, len(rows))
	}

	row := rows[m.optionCursor]
	if row.option.ID != tint.ID || row.value.ID != "rose" {
		t.Errorf("cursor starts on %s/%s, want the selected %s/rose",
			row.option.ID, row.value.ID, tint.ID)
	}

	// Accepting the page without moving has to leave the choice where it was.
	m.chooseAtCursor()

	if got := m.choices.Values[tint.ID]; got != "rose" {
		t.Errorf("accepting the page changed the choice to %q, want rose", got)
	}
}

// TestOrderArrangesAPageAcrossComponents covers the one thing a group cannot say
// on its own. Shape is asked partly by the Plasma style and partly by the
// decoration, so the manifest's own sequence puts the containers first however
// the page should read; order overrides that without moving the components, and
// options that state none keep the sequence they were declared in.
func TestOrderArrangesAPageAcrossComponents(t *testing.T) {
	th := testTheme(t)
	// One group holding both preferences, so the page is the whole set.
	th.Components[1].Options[0].Group = "Shape"
	th.Components[1].Options[1].Group = "Shape"

	transparency := th.Components[1].Options[0].ID
	tint := th.Components[1].Options[1].ID

	// Left alone the page reads in the order the options become visible, which
	// is not the order they are declared in: the colour scheme reads the tint
	// through a resolved path, so the tint is reached first.
	got := New(th).pageOptions()
	if len(got) != 2 {
		t.Fatalf("pageOptions = %d options, want 2", len(got))
	}
	if got[0].ID != tint {
		t.Fatalf("with no order the page reads %s first, want %s", got[0].ID, tint)
	}

	// Asking for the reverse has to actually reverse it — an order that merely
	// agreed with the natural sequence would prove nothing.
	th.Components[1].Options[0].Order = 1 // transparency
	th.Components[1].Options[1].Order = 2 // tint

	if got := New(th).pageOptions(); got[0].ID != transparency {
		t.Errorf("page reads %s first, want the lower order %s", got[0].ID, transparency)
	}
}

// TestPagesFollowTheirGroups covers the split: a page per group, in the order
// the groups first appear, with enter and esc walking between them.
func TestPagesFollowTheirGroups(t *testing.T) {
	m := New(pagedTheme(t))

	// Page order follows the order the options become visible, not the order
	// they sit in one component: the colour scheme reads the tint through a
	// resolved path, so Colour is reached first.
	if got := m.pages(); len(got) != 2 || got[0] != "Colour" || got[1] != "Switches" {
		t.Fatalf("pages = %v, want [Colour Switches]", got)
	}

	if view := renderView(m); !strings.Contains(view, "Step 1 of 2") {
		t.Error("a page should say where in the sequence it sits")
	}
	if m.keys.Back.Enabled() {
		t.Error("esc should be disabled on the first page")
	}

	if view := renderView(m); !strings.Contains(view, "Neutral") {
		t.Error("the first page should show the colour values")
	}

	m, _ = send(m, "enter")
	if m.page != 1 {
		t.Fatalf("page = %d, want 1 after enter", m.page)
	}

	view := renderView(m)
	if !strings.Contains(view, "Transparency") {
		t.Error("the second page should show its own options")
	}
	if strings.Contains(view, "Neutral") {
		t.Error("the second page should not show the first page's options")
	}

	// esc walks back rather than leaving the preferences.
	m, _ = send(m, "esc")
	if m.page != 0 || m.screen != screenOptions {
		t.Errorf("page = %d screen = %v, want to step back to page 0", m.page, m.screen)
	}

	// And enter off the end reaches the review.
	m, _ = send(m, "enter", "enter")
	if m.screen != screenConfirm {
		t.Errorf("screen = %v, want screenConfirm past the last page", m.screen)
	}

	// Back from the review lands on the last question asked, not the first.
	m, _ = send(m, "esc")
	if m.page != 1 {
		t.Errorf("page = %d after esc from the review, want the last page", m.page)
	}
}

// TestSingleQuestionPageDropsTheHeader keeps a page from saying the same thing
// twice: the heading already names the choice.
func TestSingleQuestionPageDropsTheHeader(t *testing.T) {
	m := New(pagedTheme(t))
	m.theme.Components[1].Options[1].Description = "surfaces and highlight"

	// Page one asks only for a colour, so the heading carries the question.
	for _, r := range m.optionRows() {
		if r.kind == rowHeader {
			t.Error("a page asking a single choice should not repeat it as a header")
		}
	}

	if view := renderView(m); !strings.Contains(view, "Step 1 of 2 · surfaces and highlight") {
		t.Errorf("the option's description should move into the subtitle, got:\n%s", view)
	}
}

// rowFor puts the cursor on a named value of a named option.
func rowFor(t *testing.T, m Model, option, value string) Model {
	t.Helper()

	for i, r := range m.optionRows() {
		if r.kind == rowValue && r.option.ID == option && r.value.ID == value {
			m.optionCursor = i

			return m
		}
	}

	t.Fatalf("no %q row for option %q", value, option)

	return m
}

// TestConfirmScreenReadsBackTheChoices covers what the review is for: the
// answers in the words they were asked in. Components and their destinations
// are deliberately absent — the user chose preferences, not packages, and
// asking them to check a path tells them nothing about whether they got what
// they wanted.
func TestConfirmScreenReadsBackTheChoices(t *testing.T) {
	m := New(testTheme(t))

	m = optionAt(t, m, "transparency")
	m, _ = send(m, " ", "enter")

	if m.screen != screenConfirm {
		t.Fatalf("screen = %v, want screenConfirm", m.screen)
	}

	view := plain(renderView(m))
	if !strings.Contains(view, "Transparency: off") {
		t.Error("confirm view should report the chosen options")
	}
	if !strings.Contains(view, "Colour: Neutral") {
		t.Error("confirm view should report the chosen value of a select")
	}
	if !strings.Contains(view, "System Settings") {
		t.Error("confirm view should say the theme is not applied")
	}

	for _, unwanted := range []string{"color-schemes", "plasma/desktoptheme", "~/.local"} {
		if strings.Contains(view, unwanted) {
			t.Errorf("confirm view should not show file paths, found %q", unwanted)
		}
	}
}

// TestConfirmScreenShowsTheColourChosen is the swatch following the user to the
// last screen. Reading "Rose" back is not a check on anything if the reason
// they picked it was the colour.
func TestConfirmScreenShowsTheColourChosen(t *testing.T) {
	th := testTheme(t)

	for i, o := range th.Components[1].Options {
		if o.ID == "tint" {
			th.Components[1].Options[i].Values[0].Swatch = []string{"#292929", "#383838"}
		}
	}

	m := New(th)
	m, _ = send(m, "enter")

	if view := renderView(m); !strings.Contains(view, "\x1b[48;2;41;41;41m") {
		t.Error("the review should draw the colour of the palette chosen")
	}
}

// TestEscapeReturnsToPreferences checks the one step back that still exists.
// The preferences are the first screen now, so esc there has nowhere to go and
// the binding is disabled rather than silently doing nothing.
func TestEscapeReturnsToPreferences(t *testing.T) {
	m := New(testTheme(t))

	m, _ = send(m, "enter", "esc")
	if m.screen != screenOptions {
		t.Fatalf("screen = %v, want screenOptions after esc from the review", m.screen)
	}

	if m.keys.Back.Enabled() {
		t.Error("esc should be disabled on the first screen")
	}
}

// TestInstallWalksTheQueue drives the whole install by hand: it runs each
// command the model returns and feeds the resulting message back in, which is
// what the Bubble Tea runtime does.
func TestInstallWalksTheQueue(t *testing.T) {
	m := New(testTheme(t))

	m, _ = send(m, "enter") // through the preferences, to the review
	next, cmd := m.Update(keyPress("enter"))
	m = next.(Model)

	if m.screen != screenInstall {
		t.Fatalf("screen = %v, want screenInstall", m.screen)
	}
	if len(m.queue) != 2 {
		t.Fatalf("queue has %d steps, want 2", len(m.queue))
	}
	if m.items[m.queue[0]].status != statusRunning {
		t.Error("the first queued component should be running")
	}
	if m.keys.Quit.Enabled() {
		t.Error("quit should be disabled while installing")
	}

	// Run the install to completion.
	for step := range m.queue {
		msg := runInstallStep(t, cmd)
		if msg == nil {
			t.Fatalf("step %d produced no stepDoneMsg", step)
		}

		next, cmd = m.Update(msg)
		m = next.(Model)
	}

	succeeded, failed := m.results()
	if succeeded != 2 || failed != 0 {
		t.Errorf("results() = (%d, %d), want (2, 0)", succeeded, failed)
	}
	if !m.finishing {
		t.Error("the model should be waiting for the progress bar to catch up")
	}

	// The progress bar finishing is what moves us to the done screen. The last
	// command the model handed back animates it; keep pumping frames until it
	// settles.
	for range 500 {
		if cmd == nil || m.screen == screenDone {
			break
		}

		next, cmd = m.Update(cmd())
		m = next.(Model)
	}

	if m.screen != screenDone {
		t.Fatal("never reached the done screen")
	}
	if view := renderView(m); !strings.Contains(view, "2 installed") {
		t.Errorf("done view should report 2 installed, got:\n%s", view)
	}
}

func TestRestartFromDone(t *testing.T) {
	m := New(testTheme(t))
	m.screen = screenDone
	m.queue = []int{0}
	m.items[0].status = statusOK
	m.updateBindings()

	m, _ = send(m, "r")

	if m.screen != screenOptions {
		t.Errorf("screen = %v, want screenOptions", m.screen)
	}
	if m.items[0].status != statusPending {
		t.Error("restart should clear step statuses")
	}
}

func TestErrorModelShowsTheReason(t *testing.T) {
	m := NewError(errMissingAssets{})

	view := renderView(m)
	if !strings.Contains(view, "Could not load the theme") {
		t.Error("error view should say the theme could not be loaded")
	}
	if !strings.Contains(view, "no theme.json") {
		t.Error("error view should include the underlying error")
	}
}

type errMissingAssets struct{}

func (errMissingAssets) Error() string { return "no theme.json found" }

// runInstallStep executes a command chain until it yields a stepDoneMsg.
func runInstallStep(t *testing.T, cmd tea.Cmd) tea.Msg {
	t.Helper()

	if cmd == nil {
		return nil
	}

	deadline := time.Now().Add(10 * time.Second)

	pending := []tea.Cmd{cmd}
	for len(pending) > 0 && time.Now().Before(deadline) {
		next := pending[0]
		pending = pending[1:]

		if next == nil {
			continue
		}

		switch msg := next().(type) {
		case stepDoneMsg:
			return msg
		case tea.BatchMsg:
			pending = append(pending, msg...)
		}
	}

	return nil
}

func renderView(m Model) string {
	return m.View().Content
}

// ansiEscape matches the styling lipgloss writes around every rendered segment.
var ansiEscape = regexp.MustCompile("\x1b\\[[0-9;]*m")

// plain drops the styling, so a test can match text that spans two styled
// segments — a label and its value, say — without depending on where the
// escapes happen to fall between them.
func plain(s string) string { return ansiEscape.ReplaceAllString(s, "") }

// TestConditionalComponentFollowsItsPreference is the mechanism that replaced a
// component checklist: the question is asked once, among the preferences, and
// the component follows the answer. The failure it guards is a stale selection
// — the answer changing while the install queue keeps whatever it decided when
// the model was built.
func TestConditionalComponentFollowsItsPreference(t *testing.T) {
	m := New(testTheme(t))

	extras := func(m Model) bool {
		for _, it := range m.items {
			if it.component.ID == "extras" {
				return it.selected
			}
		}

		t.Fatal("no extras component in the fixture")

		return false
	}

	if extras(m) {
		t.Error("extras is selected before its preference asks for it")
	}

	m = rowFor(t, m, "tint", "rose")
	m, _ = send(m, " ")

	if !extras(m) {
		t.Error("choosing rose should select extras")
	}

	m = rowFor(t, m, "tint", "neutral")
	m, _ = send(m, " ")

	if extras(m) {
		t.Error("choosing neutral again should deselect extras")
	}
}

// TestPaletteRowsCarryTheirColour covers the one thing a name cannot do: a
// palette is a colour, and "Plum" only tells you which colour if you already
// know. The boxes are drawn from the manifest's swatches, so what is on screen
// and what gets installed come from the same place.
func TestPaletteRowsCarryTheirColour(t *testing.T) {
	th := testTheme(t)

	for i, o := range th.Components[1].Options {
		if o.ID == "tint" {
			th.Components[1].Options[i].Values[2].Swatch = []string{"#2b2729", "#3b3538"}
		}
	}

	view := renderView(New(th))

	// The background escapes for the pair, which is what two filled boxes are.
	for _, want := range []string{"\x1b[48;2;43;39;41m", "\x1b[48;2;59;53;56m"} {
		if !strings.Contains(view, want) {
			t.Errorf("the rose row should draw both of its surfaces")
		}
	}

	// A value with no swatch draws no box, rather than an empty one that would
	// leave the rows misaligned with each other.
	if got := swatch(nil); got != "" {
		t.Errorf("swatch(nil) = %q, want nothing at all", got)
	}
}
