package ui

import (
	"fmt"
	"strings"
)

// installView shows one line per queued component and an overall progress bar.
func (m Model) installView() string {
	var b strings.Builder

	b.WriteString(m.heading("Installing"))
	b.WriteString("\n\n")

	for _, index := range m.queue {
		b.WriteString(m.stepRow(m.items[index]))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(m.progress.View())
	b.WriteString("  ")
	b.WriteString(m.styles.subtitle.Render(fmt.Sprintf(
		"%d/%d", m.completedCount(), len(m.queue),
	)))

	return b.String()
}

// stepRow renders one component's progress: a marker, its name, and any error.
func (m Model) stepRow(it item) string {
	var marker, name string

	switch it.status {
	case statusRunning:
		// The spinner renders with a trailing space; trim it so the markers line
		// up with the ✓ and ✗ of the other rows.
		marker = strings.TrimSpace(m.spinner.View())
		name = m.styles.selected.Render(it.component.Name)

	case statusOK:
		marker = m.styles.success.Render("✓")
		name = m.styles.item.Render(it.component.Name)

	case statusFailed:
		marker = m.styles.failure.Render("✗")
		name = m.styles.failure.Render(it.component.Name)

	default:
		marker = m.styles.dimmed.Render("·")
		name = m.styles.dimmed.Render(it.component.Name)
	}

	row := "  " + marker + " " + name
	if it.err != nil {
		row += m.styles.failure.Render("  " + it.err.Error())
	}

	return row
}

// completedCount is how many queued steps have finished, either way.
func (m Model) completedCount() int {
	var n int
	for _, index := range m.queue {
		if m.items[index].status == statusOK || m.items[index].status == statusFailed {
			n++
		}
	}

	return n
}
