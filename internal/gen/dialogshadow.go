package main

import (
	"fmt"
	"math"
	"strings"
)

// dropShadow is the shadow Plasma draws under the launcher and every system
// tray popup.
//
// Plasma asks the compositor for it itself: DialogShadows reads eight
// shadow-<side> tiles and four shadow-hint-<side>-margin rects out of
// dialogs/background and hands them to KWin, which lays each tile against the
// popup with its outer edge the margin's distance outside. Whatever a tile holds
// past the margin sits underneath the popup — and the popups are translucent, so
// anything drawn there shows through as a dark band inside the edge. Breeze's own
// tiles are empty there for the same reason, and these cut the popup's rounded
// shape out exactly.
//
// The field is one blurred box the shape of the popup, held in from its edges by
// an inset and pushed down a little.
type dropShadow struct {
	Color    string  // stop colour
	Strength float64 // opacity where the field has settled, well inside the box
	OffsetY  int     // how far the box sits below the popup
	Radius   int     // blur radius, Breeze's convention: stdDev = Radius/2

	// Inset shrinks the box inside the popup's outline on every side, before
	// the offset moves it, so less of the blur shows past the edges. The box
	// stays concentric with the popup: its corners keep the popup's centres and
	// lose the inset from their radius.
	Inset int

	Corner float64 // the popup's corner radius, which the box shares
}

// dropShadowArc is how many segments a quarter circle is flattened into. The
// cut-out is a polygon so it can be clipped into the regions below with plain
// arithmetic; at sixteen segments a 10px arc strays under a tenth of a pixel.
const dropShadowArc = 16

// layer is the blur as a shadowLayer, so the profile is the decoration's own.
func (s dropShadow) layer() shadowLayer {
	return shadowLayer{Radius: s.Radius, Opacity: s.Strength}
}

// Margin is how far the shadow reaches outside the popup: three standard
// deviations past the box, on the side the offset pushes it towards, less
// whatever the inset holds the box back by. Past that the profile is under a
// thousandth of the strength.
func (s dropShadow) Margin() int {
	if m := int(math.Ceil(1.5*float64(s.Radius))) + abs(s.OffsetY) - s.Inset; m > 1 {
		return m
	}
	return 1
}

// Tile is the size of every shadow tile.
//
// A corner tile has to reach along both edges until the blur has settled to the
// edge's own profile, or the stretched edge tile beside it — which cannot vary
// along its length — would meet it at a step. The box's straight edge starts the
// inset in from the popup's, and at a top corner the offset moves it another
// few pixels along; from there the field needs settle more pixels to come within
// half a level of the edge. The tile also has to hold the popup's whole arc, for
// the cut-out, which at a small blur is the larger of the two.
func (s dropShadow) Tile() int {
	along := s.Inset + abs(s.OffsetY) + s.settle()
	if arc := int(math.Ceil(s.Corner)) + abs(s.OffsetY); arc > along {
		along = arc
	}

	return s.Margin() + along
}

// settle is how far past the start of the box's straight edge the field has to
// run before what is still missing from it is under half an alpha level.
func (s dropShadow) settle() int {
	for d := 0; ; d++ {
		if 255*s.Strength*s.layer().profile(float64(d)) < 0.5 {
			return d
		}
	}
}

// alpha is the shadow's opacity d pixels outside the box's edge.
func (s dropShadow) alpha(d float64) float64 {
	return s.Strength * s.layer().profile(d)
}

type point struct{ X, Y float64 }

// clip keeps the part of a polygon on one side of an axis-aligned line
// (Sutherland–Hodgman). The polygons here are concave only at the arc, and a
// half-plane never splits them in two, so the result stays one outline.
func clip(poly []point, keep func(point) bool, cross func(a, b point) point) []point {
	var out []point

	for i, cur := range poly {
		prev := poly[(i+len(poly)-1)%len(poly)]

		switch {
		case keep(cur):
			if !keep(prev) {
				out = append(out, cross(prev, cur))
			}
			out = append(out, cur)
		case keep(prev):
			out = append(out, cross(prev, cur))
		}
	}

	return out
}

