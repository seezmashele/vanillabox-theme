package ui

import (
	"strings"

	"vanillabox/internal/theme"
)

// confirmView is the last stop before installing: the answers, read back.
//
// It lists choices rather than components and destinations. What a preference
// is called is what the user was asked, so reading the answers back in those
// words is the only way they can check them against what they meant — where a
// list of packages and paths asks them to verify something they never chose.
func (m Model) confirmView() string {
	var b strings.Builder

	b.WriteString(m.heading("Review"))
	b.WriteString("\n")
	b.WriteString(m.styles.subtitle.Render("Your choices:"))
	b.WriteString("\n\n")

	// Walked page by page, so the answers come back in the order they were
	// asked rather than in the order the manifest happens to declare them.
	// pageOptions reads m.page, and the receiver is a copy, so moving it here
	// costs nothing and is not visible to the caller.
	for i := range m.pages() {
		m.page = i

		for _, o := range m.pageOptions() {
			// The swatch brings its own leading space, so the colon does not.
			b.WriteString(m.styles.item.Render("  " + o.Name + ":"))
			b.WriteString(swatch(m.selectedSwatch(o)))
			b.WriteString(m.styles.selected.Render(" " + m.optionSummary(o)))
			b.WriteString("\n")
		}
	}

	// A component that will not be installed is worth saying, but only when it
	// is not the user's doing: a gap in the asset tree is a broken build, and
	// this is the only place it gets said. A component they turned down is
	// already above, as the answer that turned it down.
	for _, it := range m.items {
		if it.selected || it.component.Available {
			continue
		}

		b.WriteString(m.styles.dimmed.Render(
			"  " + it.component.Name + " — unavailable, will be skipped",
		))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(m.styles.subtitle.Render(
		"Files are copied only. Nothing is applied — you pick the theme in System Settings.",
	))
	b.WriteString("\n")

	return b.String()
}

// selectedSwatch is the colours of the value a choice currently holds, so the
// review can show the palette rather than only name it. Anything without them
// renders no box at all.
func (m Model) selectedSwatch(o theme.Option) []string {
	if o.Kind != theme.KindSelect {
		return nil
	}

	for _, v := range o.Values {
		if v.ID == m.choices.Values[o.ID] {
			return v.Swatch
		}
	}

	return nil
}

// optionSummary renders an option's state the way the review screen lists it:
// a switch as on or off, a choice by the name of the value picked.
func (m Model) optionSummary(o theme.Option) string {
	if o.Kind == theme.KindSelect {
		return m.selectedValueName(o)
	}

	if m.choices.Toggles[o.ID] {
		return "on"
	}

	return "off"
}
