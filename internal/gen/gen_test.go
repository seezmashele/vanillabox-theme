package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// TestGeneratedFilesAreCommitted is the generator's whole safety property: what
// the tokens describe and what the repository ships are the same bytes. It
// fails whenever assets/ is edited by hand instead of through spec/tokens.json,
// which is the mistake the generator exists to make impossible.
func TestGeneratedFilesAreCommitted(t *testing.T) {
	const root = "../.."

	tk, err := loadTokens(filepath.Join(root, "spec", "tokens.json"))
	if err != nil {
		t.Fatalf("loadTokens: %v", err)
	}

	files, err := allFiles(tk)
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	if len(files) == 0 {
		t.Fatal("the generator produced nothing")
	}

	for path, want := range files {
		got, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			t.Errorf("%s: %v — run go generate ./...", path, err)

			continue
		}

		if string(got) != want {
			t.Errorf("%s is out of date with spec/tokens.json — run go generate ./...", path)
		}
	}
}

// TestSquareSurfacesDropTheirCurves checks the axis the generator was built for
// before any variant depends on it: at radius zero the corner tiles must be
// plain lines, or a square variant would ship rounded artwork.
func TestSquareSurfacesDropTheirCurves(t *testing.T) {
	square := frame{
		Size: 44, Canvas: 60, Tile: 10, Radius: 0,
		Fallback: "#292929", Mask: true, HintSize: 8, HintY: 48,
	}

	svg := square.render()

	if want := `d="M0,10 L0,0 L10,0 L10,10 Z"`; !strings.Contains(svg, want) {
		t.Errorf("radius 0 should emit a plain corner tile, want %s in:\n%s", want, firstTile(svg))
	}
	if strings.Contains(svg, " C") {
		t.Error("radius 0 emitted a curve command")
	}
}

func firstTile(svg string) string {
	return element(svg, "topleft")
}

// element returns the one line of the sheet carrying the given tile id.
// It walks the markup rather than to the end of the line, because the panel
// writes all nine of its tiles onto one.
func element(svg, id string) string {
	at := strings.Index(svg, `id="`+id+`"`)
	if at < 0 {
		return ""
	}

	start := strings.LastIndex(svg[:at], "<")

	close := "/>"
	if strings.HasPrefix(svg[start:], "<g") {
		close = "</g>"
	}

	end := strings.Index(svg[start:], close)
	if end < 0 {
		return ""
	}

	return svg[start : start+end+len(close)]
}

// outlined stands in for the tooltip, the one container that carries a border.
// Its numbers are a fixture rather than the shipped ones — these tests are about
// how an outlined frame is built, not about the radius or the opacity the
// tooltip happens to ship, which TestTheTooltipKeepsAVisibleEdge covers.
func outlined() frame {
	return frame{
		Size: 44, Canvas: 60, Tile: 10, Radius: 8,
		Fallback: "#2f2f2f", Mask: true, HintSize: 4, HintY: 48,
		Border: "0.1", BorderFallback: "#e8e4dd", Shadow: true,
	}
}

// TestShadowPrefixIsPresentAndEmpty is what keeps the tooltip to one border.
//
// org.kde.plasma.components.ToolTip draws this sheet twice, once with prefix
// "shadow" for the drop shadow and once plain. KSvg decides a prefix exists by
// looking for <prefix>-center and clears the prefix when it finds none, so a
// sheet with no shadow tiles draws its own frame a second time, inflated by the
// margins — which is one border while the frame is a flat fill and two the
// moment it has an outline.
func TestShadowPrefixIsPresentAndEmpty(t *testing.T) {
	svg := outlined().render()

	// The probe KSvg uses. Everything else here depends on it being found.
	if !strings.Contains(svg, `id="shadow-center"`) {
		t.Fatal("no shadow-center: KSvg would clear the prefix and draw the frame twice")
	}

	for _, name := range []string{
		"topleft", "topright", "bottomleft", "bottomright",
		"top", "bottom", "left", "right", "center",
	} {
		el := element(svg, "shadow-"+name)
		if el == "" {
			t.Errorf("no shadow-%s in the sheet", name)

			continue
		}
		if !strings.Contains(el, `style="fill:none"`) {
			t.Errorf("shadow-%s paints something, so it would show as a second border:\n%s", name, el)
		}
	}

	// The prefix reports the frame's own insets, so adding it does not move the
	// tooltip: these are the margins KSvg already derived from the fallback.
	for _, side := range []string{"top", "bottom", "left", "right"} {
		if !strings.Contains(svg, `id="shadow-hint-`+side+`-margin"`) {
			t.Errorf("no shadow-hint-%s-margin", side)
		}
	}
}

// TestOutlineIsConcentricWithTheSurface covers how the border is built: the
// surface is the whole tile and the outline is a ring between that shape and
// the same shape a pixel in, so the two curves stay parallel. Drawing the inner
// path at the outer radius would leave the border thicker at the corners than
// along the edges.
func TestOutlineIsConcentricWithTheSurface(t *testing.T) {
	svg := outlined().render()
	tile := element(svg, "topleft")

	// Radius 8 at the outer edge, 7 a pixel in — and cornerFactor puts the
	// inner control point at 1+7*0.4478.
	outer := `M0,10 L0,8 C0,3.582 3.582,0 8,0 L10,0 L10,10 Z`
	inner := `M1,10 L1,8 C1,4.135 4.135,1 8,1 L10,1 L10,10 Z`

	if !strings.Contains(tile, fmt.Sprintf(`d="%s" class="ColorScheme-Background"`, outer)) {
		t.Errorf("surface should cover the whole corner, want %s in:\n%s", outer, tile)
	}
	// Both subpaths in one d, so even-odd drops the inner shape out and leaves a
	// ring. Two separate paths would paint the inner shape rather than cut it.
	if !strings.Contains(tile, fmt.Sprintf(`d="%s %s" fill-rule="evenodd"`, outer, inner)) {
		t.Errorf("outline should be a ring a pixel inside it, want %s in:\n%s", inner, tile)
	}
	// The outline goes through the stylesheet so it follows the tint rather than
	// baking one palette's border into artwork every palette shares.
	if !strings.Contains(svg, `.ColorScheme-Text { color:#e8e4dd; }`) {
		t.Errorf("the outline's class should be declared in the stylesheet:\n%s", svg)
	}
}

