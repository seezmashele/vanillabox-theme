package main

import (
	"fmt"
	"math"
	"strings"
)

// Aurorae button glyphs. These are artwork rather than anything derived from a
// token, so they are held verbatim and only their size, colour and state
// opacities are decided here.
//
// They are Phosphor Icons 2.1.1 (MIT), bold weight: x, minus, square and copy.
// Bold rather than regular because the regular weight's 16-unit stroke renders
// at 0.75px at the button's glyph size, which antialiases to a soft grey line;
// bold's 24-unit stroke lands just over a pixel.
const (
	glyphClose    = "M208.49,191.51a12,12,0,0,1-17,17L128,145,64.49,208.49a12,12,0,0,1-17-17L111,128,47.51,64.49a12,12,0,0,1,17-17L128,111l63.51-63.52a12,12,0,0,1,17,17L145,128Z"
	glyphMinimize = "M228,128a12,12,0,0,1-12,12H40a12,12,0,0,1,0-24H216A12,12,0,0,1,228,128Z"
	glyphMaximize = "M208,28H48A20,20,0,0,0,28,48V208a20,20,0,0,0,20,20H208a20,20,0,0,0,20-20V48A20,20,0,0,0,208,28Zm-4,176H52V52H204Z"
	glyphRestore  = "M216,28H88A12,12,0,0,0,76,40V76H40A12,12,0,0,0,28,88V216a12,12,0,0,0,12,12H168a12,12,0,0,0,12-12V180h36a12,12,0,0,0,12-12V40A12,12,0,0,0,216,28ZM156,204H52V100H156Zm48-48H180V88a12,12,0,0,0-12-12H100V52H204Z"
)

// decorationNote explains why the bottom corners are square. It is reproduced
// verbatim because the constraint it records is not obvious from the artwork,
// and someone will otherwise try to round them again.
const decorationNote = `<!-- The top two corners carry a 10px radius, drawn inside a corner tile at
     least that wide. Each corner draws the border colour first and lays the
     background over it inset by 1px at radius 9, so the two antialiased edges
     stay concentric and no seam shows along the curve.

     The bottom corners are square, and deliberately so. Rounding them needs a
     bottom border to draw the curve in, and anything narrower than the radius
     lets the client window's square corner show through it. Breeze avoids that
     by calling KDecoration3::Decoration::setBorderRadius, which makes KWin clip
     the client itself — neither Aurorae plugin exposes that API, so an SVG
     theme cannot round the bottom without a visible bottom strip. -->`

// button renders one Aurorae titlebar button: a 24x24 canvas holding four
// states. The resting and deactivated states show only the glyph; hover and
// pressed lay a plate beneath it, rounded only if the style asks for it.
type button struct {
	Glyph string

	// GlyphSize is how big the symbol should be on screen, in pixels, and Box is
	// the square the tile is scaled into. Aurorae stretches the 24x24 tile to the
	// button box, so a size given in tile units would mean something different
	// for every button size; giving it in rendered pixels means the number in the
	// tokens is the number you measure.
	GlyphSize float64
	Box       float64

	PlateFill      string
	HoverOpacity   string
	PressedOpacity string
	Radius         float64

	GlyphFill  string
	RestOpen   string // glyph opacity at rest
	DimOpacity string // glyph opacity when the window is inactive
}

