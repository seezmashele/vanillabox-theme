// Command gen writes the parts of the theme that follow from spec/tokens.json.
//
// Only files that participate in a variant axis are generated. Artwork that no
// axis touches — tabbar, line, plasmoidheading, tasks, and the widget artwork
// whose colours are already resolved at paint time — stays hand-maintained,
// because generating a file that never varies buys nothing and costs a template.
//
// Run it with `go generate ./...` from the repository root. Output is committed;
// see DESIGN.md.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"strings"
)

type tokens struct {
	Theme      identity                     `json:"theme"`
	Foreground map[string]string            `json:"foreground"`
	Surfaces   map[string]map[string]string `json:"surfaces"`
	Palettes   map[string]palette           `json:"palettes"`
	Status     map[string]string            `json:"status"`

	ContainerShape   map[string]map[string]float64 `json:"containerShape"`
	ElementShape     map[string]map[string]float64 `json:"elementShape"`
	DecorationShape  map[string]map[string]float64 `json:"decorationShape"`
	DecorationShadow map[string]shadowSpec         `json:"decorationShadow"`
	PopupShadow      popupShadowSpec               `json:"popupShadow"`
	ButtonStyles     map[string]buttonStyle        `json:"buttonStyles"`
	Opacity          map[string]float64            `json:"opacity"`
}

// shadowSpec is one point on the shadow axis: how far the band reaches beyond
// the window, and what it is painted in. Extent zero is no band at all.
// popupShadowSpec is the shadow under the launcher and the system tray popups.
// See dropShadow for how it is drawn.
type popupShadowSpec struct {
	Color    string  `json:"color"`
	Strength float64 `json:"strength"`
	OffsetY  int     `json:"offsetY"`
	Radius   int     `json:"radius"`
}

type shadowSpec struct {
	Size     string            `json:"size"`
	Color    string            `json:"color"`
	Strength float64           `json:"strength"`
	OffsetY  int               `json:"offsetY"`
	Overlap  int               `json:"overlap"`
	Layers   []shadowLayerSpec `json:"layers"`
}

// shadowLayerSpec is one ShadowParams entry out of Breeze's s_shadowParams.
type shadowLayerSpec struct {
	OffsetY int     `json:"offsetY"`
	Radius  int     `json:"radius"`
	Opacity float64 `json:"opacity"`
}

// blurExtent is Breeze's calculateBlurExtent, which is how far a blur of this
// radius reaches: calculateBlurRadius(calculateBlurStdDev(radius)), with
// stdDev = radius/2 and the SVG feGaussianBlur scale factor Breeze cites.
func blurExtent(radius int) int {
	const gaussianScaleFactor = (3.0 * math.Sqrt2 * math.SqrtPi / 4.0) * 1.5

	if radius == 0 {
		return 0
	}

	e := int(math.Floor(float64(radius)*0.5*gaussianScaleFactor + 0.5))
	if e < 2 {
		e = 2
	}

	return e
}

// padding derives the four bands exactly as Breeze's createShadowObject does:
// a box big enough for the widest blur, a canvas big enough for every shadow at
// its own offset, the box centred in it, and then the overlap and the composite
// offset taken back off each side.
func (sh shadowSpec) padding() (left, right, top, bottom int) {
	if len(sh.Layers) == 0 {
		return 0, 0, 0, 0
	}

	widest := 0
	for _, l := range sh.Layers {
		if e := blurExtent(l.Radius); e > widest {
			widest = e
		}
	}
	box := 2*widest + 1

	canvasW, canvasH := 0, 0
	for _, l := range sh.Layers {
		e := blurExtent(l.Radius)
		if w := box + 2*e; w > canvasW {
			canvasW = w
		}
		if h := box + 2*e + abs(l.OffsetY); h > canvasH {
			canvasH = h
		}
	}

	left = (canvasW-box)/2 - sh.Overlap
	right = left
	top = (canvasH-box)/2 - sh.Overlap - sh.OffsetY
	bottom = (canvasH-box)/2 - sh.Overlap + sh.OffsetY

	return left, right, top, bottom
}