// TestOutlineIsPaintedOverTheSurface is the bug the concentricity test could not
// see. The outline is a tenth of the text colour, so it is only a border while
// it has the surface underneath it to be a tenth of. Painted first, with the
// background inset over it, the frame's outermost pixel is ninety percent
// transparent and the border reads as a gap onto whatever is behind the tooltip.
//
// The window decoration does stack them the other way round, and that is where
// this went wrong: its border is an opaque literal, so the order does not matter
// to it.
func TestOutlineIsPaintedOverTheSurface(t *testing.T) {
	svg := outlined().render()

	// The corners carry the ring and the edges a plain strip, so check one of
	// each: in both the surface has to be the first shape in the group.
	for _, id := range []string{"topleft", "topright", "bottomleft", "bottomright", "top", "bottom", "left", "right"} {
		tile := element(svg, id)
		surface := strings.Index(tile, "ColorScheme-Background")
		outline := strings.Index(tile, "ColorScheme-Text")

		switch {
		case surface < 0:
			t.Errorf("%s paints no surface under its outline:\n%s", id, tile)
		case outline < 0:
			t.Errorf("%s carries no outline:\n%s", id, tile)
		case outline < surface:
			t.Errorf("%s paints its outline under the surface, so the border is transparent:\n%s", id, tile)
		}
	}
}

// TestOutlineLeavesTheTileSeamsBare keeps the border to the frame's outer edge.
// Every tile is drawn against its neighbours, so an outline on a shared boundary
// would draw a line across the middle of the tooltip.
func TestOutlineLeavesTheTileSeamsBare(t *testing.T) {
	svg := outlined().render()

	// The centre is interior on all four sides and never carries an outline.
	if tile := element(svg, "center"); strings.Contains(tile, "ColorScheme-Text") {
		t.Errorf("the centre carries an outline:\n%s", tile)
	}
	// A strip's outline is one pixel on the frame's edge, not the whole tile:
	// top spans the full 10px tile and its border only the first row.
	tile := element(svg, "top")
	if !strings.Contains(tile, `<rect x="10" y="0" width="24" height="10" class="ColorScheme-Background"`) {
		t.Errorf("top's surface should cover the whole tile:\n%s", tile)
	}
	if !strings.Contains(tile, `<rect x="10" y="0" width="24" height="1" class="ColorScheme-Text"`) {
		t.Errorf("top's outline should be one pixel on the frame's edge:\n%s", tile)
	}
}

// TestOutlineLeavesTheMaskWhole is the half of the border that is invisible
// until it is wrong: the mask is the blur region, not artwork, so an outline
// drawn into it would cut a ring out of the blur instead of showing as a
// border.
func TestOutlineLeavesTheMaskWhole(t *testing.T) {
	svg := outlined().render()

	for _, id := range []string{"mask-topleft", "mask-top", "mask-center"} {
		tile := element(svg, id)
		if tile == "" {
			t.Errorf("no %s in the sheet", id)

			continue
		}
		if strings.Contains(tile, "ColorScheme-Text") {
			t.Errorf("%s carries the outline:\n%s", id, tile)
		}
	}
}

// TestUnoutlinedFramesAreUntouched pins the panel and popup artwork against the
// border work: they share the corner builders with the tooltip, and a frame
// with no outline must still emit one flat shape per tile.
func TestUnoutlinedFramesAreUntouched(t *testing.T) {
	plain := outlined()
	plain.Border, plain.BorderFallback = "", ""

	tile := element(plain.render(), "topleft")

	want := `<g id="topleft"><path d="M0,10 L0,8 C0,3.582 3.582,0 8,0 L10,0 L10,10 Z" class="ColorScheme-Background" style="fill:currentColor"/></g>`
	if tile != want {
		t.Errorf("a frame with no outline changed shape:\n got %s\nwant %s", tile, want)
	}
}

// TestSidebarMovesOnlyTheWindowBackground pins what the sidebar choice is
// allowed to touch. KColorScheme has no sidebar role — a places panel paints
// with the window background — so the option works by moving that one role, and
// the risk is it dragging the rest of the scheme with it.
//
// Header staying put is the point: a sidebar merged with the file list still
// wants a toolbar above it that is not.
func TestSidebarMovesOnlyTheWindowBackground(t *testing.T) {
	tk := testTokens(t)

	windowed, err := appScheme(tk, defaultPalette, sidebarWindow)
	if err != nil {
		t.Fatalf("appScheme: %v", err)
	}
	viewed, err := appScheme(tk, defaultPalette, sidebarView)
	if err != nil {
		t.Fatalf("appScheme: %v", err)
	}

	assertWindowBackgroundMove(t, tk, "sidebar", windowed, viewed)
}

// TestPanelTintMovesOnlyTheShell is the same pinning for the shell's copy of
// the scheme, which asks its own question: the panel, the launcher and applet
// popups all resolve ColorScheme-Background against [Colors:Window], so they
// move together and nothing else may.
func TestPanelTintMovesOnlyTheShell(t *testing.T) {
	tk := testTokens(t)

	chrome, err := shellScheme(tk, defaultPalette, panelChrome)
	if err != nil {
		t.Fatalf("shellScheme: %v", err)
	}
	dark, err := shellScheme(tk, defaultPalette, panelDark)
	if err != nil {
		t.Fatalf("shellScheme: %v", err)
	}

	assertWindowBackgroundMove(t, tk, "panel-tint", chrome, dark)
}