func (b button) render() string {
	// A fully transparent rect keeps the button's hit area at full tile size
	// whatever the glyph covers.
	hit := `<rect x="0" y="0" width="24" height="24" fill="#000" fill-opacity="0"/>`

	// The glyph paths are drawn on a 256-unit grid. Converting the wanted screen
	// size back into tile units undoes the stretch Aurorae will apply, and the
	// inset that centres it is half of whatever the tile has left over.
	glyph := func(opacity string) string {
		tile := b.GlyphSize * 24 / b.Box
		inset := (24 - tile) / 2

		return fmt.Sprintf(
			`<g transform="translate(%s,%s) scale(%s)"><path d="%s" fill="%s" opacity="%s"/></g>`,
			n(inset), n(inset), exact(tile/256), b.Glyph, b.GlyphFill, opacity)
	}
	plate := func(opacity string) string {
		corners := ""
		if b.Radius > 0 {
			corners = fmt.Sprintf(` rx="%s" ry="%s"`, n(b.Radius), n(b.Radius))
		}

		return fmt.Sprintf(`<rect x="0" y="0" width="24" height="24" fill="%s"%s opacity="%s"/>`,
			b.PlateFill, corners, opacity)
	}
	group := func(id string, body ...string) string {
		return fmt.Sprintf(`<g id="%s-center">%s</g>`, id, strings.Join(body, ""))
	}

	var s strings.Builder

	s.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24">` + "\n")
	s.WriteString(group("active", hit, glyph(b.RestOpen)))
	s.WriteString(group("hover", hit, plate(b.HoverOpacity), glyph("1")))
	s.WriteString(group("pressed", hit, plate(b.PressedOpacity), glyph("1")))
	s.WriteString(group("deactivated", hit, glyph(b.DimOpacity)))
	s.WriteString("\n</svg>\n")

	return s.String()
}

// circleButton renders a macOS-style titlebar light: a filled circle, grey at
// rest and coloured on hover, with no glyph at any point.
//
// Hover is per-button and cannot be otherwise. On macOS, pointing at any of the
// three lights up all three; Aurorae renders each button from its own SVG with
// no knowledge of its neighbours, so here only the one under the pointer takes
// colour. That is a limit of the format, not a choice.
type circleButton struct {
	Radius  float64
	Rest    string // grey, window focused and pointer elsewhere
	Dim     string // window unfocused
	Hover   string // the light's own colour
	Pressed string // opacity applied to the hover colour
}

func (c circleButton) render() string {
	hit := `<rect x="0" y="0" width="24" height="24" fill="#000" fill-opacity="0"/>`

	circle := func(fill, opacity string) string {
		alpha := ""
		if opacity != "" {
			alpha = fmt.Sprintf(` opacity="%s"`, opacity)
		}

		return fmt.Sprintf(`<circle cx="12" cy="12" r="%s" fill="%s"%s/>`, n(c.Radius), fill, alpha)
	}
	group := func(id string, body ...string) string {
		return fmt.Sprintf(`<g id="%s-center">%s</g>`, id, strings.Join(body, ""))
	}

	var s strings.Builder

	s.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24">` + "\n")
	s.WriteString(group("active", hit, circle(c.Rest, "")))
	s.WriteString(group("hover", hit, circle(c.Hover, "")))
	s.WriteString(group("pressed", hit, circle(c.Hover, c.Pressed)))
	s.WriteString(group("deactivated", hit, circle(c.Dim, "")))
	s.WriteString("\n</svg>\n")

	return s.String()
}

