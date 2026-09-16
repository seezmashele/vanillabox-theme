package main

import (
	"fmt"
	"strconv"
	"strings"
)

// cornerFactor places a cubic corner's control points, at radius*cornerFactor in
// from the corner. It is the complement of the usual circle-to-cubic constant,
// written directly because the original hand-drawn artwork's rounding pinned it:
// only 0.4478 reproduced both the 3.582 those frames used at radius 8 and the
// 3.135 the window decoration used at radius 7 for its inset border. The shipped
// radii have moved since; the constant is what those files measured.
const cornerFactor = 0.4478

// maskCornerInset is how far inside the background's corner the mask's corner
// arc is drawn, in SVG pixels.
//
// The mask is a region, not artwork. KSvg renders the mask- prefixed frame and
// thresholds it to one bit — QRegion(QBitmap(alphaMask.mask())) in
// ksvg/framesvg.cpp — so the mask's corner is a hard staircase where the
// background's is antialiased. Drawn at the same radius the staircase lands
// outside the background's smooth edge, and the pixels between the two are mask
// the artwork does not cover: a blurred fringe tracing a slightly tighter curve
// than the corner it belongs to. Breeze carries the same pixel for the same
// reason and says so in a note inside its own background.svg:
//
//	The corners of the mask are 1px smaller because they are not antialiased,
//	whereas the svg corners are; if the mask is the same size of the svg, then,
//	some of the corner pixels of the mask will be visible even though they
//	should be covered by the svg.
//
// One pixel because it covers an antialiasing ramp, which is a pixel wide
// whatever the radius is — so this does not move when the radii do. Only the
// corners take it: Plasma tried shrinking the whole mask first and got a dark
// outline along every edge with the wallpaper showing through the outermost
// pixel, which is why the mask's edge and centre tiles are the background's own.
// See https://invent.kde.org/frameworks/plasma-framework/-/merge_requests/644.
const maskCornerInset = 1

// frame describes a nine-tile FrameSvg: four corners, four edges and a centre,
// laid out in a square of Size with tiles Tile across, followed by the margin
// hints Plasma reads the frame's insets from.
//
// Tile has to be at least Radius: the corner arc is drawn inside its own tile,
// and a radius overflowing it would spill into the stretched edge tile beside.
//
// Two idioms appear in the theme and they are not interchangeable. Dialog-like
// frames leave a straight run between the corner arc and the tile edge, because
// Tile exceeds Radius. The panel's tiles are exactly the radius, so its corners
// collapse to an arc and a single closing line. Emitting one from the other's
// template would change the path data, so each keeps its own builder.
type frame struct {
	Size   int     // the frame square: 44 for dialogs, 40 for the panel
	Canvas int     // SVG height, which leaves room for the hint row
	Tile   int     // corner and edge tile size
	Radius float64 // corner radius

	Fallback string // stylesheet colour, shown by editors and ignored at paint time
	Opacity  string // fill-opacity, or empty for fully opaque

	// Border is the opacity of a one-pixel outline drawn in the colour scheme's
	// text colour, written literally as the artwork writes its opacities, or
	// empty for a frame with no outline. A literal colour would bake one
	// palette's border into artwork every palette shares, so the outline goes
	// through the stylesheet like everything else here.
	//
	// The background covers the tile's whole shape and the outline is laid over
	// it as a ring, so the two antialiased curves stay concentric and the border
	// pixel is the surface tinted by the outline. The window decoration stacks
	// these the other way round — outline first, background inset over it — and
	// it can, because its border is an opaque literal. An outline that is a
	// tenth of anything needs the surface underneath it to be a tenth of; with
	// nothing under it the frame's outermost pixel is ninety percent
	// transparent, which is a gap rather than an edge.
	Border         string
	BorderFallback string // stylesheet colour for the outline

	// Shadow emits a shadow- prefixed tile set that paints nothing.
	//
	// Plasma's tooltip draws this artwork twice: once with prefix "shadow" for
	// the drop shadow and once plain for the background. KSvg decides a prefix
	// exists by looking for <prefix>-center, and clears the prefix when it finds
	// none — so a sheet with no shadow tiles draws its own frame a second time,
	// inflated by the frame's margins. That was invisible while the frame was a
	// flat fill and is two concentric borders the moment it has an outline.
	//
	// Shipping the prefix empty is the only way to say "there is no shadow
	// here", for the same reason widgets/scrollwidget.svg exists at all.
	Shadow bool

	Mask     bool // emit the mask- copies used for blur regions
	HintSize int  // size of the four margin hints
	HintY    int  // baseline the hints sit on

	// Inline puts every tile on one line, as the panel artwork does. The
	// difference is cosmetic but it is load-bearing for byte-for-byte
	// regeneration, so it is described rather than normalised away.
	Inline bool
}