// TestTheTwoSchemesTakeSeparateAxes is the decoupling itself. The sidebar
// question is about a dock panel in a Qt application and the panel question is
// about the desktop; they move the same role in two different files, and the
// files have to be able to disagree. Sharing a flag once made choosing the
// merged sidebar darken the panel as a side effect.
func TestTheTwoSchemesTakeSeparateAxes(t *testing.T) {
	tk := testTokens(t)

	for _, palette := range []string{defaultPalette, "forest"} {
		app := make([]string, 0, 2)
		for _, sidebar := range []string{sidebarWindow, sidebarView} {
			s, err := appScheme(tk, palette, sidebar)
			if err != nil {
				t.Fatalf("appScheme: %v", err)
			}
			app = append(app, s)
		}

		shell := make([]string, 0, 2)
		for _, panel := range []string{panelChrome, panelDark} {
			s, err := shellScheme(tk, palette, panel)
			if err != nil {
				t.Fatalf("shellScheme: %v", err)
			}
			shell = append(shell, s)
		}

		// Each axis moves its own file, and the manifest resolves the two from
		// separate trees, so neither can reach the other's.
		if app[0] == app[1] {
			t.Errorf("%s: the sidebar choice leaves the application scheme unchanged", palette)
		}
		if shell[0] == shell[1] {
			t.Errorf("%s: the panel choice leaves the shell scheme unchanged", palette)
		}
	}
}

// TestTooltipSitsOffThePopupBackground guards the surface the tooltip is for.
// It carries a colour of its own, below every other surface in the set, so that
// it reads as a dark card over whatever it covers rather than as one more shade
// of it.
//
// Both panel tints are checked, and the second is the one that matters. The
// tooltip took the view colour until the darker-panels option was added, which
// puts the popups it appears over on exactly that — so with that option on, a
// tooltip over a popup painted the same colour as the popup and showed only its
// outline. Nothing caught it, because this only looked at the shipped tint.
func TestTooltipSitsOffThePopupBackground(t *testing.T) {
	tk := testTokens(t)

	for palette := range tk.Palettes {
		for _, tint := range []string{panelChrome, panelDark} {
			t.Run(palette+"-"+tint, func(t *testing.T) {
				shell, err := shellScheme(tk, palette, tint)
				if err != nil {
					t.Fatalf("shellScheme: %v", err)
				}

				popup, tooltip := background(shell, "Window"), background(shell, "Tooltip")
				if popup == tooltip {
					t.Errorf("popups and tooltips both paint %s, so a tooltip over a popup "+
						"shows only its outline", popup)
				}
			})
		}
	}
}

// TestHoverCannotGrowPastTheButton pins the one control state Plasma draws
// outside the control it belongs to. ButtonHover.qml fills the button and then
// pushes all four edges back out by the hover prefix's own margins, so every
// margin that prefix reports becomes overhang — the theme once shipped a plate
// 12px wider and taller than the button under it.
//
// hint-no-border-padding is what answers it: KSvg returns zero for a margin
// query on a prefix carrying it, while the border tiles keep their own widths
// and still paint. That is the only combination that puts the wash on the
// button's rect and still lets its corners follow the button's.
func TestHoverCannotGrowPastTheButton(t *testing.T) {
	tk := testTokens(t)

	files, err := elements(tk, defaultPalette, defaultElements)
	if err != nil {
		t.Fatalf("elements: %v", err)
	}
	button := files["widgets/button.svg"]

	if !strings.Contains(button, `id="hover-hint-no-border-padding"`) {
		t.Error("the hover state does not disclaim its border padding, so Plasma will draw it " +
			"past the button by the margins its tiles imply")
	}

	// KSvg honours the hint unprefixed as well, where it would zero the margins
	// of every prefix in the sheet — including normal's, which are a raised
	// button's own padding.
	if strings.Contains(button, `id="hint-no-border-padding"`) {
		t.Error("hint-no-border-padding is unprefixed, so it reaches every state in the sheet")
	}

	borders := []string{
		"topleft", "top", "topright", "left", "right", "bottomleft", "bottom", "bottomright",
	}

	for _, prefix := range []string{"hover", "normal", "pressed", "toolbutton-hover"} {
		for _, tile := range borders {
			if id := `id="` + prefix + "-" + tile + `"`; !strings.Contains(button, id) {
				t.Errorf("%s is missing %s, so it has no corner to follow the button's with",
					prefix, id)
			}
		}
	}
}

// TestHoverCornersFollowTheElementShape is the complaint that outlived the
// first fix: a hover that lands on the button but squares off its corners
// reads as the button changing shape under the pointer. The shape is no longer
// a preference, so what this pins now is that the hover plate is drawn from the
// same radius as the button under it rather than from a figure of its own.
func TestHoverCornersFollowTheElementShape(t *testing.T) {
	tk := testTokens(t)

	files, err := elements(tk, defaultPalette, defaultElements)
	if err != nil {
		t.Fatalf("elements: %v", err)
	}

	button := files["widgets/button.svg"]

	hover, ok := cut(button, `<g id="hover-topleft">`, "</g>")
	if !ok {
		t.Fatal("the hover state has no top-left corner")
	}

	normal, ok := cut(button, `<g id="normal-topleft">`, "</g>")
	if !ok {
		t.Fatal("the normal state has no top-left corner")
	}

	if !strings.Contains(hover, " A") {
		t.Error("the hover corner is not arced, so it squares the button off under the pointer")
	}

	// The two states are stacked at different offsets down the sheet, so their
	// arcs end at different points. The radii the arcs are drawn at are the part
	// the element shape decides, and those have to agree.
	if hoverR, normalR := arcRadii(hover), arcRadii(normal); hoverR != normalR {
		t.Errorf("hover corner is drawn at radius %q and the normal state under it at %q — the "+
			"plate has to follow the button's radius, not one of its own", hoverR, normalR)
	}
}