// reach is how far inside a box a blurred edge has to run before it is
// indistinguishable from full. 2.6 standard deviations leaves under half a
// percent, which is below a rounding step at these opacities.
func (sh shadowSpec) reach() int {
	widest := 0.0
	for _, l := range sh.Layers {
		if s := float64(l.Radius) * 0.5; s > widest {
			widest = s
		}
	}

	return int(math.Ceil(2.6 * widest))
}

func abs(v int) int {
	if v < 0 {
		return -v
	}

	return v
}

// palette is a named colour variant: a surface set and the accent chosen to go
// with it. Pairing them here rather than offering two independent choices is
// what lets the installer ask one question instead of two.
type palette struct {
	Surfaces string `json:"surfaces"`
	Accent   string `json:"accent"`

	// OnHighlight overrides foreground.onHighlight for this palette alone, and
	// is the one exception to holding the foregrounds still.
	//
	// The accent is the selection background, and it is the only colour a
	// palette moves that text has to sit on top of rather than beside. An accent
	// dark enough stops carrying the shared dark on-highlight colour, and no
	// amount of tuning elsewhere fixes it: the two are the same decision. A
	// palette that wants to go that dark says so here rather than dragging every
	// other palette's foregrounds into the question.
	OnHighlight string `json:"onHighlight,omitempty"`
}

// buttonStyle is one titlebar button treatment. The plate opacities differ
// between close and the rest because close is the only button that earns a
// colour of its own.
type buttonStyle struct {
	// Kind picks the treatment: "glyph" shows a symbol always and gains a plate
	// on hover; "circle" shows no symbol and gains colour on hover.
	Kind string `json:"kind"`

	PlateRadius float64 `json:"plateRadius"`

	ClosePlate   string `json:"closePlate"`
	CloseHover   string `json:"closeHover"`
	ClosePressed string `json:"closePressed"`

	PlainHover   string `json:"plainHover"`
	PlainPressed string `json:"plainPressed"`

	// GlyphSize is how big the symbol is on screen, in pixels. It assumes a
	// square button box, which is what keeps the glyph from being stretched.
	GlyphSize float64 `json:"glyphSize"`

	Rest string `json:"rest"`
	Dim  string `json:"dim"`

	// The circle treatment.
	CircleRadius   float64 `json:"circleRadius"`
	RestColor      string  `json:"restColor"`
	DimColor       string  `json:"dimColor"`
	Close          string  `json:"close"`
	Minimize       string  `json:"minimize"`
	Maximize       string  `json:"maximize"`
	PressedOpacity string  `json:"pressedOpacity"`

	Width  int `json:"width"`
	Height int `json:"height"`

	// MenuWidth sizes the titlebar's application icon. Aurorae gives every
	// button the same height, so this is the only per-button size there is: the
	// icon shrinks by width and the leftover height centres it, which is where
	// its breathing room above and below comes from.
	MenuWidth int `json:"menuWidth"`

	// Spacing is the gap Aurorae leaves between adjacent buttons. The hover
	// plates are as wide as the button box, so without it neighbouring plates
	// touch.
	Spacing int `json:"buttonSpacing"`

	// NudgeTop is added to the margin that centres the button in the titlebar.
	// Centred and looking centred are not always the same thing, and this is
	// where that difference is admitted to rather than hidden in the artwork.
	NudgeTop int `json:"nudgeTop"`

	ButtonsOnLeft  string `json:"buttonsOnLeft"`
	ButtonsOnRight string `json:"buttonsOnRight"`
}