func (f frame) render() string {
	var b strings.Builder

	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d">`+"\n", f.Size, f.Canvas)
	b.WriteString(`<defs><style type="text/css" id="current-color-scheme">` + "\n")
	fmt.Fprintf(&b, ".ColorScheme-Background { color:%s; }\n", f.Fallback)
	if f.Border != "" {
		fmt.Fprintf(&b, ".ColorScheme-Text { color:%s; }\n", f.BorderFallback)
	}
	b.WriteString("</style></defs>\n")

	fill := "fill:currentColor"
	if f.Opacity != "" {
		fill += ";fill-opacity:" + f.Opacity
	}
	paint := `class="ColorScheme-Background" style="` + fill + `"`

	sep := "\n"
	if f.Inline {
		sep = ""
	}

	for _, tile := range f.tiles(paint, f.Border) {
		b.WriteString(tile + sep)
	}
	if f.Inline {
		b.WriteString("\n")
	}

	// The mask is the blur region rather than artwork. Its edges stay flush with
	// the frame's, and only its corners pull in — see maskCornerInset.
	if f.Mask {
		for _, tile := range f.maskTiles(`style="fill:#ffffff"`) {
			b.WriteString(strings.Replace(tile, `id="`, `id="mask-`, 1) + "\n")
		}
	}

	// The empty shadow prefix, tiles and margin hints together: the hints repeat
	// the frame's own, so the prefix reports the insets KSvg already derives
	// from the fallback and the tooltip's geometry does not move.
	if f.Shadow {
		for _, el := range append(f.nullTiles("shadow-"), f.hints("shadow-")...) {
			b.WriteString(el + "\n")
		}
	}

	for _, hint := range f.hints("") {
		b.WriteString(hint)
		if !f.Inline {
			b.WriteString("\n")
		}
	}
	if f.Inline {
		b.WriteString("\n")
	}

	b.WriteString("</svg>\n")

	return b.String()
}

// tiles returns the nine elements in the order the artwork uses: corners first,
// then edges, then the centre. border is the outline's opacity, or empty for a
// frame that has none.
func (f frame) tiles(paint, border string) []string {
	tl, tr, bl, br := f.corners(0)

	return f.tilesFrom(paint, border, tl, tr, bl, br)
}

// maskTiles returns the nine tiles of the mask- prefix: the frame's own
// geometry everywhere except the corners, which are pulled in by
// maskCornerInset. The mask carries no outline — it is a region, and an outline
// drawn into it would punch a ring out of the blur instead of showing up as a
// border.
func (f frame) maskTiles(paint string) []string {
	tl, tr, bl, br := f.maskCorners()

	return f.tilesFrom(paint, "", tl, tr, bl, br)
}

// tilesFrom lays the nine elements out around corner paths it is handed.
func (f frame) tilesFrom(paint, border string, tl, tr, bl, br string) []string {
	t := f.Tile
	s := f.Size
	inner := s - 2*t

	rect := func(id string, x, y, w, h int) string {
		return fmt.Sprintf(`<rect id="%s" x="%d" y="%d" width="%d" height="%d" %s/>`, id, x, y, w, h, paint)
	}

	if border == "" {
		group := func(id, d string) string {
			return fmt.Sprintf(`<g id="%s"><path d="%s" %s/></g>`, id, d, paint)
		}

		return []string{
			group("topleft", tl),
			group("topright", tr),
			group("bottomleft", bl),
			group("bottomright", br),
			rect("top", t, 0, inner, t),
			rect("bottom", t, s-t, inner, t),
			rect("left", 0, t, t, inner),
			rect("right", s-t, t, t, inner),
			rect("center", t, t, inner, inner),
		}
	}

	// An outlined tile is two shapes under the one id Plasma looks up: the
	// background covering the whole tile, and the outline over it, a ring a
	// pixel wide against the frame's outer edge and nothing at all along the
	// tile boundaries it shares with its neighbours.
	itl, itr, ibl, ibr := f.corners(1)
	edge := fmt.Sprintf(`class="ColorScheme-Text" style="fill:currentColor" opacity="%s"`, border)

	// A corner's ring is its two paths in one, wound so the even-odd rule drops
	// the inner shape out of the outer one. They already meet flush along the
	// shared tile boundaries, where the two coincident edges cancel and leave
	// the ring open — the same three sides the strips leave bare.
	corner := func(id, outer, inset string) string {
		return fmt.Sprintf(`<g id="%s"><path d="%s" %s/><path d="%s %s" fill-rule="evenodd" %s/></g>`,
			id, outer, paint, outer, inset, edge)
	}
	strip := func(id string, x, y, w, h, bx, by, bw, bh int) string {
		return fmt.Sprintf(
			`<g id="%s"><rect x="%d" y="%d" width="%d" height="%d" %s/><rect x="%d" y="%d" width="%d" height="%d" %s/></g>`,
			id, x, y, w, h, paint, bx, by, bw, bh, edge)
	}

	return []string{
		corner("topleft", tl, itl),
		corner("topright", tr, itr),
		corner("bottomleft", bl, ibl),
		corner("bottomright", br, ibr),
		strip("top", t, 0, inner, t, t, 0, inner, 1),
		strip("bottom", t, s-t, inner, t, t, s-1, inner, 1),
		strip("left", 0, t, t, inner, 0, t, 1, inner),
		strip("right", s-t, t, t, inner, s-1, t, 1, inner),
		rect("center", t, t, inner, inner),
	}
}