func clipU(poly []point, at float64, below bool) []point {
	return clip(poly,
		func(p point) bool { return (p.X < at) == below },
		func(a, b point) point {
			t := (at - a.X) / (b.X - a.X)
			return point{at, a.Y + t*(b.Y-a.Y)}
		})
}

func clipV(poly []point, at float64, below bool) []point {
	return clip(poly,
		func(p point) bool { return (p.Y < at) == below },
		func(a, b point) point {
			t := (at - a.Y) / (b.Y - a.Y)
			return point{a.X + t*(b.X-a.X), at}
		})
}

// shadowTile is one tile drawn in its own corner-relative frame, where u and v
// run inwards from the tile's outer edges and the popup starts at Margin on both.
// place maps that frame onto the sheet, mirroring it for the right and bottom.
type shadowTile struct {
	s     dropShadow
	name  string
	place func(u, v float64) point

	// e is the box's offset measured inwards along v: the offset pushes the box
	// down, which is inwards for a top tile and outwards for a bottom one.
	e float64
}

func (t shadowTile) path(poly []point) string {
	if len(poly) < 3 {
		return ""
	}

	var b strings.Builder
	for i, p := range poly {
		q := t.place(p.X, p.Y)
		if i == 0 {
			fmt.Fprintf(&b, "M%s,%s", n(q.X), n(q.Y))
		} else {
			fmt.Fprintf(&b, " L%s,%s", n(q.X), n(q.Y))
		}
	}
	b.WriteString(" Z")

	return b.String()
}

// linear is a gradient along one local axis from the tile's outer edge over
// length pixels, carrying the profile at the box's distance for each point.
func (t shadowTile) linear(id string, length float64, along func(k float64) point, dist func(k float64) float64) string {
	a, z := along(0), along(length)

	var b strings.Builder
	fmt.Fprintf(&b, `<linearGradient id="%s" gradientUnits="userSpaceOnUse" x1="%s" y1="%s" x2="%s" y2="%s">`,
		id, n(a.X), n(a.Y), n(z.X), n(z.Y))
	for k := 0.0; k <= length; k += sampleStep {
		fmt.Fprintf(&b, `<stop offset="%s" stop-color="%s" stop-opacity="%s"/>`,
			n(k/length), t.s.Color, n(t.s.alpha(dist(k))))
	}
	b.WriteString("</linearGradient>")

	return b.String()
}

