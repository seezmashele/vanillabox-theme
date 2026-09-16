package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"vanillabox/internal/theme"
)

// optionsView is the preferences step, and now the first thing the user sees:
// every choice with every alternative under it, one of each marked.
func (m Model) optionsView() string {
	var b strings.Builder

	pages := m.pages()
	title, step := "Preferences", ""
	if m.page < len(pages) {
		title = pages[m.page]
		step = fmt.Sprintf("Step %d of %d", m.page+1, len(pages))
	}

	b.WriteString(m.heading(title))
	b.WriteString("\n")
	b.WriteString(m.styles.subtitle.Render(m.pageSubtitle(step)))
	b.WriteString("\n\n")

	rows := m.optionRows()
	visible := m.visibleRows(len(rows))
	end := min(m.optionScroll+visible, len(rows))

	// A header left dangling at the bottom names a group whose values are all
	// off screen, which is the same problem as values without a header. Drop it
	// and let the marker below account for it.
	for end-1 > m.optionCursor && !rows[end-1].selectable() {
		end--
	}

	// The markers say which way the rest of the list lies, rather than only that
	// there is more of it.
	if m.optionScroll > 0 {
		b.WriteString(m.styles.dimmed.Render(fmt.Sprintf("  ⋮ %d above", m.optionScroll)))
		b.WriteString("\n")
	}

	for i := m.optionScroll; i < end; i++ {
		b.WriteString(m.renderRow(i, rows[i]))
		b.WriteString("\n")
	}

	if below := len(rows) - end; below > 0 {
		b.WriteString(m.styles.dimmed.Render(fmt.Sprintf("  ⋮ %d below", below)))
		b.WriteString("\n")
	}

	return b.String()
}

// pageSubtitle says where in the sequence this page sits, and on a page asking
// a single question carries that question's description — which would otherwise
// be lost with the header row the page heading replaces.
func (m Model) pageSubtitle(step string) string {
	if step == "" {
		return "How the theme's files should be written"
	}

	if options := m.pageOptions(); len(options) == 1 && options[0].Description != "" {
		return step + " · " + options[0].Description
	}

	return step
}

func (m Model) renderRow(i int, row optionRow) string {
	cursor := "  "
	if i == m.optionCursor {
		cursor = m.styles.accent.Render("❯ ")
	}

	switch row.kind {
	case rowSpacer:
		return ""

	case rowHeader:
		head := m.styles.heading.Render(row.option.Name)
		if row.option.Description != "" {
			head += " " + m.styles.dimmed.Render(row.option.Description)
		}

		// Headers are not selectable, so they never carry the cursor column.
		return "  " + head

	case rowValue:
		mark, style := "( )", m.styles.item
		if m.choices.Values[row.option.ID] == row.value.ID {
			mark, style = "(•)", m.styles.selected
		}

		out := cursor + "  " + style.Render(mark) + swatch(row.value.Swatch) + style.Render(" "+row.value.Name)
		if row.value.Description != "" {
			out += " " + m.styles.dimmed.Render(row.value.Description)
		}

		return out

	default:
		return cursor + m.renderToggle(row.option)
	}
}

// swatch draws the colours a value stands for as adjacent filled boxes, so a
// palette can be seen rather than only read. A value with no colours renders
// nothing at all — not an empty box, which would put every other row out of
// alignment with itself for no reason.
//
// The boxes are backgrounds rather than block glyphs: a filled cell is the same
// size in every font, where ▉ and █ are not. They are two cells wide apiece
// because these are dark, barely-tinted surfaces, and a single cell of one is
// not enough area to tell it from the next palette's.
func swatch(colours []string) string {
	if len(colours) == 0 {
		return ""
	}

	var b strings.Builder

	b.WriteString(" ")
	for _, c := range colours {
		b.WriteString(lipgloss.NewStyle().Background(lipgloss.Color(c)).Render("  "))
	}

	return b.String()
}

func (m Model) renderToggle(o theme.Option) string {
	box, style := "[ ]", m.styles.item
	if m.choices.Toggles[o.ID] {
		box, style = "[x]", m.styles.selected
	}

	out := style.Render(box + " " + o.Name)
	if o.Description != "" {
		out += " " + m.styles.dimmed.Render(o.Description)
	}

	return out
}

// selectedValueName is the display name of a select's current value, falling
// back to the stored id if the manifest and the choice have drifted apart.
func (m Model) selectedValueName(o theme.Option) string {
	id := m.choices.Values[o.ID]

	for _, v := range o.Values {
		if v.ID == id {
			return v.Name
		}
	}

	return id
}
