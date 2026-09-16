package main

import (
	"fmt"
	"strconv"
	"strings"
)

// colorSet is one [Colors:*] section. Only the backgrounds and the focus
// decoration vary between sections; every other role is shared, which is why
// they are filled in by scheme rather than repeated per section.
type colorSet struct {
	Name             string
	BackgroundAlt    string
	BackgroundNormal string
	DecorationFocus  string

	// Selection inverts: its foreground sits on the highlight rather than on a
	// surface, so it carries its own text colours.
	ForegroundNormal   string
	ForegroundInactive string
}

// scheme renders a KColorScheme ini. The same content serves the application
// colour scheme and the Plasma style's own colors file; they differ only in the
// [General] ColorScheme key, which names the scheme by id in one and by display
// name in the other, and in what they answer for WindowOnView.
type scheme struct {
	palette map[string]string
	accent  string
	status  map[string]string

	// WindowOnView puts the window background on the view colour. Two different
	// questions ask for it, one per file, which is why it is spelled as the role
	// it moves rather than as either of them.
	//
	// In the application scheme it is the sidebar option. KColorScheme has no
	// sidebar role: Dolphin's places panel — and every other dock panel in a KDE
	// app — paints with the window background, which is why it matches the
	// toolbar rather than the list. Moving that one role is the only lever there
	// is, so it reaches every window background, not just panels.
	//
	// In the Plasma style's copy it is the darker-panels option. The shell's
	// backgrounds — the panel strip, the launcher, applet popups — all carry
	// ColorScheme-Background, which resolves against this same role.
	//
	// Header deliberately stays behind, which is the one place this breaks the
	// rule that Window, Header and Complementary move together: keeping the
	// toolbar on the chrome colour is the whole point of the sidebar option,
	// since a sidebar that merges with the list still wants a strip above it
	// that does not. Complementary follows Window so the pair a widget might
	// resolve a plain surface against cannot disagree.
	WindowOnView bool

	SchemeKey string
	Name      string
}

func (s scheme) render() string {
	p, st := s.palette, s.status
	a := map[string]string{"highlight": s.accent}

	window := p["background"]
	if s.WindowOnView {
		window = p["view"]
	}

	sets := []colorSet{
		{Name: "Button", BackgroundAlt: p["backgroundAlt"], BackgroundNormal: p["elevated"], DecorationFocus: p["elevated"]},
		{Name: "Complementary", BackgroundAlt: p["backgroundAlt"], BackgroundNormal: window, DecorationFocus: p["focusDim"]},
		// Header is the one background that stays on the chrome colour: the
		// toolbar above a merged panel and list is what keeps them readable.
		{Name: "Header", BackgroundAlt: p["backgroundAlt"], BackgroundNormal: p["background"], DecorationFocus: p["background"]},
		{
			Name: "Selection", BackgroundAlt: a["highlight"], BackgroundNormal: a["highlight"],
			DecorationFocus:  a["highlight"],
			ForegroundNormal: p["onHighlight"], ForegroundInactive: p["onHighlight"],
		},
		// The tooltip is the one surface with a colour of its own, below every
		// other surface in the set. It appears over arbitrary content, so it
		// reads better as a dark card than as one more shade of the window it
		// covers — and it cannot borrow the view colour to get there, because
		// the darker-panels option puts the popups it appears over on exactly
		// that, which would leave a tooltip over a popup showing only its
		// outline. Focus tracks the background, as it does in every set that
		// does not want a visible focus ring.
		{Name: "Tooltip", BackgroundAlt: p["elevatedAlt"], BackgroundNormal: p["tooltip"], DecorationFocus: p["tooltip"]},
		{Name: "View", BackgroundAlt: p["backgroundAlt"], BackgroundNormal: p["view"], DecorationFocus: p["focusDim"]},
		{Name: "Window", BackgroundAlt: p["backgroundAlt"], BackgroundNormal: window, DecorationFocus: window},
	}

	var b strings.Builder

	b.WriteString(`[ColorEffects:Disabled]
ColorAmount=0
ColorEffect=0
ContrastAmount=0.65
ContrastEffect=1
IntensityAmount=0.1
IntensityEffect=2

[ColorEffects:Inactive]
ChangeSelectionColor=true
ColorAmount=0.025
ColorEffect=2
ContrastAmount=0.1
ContrastEffect=2
Enable=false
IntensityAmount=0
IntensityEffect=0

`)

	for _, set := range sets {
		normal, inactive := p["text"], p["textInactive"]
		if set.ForegroundNormal != "" {
			normal, inactive = set.ForegroundNormal, set.ForegroundInactive
		}

		fmt.Fprintf(&b, "[Colors:%s]\n", set.Name)
		for _, kv := range [][2]string{
			{"BackgroundAlternate", set.BackgroundAlt},
			{"BackgroundNormal", set.BackgroundNormal},
			{"DecorationFocus", set.DecorationFocus},
			{"DecorationHover", st["hover"]},
			{"ForegroundActive", a["highlight"]},
			{"ForegroundInactive", inactive},
			{"ForegroundLink", a["highlight"]},
			{"ForegroundNegative", st["negative"]},
			{"ForegroundNeutral", st["neutral"]},
			{"ForegroundNormal", normal},
			{"ForegroundPositive", st["positive"]},
			{"ForegroundVisited", st["visited"]},
		} {
			fmt.Fprintf(&b, "%s=%s\n", kv[0], rgb(kv[1]))
		}
		b.WriteString("\n")
	}

	fmt.Fprintf(&b, "[General]\nColorScheme=%s\nName=%s\nshadeSortColumn=true\n\n", s.SchemeKey, s.Name)
	b.WriteString("[KDE]\ncontrast=4\n\n")

	fmt.Fprintf(&b, "[WM]\nactiveBackground=%s\nactiveBlend=%s\nactiveForeground=%s\n",
		rgb(p["background"]), rgb(p["text"]), rgb(p["text"]))
	fmt.Fprintf(&b, "inactiveBackground=%s\ninactiveBlend=%s\ninactiveForeground=%s\n",
		rgb(p["background"]), rgb(p["textInactive"]), rgb(p["textInactive"]))

	return b.String()
}

// rgb converts #rrggbb into the comma-separated triple KColorScheme files use.
func rgb(hex string) string {
	v, err := strconv.ParseUint(strings.TrimPrefix(hex, "#"), 16, 32)
	if err != nil {
		panic("tokens: not a #rrggbb colour: " + hex)
	}

	return fmt.Sprintf("%d,%d,%d", v>>16&0xff, v>>8&0xff, v&0xff)
}