const (
	style      = "assets/plasma/desktoptheme/vanilla-box-dark"
	schemeDir  = "assets/color-schemes"
	auroraeDir = "assets/aurorae/themes/VanillaBoxDark"
	lookFeel   = "assets/plasma/look-and-feel/org.vanillabox.dark"
	variantDir = "assets/variants"

	// The theme as shipped. Every other point in the variant space is written
	// under variants/ and copied over an install by the option that names it.
	defaultPalette = "neutral"

	// Rounded throughout, and not as a default anyone can move: the surfaces a
	// thing sits in and the things sitting in them share one scale. The titlebar
	// carries a known compromise for it — Aurorae cannot round a window's bottom
	// corners, so a rounded titlebar tops a body that still ends square. That is
	// accepted as the shipped look and the only one.
	defaultContainers = "rounded"
	defaultElements   = "rounded"
	defaultTitlebar   = "rounded"
	defaultButtons    = "windows"

	// Shadows ship on. Aurorae draws them from the decoration's own padding, so
	// the cost is a band of translucent pixels around every window rather than
	// anything the compositor has to be asked for.
	defaultShadow = "on"

	// The two points on the sidebar axis. A places panel paints with the window
	// background, so "view" is the one that moves that background onto the view
	// colour and "window" is leaving it where it is.
	//
	// "window" is the default: KDE's own arrangement, a sidebar on the window
	// colour beside a darker file list, which leaves dialogs, settings pages and
	// every other window background where they are. "view" is here for anyone
	// who wants the sidebar merged into the file list and accepts the whole
	// window background moving with it.
	sidebarWindow  = "window"
	sidebarView    = "view"
	defaultSidebar = sidebarWindow

	// The two points on the panel tint axis, which is the same move made in the
	// shell's copy of the scheme rather than in the application one.
	//
	// It is a separate axis because the two files have separate readers and
	// nothing in Plasma ties them: the sidebar question is about a dock panel in
	// a Qt application, and the shell has no sidebar to answer it for. Off is
	// the default, which leaves the panel and the launcher on the chrome colour
	// the toolbars and the titlebar already use. On is worth having anyway — a
	// desktop where every surface but the toolbars is the view colour is a
	// coherent look, just not the shipped one.
	panelChrome  = "off"
	panelDark    = "on"
	defaultPanel = panelChrome

	// The titlebar row is the rc's TitleHeight plus the frame's own top border.
	titleHeight = 30
	titleBorder = 1

	// The frame's own corner tile when nothing else forces it wider, and the
	// stretchable run between the two corners.
	decoTile = 12
	decoMid  = 24
)

func main() {
	root := flag.String("root", ".", "repository root to write into")
	flag.Parse()

	if err := run(*root); err != nil {
		fmt.Fprintln(os.Stderr, "gen:", err)
		os.Exit(1)
	}
}

func run(root string) error {
	tk, err := loadTokens(filepath.Join(root, "spec", "tokens.json"))
	if err != nil {
		return err
	}

	files, err := allFiles(tk)
	if err != nil {
		return err
	}

	for path, content := range files {
		full := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			return err
		}
	}

	return prune(root, files, variantDir)
}

// prune deletes anything under the wholly-generated trees that the generator
// did not just write.
//
// Renaming a variant otherwise leaves the old one behind, committed and
// installable, describing a combination the manifest no longer offers. Only
// this tree is swept: the rest of assets/ mixes generated files with
// hand-maintained ones, and nothing there should be deleted by a build.
func prune(root string, written map[string]string, trees ...string) error {
	for _, tree := range trees {
		if err := pruneTree(root, written, tree); err != nil {
			return err
		}
	}

	return nil
}

func pruneTree(root string, written map[string]string, tree string) error {
	dir := filepath.Join(root, filepath.FromSlash(tree))
	if _, err := os.Stat(dir); err != nil {
		return nil
	}

	var stale []string

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if _, ok := written[filepath.ToSlash(rel)]; !ok {
			stale = append(stale, path)
		}

		return nil
	})
	if err != nil {
		return err
	}

	for _, path := range stale {
		if err := os.Remove(path); err != nil {
			return err
		}
	}

	// Directories the removals emptied would otherwise linger as the shape of a
	// variant that no longer exists.
	return removeEmptyDirs(dir)
}

func removeEmptyDirs(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if err := removeEmptyDirs(filepath.Join(dir, e.Name())); err != nil {
			return err
		}
	}

	if entries, err = os.ReadDir(dir); err == nil && len(entries) == 0 {
		return os.Remove(dir)
	}

	return nil
}