// arcRadii pulls the rx,ry off a corner path's elliptical-arc command, which is
// the part of it the element radius decides. The rest of the command is the
// endpoint, which moves with the state's offset down the sheet.
func arcRadii(corner string) string {
	i := strings.Index(corner, " A")
	if i < 0 {
		return ""
	}

	rest := corner[i+2:]
	if j := strings.Index(rest, " "); j >= 0 {
		return rest[:j]
	}

	return rest
}

// TestTheButtonSurfaceReadsAsRaised pins the lift a button has to keep over the
// window it sits on, after the translucency has taken its share back off. The
// two are one decision: the theme once shipped an opaque surface six units up
// and it read as no background at all, and lowering opacity.button alone would
// walk it back there without touching a colour.
func TestTheButtonSurfaceReadsAsRaised(t *testing.T) {
	tk := testTokens(t)

	// Far enough to be seen on a near-black surface. Eyeballed rather than
	// derived, like the surfaces themselves.
	const clear = 10

	alpha := tk.Opacity["button"]
	if alpha <= 0 || alpha > 1 {
		t.Fatalf("opacity.button is %v, which is not an opacity", alpha)
	}

	for name, surfaces := range tk.Surfaces {
		background, elevated := luma(t, surfaces["background"]), luma(t, surfaces["elevated"])

		// What the eye gets: the surface composited onto the window behind it.
		if lift := alpha * float64(elevated-background); lift < clear {
			t.Errorf("%s: the button surface lands %.1f above the window, which is not enough "+
				"to read as raised (want %d) — raise elevated or opacity.button",
				name, lift, clear)
		}
	}
}

// luma is a surface's average channel, which is enough to order near-blacks
// that differ only in tint.
func luma(t *testing.T, hex string) int {
	t.Helper()

	v, err := strconv.ParseUint(strings.TrimPrefix(hex, "#"), 16, 32)
	if err != nil {
		t.Fatalf("not a #rrggbb colour: %s", hex)
	}

	return (int(v>>16&0xff) + int(v>>8&0xff) + int(v&0xff)) / 3
}

// TestButtonSurfaceIsTranslucent pins the one control fill that lets what it
// sits on show through, in both the states that draw it. A pressed button that
// went opaque would read as a different surface rather than a pressed one.
func TestButtonSurfaceIsTranslucent(t *testing.T) {
	tk := testTokens(t)

	want := n(tk.Opacity["button"])
	if want == "1" || want == "" {
		t.Fatalf("opacity.button is %q, which is not translucent at all", want)
	}

	files, err := elements(tk, defaultPalette, defaultElements)
	if err != nil {
		t.Fatalf("elements: %v", err)
	}

	for _, prefix := range []string{"normal", "pressed"} {
		centre, ok := cut(files["widgets/button.svg"], `<g id="`+prefix+`-center">`, "</g>")
		if !ok {
			t.Fatalf("%s has no centre tile", prefix)
		}

		surface := `class="ColorScheme-ButtonBackground" style="fill:currentColor" opacity="` + want + `"`
		if !strings.Contains(centre, surface) {
			t.Errorf("%s does not paint its surface at opacity %s: %s", prefix, want, centre)
		}
	}
}

// cut returns what lies between the first open and the next close after it.
func cut(s, open, close string) (string, bool) {
	_, rest, ok := strings.Cut(s, open)
	if !ok {
		return "", false
	}

	inner, _, ok := strings.Cut(rest, close)

	return inner, ok
}

// assertWindowBackgroundMove checks one scheme rendered at both ends of an axis
// that moves the window background. Both axes move the same role in the same
// way, so they are worth holding to one table: what varies between them is only
// which file is asking.
func assertWindowBackgroundMove(t *testing.T, tk *tokens, axis, held, moved string) {
	t.Helper()

	surfaces := tk.Surfaces[tk.Palettes[defaultPalette].Surfaces]
	chrome, view := rgb(surfaces["background"]), rgb(surfaces["view"])
	tip := rgb(surfaces["tooltip"])

	for _, tc := range []struct {
		section     string
		held, moved string
		description string
	}{
		{"Window", chrome, view, "the role the choice paints with"},
		{"Complementary", chrome, view, "follows Window so the two cannot disagree"},
		{"Header", chrome, chrome, "the toolbar stays on the chrome colour"},
		{"View", view, view, "already the view colour; the option is what meets it"},
		{"Tooltip", tip, tip, "its own colour, below every surface either axis moves"},
	} {
		if got := background(moved, tc.section); got != tc.moved {
			t.Errorf("%s moved: [Colors:%s] BackgroundNormal = %s, want %s (%s)",
				axis, tc.section, got, tc.moved, tc.description)
		}
		if got := background(held, tc.section); got != tc.held {
			t.Errorf("%s held: [Colors:%s] BackgroundNormal = %s, want %s (%s)",
				axis, tc.section, got, tc.held, tc.description)
		}
	}
}

// testTokens loads the theme's own tokens, which is what every test that cares
// about a colour measures against.
func testTokens(t *testing.T) *tokens {
	t.Helper()

	tk, err := loadTokens(filepath.Join("../..", "spec", "tokens.json"))
	if err != nil {
		t.Fatalf("loadTokens: %v", err)
	}

	return tk
}

// background reads one section's BackgroundNormal out of a rendered scheme.
func background(ini, section string) string {
	rest, ok := strings.CutPrefix(ini[strings.Index(ini, "[Colors:"+section+"]"):], "[Colors:"+section+"]\n")
	if !ok {
		return ""
	}

	for _, line := range strings.Split(rest, "\n") {
		if v, found := strings.CutPrefix(line, "BackgroundNormal="); found {
			return v
		}
		if strings.HasPrefix(line, "[") {
			break
		}
	}

	return ""
}