// auroraeRC renders the decoration's layout file. Its metrics depend on both
// the decoration shape and the button style, which is why it is the one file
// resolved from a pair of axes rather than laid down by an overlay.
func auroraeRC(palette map[string]string, style buttonStyle, titleHeight int, sh shadowSpec) string {
	// Padding is the frame Aurorae leaves around the decoration for shadows and
	// resize handles, and the title edges are the insets inside it.
	// Aurorae fills the padding band with the decoration's nine tiles, so the
	// band is both the frame's own one-pixel border and the room the shadow
	// needs. Breeze's bands are asymmetric, the shadow being offset downwards.
	padL, padR, padT, padB := sh.padding()
	padL, padR, padT, padB = padL+1, padR+1, padT+1, padB+1

	const (
		titleEdgeTop    = 0
		titleEdgeBottom = 0
		titleEdgeLeft   = 8
		titleEdgeRight  = 6
	)

	// Aurorae places a button at ButtonMarginTop from the top of the titlebar and
	// leaves the rest of the slack below it, so a zero margin sits every button
	// high. Deriving the margin centres whatever height a style asks for.
	marginTop := (titleHeight-style.Height)/2 + style.NudgeTop

	// A maximised window is laid out from an entirely separate set of keys, and
	// every one of them defaults to zero rather than to its ordinary
	// counterpart. AuroraeButtonGroup.qml computes the button offset as
	//
	//     maximised ? titleEdgeTopMaximized + buttonMarginTopMaximized
	//               : titleEdgeTop + padding.top + buttonMarginTop
	//
	// so leaving them unset moved every button on maximise.
	//
	// The maximised keys mirror the ordinary ones exactly, and deliberately do
	// not add the padding the other branch adds. Padding is the frame outside the
	// window, so the decoration's origin sits that far out from the window edge
	// when a window is not maximised and exactly on it when it is: the ordinary
	// branch adds padding to reach the same place the maximised branch already
	// starts from. Adding it to both put every button a pixel out.
	//
	// The edges also feed the titlebar height — borderTopMaximized is
	// titleEdgeTopMaximized + TitleHeight + titleEdgeBottomMaximized — so a
	// padded edge would additionally make a maximised titlebar 2px taller than a
	// restored one.
	maxTop := titleEdgeTop
	maxBottom := titleEdgeBottom
	maxLeft := titleEdgeLeft
	maxRight := titleEdgeRight

	// The application icon is the only button whose width is set separately, and
	// the left edge is wider than the right so it is not tucked into the corner.
	menuWidth := style.MenuWidth
	if menuWidth == 0 {
		menuWidth = style.Width
	}

	return fmt.Sprintf(`[General]
ActiveTextColor=%s
InactiveTextColor=%s
TitleAlignment=Center
TitleVerticalAlignment=Center
Animation=0

[Layout]
BorderLeft=0
BorderRight=0
BorderBottom=0
TitleEdgeTop=%d
TitleEdgeBottom=%d
TitleEdgeLeft=%d
TitleEdgeRight=%d
TitleEdgeTopMaximized=%d
TitleEdgeBottomMaximized=%d
TitleEdgeLeftMaximized=%d
TitleEdgeRightMaximized=%d
TitleBorderLeft=4
TitleBorderRight=4
TitleHeight=%d
ButtonWidth=%d
ButtonWidthMenu=%d
ButtonHeight=%d
ButtonSpacing=%d
ButtonMarginTop=%d
ButtonMarginTopMaximized=%d
ExplicitButtonSpacer=6
PaddingTop=%d
PaddingBottom=%d
PaddingLeft=%d
PaddingRight=%d
`, rgb(palette["text"]), rgb(palette["textInactive"]),
		titleEdgeTop, titleEdgeBottom, titleEdgeLeft, titleEdgeRight,
		maxTop, maxBottom, maxLeft, maxRight,
		titleHeight, style.Width, menuWidth, style.Height, style.Spacing,
		marginTop, marginTop,
		padT, padB, padL, padR)
}

// lookAndFeelDefaults renders the settings KDE applies when the global theme is
// chosen. The accent lives here, and so would a button order: Mac-style buttons
// sit on the left, which is a KWin setting rather than anything in the Aurorae
// theme. An empty order leaves KWin's own default in place.
func lookAndFeelDefaults(accent, buttonsOnLeft, buttonsOnRight string) string {
	var b strings.Builder

	fmt.Fprintf(&b, `[kdeglobals][General]
ColorScheme=VanillaBoxDark
accentColorFromWallpaper=false
AccentColor=%s
`, rgb(accent))

	fmt.Fprint(&b, `
[plasmarc][Theme]
name=vanilla-box-dark

[kwinrc][org.kde.kdecoration2]
library=org.kde.kwin.aurorae.v2
theme=__aurorae__svg__VanillaBoxDark
BorderSize=None
BorderSizeAuto=false
`)

	if buttonsOnLeft != "" || buttonsOnRight != "" {
		fmt.Fprintf(&b, "ButtonsOnLeft=%s\nButtonsOnRight=%s\n", buttonsOnLeft, buttonsOnRight)
	}

	return b.String()
}