// colours resolves a palette into the surfaces it names merged with the
// foregrounds every palette shares, plus its accent.
func (tk *tokens) colours(name string) (map[string]string, string, error) {
	p, ok := tk.Palettes[name]
	if !ok {
		return nil, "", fmt.Errorf("no palette %q", name)
	}

	surfaces, ok := tk.Surfaces[p.Surfaces]
	if !ok {
		return nil, "", fmt.Errorf("palette %q names unknown surfaces %q", name, p.Surfaces)
	}

	merged := make(map[string]string, len(surfaces)+len(tk.Foreground))
	for k, v := range surfaces {
		merged[k] = v
	}
	for k, v := range tk.Foreground {
		merged[k] = v
	}
	if p.OnHighlight != "" {
		merged["onHighlight"] = p.OnHighlight
	}

	return merged, p.Accent, nil
}

// allFiles is everything the generator writes: the theme as shipped and the
// variant trees the installer picks from.
func allFiles(tk *tokens) (map[string]string, error) {
	out, err := build(tk, defaultPalette, defaultContainers, defaultElements, defaultTitlebar, defaultButtons, defaultShadow)
	if err != nil {
		return nil, err
	}

	extra, err := variants(tk)
	if err != nil {
		return nil, err
	}
	for path, content := range extra {
		out[path] = content
	}

	return out, nil
}

// variants writes one file per point in each axis that a component resolves at
// install time. The colours are a product of tint and accent because the shell
// reads the theme's own colors file rather than resolving the KDE accent
// through kdeglobals — baking it is correct either way, and costs only ini.
func variants(tk *tokens) (map[string]string, error) {
	out := map[string]string{}

	for name := range tk.Palettes {
		// The two schemes take one axis each rather than one between them. Both
		// carry the window background, but they are read by different things and
		// asked about separately: applications answer the sidebar question, the
		// shell answers the panel one.
		for _, sidebar := range []string{sidebarWindow, sidebarView} {
			app, err := appScheme(tk, name, sidebar)
			if err != nil {
				return nil, err
			}
			out[variantDir+"/colors/app/"+name+"-"+sidebar+"/VanillaBoxDark.colors"] = app
		}

		for _, panel := range []string{panelChrome, panelDark} {
			shell, err := shellScheme(tk, name, panel)
			if err != nil {
				return nil, err
			}
			out[variantDir+"/colors/shell/"+name+"-"+panel+"/colors"] = shell
		}

		// The window decoration is the only artwork that paints a palette colour
		// rather than deferring to the scheme, so it is the only thing a tint
		// generates beyond ini files. The shadow is the other axis that reaches
		// it, which makes it a product of the two.
		for shadow := range tk.DecorationShadow {
			deco, err := tk.decoration(name, defaultTitlebar, shadow)
			if err != nil {
				return nil, err
			}
			out[variantDir+"/decoration/"+name+"-"+shadow+"/decoration.svg"] = deco
		}
	}

	for name, p := range tk.Palettes {
		out[variantDir+"/defaults/"+name+"/defaults"] = lookAndFeelDefaults(p.Accent, "", "")
	}

	for name := range tk.ButtonStyles {
		for shadow := range tk.DecorationShadow {
			files, err := tk.titlebarButtons(name, shadow)
			if err != nil {
				return nil, err
			}
			for path, content := range files {
				out[variantDir+"/buttons/"+name+"-"+shadow+"/"+path] = content
			}
		}
	}

	// Only the opaque frames are written as variants. Corners are not a
	// preference any more, so nothing has to be able to re-lay the backgrounds
	// in another shape — but the three transparency switches still need an
	// opaque copy of the surface each one turns solid, and those copies cannot
	// come from the install itself, which is the translucent artwork they are
	// there to replace. The solid/ prefix Plasma falls back to is shipped in
	// the base and no option reaches it, so it stays out of here.
	files, err := containers(tk, defaultPalette, defaultContainers)
	if err != nil {
		return nil, err
	}
	for path, content := range files {
		if opaque, ok := strings.CutPrefix(path, "opaque/"); ok {
			out[variantDir+"/containers/opaque/"+opaque] = content
		}
	}

	return out, nil
}

