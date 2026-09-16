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
// The field is Breeze's popup shadow measured off its artwork: one blurred box,
// pushed down a little, at a quarter strength.
type dropShadow struct {
	Color    string  // stop colour
	Strength float64 // opacity where the field has settled, well inside the box
	OffsetY  int     // how far the box sits below the popup
	Radius   int     // blur radius, Breeze's convention: stdDev = Radius/2

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
// deviations past the box, on the side the offset pushes it towards. Past that
// the profile is under a thousandth of the strength.
func (s dropShadow) Margin() int {
	return int(math.Ceil(1.5*float64(s.Radius))) + abs(s.OffsetY)
}

// Tile is the size of every shadow tile. A corner tile has to hold all of the
// curvature — the popup's arc and the box's, which the offset moves along the
// edge — or the stretched edge tile beside it would be asked to draw a curve it
// cannot vary along.
func (s dropShadow) Tile() int {
	return s.Margin() + int(math.Ceil(s.Corner)) + abs(s.OffsetY)
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

// linear is a gradient along one local axis, from the tile's outer edge to the
// popup's, carrying the profile at the box's distance for each point.
func (t shadowTile) linear(id string, along func(k float64) point, dist func(k float64) float64) string {
	m := float64(t.s.Margin())
	a, z := along(0), along(m)

	var b strings.Builder
	fmt.Fprintf(&b, `<linearGradient id="%s" gradientUnits="userSpaceOnUse" x1="%s" y1="%s" x2="%s" y2="%s">`,
		id, n(a.X), n(a.Y), n(z.X), n(z.Y))
	for k := 0.0; k <= m; k += sampleStep {
		fmt.Fprintf(&b, `<stop offset="%s" stop-color="%s" stop-opacity="%s"/>`,
			n(k/m), t.s.Color, n(t.s.alpha(dist(k))))
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

	// Distance outside the box's straight edges, in local coordinates.
	outsideV := func(v float64) float64 { return (m + t.e) - v }
	outsideU := func(u float64) float64 { return m - u }

	if !corner {
		// An edge tile varies along v only; left and right tiles are drawn with
		// u and v swapped by place, so this one shape serves all four.
		band := []point{{0, 0}, {size, 0}, {size, m}, {0, m}}
		defs = append(defs, t.linear(id("edge"),
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

	// The box's own corner centre, moved along v by the offset. Past it on
	// either axis the nearest part of the box is a straight edge; inside both
	// it is the arc, and the field is radial about this point.
	cv := m + r + t.e

	radial := clipV(clipU(outside, cu, true), cv, true)
	alongV := clipU(outside, cu, false)
	alongU := clipV(clipU(outside, cu, true), cv, false)

	c := t.place(cu, cv)
	reach := r + 1.5*float64(s.Radius) + math.Abs(t.e) + 1

	var g strings.Builder
	fmt.Fprintf(&g, `<radialGradient id="%s" gradientUnits="userSpaceOnUse" cx="%s" cy="%s" r="%s">`,
		id("arc"), n(c.X), n(c.Y), n(reach))
	for k := 0.0; k <= reach; k += sampleStep {
		fmt.Fprintf(&g, `<stop offset="%s" stop-color="%s" stop-opacity="%s"/>`,
			n(k/reach), s.Color, n(s.alpha(k-r)))
	}
	g.WriteString("</radialGradient>")

	defs = append(defs, g.String(),
		t.linear(id("v"), func(k float64) point { return t.place(0, k) }, outsideV),
		t.linear(id("u"), func(k float64) point { return t.place(k, 0) }, outsideU),
	)

	element = fmt.Sprintf(`<g id="shadow-%s">%s%s%s%s</g>`, t.name, box,
		shape(t.path(radial), id("arc")), shape(t.path(alongV), id("v")), shape(t.path(alongU), id("u")))

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