// TestSwatchesMatchTheSurfaces keeps the two hand-written copies of a colour in
// step. The manifest carries swatches so the preferences screen can draw the
// palette it is offering, and spec/tokens.json is where those colours actually
// come from — a swatch that drifts shows the user one colour and installs
// another, which is worse than showing none.
//
// The pair is the panel background and the elevated surface, in that order:
// what the desktop is mostly made of, and the shade sitting on top of it.
func TestSwatchesMatchTheSurfaces(t *testing.T) {
	const root = "../.."

	tk, err := loadTokens(filepath.Join(root, "spec", "tokens.json"))
	if err != nil {
		t.Fatalf("loadTokens: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, "assets", "theme.json"))
	if err != nil {
		t.Fatal(err)
	}

	var manifest struct {
		Components []struct {
			Options []struct {
				ID     string `json:"id"`
				Values []struct {
					ID     string   `json:"id"`
					Swatch []string `json:"swatch"`
				} `json:"values"`
			} `json:"options"`
		} `json:"components"`
	}

	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}

	var seen int

	for _, c := range manifest.Components {
		for _, o := range c.Options {
			if o.ID != "palette" {
				continue
			}

			for _, v := range o.Values {
				p, ok := tk.Palettes[v.ID]
				if !ok {
					t.Errorf("theme.json offers palette %q, which spec/tokens.json does not define", v.ID)

					continue
				}

				seen++

				surfaces, ok := tk.Surfaces[p.Surfaces]
				if !ok {
					t.Errorf("palette %q names unknown surfaces %q", v.ID, p.Surfaces)

					continue
				}

				want := []string{surfaces["background"], surfaces["elevatedAlt"]}
				if len(v.Swatch) != len(want) {
					t.Errorf("%s has %d swatches, want %d", v.ID, len(v.Swatch), len(want))

					continue
				}

				for i, got := range v.Swatch {
					if got != want[i] {
						t.Errorf("%s swatch %d is %s but its surface is %s — "+
							"edit spec/tokens.json and assets/theme.json together", v.ID, i, got, want[i])
					}
				}
			}
		}
	}

	if seen != len(tk.Palettes) {
		t.Errorf("the manifest offers %d palettes, spec/tokens.json defines %d", seen, len(tk.Palettes))
	}
}

// TestManifestDefaultsMatchTheShippedArtwork ties the option the installer
// preselects to the variant the generator bakes into the shipped tree. They are
// two separate statements of the same intent and they drifted once already: the
// style shipped square containers while theme.json preselected "rounded", so a
// default install laid the rounded overlay over a square base and the constant
// here described a default nobody received.
//
// Where a value carries no overlay the drift is not even redundant work. The
// shadow is resolved from {palette}-{shadows}, so its defaultValue is the only
// thing that decides what a fresh install renders.
func TestManifestDefaultsMatchTheShippedArtwork(t *testing.T) {
	const root = "../.."

	data, err := os.ReadFile(filepath.Join(root, "assets", "theme.json"))
	if err != nil {
		t.Fatal(err)
	}

	var manifest struct {
		Components []struct {
			Options []struct {
				ID           string `json:"id"`
				DefaultValue string `json:"defaultValue"`
			} `json:"options"`
		} `json:"components"`
	}

	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}

	want := map[string]string{
		"palette":    defaultPalette,
		"sidebar":    defaultSidebar,
		"panel-tint": defaultPanel,
		"buttons":    defaultButtons,
	}

	seen := map[string]bool{}

	for _, c := range manifest.Components {
		for _, o := range c.Options {
			w, ok := want[o.ID]
			if !ok {
				continue
			}

			seen[o.ID] = true

			if o.DefaultValue != w {
				t.Errorf("theme.json preselects %q for %s but the generator ships %q — "+
					"edit assets/theme.json and internal/gen/main.go together", o.DefaultValue, o.ID, w)
			}
		}
	}

	for id := range want {
		if !seen[id] {
			t.Errorf("theme.json declares no %s option", id)
		}
	}
}