// hints returns the four margin rects Plasma reads the frame's insets from,
// under the given id prefix.
func (f frame) hints(prefix string) []string {
	out := make([]string, 0, 4)

	for i, name := range []string{"top", "bottom", "left", "right"} {
		out = append(out, fmt.Sprintf(
			`<rect id="%shint-%s-margin" x="%d" y="%d" width="%d" height="%d" style="fill:none"/>`,
			prefix, name, i*10, f.HintY, f.HintSize, f.HintSize))
	}

	return out
}

// nullTiles returns the nine tiles of a prefix that paints nothing: bare rects
// on the frame's own tile boxes, which is all KSvg needs to find the prefix and
// take its geometry from it. They sit on the same coordinates as the painted
// tiles, as the mask- copies already do — an element is fetched by id, so what
// it overlaps in the sheet never comes up.
//
// The corners are rects rather than the painted paths because a tile that draws
// nothing has no corner to round.
func (f frame) nullTiles(prefix string) []string {
	t := f.Tile
	s := f.Size
	inner := s - 2*t

	rect := func(name string, x, y, w, h int) string {
		return fmt.Sprintf(`<rect id="%s%s" x="%d" y="%d" width="%d" height="%d" style="fill:none"/>`,
			prefix, name, x, y, w, h)
	}

	return []string{
		rect("topleft", 0, 0, t, t),
		rect("topright", s-t, 0, t, t),
		rect("bottomleft", 0, s-t, t, t),
		rect("bottomright", s-t, s-t, t, t),
		rect("top", t, 0, inner, t),
		rect("bottom", t, s-t, inner, t),
		rect("left", 0, t, t, inner),
		rect("right", s-t, t, t, inner),
		rect("center", t, t, inner, inner),
	}
}

// corners returns the four corner paths, pulled in from the frame's outer edge
// by inset pixels.
//
// The idiom is picked from the frame's own radius rather than the inset one. An
// inset path is the same corner drawn a pixel in, and re-deciding on the inset
// radius would hand a bordered frame's inner path to a different template than
// its outer one.
func (f frame) corners(inset float64) (tl, tr, bl, br string) {
	switch {
	case f.Radius == 0:
		// A square corner is the same filled tile in either idiom, and emitting
		// it as a zero-length cubic would ship curves that render flat.
		return f.squareCorners(inset)
	case f.Tile == int(f.Radius):
		return f.panelCorners(inset)
	}

	return f.dialogCorners(inset)
}