// decoration renders the window frame and the shadow band around it.
//
// The sheet is a nine-slice whose fixed tiles have to be big enough to hold
// everything that varies. That is what sizes them: a Gaussian shadow's corner
// settles to its edge value only after about 3*sigma, and a stretched tile
// cannot vary, so the corner tiles reach that far inside the shadow's box. The
// excess sits over the client, which draws on top of it.
type decoration struct {
	Width   int // the frame's own nine-slice width: 2*Tile plus a stretchable run
	Tile    int // the frame's corner tile; at least the radius, and at least Overlap+Reach
	TitleH  int // titlebar row including its top border
	BodyH   int // the frame's body, between titlebar and bottom border
	Radius  float64
	Border  string
	Backgnd string

	// Breeze's padding, asymmetric because its shadow is offset downwards.
	PadL, PadR, PadT, PadB int

	// Overlap is Metrics::Shadow_Overlap: Breeze deflates each shadow box by
	// this much, so the shadow starts just inside the window.
	Overlap int

	// Reach is how far inside a box the profile has to run before it is
	// indistinguishable from full. The fixed tiles are sized from it.
	Reach int

	Layers      []shadowLayer
	ShadowColor string
}

const stateGap = 4

func (d decoration) frameH() int  { return d.TitleH + d.BodyH + 1 }
func (d decoration) canvasW() int { return d.PadL + d.Width + d.PadR }
func (d decoration) stateH() int  { return d.PadT + d.frameH() + d.PadB }
func (d decoration) cornerW() int { return d.PadL + d.Tile }

// x0 and x1 are the shadow boxes' side edges, shared by every layer since the
// offset is vertical only.
func (d decoration) x0() int { return d.PadL + d.Overlap }
func (d decoration) x1() int { return d.PadL + d.Width - d.Overlap }

// y0 and y1 are one layer's top and bottom edges, on the sheet. They take the
// state's origin because the two states sit at different heights: everything
// else here is relative to the frame, but a gradient is in user space and has
// to be placed where the state actually is.
func (d decoration) y0(l shadowLayer, y int) int { return y + d.PadT + d.Overlap + l.DY }
func (d decoration) y1(l shadowLayer, y int) int {
	return y + d.PadT + d.frameH() - d.Overlap + l.DY
}

// row1 and row2 are where the stretchable middle row begins and ends. The
// bottom row reaches up past the frame's own bottom so the shadow's lower
// corner has fixed tile to settle in; the client covers the overlap.
func (d decoration) row1() int { return d.PadT + d.TitleH }
func (d decoration) row2() int {
	lowest := d.stateH()
	for _, l := range d.Layers {
		if v := d.y1(l, 0) - d.Reach; v < lowest {
			lowest = v
		}
	}
	if lowest <= d.row1() {
		lowest = d.row1() + 1
	}

	return lowest
}

// step is where the inactive state starts. It has to clear the active state, or
// the two are drawn over each other and their shadows add up.
func (d decoration) step() int { return d.stateH() + stateGap }

func (d decoration) render() string {
	var b strings.Builder

	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d">`+"\n",
		d.canvasW(), d.step()+d.stateH())
	b.WriteString(decorationNote + "\n")

	states := []struct {
		Prefix string
		Y      int
	}{{"", 0}, {"inactive-", d.step()}}

	var defs []string
	for _, st := range states {
		defs = append(defs, d.shadowDefs(st.Prefix, st.Y)...)
	}
	if len(defs) > 0 {
		b.WriteString("<defs>" + strings.Join(defs, "") + "</defs>\n")
	}

	for _, st := range states {
		for _, el := range d.state(st.Prefix, st.Y) {
			b.WriteString(el + "\n")
		}
	}

	b.WriteString("</svg>\n")

	return b.String()
}