// TestShadowMatchesBreeze pins the numbers to their source. The shadow is
// Breeze's, so the values that matter are Breeze's: the padding its
// createShadowObject derives for ShadowLarge, and the composited alpha its two
// blurred boxes produce at the window's edge. Both are computed here the long
// way round, from the constants in spec/tokens.json, so a change to either the
// tokens or the derivation has to be deliberate.
func TestShadowMatchesBreeze(t *testing.T) {
	const root = "../.."

	tk, err := loadTokens(filepath.Join(root, "spec", "tokens.json"))
	if err != nil {
		t.Fatalf("loadTokens: %v", err)
	}

	t.Run("padding", func(t *testing.T) {
		// From breezedecoration.cpp with Metrics::Shadow_Overlap = 3:
		// ShadowSmall's blur extents are 23 and 11, box 47, canvas 93,
		// composite offset (0,4).
		l, r, top, bottom := tk.DecorationShadow["on"].padding()
		for _, c := range []struct {
			name      string
			got, want int
		}{{"left", l, 20}, {"right", r, 20}, {"top", top, 16}, {"bottom", bottom, 24}} {
			if c.got != c.want {
				t.Errorf("%s padding %d, want Breeze's %d", c.name, c.got, c.want)
			}
		}
	})

	t.Run("shipped strength", func(t *testing.T) {
		// Breeze's own default is 255/255. This theme runs a little lighter;
		// the value is still within Breeze's range, which is 25 to 255.
		got := tk.DecorationShadow["on"].Strength
		if got != 0.85 {
			t.Errorf("shadow strength %v, want 0.85", got)
		}
		if got < 25.0/255 || got > 1 {
			t.Errorf("strength %v is outside Breeze's 25..255 range", got)
		}
	})

	t.Run("blur extent", func(t *testing.T) {
		for _, c := range []struct{ radius, want int }{{48, 68}, {32, 45}, {24, 34}, {16, 23}, {8, 11}} {
			if got := blurExtent(c.radius); got != c.want {
				t.Errorf("blurExtent(%d) = %d, want %d", c.radius, got, c.want)
			}
		}
	})

	// The profile is Breeze's, so it is checked at Breeze's own full strength.
	// ShadowStrength is the one value this theme sets for itself, and it only
	// scales the result, so it is pinned separately below.
	t.Run("alpha at the window edge", func(t *testing.T) {
		sh := tk.DecorationShadow["on"]
		sh.Strength = 1.0

		// Each layer's box is inset by the overlap, so at the window's own edge
		// the shadow is already that far in.
		// sign is +1 above the window and -1 below it: the shadow is offset
		// downwards, so the same offset weakens the top and deepens the bottom.
		// Each layer carries its own offset on top of the composite one.
		composite := func(sign int) float64 {
			var a float64
			for _, spec := range sh.Layers {
				l := shadowLayer{Radius: spec.Radius, Opacity: spec.Opacity * sh.Strength}
				d := sh.Overlap + sign*(sh.OffsetY+spec.OffsetY)
				a += l.Opacity * l.profile(float64(d)) * (1 - a)
			}

			return a
		}

		for _, c := range []struct {
			name string
			got  float64
			want float64
		}{
			{"side", composite(0), 0.412},
			{"top", composite(1), 0.225},
			{"bottom", composite(-1), 0.622},
		} {
			if math.Abs(c.got-c.want) > 0.002 {
				t.Errorf("%s alpha at the window edge = %.3f, want Breeze's %.3f", c.name, c.got, c.want)
			}
		}
	})
}

// TestStatesDoNotOverlapOnTheSheet keeps the inactive copy clear of the active
// one. The step between them was a constant sized for artwork with no shadow
// band; a band on every side makes a state taller than that, and the two
// shadows then add up where they overlap.
func TestStatesDoNotOverlapOnTheSheet(t *testing.T) {
	const root = "../.."

	tk, err := loadTokens(filepath.Join(root, "spec", "tokens.json"))
	if err != nil {
		t.Fatalf("loadTokens: %v", err)
	}

	for name, sh := range tk.DecorationShadow {
		t.Run(name, func(t *testing.T) {
			l, r, top, bottom := sh.padding()
			tile := decoTile
			if m := sh.Overlap + sh.reach(); m > tile {
				tile = m
			}
			d := decoration{
				Width: 2*tile + decoMid, Tile: tile, TitleH: 31, BodyH: 24,
				PadL: l, PadR: r, PadT: top, PadB: bottom,
				Overlap: sh.Overlap, Reach: sh.reach(),
			}

			if d.step() < d.stateH() {
				t.Errorf("step %d is inside a state %d tall: the two would be drawn on top of each other",
					d.step(), d.stateH())
			}
		})
	}
}

// TestShadowFitsTheFixedTiles is why the theme carries Breeze's Small shadow
// rather than the Large one it ships.
//
// A blurred corner is the product of two profiles, and each settles only after
// about 3 standard deviations. A stretched tile cannot vary, so that settling
// has to happen inside the fixed tiles: sideways within the corner tile, which
// can be made as wide as it likes, but vertically within the titlebar row,
// which cannot — it is the titlebar. Large's sigma is 24 and wants 72px against
// a 30px titlebar; Small's is 8 and wants 21px, which fits.
//
// Without this, a too-large shadow does not fail loudly. It leaves a step where
// the corner tile meets the stretched edge beside it.
func TestShadowFitsTheFixedTiles(t *testing.T) {
	const root = "../.."

	tk, err := loadTokens(filepath.Join(root, "spec", "tokens.json"))
	if err != nil {
		t.Fatalf("loadTokens: %v", err)
	}

	sh := tk.DecorationShadow["on"]
	reach := sh.reach()

	// The lowest a layer's top edge sits, which is the one with least room.
	deepest := 0
	for _, l := range sh.Layers {
		if d := sh.Overlap + sh.OffsetY + l.OffsetY; d > deepest {
			deepest = d
		}
	}

	if room := titleHeight + titleBorder - deepest; room < reach {
		t.Errorf("the titlebar row leaves %dpx for a shadow needing %dpx to settle: "+
			"the corner would step where it meets the edge tile", room, reach)
	}

	l, r, top, bottom := sh.padding()
	tile := decoTile
	if m := sh.Overlap + reach; m > tile {
		tile = m
	}
	d := decoration{
		Width: 2*tile + decoMid, Tile: tile, TitleH: titleHeight + titleBorder, BodyH: 24,
		PadL: l, PadR: r, PadT: top, PadB: bottom,
		Overlap: sh.Overlap, Reach: reach,
		Layers: []shadowLayer{{DY: sh.OffsetY, Radius: sh.Layers[0].Radius}},
	}

	if d.row2() <= d.row1() {
		t.Errorf("no stretchable row left: rows %d and %d", d.row1(), d.row2())
	}
}

