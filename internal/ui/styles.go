package ui

import (
	"image/color"

	"charm.land/bubbles/v2/progress"
	"charm.land/lipgloss/v2"
)

// styles holds every style the UI draws with, resolved once against the
// terminal's background so the same code reads well on light and dark themes.
type styles struct {
	app       lipgloss.Style
	title     lipgloss.Style
	subtitle  lipgloss.Style
	heading   lipgloss.Style
	item      lipgloss.Style
	selected  lipgloss.Style
	dimmed    lipgloss.Style
	accent    lipgloss.Style
	success   lipgloss.Style
	failure   lipgloss.Style
	warning   lipgloss.Style
	path      lipgloss.Style
	help      lipgloss.Style
	errorBody lipgloss.Style
}

func newStyles(isDark bool) styles {
	c := lipgloss.LightDark(isDark)

	var (
		fg     = c(lipgloss.Color("#2b2b2b"), lipgloss.Color("#e8e2d6"))
		muted  = c(lipgloss.Color("#7c7468"), lipgloss.Color("#8a8177"))
		accent = c(lipgloss.Color("#a8761f"), lipgloss.Color("#d6b47c"))
		good   = c(lipgloss.Color("#2f7d4f"), lipgloss.Color("#7fc99a"))
		bad    = c(lipgloss.Color("#a8322f"), lipgloss.Color("#e58a86"))
		warn   = c(lipgloss.Color("#8a6a1f"), lipgloss.Color("#d8c07a"))
	)

	base := lipgloss.NewStyle().Foreground(fg)

	return styles{
		app:       lipgloss.NewStyle().Padding(1, 2),
		title:     base.Bold(true).Foreground(accent),
		subtitle:  base.Foreground(muted),
		heading:   base.Bold(true),
		item:      base,
		selected:  base.Bold(true).Foreground(accent),
		dimmed:    base.Foreground(muted),
		accent:    base.Foreground(accent),
		success:   base.Foreground(good),
		failure:   base.Foreground(bad),
		warning:   base.Foreground(warn),
		path:      base.Foreground(muted).Italic(true),
		help:      base.Foreground(muted),
		errorBody: base.Foreground(bad),
	}
}

// newProgress builds the install progress bar, filling from the theme's accent
// to its success color.
func newProgress(s styles) progress.Model {
	return progress.New(
		progress.WithoutPercentage(),
		progress.WithColors([]color.Color{
			s.accent.GetForeground(),
			s.success.GetForeground(),
		}...),
	)
}