// containers renders the artwork a container radius reaches: the panel, popup
// and tooltip backgrounds, across all three prefixes.
//
// Colours here are only the stylesheet fallbacks an editor shows, so the tint
// makes no difference to the bytes and the default one is used throughout.
func containers(tk *tokens, name, shape string) (map[string]string, error) {
	palette, _, err := tk.colours(name)
	if err != nil {
		return nil, err
	}
	radii, ok := tk.ContainerShape[shape]
	if !ok {
		return nil, fmt.Errorf("no container shape %q", shape)
	}

	out := map[string]string{}
	for _, prefix := range []string{"", "opaque/", "solid/"} {
		for path, content := range frames(tk, palette, radii, prefix == "") {
			out[prefix+path] = content
		}
	}

	return out, nil
}

// elements renders the artwork an element radius reaches: the stacked control
// states for buttons, text fields and list items.
//
// These carry no opaque or solid copies. Plasma only falls back to those
// prefixes for backgrounds, and a control has none of its own to make opaque.
func elements(tk *tokens, name, shape string) (map[string]string, error) {
	palette, accent, err := tk.colours(name)
	if err != nil {
		return nil, err
	}
	radii, ok := tk.ElementShape[shape]
	if !ok {
		return nil, fmt.Errorf("no element shape %q", shape)
	}

	out := map[string]string{}
	for path, c := range controls(palette, accent, tk.Status, radii["button"], tk.Opacity["button"]) {
		out["widgets/"+path] = c.render()
	}

	return out, nil
}

// frames renders the four background surfaces at one radius. translucent picks
// the theme root's opacities; the opaque/ and solid/ prefixes Plasma falls back
// to when compositing is off never carry any.
func frames(tk *tokens, palette map[string]string, radii map[string]float64, translucent bool) map[string]string {
	popup := frame{
		Size: 44, Canvas: 60, Tile: 14, Radius: radii["popup"],
		Fallback: palette["background"], Mask: true, HintSize: 8, HintY: 48,
	}
	// The tooltip is the one container that appears over arbitrary content
	// rather than over the desktop or a panel it already contrasts with, so it
	// is the one that carries an outline to sit off whatever is behind it.
	tooltip := frame{
		Size: 44, Canvas: 60, Tile: 14, Radius: radii["tooltip"],
		Fallback: palette["tooltip"], Mask: true, HintSize: 4, HintY: 48,
		Border: opacity(tk.Opacity["tooltipBorder"]), BorderFallback: palette["text"], Shadow: true,
	}
	// The panel carries a mask like the other three. Without one KSvg falls back
	// to the background pixmap itself — alphaMask() returns frame->cachedBackground
	// when there is no mask- prefix — so the blur region would be thresholded out
	// of translucent artwork rather than off a clean shape.
	panel := frame{
		Size: 40, Canvas: 56, Tile: 12, Radius: radii["panel"],
		Fallback: palette["background"], Mask: true, HintSize: 2, HintY: 44, Inline: true,
	}

	if translucent {
		popup.Opacity = opacity(tk.Opacity["popup"])
		tooltip.Opacity = opacity(tk.Opacity["tooltip"])
		panel.Opacity = opacity(tk.Opacity["panel"])

	}

	// The launcher and the tray popups are windows, and Plasma asks the
	// compositor to shadow them from this file's shadow- tiles.
	dialog := popup
	dialog.DropShadow = &dropShadow{
		Color: tk.PopupShadow.Color, Strength: tk.PopupShadow.Strength,
		OffsetY: tk.PopupShadow.OffsetY, Radius: tk.PopupShadow.Radius,
		Corner: radii["popup"],
	}

	return map[string]string{
		"widgets/background.svg":       popup.render(),
		"dialogs/background.svg":       dialog.render(),
		"widgets/tooltip.svg":          tooltip.render(),
		"widgets/panel-background.svg": panel.render(),
	}
}