func (d decoration) state(prefix string, y int) []string {
	w := d.canvasW()
	cw := d.cornerW()
	left, right := d.PadL, d.PadL+d.Width // the frame's own edges
	r1, r2 := y+d.row1(), y+d.row2()
	foot := y + d.PadT + d.frameH() // just past the frame's bottom border
	mid := w - 2*cw

	id := func(name string) string { return "decoration-" + prefix + name }
	border := fmt.Sprintf(`fill="%s"`, d.Border)
	backgnd := fmt.Sprintf(`class="ColorScheme-Background" fill="%s"`, d.Backgnd)

	rect := func(x, yy, ww, hh int, style string) string {
		if ww <= 0 || hh <= 0 {
			return ""
		}

		return fmt.Sprintf(`<rect x="%d" y="%d" width="%d" height="%d" %s/>`, x, yy, ww, hh, style)
	}

	outerL, outerR, innerL, innerR := d.corners(y)
	sh := d.shadowTiles(prefix, y)

	// The frame's straight left and right edges: one pixel of border, then
	// background out to the tile's inner edge.
	edgeL := func(top, h int) string {
		return rect(left, top, 1, h, border) + rect(left+1, top, cw-left-1, h, backgnd)
	}
	edgeR := func(top, h int) string {
		return rect(w-cw, top, right-1-(w-cw), h, backgnd) + rect(right-1, top, 1, h, border)
	}

	return []string{
		fmt.Sprintf(`<g id="%s">%s<path d="%s" %s/><path d="%s" %s/></g>`,
			id("topleft"), sh["topleft"], outerL, border, innerL, backgnd),
		fmt.Sprintf(`<g id="%s">%s%s%s</g>`, id("top"), sh["top"],
			rect(cw, y+d.PadT, mid, 1, border),
			rect(cw, y+d.PadT+1, mid, r1-(y+d.PadT+1), backgnd)),
		fmt.Sprintf(`<g id="%s">%s<path d="%s" %s/><path d="%s" %s/></g>`,
			id("topright"), sh["topright"], outerR, border, innerR, backgnd),

		fmt.Sprintf(`<g id="%s">%s%s</g>`, id("left"), sh["left"], edgeL(r1, r2-r1)),
		fmt.Sprintf(`<g id="%s">%s</g>`, id("center"), rect(cw, r1, mid, r2-r1, backgnd)),
		fmt.Sprintf(`<g id="%s">%s%s</g>`, id("right"), sh["right"], edgeR(r1, r2-r1)),

		// The bottom row starts above the frame's bottom, so it carries the last
		// of the body as well as the border.
		fmt.Sprintf(`<g id="%s">%s%s%s</g>`, id("bottomleft"), sh["bottomleft"],
			edgeL(r2, foot-1-r2), rect(left, foot-1, cw-left, 1, border)),
		fmt.Sprintf(`<g id="%s">%s%s%s</g>`, id("bottom"), sh["bottom"],
			rect(cw, r2, mid, foot-1-r2, backgnd), rect(cw, foot-1, mid, 1, border)),
		fmt.Sprintf(`<g id="%s">%s%s%s</g>`, id("bottomright"), sh["bottomright"],
			edgeR(r2, foot-1-r2), rect(w-cw, foot-1, right-(w-cw), 1, border)),
	}
}