// render returns the tile's gradients and its element. Every tile carries an
// unpainted rect over its whole box, as the margin hints do, because KSvg sizes
// an element by its geometry and a tile that shrank to its shadow would be laid
// out wrong.
func (t shadowTile) render(corner bool) (defs []string, element string) {
	s := t.s
	m, size, r := float64(s.Margin()), float64(s.Tile()), s.Corner

	id := func(part string) string { return "dropshadow-" + t.name + "-" + part }
	origin, far := t.place(0, 0), t.place(size, size)
	box := fmt.Sprintf(`<rect x="%s" y="%s" width="%s" height="%s" style="fill:none"/>`,
		n(math.Min(origin.X, far.X)), n(math.Min(origin.Y, far.Y)), n(size), n(size))

	shape := func(d, grad string) string {
		if d == "" {
			return ""
		}
		return fmt.Sprintf(`<path d="%s" fill="url(#%s)"/>`, d, grad)
	}

	// Distance outside the box's straight edges, in local coordinates. The box
	// starts the inset further in than the popup does.
	in := float64(s.Inset)
	outsideV := func(v float64) float64 { return (m + t.e + in) - v }
	outsideU := func(u float64) float64 { return (m + in) - u }

	if !corner {
		// An edge tile varies along v only; left and right tiles are drawn with
		// u and v swapped by place, so this one shape serves all four.
		band := []point{{0, 0}, {size, 0}, {size, m}, {0, m}}
		defs = append(defs, t.linear(id("edge"), m,
			func(k float64) point { return t.place(0, k) }, outsideV))

		return defs, fmt.Sprintf(`<g id="shadow-%s">%s%s</g>`, t.name, box, shape(t.path(band), id("edge")))
	}

	// Outside the popup: the tile with the popup's quarter cut out of it.
	cu, cvPopup := m+r, m+r
	outside := []point{{0, 0}, {size, 0}, {size, m}, {cu, m}}
	for i := 1; i < dropShadowArc; i++ {
		a := math.Pi / 2 * float64(i) / dropShadowArc
		outside = append(outside, point{cu - r*math.Sin(a), cvPopup - r*math.Cos(a)})
	}
	outside = append(outside, point{m, cvPopup}, point{m, size}, point{0, size})

	// The field is a blurred rectangle, which is separable: the product of the
	// profile across the box's side edge and the profile across its top or
	// bottom. The box's own corners are the popup's less the inset, and at the
	// blur this shadow uses that rounding changes the field by about a level —
	// treating the corner as radial about its arc instead made it twice as dark.
	//
	// SVG cannot multiply two gradients, but it can scale one, so the tile is
	// drawn as one-pixel rows: each carries the horizontal profile and is dimmed
	// by the vertical one at that row, as the window decoration's corners are.
	// Each row is the band of the cut-out outline it covers, so the popup's arc
	// stays exact.
	defs = append(defs, t.linear(id("u"), size,
		func(k float64) point { return t.place(k, 0) }, outsideU))

	var rows strings.Builder
	for k := 0.0; k < size; k++ {
		o := s.layer().profile(outsideV(k + 0.5))
		if o*s.Strength*255 < 0.05 {
			continue
		}

		band := clipV(clipV(outside, k, false), k+1, true)
		if d := t.path(band); d != "" {
			fmt.Fprintf(&rows, `<path d="%s" fill="url(#%s)" opacity="%s"/>`, d, id("u"), n(o))
		}
	}

	element = fmt.Sprintf(`<g id="shadow-%s">%s%s</g>`, t.name, box, rows.String())

	return defs, element
}

// sheet renders the shadow's gradients, eight tiles and four margin hints,
// laid out in a three-by-three block with its top-left corner at (x, y).
func (s dropShadow) sheet(x, y int) (defs []string, elements []string) {
	size := float64(s.Tile())
	x0, y0 := float64(x), float64(y)

	// Each tile's frame: left or right mirrors u, top or bottom mirrors v, and
	// the side tiles swap the axes so the edge shape runs across them.
	at := func(col, row float64, flipU, flipV, swap bool) func(u, v float64) point {
		return func(u, v float64) point {
			if swap {
				u, v = v, u
			}
			if flipU {
				u = size - u
			}
			if flipV {
				v = size - v
			}
			return point{x0 + col*size + u, y0 + row*size + v}
		}
	}

	e := float64(s.OffsetY)
	tiles := []struct {
		tile   shadowTile
		corner bool
	}{
		{shadowTile{s, "topleft", at(0, 0, false, false, false), e}, true},
		{shadowTile{s, "topright", at(2, 0, true, false, false), e}, true},
		{shadowTile{s, "bottomleft", at(0, 2, false, true, false), -e}, true},
		{shadowTile{s, "bottomright", at(2, 2, true, true, false), -e}, true},
		{shadowTile{s, "top", at(1, 0, false, false, false), e}, false},
		{shadowTile{s, "bottom", at(1, 2, false, true, false), -e}, false},
		{shadowTile{s, "left", at(0, 1, false, false, true), 0}, false},
		{shadowTile{s, "right", at(2, 1, true, false, true), 0}, false},
	}

	for _, t := range tiles {
		d, el := t.tile.render(t.corner)
		defs = append(defs, d...)
		elements = append(elements, el)
	}

	m := s.Margin()
	hy := y + 3*s.Tile() + 2
	for i, side := range []string{"top", "bottom", "left", "right"} {
		elements = append(elements, fmt.Sprintf(
			`<rect id="shadow-hint-%s-margin" x="%d" y="%d" width="%d" height="%d" style="fill:none"/>`,
			side, x+i*(m+2), hy, m, m))
	}

	return defs, elements
}