// appScheme renders the colour scheme applications read, for one tint and one
// answer to the sidebar question.
func appScheme(tk *tokens, name, sidebar string) (string, error) {
	s, err := tk.baseScheme(name)
	if err != nil {
		return "", err
	}

	// The id, which is what kdeglobals names and what the look-and-feel package
	// asks for.
	s.SchemeKey = "VanillaBoxDark"
	s.WindowOnView = sidebar == sidebarView

	return s.render(), nil
}

// shellScheme renders the Plasma style's own copy, for one tint and one answer
// to the panel question. Without this file the shell falls back to the system
// scheme and a tint reaches applications but not the panel.
func shellScheme(tk *tokens, name, panel string) (string, error) {
	s, err := tk.baseScheme(name)
	if err != nil {
		return "", err
	}

	// Plasma's own styles name their scheme by display name here, not by id.
	s.SchemeKey = "Vanilla Box Dark"
	s.WindowOnView = panel == panelDark

	return s.render(), nil
}

// baseScheme is everything the two schemes agree on: the surfaces, the accent
// and the status colours for one palette.
func (tk *tokens) baseScheme(name string) (scheme, error) {
	palette, accent, err := tk.colours(name)
	if err != nil {
		return scheme{}, err
	}

	return scheme{
		palette: palette, accent: accent, status: tk.Status, Name: "Vanilla Box Dark",
	}, nil
}

// titlebarButtons renders the four buttons and the layout file for one button
// style. The rc travels with them because its metrics are the button sizes, so
// the two cannot disagree about how big a button is.
//
// Buttons do not multiply by tint: they are painted in the foreground colours,
// and those are held still across every tint. They do multiply by the shadow,
// though nothing about a button changes with it: the rc's Padding is what the
// shadow band has to agree with, and the rc rides in this overlay. The four
// button SVGs are the same bytes in both copies.
func (tk *tokens) titlebarButtons(name, shadow string) (map[string]string, error) {
	bs, ok := tk.ButtonStyles[name]
	if !ok {
		return nil, fmt.Errorf("no button style %q", name)
	}
	sh, ok := tk.DecorationShadow[shadow]
	if !ok {
		return nil, fmt.Errorf("no decoration shadow %q", shadow)
	}

	palette, _, err := tk.colours(defaultPalette)
	if err != nil {
		return nil, err
	}

	out := map[string]string{"VanillaBoxDarkrc": auroraeRC(palette, bs, titleHeight, sh)}

	if bs.Kind == "circle" {
		// Maximize and restore are the same light: the button changes what it
		// does, not what it means.
		for file, colour := range map[string]string{
			"close": bs.Close, "minimize": bs.Minimize,
			"maximize": bs.Maximize, "restore": bs.Maximize,
		} {
			out[file+".svg"] = circleButton{
				Radius: bs.CircleRadius, Rest: bs.RestColor, Dim: bs.DimColor,
				Hover: colour, Pressed: bs.PressedOpacity,
			}.render()
		}

		return out, nil
	}

	plain := button{
		PlateFill: palette["text"], HoverOpacity: bs.PlainHover, PressedOpacity: bs.PlainPressed,
		Radius: bs.PlateRadius, GlyphFill: palette["text"], RestOpen: bs.Rest, DimOpacity: bs.Dim,
		GlyphSize: bs.GlyphSize, Box: float64(bs.Width),
	}

	for file, glyph := range map[string]string{
		"minimize": glyphMinimize, "maximize": glyphMaximize, "restore": glyphRestore,
	} {
		b := plain
		b.Glyph = glyph
		out[file+".svg"] = b.render()
	}

	closeBtn := plain
	closeBtn.PlateFill = bs.ClosePlate
	closeBtn.HoverOpacity = bs.CloseHover
	closeBtn.PressedOpacity = bs.ClosePressed
	closeBtn.Glyph = glyphClose
	out["close.svg"] = closeBtn.render()

	return out, nil
}