// corners returns the outer border path and the inset background path for both
// top corners. Each runs from the frame's edge, round the curve, and straight
// on to the tile's inner edge — the tile is wider than the radius now, because
// the shadow needs it to be.
func (d decoration) corners(y int) (outerL, outerR, innerL, innerR string) {
	w := float64(d.canvasW())
	cw := float64(d.cornerW())
	left, right := float64(d.PadL), float64(d.PadL+d.Width)
	top, title := float64(y+d.PadT), float64(y+d.row1())
	r := d.Radius

	if r == 0 {
		outerL = fmt.Sprintf("M%s,%s L%s,%s L%s,%s L%s,%s Z", n(left), n(title), n(left), n(top), n(cw), n(top), n(cw), n(title))
		outerR = fmt.Sprintf("M%s,%s L%s,%s L%s,%s L%s,%s Z", n(right), n(title), n(right), n(top), n(w-cw), n(top), n(w-cw), n(title))
		innerL = fmt.Sprintf("M%s,%s L%s,%s L%s,%s L%s,%s Z", n(left+1), n(title), n(left+1), n(top+1), n(cw), n(top+1), n(cw), n(title))
		innerR = fmt.Sprintf("M%s,%s L%s,%s L%s,%s L%s,%s Z", n(right-1), n(title), n(right-1), n(top+1), n(w-cw), n(top+1), n(w-cw), n(title))

		return outerL, outerR, innerL, innerR
	}

	c := r * cornerFactor
	ri := r - 1
	ci := ri * cornerFactor

	outerL = fmt.Sprintf("M%s,%s L%s,%s C%s,%s %s,%s %s,%s L%s,%s L%s,%s Z",
		n(left), n(title), n(left), n(top+r), n(left), n(top+c), n(left+c), n(top), n(left+r), n(top), n(cw), n(top), n(cw), n(title))
	outerR = fmt.Sprintf("M%s,%s L%s,%s C%s,%s %s,%s %s,%s L%s,%s L%s,%s Z",
		n(right), n(title), n(right), n(top+r), n(right), n(top+c), n(right-c), n(top), n(right-r), n(top), n(w-cw), n(top), n(w-cw), n(title))
	innerL = fmt.Sprintf("M%s,%s L%s,%s C%s,%s %s,%s %s,%s L%s,%s L%s,%s Z",
		n(left+1), n(title), n(left+1), n(top+1+ri), n(left+1), n(top+1+ci), n(left+1+ci), n(top+1), n(left+1+ri), n(top+1), n(cw), n(top+1), n(cw), n(title))
	innerR = fmt.Sprintf("M%s,%s L%s,%s C%s,%s %s,%s %s,%s L%s,%s L%s,%s Z",
		n(right-1), n(title), n(right-1), n(top+1+ri), n(right-1), n(top+1+ci), n(right-1-ci), n(top+1), n(right-1-ri), n(top+1), n(w-cw), n(top+1), n(w-cw), n(title))

	return outerL, outerR, innerL, innerR
}

// shadowLayer is one of the two blurred boxes Breeze composites, as
// s_shadowParams names them.
//
// Breeze blurs with three box passes standing in for a Gaussian of
// stdDev = radius/2, and a Gaussian-blurred rectangle is separable: its field
// is the product of a horizontal profile and a vertical one, each of them the
// Gaussian integral of a single edge. That is what makes it drawable without a
// blur, which matters because QtSvg supports no filters.
//
// Taking the profile of the distance to the nearest edge instead — treating a
// corner as radial — is wrong, and wrong in a way that shows: on the diagonal
// it gives P(d) where the truth is P(d/sqrt2)^2, roughly twice the opacity, so
// the shadow bunches at the corners.
type shadowLayer struct {
	DY      int // the layer's box, relative to the window's edges
	Radius  int // Breeze's blur radius
	Opacity float64
}

// profile is the share of the layer's opacity d pixels past one edge. Negative
// d is inside the box, where it climbs towards 1.
func (l shadowLayer) profile(d float64) float64 {
	if l.Radius == 0 {
		return 0
	}

	return 0.5 * math.Erfc(d/(float64(l.Radius)*0.5*math.Sqrt2))
}

// sampleStep is how far apart resampled gradient stops sit, in pixels. SVG
// interpolates linearly between stops, so a stop per pixel keeps the error
// below a rounding step.
const sampleStep = 1.0

// ramp renders a gradient's stops, walking its geometry and asking the profile
// what belongs at each point. The stops carry the layer's opacity; the other
// axis is applied per strip as fill-opacity.
func (d decoration) ramp(l shadowLayer, length, d0, d1 float64) string {
	var b strings.Builder

	count := int(length/sampleStep) + 1
	if count < 8 {
		count = 8
	}

	for i := 0; i <= count; i++ {
		o := float64(i) / float64(count)
		fmt.Fprintf(&b, `<stop offset="%s" stop-color="%s" stop-opacity="%s"/>`,
			n(o), d.ShadowColor, n(l.Opacity*l.profile(d0+o*(d1-d0))))
	}

	return b.String()
}

