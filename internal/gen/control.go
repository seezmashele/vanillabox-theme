package main

import (
	"fmt"
	"strings"
)

// Controls are nine-tile sets stacked down a 24-wide canvas, one set per state,
// 24 tall with a 4px gap between them. Their corners are arcs rather than the
// cubics the window frames use, so they get their own builder.
const (
	controlWidth = 24
	controlSet   = 24
	controlStep  = 28
)

// layer is one painted pass over a tile. States that read as a wash over a
// surface — a button's hover, say — draw the surface first and the wash second,
// which is why a tile takes a list rather than a single paint.
type layer struct {
	Class   string
	Opacity string // written literally: the artwork uses "1.0" and "0" as-is
}

// tileSet is one state of a control: nine tiles at a vertical offset, sharing a
// prefix. A set with no layers is a placeholder — nine bare rects with no fill,
// which is how the theme says "this state draws nothing" without giving up the
// element ids Plasma looks for.
type tileSet struct {
	Prefix string
	Y      int
	Tile   int
	Radius float64
	Layers []layer

	// Margin is the padding this prefix reports, written as explicit
	// hint-*-margin elements. KSvg falls back to the tile size when they are
	// absent, which ties a control's padding to its corner radius — raising the
	// radius needs a wider tile to draw it in, and that would silently make
	// every button bigger. Breeze declares its margins for the same reason.
	// Zero emits no hints, leaving the tile to speak: that is what the button's
	// hover halo wants, since hover-hint-no-border-padding already zeroes it.
	// See docs/plasma-controls.md.
	Margin int
}

type control struct {
	Height int
	Style  string
	Sets   []tileSet
	Hints  []string
}

func (c control) render() string {
	var b strings.Builder

	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d">`+"\n",
		controlWidth, c.Height)
	fmt.Fprintf(&b, `<defs><style type="text/css" id="current-color-scheme">%s</style></defs>`+"\n", c.Style)

	for _, set := range c.Sets {
		for _, el := range set.render() {
			b.WriteString(el + "\n")
		}
	}

	for _, id := range c.Hints {
		fmt.Fprintf(&b, `<rect id="%s" x="0" y="0" width="1" height="1" style="fill:none"/>`+"\n", id)
	}

	b.WriteString("</svg>\n")

	return b.String()
}

// tileGeometry is a tile's box within the set, before the set's offset is added.
type tileGeometry struct {
	Name       string
	X, Y, W, H int
	Corner     bool
	Path       string
}

func (s tileSet) geometry() []tileGeometry {
	t := s.Tile
	inner := controlWidth - 2*t
	right := controlWidth - t
	bottom := controlSet - t

	return []tileGeometry{
		{Name: "topleft", X: 0, Y: 0, W: t, H: t, Corner: true, Path: s.corner("topleft")},
		{Name: "top", X: t, Y: 0, W: inner, H: t},
		{Name: "topright", X: right, Y: 0, W: t, H: t, Corner: true, Path: s.corner("topright")},
		{Name: "left", X: 0, Y: t, W: t, H: inner},
		{Name: "center", X: t, Y: t, W: inner, H: inner},
		{Name: "right", X: right, Y: t, W: t, H: inner},
		{Name: "bottomleft", X: 0, Y: bottom, W: t, H: t, Corner: true, Path: s.corner("bottomleft")},
		{Name: "bottom", X: t, Y: bottom, W: inner, H: t},
		{Name: "bottomright", X: right, Y: bottom, W: t, H: t, Corner: true, Path: s.corner("bottomright")},
	}
}

// corner returns the arc path for a rounded corner, or an empty string when the
// radius is zero and the corner is just another rectangle.
func (s tileSet) corner(name string) string {
	if s.Radius == 0 {
		return ""
	}

	r := n(s.Radius)
	y := s.Y
	w := controlWidth
	h := controlSet
	arc := fmt.Sprintf("A%s,%s 0 0 1", r, r)

	switch name {
	case "topleft":
		return fmt.Sprintf("M0,%d %s %s,%d L%s,%d Z", y+int(s.Radius), arc, r, y, r, y+int(s.Radius))
	case "topright":
		return fmt.Sprintf("M%d,%d %s %d,%d L%d,%d Z",
			w-int(s.Radius), y, arc, w, y+int(s.Radius), w-int(s.Radius), y+int(s.Radius))
	case "bottomleft":
		return fmt.Sprintf("M%s,%d %s 0,%d L%s,%d Z",
			r, y+h, arc, y+h-int(s.Radius), r, y+h-int(s.Radius))
	case "bottomright":
		return fmt.Sprintf("M%d,%d %s %d,%d L%d,%d Z",
			w, y+h-int(s.Radius), arc, w-int(s.Radius), y+h, w-int(s.Radius), y+h-int(s.Radius))
	}

	return ""
}

func (s tileSet) render() []string {
	out := make([]string, 0, 13)

	for i, side := range []string{"left", "right", "top", "bottom"} {
		if s.Margin == 0 {
			break
		}

		out = append(out, fmt.Sprintf(
			`<rect id="%s-hint-%s-margin" x="%d" y="%d" width="%d" height="%d" style="fill:none"/>`,
			s.Prefix, side, i*s.Margin, s.Y, s.Margin, s.Margin))
	}

	for _, g := range s.geometry() {
		id := s.Prefix + "-" + g.Name

		if len(s.Layers) == 0 {
			out = append(out, fmt.Sprintf(
				`<rect id="%s" x="%d" y="%d" width="%d" height="%d" style="fill:none"/>`,
				id, g.X, s.Y+g.Y, g.W, g.H))

			continue
		}

		var painted strings.Builder
		for _, l := range s.Layers {
			paint := fmt.Sprintf(`class="%s" style="fill:currentColor" opacity="%s"`, l.Class, l.Opacity)

			if g.Corner && g.Path != "" {
				fmt.Fprintf(&painted, `<path d="%s" %s/>`, g.Path, paint)
			} else {
				fmt.Fprintf(&painted, `<rect x="%d" y="%d" width="%d" height="%d" %s/>`,
					g.X, s.Y+g.Y, g.W, g.H, paint)
			}
		}

		out = append(out, fmt.Sprintf(`<g id="%s">%s</g>`, id, painted.String()))
	}

	return out
}