// decoration renders the window frame for one tint and decoration shape.
func (tk *tokens) decoration(name, decoShape, shadow string) (string, error) {
	palette, _, err := tk.colours(name)
	if err != nil {
		return "", err
	}
	deco, ok := tk.DecorationShape[decoShape]
	if !ok {
		return "", fmt.Errorf("no decoration shape %q", decoShape)
	}
	sh, ok := tk.DecorationShadow[shadow]
	if !ok {
		return "", fmt.Errorf("no decoration shadow %q", shadow)
	}

	padL, padR, padT, padB := sh.padding()

	layers := make([]shadowLayer, 0, len(sh.Layers))
	for _, l := range sh.Layers {
		layers = append(layers, shadowLayer{
			// Breeze's box is deflated by the overlap and then pushed down by
			// the composite offset and the layer's own, so relative to the
			// window each layer's box starts this far below its top edge.
			DY:      sh.OffsetY + l.OffsetY,
			Radius:  l.Radius,
			Opacity: l.Opacity * sh.Strength,
		})
	}

	// The fixed tiles have to hold the shadow's corner, which settles only after
	// about 3*sigma. That sets the corner tile, and with it the frame's width.
	reach := sh.reach()
	tile := decoTile
	if min := sh.Overlap + reach; min > tile {
		tile = min
	}

	return decoration{
		Width: 2*tile + decoMid, Tile: tile,
		TitleH: titleHeight + titleBorder, BodyH: 24,
		Radius: deco["titlebar"], Border: palette["elevatedAlt"], Backgnd: palette["background"],
		PadL: padL, PadR: padR, PadT: padT, PadB: padB,
		Overlap: sh.Overlap, Reach: reach, Layers: layers, ShadowColor: sh.Color,
	}.render(), nil
}

// build returns every generated file for one point in the variant space, keyed
// by repository-relative slash path.
func build(tk *tokens, name, containerShape, elementShape, decoShape, buttons, shadow string) (map[string]string, error) {
	_, acc, err := tk.colours(name)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}

	// The shipped pair, each at its own default: applications get the merged
	// sidebar, the shell keeps its panel on the chrome colour.
	app, err := appScheme(tk, name, defaultSidebar)
	if err != nil {
		return nil, err
	}
	out[schemeDir+"/VanillaBoxDark.colors"] = app

	shell, err := shellScheme(tk, name, defaultPanel)
	if err != nil {
		return nil, err
	}
	out[style+"/colors"] = shell

	// The theme root carries the translucent artwork. Plasma falls back to the
	// opaque/ and solid/ prefixes itself when compositing is off, so both ship
	// whatever the transparency options are set to.
	cont, err := containers(tk, name, containerShape)
	if err != nil {
		return nil, err
	}
	for path, content := range cont {
		out[style+"/"+path] = content
	}

	elem, err := elements(tk, name, elementShape)
	if err != nil {
		return nil, err
	}
	for path, content := range elem {
		out[style+"/"+path] = content
	}

	deco, err := tk.decoration(name, decoShape, shadow)
	if err != nil {
		return nil, err
	}
	out[auroraeDir+"/decoration.svg"] = deco

	btns, err := tk.titlebarButtons(buttons, shadow)
	if err != nil {
		return nil, err
	}
	for path, content := range btns {
		out[auroraeDir+"/"+path] = content
	}

	out[lookFeel+"/contents/defaults"] = lookAndFeelDefaults(acc, "", "")

	// Identity: the same handful of facts KDE wants in three formats, plus the
	// installer's own copy of the version.
	id := tk.Theme
	out[style+"/metadata.json"] = id.kPluginMetadata(id.StyleID, "X-Plasma-API", "5.0")
	out[lookFeel+"/metadata.json"] = id.kPluginMetadata(
		id.LookAndFeelID, "KPackageStructure", "Plasma/LookAndFeel")
	out[auroraeDir+"/metadata.desktop"] = id.desktopEntry()
	out["internal/theme/version.go"] = id.versionSource()

	return out, nil
}