// TestInactiveStateMatchesActive covers a bug that only showed on unfocused
// windows. The shadow's geometry is mostly relative to the frame, but a
// gradient is in user space and has to be placed where the state actually sits
// on the sheet. The helpers that give a layer's top and bottom edges were
// returning the active state's coordinates for both, so the inactive copy's
// gradients pointed a full step away: its top band came out at full opacity and
// its bottom corners lost their shadow entirely.
//
// The two states are drawn from the same colours here, so the artwork must be
// identical bar the step between them.
func TestInactiveStateMatchesActive(t *testing.T) {
	const root = "../.."

	tk, err := loadTokens(filepath.Join(root, "spec", "tokens.json"))
	if err != nil {
		t.Fatalf("loadTokens: %v", err)
	}

	svg, err := tk.decoration(defaultPalette, defaultTitlebar, defaultShadow)
	if err != nil {
		t.Fatalf("decoration: %v", err)
	}

	// Every vertical gradient exists twice, and the inactive copy has to sit
	// exactly one step lower with the same stops.
	re := regexp.MustCompile(`<linearGradient id="shadow-(inactive-)?(\d+)-(top|bottom)" ` +
		`gradientUnits="userSpaceOnUse" x1="\d+" y1="(\d+)" x2="\d+" y2="(\d+)">([^<]*(?:<stop[^>]*/>)*)`)

	type grad struct{ y1, y2 int }
	active, inactive := map[string]grad{}, map[string]grad{}
	stops := map[string]string{}

	for _, m := range re.FindAllStringSubmatch(svg, -1) {
		key := m[2] + "-" + m[3]
		y1, _ := strconv.Atoi(m[4])
		y2, _ := strconv.Atoi(m[5])
		if m[1] == "" {
			active[key] = grad{y1, y2}
			stops[key] = m[6]
		} else {
			inactive[key] = grad{y1, y2}
			if s := stops[key]; s != "" && s != m[6] {
				t.Errorf("%s: the two states have different stops", key)
			}
		}
	}

	if len(active) == 0 || len(active) != len(inactive) {
		t.Fatalf("found %d active and %d inactive vertical gradients", len(active), len(inactive))
	}

	for key, a := range active {
		i, ok := inactive[key]
		if !ok {
			t.Errorf("no inactive copy of %s", key)

			continue
		}
		if d1, d2 := i.y1-a.y1, i.y2-a.y2; d1 != d2 {
			t.Errorf("%s: inactive copy is skewed, y1 moved %d and y2 moved %d", key, d1, d2)
		}
	}
}

// TestMaskCornersArePulledInsideTheBackground is the fix for the fringe that
// shows around a rounded popup's corners.
//
// The mask is thresholded to a one-bit region, so its corner is a staircase
// where the artwork's is antialiased. At equal radii the staircase lands
// outside the artwork and those pixels read as blur the surface does not cover.
// Breeze carries the same inset and documents it inside its own background.svg.
func TestMaskCornersArePulledInsideTheBackground(t *testing.T) {
	// Pinned as a literal rather than taken from the constant, which would let
	// this agree with itself if the inset were ever zeroed. It covers an
	// antialiasing ramp, which is one pixel wide whatever the radius is, so it
	// is the one number here that does not move when the radii do.
	if maskCornerInset != 1 {
		t.Fatalf("maskCornerInset is %v, want 1 — the inset covers a one-pixel "+
			"antialiasing ramp and does not scale with the radius", maskCornerInset)
	}

	for _, f := range maskedFrames() {
		t.Run(f.name, func(t *testing.T) {
			svg := f.render()

			for _, corner := range []string{"topleft", "topright", "bottomleft", "bottomright"} {
				bg, mask := element(svg, corner), element(svg, "mask-"+corner)
				if bg == "" || mask == "" {
					t.Fatalf("%s: missing background or mask corner", corner)
				}

				want := f.Radius - 1
				if got := arcRadius(t, mask); got != want {
					t.Errorf("mask-%s is drawn at radius %v, want %v — a pixel inside the "+
						"background's %v, or its corner staircase shows outside the artwork",
						corner, got, want, f.Radius)
				}
				if got := arcRadius(t, bg); got != f.Radius {
					t.Errorf("%s is drawn at radius %v, want the frame's %v", corner, got, f.Radius)
				}
			}
		})
	}
}

// TestMaskKeepsTheFramesOuterEdge is the half of the fix that is easy to lose.
//
// Plasma shrank the whole mask first and got a dark outline down every side
// with the wallpaper showing through the outermost pixel. Only the arc moves:
// the mask's straight runs, edges and centre are the frame's own, so the mask
// still covers the outer pixel everywhere the corner is not curving.
func TestMaskKeepsTheFramesOuterEdge(t *testing.T) {
	for _, f := range maskedFrames() {
		t.Run(f.name, func(t *testing.T) {
			svg := f.render()

			// The edges and the centre are copied outright, so they have to
			// match the background's geometry exactly.
			for _, name := range []string{"top", "bottom", "left", "right", "center"} {
				bg, mask := element(svg, name), element(svg, "mask-"+name)
				if geometryOf(bg) != geometryOf(mask) {
					t.Errorf("mask-%s has moved off the background's %s:\n bg   %s\n mask %s",
						name, name, geometryOf(bg), geometryOf(mask))
				}
			}

			// Every mask corner still opens and closes on the frame's outer
			// edge: a path that started at the inset would pull the mask off the
			// outermost pixel along the straight part of the side.
			size := n(float64(f.Size))
			for _, corner := range []string{"topleft", "topright", "bottomleft", "bottomright"} {
				mask := element(svg, "mask-"+corner)

				var touches bool
				for _, edge := range []string{"M0,", ",0 ", ",0 L", "L0,", "M" + size, "," + size} {
					if strings.Contains(mask, edge) {
						touches = true

						break
					}
				}
				if !touches {
					t.Errorf("mask-%s never reaches the frame's outer edge:\n%s", corner, mask)
				}
			}
		})
	}
}

// TestEveryTranslucentFrameCarriesAMask guards the panel's own version of the
// artefact. With no mask- prefix KSvg falls back to the background pixmap
// itself — alphaMask() returns frame->cachedBackground — so a translucent frame
// without one has its blur region thresholded out of 85%-opaque artwork.
func TestEveryTranslucentFrameCarriesAMask(t *testing.T) {
	tk := testTokens(t)

	palette, _, err := tk.colours(defaultPalette)
	if err != nil {
		t.Fatalf("colours: %v", err)
	}

	for path, svg := range frames(tk, palette, tk.ContainerShape[defaultContainers], true) {
		if !strings.Contains(svg, "fill-opacity") {
			continue
		}
		if !strings.Contains(svg, `id="mask-center"`) {
			t.Errorf("%s is translucent and has no mask-center, so KSvg will take its blur "+
				"region from the translucent artwork", path)
		}
	}
}

