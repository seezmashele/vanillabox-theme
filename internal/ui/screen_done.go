package ui

import (
	"fmt"
	"strings"
)

// doneView is the closing summary.
func (m Model) doneView() string {
	succeeded, failed := m.results()

	var b strings.Builder

	if failed == 0 {
		b.WriteString(m.heading("Done"))
	} else {
		b.WriteString(m.heading("Finished with errors"))
	}
	b.WriteString("\n")

	b.WriteString(m.styles.success.Render(fmt.Sprintf("  ✓ %d installed", succeeded)))
	b.WriteString("\n")

	if failed > 0 {
		b.WriteString(m.styles.failure.Render(fmt.Sprintf("  ✗ %d failed", failed)))
		b.WriteString("\n")

		for _, index := range m.queue {
			if m.items[index].status != statusFailed {
				continue
			}

			b.WriteString(m.styles.failure.Render(fmt.Sprintf(
				"      %s: %v", m.items[index].component.Name, m.items[index].err,
			)))
			b.WriteString("\n")
		}
	}

	if succeeded > 0 {
		b.WriteString("\n")
		b.WriteString(m.styles.subtitle.Render(
			"Now pick the theme in System Settings → Colors & Themes.",
		))
		b.WriteString("\n")
	}

	return b.String()
}

// results counts how the queued steps turned out.
func (m Model) results() (succeeded, failed int) {
	for _, index := range m.queue {
		switch m.items[index].status {
		case statusOK:
			succeeded++
		case statusFailed:
			failed++
		}
	}

	return succeeded, failed
}

// errorView is shown when the theme could not be loaded at all.
func (m Model) errorView() string {
	var b strings.Builder

	b.WriteString(m.heading("Could not load the theme"))
	b.WriteString("\n")
	b.WriteString(m.styles.errorBody.Render(m.fatal.Error()))
	b.WriteString("\n")

	return b.String()
}