// controls builds the widget artwork that shares the stacked nine-tile idiom:
// buttons, text fields and list items.
func controls(
	palette map[string]string, accent string, status map[string]string,
	radius, buttonOpacity float64,
) map[string]control {
	// The button's surface is the one control fill that is not fully opaque, so
	// what it sits on shows through it a little. Written per layer rather than
	// on the tile, because the wash stacked over it keeps its own value.
	surface := n(buttonOpacity)

	sheet := fmt.Sprintf(
		".ColorScheme-Text { color:%s; }.ColorScheme-Highlight { color:%s; }"+
			".ColorScheme-ButtonBackground { color:%s; }.ColorScheme-ButtonHover { color:%s; }",
		palette["text"], accent, palette["elevated"], status["hover"])

	const (
		text    = "ColorScheme-Text"
		hi      = "ColorScheme-Highlight"
		btnBg   = "ColorScheme-ButtonBackground"
		btnHvr  = "ColorScheme-ButtonHover"
		fullTop = 8

		// The padding every control reported before the corner radius grew.
		// Pinning it keeps the artwork's corners and the control's size
		// independent: fullTop is now the box a radius-8 corner needs, and this
		// is what the control actually measures.
		pad = 6
	)

	// set places a state at its slot down the canvas, so the offsets stay
	// derived rather than repeated.
	set := func(slot int, prefix string, tile, margin int, layers ...layer) tileSet {
		return tileSet{
			Prefix: prefix, Y: slot * controlStep, Tile: tile,
			Radius: radius, Layers: layers, Margin: margin,
		}
	}

	return map[string]control{
		"button.svg": {
			Height: 7 * controlStep, Style: sheet,
			Sets: []tileSet{
				set(0, "normal", fullTop, pad, layer{btnBg, surface}),
				// Hover is the one state Plasma draws outside the control it
				// belongs to: ButtonHover.qml fills the button and then pushes
				// all four edges back out by this prefix's own margins, so the
				// prefix is a halo rather than a surface. Breeze paints its
				// hover-center at opacity 0.001 and inks only the border.
				//
				// hint-no-border-padding below makes the margins report zero, so
				// there is nothing to push out by and the frame lands on the
				// button exactly — while the tiles keep their own widths and draw
				// the corners. It paints only the wash, since the normal state is
				// still drawn underneath. See docs/plasma-controls.md.
				//
				// It washes harder than the flat button below, because the two are
				// not doing the same job: this one lightens a surface the button
				// already has, where a flat button's hover is the only surface it
				// ever gets and lands straight on whatever it sits on.
				set(1, "hover", fullTop, 0, layer{text, "0.15"}),
				set(2, "pressed", fullTop, pad, layer{btnBg, surface}, layer{btnHvr, "0.45"}),
				set(3, "focus", fullTop, pad),
				set(4, "toolbutton-hover", fullTop, pad, layer{text, "0.08"}),
				set(5, "toolbutton-pressed", fullTop, pad, layer{text, "0.15"}),
				set(6, "toolbutton-focus", fullTop, pad),
			},
			Hints: []string{"hover-hint-no-border-padding"},
		},
		"lineedit.svg": {
			Height: 3 * controlStep, Style: sheet,
			Sets: []tileSet{
				set(0, "base", fullTop, pad, layer{text, "0.06"}),
				set(1, "hover", fullTop, pad, layer{text, "0"}),
				set(2, "focus", fullTop, pad, layer{text, "0"}),
			},
			Hints: []string{"hint-focus-over-base", "hint-hover-over-base"},
		},
		"viewitem.svg": {
			Height: 5 * controlStep, Style: sheet,
			Sets: []tileSet{
				// The resting state draws nothing, and its tiles are inset 3 rather
				// than 6 so a list row's margins do not follow the hover radius.
				set(0, "normal", 3, 0),
				set(1, "hover", fullTop, pad, layer{hi, "0.25"}),
				set(2, "selected", fullTop, pad, layer{hi, "0.25"}),
				set(3, "selected+hover", fullTop, pad, layer{hi, "0.25"}),
				set(4, "focus", fullTop, pad, layer{text, "0.08"}),
			},
		},
	}
}

// opacity renders a fill-opacity, treating a fully opaque surface as no
// attribute at all rather than an explicit 1.
func opacity(v float64) string {
	if v == 0 || v == 1 {
		return ""
	}

	return n(v)
}

func loadTokens(path string) (*tokens, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	tk := &tokens{}
	if err := json.Unmarshal(data, tk); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	return tk, nil
}