// shadowDefs renders one state's gradients: for each layer, the horizontal
// profile at each side and the vertical profile at top and bottom. Four in all
// per layer — the corners reuse the horizontal pair rather than adding gradients
// of their own.
func (d decoration) shadowDefs(prefix string, y int) []string {
	if len(d.Layers) == 0 {
		return nil
	}

	var out []string
	w, reach := d.canvasW(), d.Reach

	for i, l := range d.Layers {
		id := func(name string) string { return fmt.Sprintf("shadow-%s%d-%s", prefix, i, name) }

		line := func(name string, x1, y1, x2, y2 int, d0, d1 float64) string {
			length := math.Hypot(float64(x2-x1), float64(y2-y1))

			return fmt.Sprintf(
				`<linearGradient id="%s" gradientUnits="userSpaceOnUse" x1="%d" y1="%d" x2="%d" y2="%d">%s</linearGradient>`,
				id(name), x1, y1, x2, y2, d.ramp(l, length, d0, d1))
		}

		x0, x1 := d.x0(), d.x1()
		y0, y1 := d.y0(l, y), d.y1(l, y)

		// Each gradient runs from Reach inside the box out to the sheet's edge,
		// so it covers both the settled part and the falloff.
		out = append(out,
			line("left", x0+reach, 0, 0, 0, -float64(reach), float64(x0)),
			line("right", x1-reach, 0, w, 0, -float64(reach), float64(w-x1)),
			line("top", 0, y0+reach, 0, y, -float64(reach), float64(y0-y)),
			line("bottom", 0, y1-reach, 0, y+d.stateH(), -float64(reach), float64(y+d.stateH()-y1)),
		)
	}

	return out
}

// shadowTiles returns each tile's slice of the shadow, keyed by tile name, with
// the layers stacked in order so they composite as Breeze's painter does.
//
// An edge tile varies along one axis only — the other profile has settled to one
// there — so it is a single rect carrying that gradient. A corner tile varies
// along both, and the product of two gradients is not something SVG can express;
// what it can express is a gradient scaled by a constant, so a corner is drawn
// as one-pixel strips, each carrying the horizontal gradient at a fill-opacity
// of the vertical profile for that row. Fill-opacity multiplies, which is
// exactly the operation wanted, and it costs no offscreen buffer.
func (d decoration) shadowTiles(prefix string, y int) map[string]string {
	out := map[string]string{}
	if len(d.Layers) == 0 {
		return out
	}

	w, cw := d.canvasW(), d.cornerW()
	r1, r2, bottom := y+d.row1(), y+d.row2(), y+d.stateH()

	add := func(name, frag string) {
		if frag != "" {
			out[name] += frag
		}
	}

	for i, l := range d.Layers {
		id := func(name string) string { return fmt.Sprintf("shadow-%s%d-%s", prefix, i, name) }

		rect := func(grad string, x, yy, ww, hh int) string {
			if ww <= 0 || hh <= 0 {
				return ""
			}

			return fmt.Sprintf(`<rect x="%d" y="%d" width="%d" height="%d" fill="url(#%s)"/>`,
				x, yy, ww, hh, id(grad))
		}

		// strips fills a corner: the side's gradient, row by row, dimmed by the
		// vertical profile at that row.
		strips := func(grad string, x, ww, top, h int, vertical func(int) float64) string {
			var b strings.Builder
			for k := top; k < top+h; k++ {
				o := vertical(k)
				if o <= 0.0005 {
					continue
				}
				fmt.Fprintf(&b, `<rect x="%d" y="%d" width="%d" height="1" fill="url(#%s)" fill-opacity="%s"/>`,
					x, k, ww, id(grad), n(o))
			}

			return b.String()
		}

		above := func(k int) float64 { return l.profile(float64(d.y0(l, y) - k)) }
		below := func(k int) float64 { return l.profile(float64(k - d.y1(l, y))) }

		add("topleft", strips("left", 0, cw, y, r1-y, above))
		add("topright", strips("right", w-cw, cw, y, r1-y, above))
		add("top", rect("top", cw, y, w-2*cw, r1-y))

		add("left", rect("left", 0, r1, cw, r2-r1))
		add("right", rect("right", w-cw, r1, cw, r2-r1))

		add("bottomleft", strips("left", 0, cw, r2, bottom-r2, below))
		add("bottomright", strips("right", w-cw, cw, r2, bottom-r2, below))
		add("bottom", rect("bottom", cw, r2, w-2*cw, bottom-r2))
	}

	return out
}