// maskedFrames is every frame shape the theme masks, at the radii it ships.
// Both corner idioms are covered: the popup and the tooltip have tiles that
// exceed their radius, the panel's tile is exactly its radius.
func maskedFrames() []struct {
	name string
	frame
} {
	return []struct {
		name string
		frame
	}{
		{"popup", frame{Size: 44, Canvas: 60, Tile: 14, Radius: 12, Fallback: "#292929", Mask: true, HintSize: 8, HintY: 48}},
		{"tooltip", frame{Size: 44, Canvas: 60, Tile: 14, Radius: 8, Fallback: "#292929", Mask: true, HintSize: 4, HintY: 48}},
		{"panel", frame{Size: 40, Canvas: 56, Tile: 12, Radius: 12, Fallback: "#292929", Mask: true, HintSize: 2, HintY: 44}},
	}
}

// arcRadius recovers the radius a corner path was drawn at from its cubic. The
// control points sit cornerFactor of the radius in from the corner, so the span
// between the curve's start and end is the radius itself on each axis.
func arcRadius(t *testing.T, el string) float64 {
	t.Helper()

	i := strings.Index(el, " C")
	if i < 0 {
		t.Fatalf("no cubic in %s", el)
	}

	// The point before the C is where the arc starts; the last pair of the C is
	// where it ends. The radius is the distance between them on one axis.
	head := strings.Fields(el[:i])
	start := head[len(head)-1]

	body := el[i+2:]
	if j := strings.IndexAny(body, "LZ"); j >= 0 {
		body = body[:j]
	}
	pts := strings.Fields(body)
	end := pts[len(pts)-1]

	sx, sy := coords(t, start)
	ex, ey := coords(t, end)

	return math.Max(math.Abs(ex-sx), math.Abs(ey-sy))
}

// geometryOf strips an element down to the attributes that place it. The
// attributes are matched with their leading space so that "d=" does not also
// find the "id=" every element carries.
func geometryOf(el string) string {
	var out []string

	for _, attr := range []string{" x=", " y=", " width=", " height=", " d="} {
		i := strings.Index(el, attr)
		if i < 0 {
			continue
		}

		rest := el[i+len(attr)+1:]
		out = append(out, strings.TrimSpace(attr)+rest[:strings.Index(rest, `"`)])
	}

	return strings.Join(out, " ")
}

func coords(t *testing.T, pair string) (float64, float64) {
	t.Helper()

	// The first point of a path is the one right after d=", with no command
	// letter of its own to separate it from the attribute.
	if i := strings.LastIndex(pair, `"`); i >= 0 {
		pair = pair[i+1:]
	}

	pair = strings.TrimLeft(pair, "ML")

	x, y, ok := strings.Cut(pair, ",")
	if !ok {
		t.Fatalf("not a coordinate pair: %q", pair)
	}

	fx, err := strconv.ParseFloat(x, 64)
	if err != nil {
		t.Fatalf("bad x in %q: %v", pair, err)
	}

	fy, err := strconv.ParseFloat(strings.TrimSuffix(y, "Z"), 64)
	if err != nil {
		t.Fatalf("bad y in %q: %v", pair, err)
	}

	return fx, fy
}

// TestTheTooltipKeepsAVisibleEdge guards the one thing darkening the tooltip can
// take away.
//
// The tooltip is the container that appears over arbitrary content rather than
// over a surface it already contrasts with, which is the whole reason it carries
// an outline. Its background and that outline have been taken down together more
// than once, and they are separately convincing each time: a slightly darker
// card is an improvement right up to the step where the edge stops resolving and
// the tooltip becomes a shape with no boundary. Nothing else would notice, because
// every other test here asks whether the outline is built correctly rather than
// whether it can be seen.
func TestTheTooltipKeepsAVisibleEdge(t *testing.T) {
	tk := testTokens(t)

	// Enough of a gap to survive a dark wallpaper behind a translucent popup
	// under the tooltip. Eyeballed, like the surfaces themselves.
	const clear = 6

	alpha := tk.Opacity["tooltipBorder"]
	if alpha <= 0 || alpha > 1 {
		t.Fatalf("opacity.tooltipBorder is %v, which is not an opacity", alpha)
	}

	text := rgbOf(t, tk.Foreground["text"])

	for name, surfaces := range tk.Surfaces {
		bg := rgbOf(t, surfaces["tooltip"])

		var gap float64

		for i := range bg {
			// The outline is laid over the surface rather than replacing it, so
			// what the edge pixel ends up as is the two composited.
			edge := alpha*text[i] + (1-alpha)*bg[i]
			if d := edge - bg[i]; d > gap {
				gap = d
			}
		}

		if gap < clear {
			t.Errorf("%s: the tooltip's outline composites to within %.1f of its background, "+
				"so the edge stops resolving — lower opacity.tooltipBorder or "+
				"surfaces.%s.tooltip further and the tooltip loses its boundary",
				name, gap, name)
		}
	}
}

// rgbOf parses a hex colour into three channels.
func rgbOf(t *testing.T, hex string) [3]float64 {
	t.Helper()

	var out [3]float64

	if len(hex) != 7 || hex[0] != '#' {
		t.Fatalf("not a hex colour: %q", hex)
	}

	for i := range out {
		v, err := strconv.ParseUint(hex[1+i*2:3+i*2], 16, 8)
		if err != nil {
			t.Fatalf("bad channel in %q: %v", hex, err)
		}

		out[i] = float64(v)
	}

	return out
}