// maskCorners returns the four corner paths for the mask: the frame's own
// corner with its arc redrawn maskCornerInset pixels further in, stepped back
// out to the outer edge at each end of the arc.
//
// The step is where this parts company with corners(inset), which pulls the
// straight runs in along with the arc. That is right for the outline ring,
// which is drawn inside the artwork, and wrong here: the mask has to keep
// covering the outermost pixel everywhere the corner is not curving, or the
// frame loses its edge to the very artefact this is fixing. So the runs stay at
// the frame's edge and the arc alone moves, which leaves a one-pixel step at
// each end of it — on the straight part of the edge, where the background is at
// full coverage and the mask boundary does not show.
//
// A square corner has no antialiased curve to hide behind and is returned as
// it is.
func (f frame) maskCorners() (tl, tr, bl, br string) {
	if f.Radius == 0 {
		return f.corners(0)
	}

	// The idiom is picked the same way corners() picks it, and for the same
	// reason: a frame whose tile is exactly its radius has no straight run for
	// the step to sit on.
	if f.Tile == int(f.Radius) {
		return f.maskPanelCorners()
	}

	const d = maskCornerInset

	sz := float64(f.Size)
	t := float64(f.Tile)
	rr := f.Radius // where the background's arc meets the straight run
	r := rr - d    // the mask's own arc
	c := r * cornerFactor
	far := sz - d

	tl = fmt.Sprintf("M%s,%s L%s,%s L%s,%s C%s,%s %s,%s %s,%s L%s,%s L%s,%s L%s,%s Z",
		n(0), n(t), n(0), n(rr), n(d), n(rr),
		n(d), n(d+c), n(d+c), n(d), n(rr), n(d),
		n(rr), n(0), n(t), n(0), n(t), n(t))

	tr = fmt.Sprintf("M%s,%s L%s,%s L%s,%s C%s,%s %s,%s %s,%s L%s,%s L%s,%s L%s,%s Z",
		n(sz-t), n(0), n(sz-rr), n(0), n(sz-rr), n(d),
		n(far-c), n(d), n(far), n(d+c), n(far), n(rr),
		n(sz), n(rr), n(sz), n(t), n(sz-t), n(t))

	bl = fmt.Sprintf("M%s,%s L%s,%s L%s,%s L%s,%s L%s,%s C%s,%s %s,%s %s,%s L%s,%s Z",
		n(0), n(sz-t), n(t), n(sz-t), n(t), n(sz), n(rr), n(sz), n(rr), n(far),
		n(d+c), n(far), n(d), n(far-c), n(d), n(sz-rr),
		n(0), n(sz-rr))

	br = fmt.Sprintf("M%s,%s L%s,%s L%s,%s L%s,%s C%s,%s %s,%s %s,%s L%s,%s L%s,%s Z",
		n(sz-t), n(sz-t), n(sz), n(sz-t), n(sz), n(sz-rr), n(far), n(sz-rr),
		n(far), n(far-c), n(far-c), n(far), n(sz-rr), n(far),
		n(sz-rr), n(sz), n(sz-t), n(sz))

	return tl, tr, bl, br
}

// maskPanelCorners is maskCorners for a frame whose tile is exactly its radius.
//
// There is no straight run to step off here: the arc's ends are the points where
// the corner tile meets its neighbours, so pulling the arc in pulls it away from
// them. The path still travels to those points, along edges it encloses no area
// on, which keeps the tile reaching its neighbours without the mask covering the
// pixel the arc has vacated. Breeze reaches the same shape the same way, and
// accepts what it leaves: a notch of a pixel at each end of the arc, which is the
// residual darkening its merge request owns up to.
func (f frame) maskPanelCorners() (tl, tr, bl, br string) {
	const d = maskCornerInset

	sz := float64(f.Size)
	t := float64(f.Tile)
	r := f.Radius - d
	c := r * cornerFactor
	far := sz - d

	tl = fmt.Sprintf("M%s,%s L%s,%s L%s,%s L%s,%s L%s,%s C%s,%s %s,%s %s,%s Z",
		n(d), n(t), n(0), n(t), n(t), n(t), n(t), n(0), n(t), n(d),
		n(d+c), n(d), n(d), n(d+c), n(d), n(t))

	tr = fmt.Sprintf("M%s,%s L%s,%s L%s,%s L%s,%s L%s,%s C%s,%s %s,%s %s,%s Z",
		n(sz-t), n(d), n(sz-t), n(0), n(sz-t), n(t), n(sz), n(t), n(far), n(t),
		n(far), n(d+c), n(far-c), n(d), n(sz-t), n(d))

	bl = fmt.Sprintf("M%s,%s L%s,%s L%s,%s L%s,%s L%s,%s C%s,%s %s,%s %s,%s Z",
		n(d), n(sz-t), n(0), n(sz-t), n(t), n(sz-t), n(t), n(sz), n(t), n(far),
		n(d+c), n(far), n(d), n(far-c), n(d), n(sz-t))

	br = fmt.Sprintf("M%s,%s L%s,%s L%s,%s L%s,%s L%s,%s C%s,%s %s,%s %s,%s Z",
		n(sz-t), n(far), n(sz-t), n(sz), n(sz-t), n(sz-t), n(sz), n(sz-t), n(far), n(sz-t),
		n(far), n(far-c), n(far-c), n(far), n(sz-t), n(far))

	return tl, tr, bl, br
}

// squareCorners builds corners with no radius at all: four plain tiles.
func (f frame) squareCorners(d float64) (tl, tr, bl, br string) {
	s := float64(f.Size)
	t := float64(f.Tile)
	far := s - d // the outer edge, pulled in by the inset

	tl = fmt.Sprintf("M%s,%s L%s,%s L%s,%s L%s,%s Z", n(d), n(t), n(d), n(d), n(t), n(d), n(t), n(t))
	tr = fmt.Sprintf("M%s,%s L%s,%s L%s,%s L%s,%s Z", n(s-t), n(d), n(far), n(d), n(far), n(t), n(s-t), n(t))
	bl = fmt.Sprintf("M%s,%s L%s,%s L%s,%s L%s,%s Z", n(d), n(s-t), n(t), n(s-t), n(t), n(far), n(d), n(far))
	br = fmt.Sprintf("M%s,%s L%s,%s L%s,%s L%s,%s Z",
		n(s-t), n(s-t), n(far), n(s-t), n(far), n(far), n(s-t), n(far))

	return tl, tr, bl, br
}

// dialogCorners builds corners for frames whose tile is larger than the radius,
// leaving a straight run between the end of the arc and the tile boundary.
func (f frame) dialogCorners(d float64) (tl, tr, bl, br string) {
	s := float64(f.Size)
	t := float64(f.Tile)
	r := f.Radius - d
	c := r * cornerFactor
	far := s - d // the outer edge, pulled in by the inset

	tl = fmt.Sprintf("M%s,%s L%s,%s C%s,%s %s,%s %s,%s L%s,%s L%s,%s Z",
		n(d), n(t), n(d), n(d+r), n(d), n(d+c), n(d+c), n(d), n(d+r), n(d), n(t), n(d), n(t), n(t))
	tr = fmt.Sprintf("M%s,%s L%s,%s C%s,%s %s,%s %s,%s L%s,%s L%s,%s Z",
		n(s-t), n(d), n(far-r), n(d), n(far-c), n(d), n(far), n(d+c), n(far), n(d+r), n(far), n(t), n(s-t), n(t))
	bl = fmt.Sprintf("M%s,%s L%s,%s L%s,%s L%s,%s C%s,%s %s,%s %s,%s Z",
		n(d), n(s-t), n(t), n(s-t), n(t), n(far), n(d+r), n(far), n(d+c), n(far), n(d), n(far-c), n(d), n(far-r))
	br = fmt.Sprintf("M%s,%s L%s,%s L%s,%s C%s,%s %s,%s %s,%s L%s,%s Z",
		n(s-t), n(s-t), n(far), n(s-t), n(far), n(far-r), n(far), n(far-c),
		n(far-c), n(far), n(far-r), n(far), n(s-t), n(far))

	return tl, tr, bl, br
}

// panelCorners builds corners for frames whose tile is exactly the radius, where
// the straight run vanishes and the path closes directly off the arc.
func (f frame) panelCorners(d float64) (tl, tr, bl, br string) {
	s := float64(f.Size)
	t := float64(f.Tile)
	r := f.Radius - d
	c := r * cornerFactor
	far := s - d // the outer edge, pulled in by the inset

	tl = fmt.Sprintf("M%s,%s C%s,%s %s,%s %s,%s L%s,%s Z",
		n(d), n(d+r), n(d), n(d+c), n(d+c), n(d), n(d+r), n(d), n(t), n(t))
	tr = fmt.Sprintf("M%s,%s C%s,%s %s,%s %s,%s L%s,%s Z",
		n(s-t), n(d), n(far-c), n(d), n(far), n(d+c), n(far), n(d+r), n(s-t), n(t))
	bl = fmt.Sprintf("M%s,%s L%s,%s L%s,%s C%s,%s %s,%s %s,%s Z",
		n(d), n(s-t), n(t), n(s-t), n(t), n(far), n(d+c), n(far), n(d), n(far-c), n(d), n(s-t))
	br = fmt.Sprintf("M%s,%s L%s,%s C%s,%s %s,%s %s,%s Z",
		n(s-t), n(s-t), n(far), n(s-t), n(far), n(far-c), n(far-c), n(far), n(s-t), n(far))

	return tl, tr, bl, br
}

// exact formats a value at whatever precision it needs, for the handful of
// numbers that are scale factors rather than coordinates: rounding a scale to
// three decimals resizes what it is applied to.
func exact(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

// n formats a coordinate the way the artwork writes them: three decimals at
// most, and no trailing zeros or bare point.
func n(v float64) string {
	s := strconv.FormatFloat(v, 'f', 3, 64)
	s = strings.TrimRight(s, "0")

	return strings.TrimRight(s, ".")
}
