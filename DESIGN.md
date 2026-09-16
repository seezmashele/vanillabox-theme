# Variants

How Vanilla Box supports more than one look without turning into a directory of near-identical
copies. This document covers the architecture; [README.md](README.md) covers what the installer
does today.

The goal is that adding a variant is an edit to a token file, not a new tree of artwork, and that
the number of hand-maintained files stays flat as variants are added.

## The governing constraint

Two kinds of variation are possible, and they behave completely differently.

**Colour is a runtime property.** Plasma renders theme SVGs through `KSvg`, which substitutes the
`current-color-scheme` stylesheet at paint time from a `colors` file shipped inside the theme. The
proof ships with Plasma: `/usr/share/plasma/desktoptheme/breeze-dark/` contains exactly three
files — `colors`, `metadata.json`, `plasmarc` — and no artwork at all. It renders Breeze dark
purely by substitution.

**Geometry is not.** A corner radius lives in path data (`d="M0,8 C0,3.582 3.582,0 8,0"`) and in
the tile offsets around it. `KSvg` draws through `QSvgRenderer`, which implements roughly SVG 1.2
Tiny: no CSS custom properties, no stylable `rx`, no dependable `<use>`. There is no runtime knob
for shape, and no SVG trick will produce one. Square artwork means different bytes.

Treating both as one problem multiplies: 5 palettes x 2 button styles x 2 shadows is 20
combinations of 25 artwork files, and that is before a shape axis doubles it again. Separating them
makes the cost additive instead, because **each axis owns a disjoint set of files**. Shape was once
three of these axes and is now none of them — see [Corners are not an axis](#corners-are-not-an-axis)
— but the separation is what made dropping them a deletion rather than a rebuild.

## The axes

| Axis | Default | Mechanism | Files it owns |
| --- | --- | --- | --- |
| Palette | `neutral` | runtime, plus one SVG | the two `colors` files, the look-and-feel `defaults`, `decoration.svg` |
| Sidebar | `view` | runtime, as a product with the palette | `color-schemes/VanillaBoxDark.colors` |
| Panel tint | `off` | runtime, as a product with the palette | the style's `colors` |
| Button style | `windows` | baked | the four button SVGs and `VanillaBoxDarkrc` |
| Window shadow | `on` | Breeze's, baked as a product with the palette | `aurorae/decoration.svg`, the rc's `Padding*` |
| Transparency (x3) | all on | per-file overlay | one file each, from `opaque/` |

### Corners are not an axis

Corners were three axes once — container, element and titlebar shape, each offering rounded and
square. They ship as neither now: the theme rounds, and the radii come from one small scale rather
than a radius tuned per component.

| Surface | Radius | Token |
| --- | --- | --- |
| Titlebar (top corners) | 12 | `decorationShape.rounded.titlebar` |
| Panel strip | 12 | `containerShape.rounded.panel` |
| Popups, menus, applet backgrounds | 12 | `containerShape.rounded.popup` |
| Tooltips | 8 | `containerShape.rounded.tooltip` |
| Buttons, inputs, list items | 8 | `elementShape.rounded.button` |
| Titlebar button plate (Symbols) | 8 | `buttonStyles.windows.plateRadius` |

Two values, shared. What the eye reads as a surface takes 12 — the titlebar, the panel strip, and
the popups and applet backgrounds that are surfaces in their own right, the start menu and the
system tray among them. What sits *on* a surface takes 8: tooltips, buttons, inputs, list items.
That a popup and the panel land on the same figure is the point rather than a collision. A theme
reads as one system when its corners come from a scale, not when every component argues its own
case.

The tooltip is the instructive one. It is a container, and it takes the small radius anyway,
because it is a strip of text rather than a surface — 12 on something two lines tall is a lozenge,
and the radius has to be read against what carries it rather than against what the file is called.
It has a token of its own, `containerShape.rounded.tooltip`, for exactly that reason: it shared the
popup's until the popups grew.

The control radius has a ceiling the containers do not, and it is what sets the bottom of the
scale. Controls are nine-tile sets on a 24px canvas, so a corner tile has to stay well under half
the width or the two corner arcs meet and leave the stretchable centre nothing to be; 8 leaves a
centre of 8, and 12 would leave none at all. 8 is therefore the top of what a control can carry,
and the containers meet it there.

Raising it past 6 also meant declaring the controls' margins. KSvg takes a prefix's padding from
its tile size unless `hint-*-margin` says otherwise, and the tile had to grow to hold the bigger
corner — so without the hints, a rounder button would also have been a bigger one. The margins are
pinned at the 6 they already reported. See [docs/plasma-controls.md](docs/plasma-controls.md).

The titlebar accepts a known compromise for it: Aurorae cannot round a window's bottom corners, so
a rounded titlebar tops a body that still ends square. That is the shipped look and the only one.
See the Aurorae reason below.

Every choice now lists the value it installs first, which is a presentational decision rather than
a structural one: the cursor opens on the value already selected rather than on the first row, so
the listing order and `defaultValue` remain free to disagree. They simply no longer do.

Corners were three axes before they were none, and the reasoning is worth keeping because it is
what made removing them cheap. They were first one axis, on the grounds that a corner radius is a
corner radius; that was wrong in an interesting way, because the surfaces a thing sits in and the
things sitting in it are separately convincing, and holding them together made the combination
people actually reached for — rounded panels, square controls — unreachable. Splitting them cost
nothing structurally, because the two own disjoint sets of files: containers own the backgrounds,
elements own the widget artwork.

What the split finally showed was that the combinations it unlocked were not ones the theme wanted
to ship. A square popup around rounded buttons is reachable and nobody chooses it; offering it put
three prompts in front of every user to protect a look the theme does not believe in. So the axes
came out, and because each owned a disjoint set of files, taking them out was a deletion rather
than a rebuild — the artwork the defaults already produced is simply the artwork now. What remains
of that separation is the scale in [Corners are not an axis](#corners-are-not-an-axis): containers
and elements still carry their own radii, they just no longer carry their own questions.

Every value of a baked axis carries an overlay, including the default one. A value that copies
nothing would be correct only while the base artwork happens to be what it describes, and the base
has changed more than once — surfaces from rounded to square and back, buttons from symbols to
traffic lights and back again. Making every value carry its own overlay keeps the option
independent of that. The window buttons are the axis this still governs.

A palette is a surface set and the accent that goes with it, chosen together. Surfaces and
highlights were two independent axes at first, on the reasoning that they are two questions. They
are — but a curated pair is a defensible answer to both, and eighteen combinations is a great deal
of menu for something most people set once and never revisit. Five named variants say more about
what the theme is for than eighteen coordinates do.

The cost is real: rose surfaces with a steel accent is no longer reachable. If that turns out to
matter the axes split again — the generator already writes a product elsewhere and would do it here
without ceremony.

Accents match their surfaces in temperature, so a variant reads as one decision rather than two.
`neutral` is the exception and takes the grey surfaces, which is what makes it the quiet one rather
than a colour with the volume turned down. Surfaces are named separately from palettes and
referenced by name — a set can then be shared, and renaming a variant does not mean renaming the
colours it points at.

A palette moves surfaces and its accent. Text, inactive text and the colour that sits on the
highlight are held still across all five, because warm text on a blue surface reads as a mistake
rather than as a variant.

`onHighlight` is the one that does not survive the rule, and `forest` is why. An accent is the
selection background — the only colour a palette moves that text sits *on* rather than beside — and
at `#4a6d41` the shared dark on-highlight colour reads at 2.8:1, which is not a foreground. The
light `text` colour reads at 4.7:1 on the same green. The crossover is around `#5c8452`: above it
the dark colour wins, below it the light one does.

So a palette may override `onHighlight`, and only `forest` does. The alternative was moving the
foregrounds for everyone, which is the question holding them still exists to avoid — and picking a
green by what the selection text needed rather than by what the palette is called.
`TestForestInvertsItsSelectionText` pins both halves: forest takes the light colour, and the other
four are checked for still taking the dark one, because an override that leaked would look like a
theme-wide change nobody asked for.

The accent is also `ForegroundLink`, drawn *on* the background rather than under text, and that one
has no override to hide behind. Forest links sit at 2.5:1 against its surfaces where the other four
palettes are near 4.6:1. It is the price of the green: the link colour would have to stop being the
accent to fix it, which would cost the theme the thing that makes an accent read as one decision.

That also keeps a palette almost free. The only artwork it repaints is `decoration.svg`, which
paints the titlebar directly instead of deferring to the scheme; everything else either resolves
its colour at paint time or carries the palette as nothing more than an editor fallback.

Rounding the decoration carries a compromise. The comment in `aurorae/themes/VanillaBoxDark/
decoration.svg` explains that Aurorae cannot round the bottom corners of a window — rounding needs
a bottom border to draw the curve into, and anything narrower than the radius lets the client's
square corner show through. The theme accepts it: rounded top, square bottom, on every window.

One file is claimed by two axes: `decoration.svg`, which the palette paints and the shadow surrounds.
It is written as a product, `variants/decoration/<palette>-<shadow>/`. See
[Resolved files](#resolved-files).

### The shadow is artwork, not a compositor request

Aurorae has no shadow prefix. `aurorae.qml` reads exactly six: `decoration`, `decoration-inactive`,
their two maximised twins, and `innerborder` with its inactive copy. There is nowhere to declare a
shadow the way `widgets/tooltip.svg` declares one.

What it has instead is padding. `Decoration.qml` sizes the decoration as client + borders +
padding, and the `decoration` frame is anchored `fill: parent` — so the padding band is inside the
frame's own nine tiles. A shadow is therefore ordinary artwork living in the outer band of the
same sheet, and `PaddingLeft` and friends are what tell Aurorae the band is there at all. The two
have to agree exactly: too little padding crops the shadow, too much leaves empty space around
every window.

### The shadow itself is Breeze's

The look is not invented. Breeze's `createShadowObject` composites two Gaussian-blurred rounded
boxes and hands the result to the compositor through `KDecoration3::DecorationShadow` — an API
Aurorae does not expose, so its *method* is unavailable to an SVG theme. Its *values* are not:
`s_shadowParams` for `ShadowSmall` is an offset of `(0, 4)` over shadows of radius 16 at opacity 1
and radius 8 at opacity 0.4, the latter offset `(0, -2)`. Those, with `ShadowStrength` 255 and
`ShadowColor` black, are what `spec/tokens.json` carries.

They can be transplanted because a Gaussian-blurred *rectangle* has a closed form, and a separable
one. Breeze blurs with three box passes standing in for a Gaussian of `stdDev = radius / 2`; a
Gaussian is separable, so the field is the product of a horizontal profile and a vertical one, each
of them the Gaussian integral of a single edge. No blur is needed to draw it, which matters, since
QtSvg implements SVG 1.2 Tiny and supports no filters at all — `feGaussianBlur` would not draw.

The product is the part worth getting right. An edge tile varies along one axis only, its other
profile having settled, so it is one rect carrying one gradient. A corner varies along both, and
SVG cannot multiply two gradients — but it can scale one by a constant, so a corner is drawn as
one-pixel strips, each carrying the side's gradient at a `fill-opacity` of the vertical profile for
that row. `fill-opacity` multiplies, which is the operation wanted, and costs no offscreen buffer.

Taking the profile of the distance to the nearest edge instead — treating a corner as radial — is
the obvious thing and it is wrong. On the diagonal it gives `P(d)` where the truth is
`P(d/sqrt2)^2`: about twice the opacity, so the shadow bunches at the corners. It also leaves a
step wherever the nearest edge changes, which is a visible line along the bottom.

### Why Small rather than the Large Breeze ships

Each profile settles to its edge value only after about three standard deviations, and a stretched
tile cannot vary, so that settling has to happen inside the *fixed* tiles. Sideways that is free:
the corner tile is made as wide as it needs to be, and the excess sits over the client, which draws
on top of it. Vertically it is not — the fixed row is the titlebar, and the titlebar is 30px.

Large's sigma is 24 and wants 72px of it. Small's is 8 and wants 21px, which fits. A shadow that
does not fit fails quietly, as a step where the corner tile meets the edge tile beside it, so
`TestShadowFitsTheFixedTiles` measures the room against the requirement.

Two details are Breeze's and easy to miss. `Metrics::Shadow_Overlap` deflates each box by 3px, so
the shadow starts just inside the window rather than at its edge. And the composite offset pushes
the boxes down, which is why the padding is asymmetric — 16 above against 24 below.

The padding is derived rather than written down, exactly as `createShadowObject` derives it: blur
extents of 23 and 11 give a box of 47, a canvas of 93, and after the overlap and offset come off
each side, 20 at the sides. `TestShadowMatchesBreeze` recomputes all of it and the resulting alpha
at the window's edge.

The two states share one sheet, the inactive copy stacked below the active one, and the step
between them is derived rather than named. It was a constant while a state was the titlebar, the
body and a one-pixel border; a band on every side makes a state twice that, and a step shorter than
a state draws the inactive copy over the active one's lower half. The two shadows then add up —
worst where the inactive corners land, which reads as a shadow bunched at the corners instead of
following the window. `TestStatesDoNotOverlapOnTheSheet` holds the step to the state.

Maximised windows get no shadow, and want none: `Decoration.qml` drops the padding when maximised,
so the band is not drawn and there is no gap where the window meets the screen edge.

`VanillaBoxDarkrc` turned out to belong to one axis after all. Its `[Layout]` metrics are the
button sizes, and the titlebar height does not change with the corner radius — so it travels with
the button style as part of that overlay rather than needing a combination of its own. Shipping the
metrics beside the artwork they describe also makes it impossible to swap one without the other.

The shadow later gave it a second axis anyway, though not through the buttons: `Padding*` is the
one metric the shadow moves, so the overlay became `buttons/<style>-<shadow>/`. The four button
SVGs are identical across the pair — the duplication buys keeping the rc beside them.

## Colour

### Prerequisite — done

The runtime mechanism did not work on the original artwork. The backgrounds carried the class but
hardcoded the fill, so `currentColor` was never consulted and a `colors` file would have been
silently ignored for exactly the surfaces a palette needs to reach:

```
widgets/background.svg        class="ColorScheme-Background" style="fill:#292929;fill-opacity:0.85"
widgets/panel-background.svg  (no class at all)              style="fill:#292929;fill-opacity:0.85"
```

Nine files were converted to `fill:currentColor` — `widgets/background.svg`,
`widgets/panel-background.svg` and `dialogs/background.svg`, plus their `opaque/` and `solid/`
copies — with `widgets/tooltip.svg` as the model. The panel files needed the class adding as well;
they had none. `fill-opacity` is a separate attribute and was unaffected, so the translucency
design and the overlay scheme survived untouched.

Thirteen surfaces now defer to the scheme. `widgets/scrollbar.svg` is among them and is easy to
overlook, which is why `TestShippedStyleFollowsTheColorScheme` asserts the expected set by path
rather than by count.

### Two consumers, two files

| File | Consumer | Controls |
| --- | --- | --- |
| `color-schemes/VanillaBoxDark.colors` | Qt/KDE apps | window and view backgrounds, menus, buttons, text |
| `plasma/desktoptheme/vanilla-box-dark/colors` | Plasma shell | panel, popups, dialogs, tooltips |

### Rules for the generator

- `Window`, `Header` and `Complementary` backgrounds move together within a surface set. Plasma
  resolves a surface against one of the three depending on the widget's colour set; keeping them
  means never having to work out which. The current scheme already satisfies this at `41,41,41`.
  The two window-background options are the one sanctioned exception, and they break the rule in
  the narrowest way they can — see below.
- `.ColorScheme-ButtonHover` reads `[Colors:Button] DecorationHover`. `button.svg` says `#9e9e9e`
  and the scheme says `158,158,158`; both must be emitted from one token so they cannot drift.
- `Colors:Selection` comes from the palette's accent, not from its surfaces. It is the one role
  that reads from the other half of the pair.
- `Colors:Tooltip` takes `view` rather than a chrome surface. A tooltip is the one popup that
  appears over arbitrary content, so it reads as a dark card rather than as one more shade of the
  window it covers — the same reason it is the only container that carries a border. Its
  `DecorationFocus` follows its background, as it does in every set that does not want a visible
  focus ring; only `Complementary` and `View` deliberately differ.
- The colours embedded in each SVG's `current-color-scheme` block are fallbacks — what an editor
  shows. They should stay accurate for the default palette, but do not drive rendering.
  `TestShippedStyleFollowsTheColorScheme` guards the deferral itself.

### Moving the window background

Two options move `[Colors:Window] BackgroundNormal` onto the view colour. They are the same edit to
two different files, asked as two questions, and the reason they are two is worth writing down.

**The sidebar option**, in the application scheme. KColorScheme has no sidebar role. A places panel
— Dolphin's, Kate's, anything in a `QDockWidget` — paints with the window background, which is why
it matches the toolbar rather than the list beside it. So "make the sidebar the view colour" can
only be spelled as "move the window background", and that reaches every window background in every
Qt app, not just panels. Dialogs, settings pages and message boxes go dark with it.

**The panel tint option**, in the style's own `colors`. Every background the shell paints —
`panel-background.svg`, `dialogs/background.svg`, `widgets/background.svg`, and their `opaque/` and
`solid/` copies — carries `ColorScheme-Background`, which resolves against the same role. So the
panel strip, the launcher and every applet popup move together, and nothing else on the desktop
does.

`Header` deliberately does not follow, in either file. It is the one place the move-together rule
above is broken, and breaking it is the point: a sidebar merged into the file list still wants a
toolbar above it that is not, and `Header` is the role that draws it. `Complementary` does follow
`Window`, so of the two roles a widget might resolve a plain window surface against, neither can
disagree with the other.

#### Why they are two axes and not one

They shared a flag once, and choosing the merged sidebar darkened the panel and the launcher as a
side effect. Nothing in Plasma ties the two files, and two things argue for keeping them apart:

- **The shell has no `Header` to fall back on.** The escape hatch that makes the move tolerable in
  an application — a toolbar left on the chrome colour above the merged panel and list — does not
  exist on the desktop. `plasmoidheading.svg` is `fill:none` throughout, so nothing in the style
  paints `Header` at all, and a dark panel leaves no chrome-coloured surface anywhere on screen.
- **`Colors:Tooltip` stops meaning anything.** The tooltip is deliberately on the view colour so it
  reads as a card over what it covers. Move the popups there too and the two are the same colour,
  leaving a tooltip over a launcher distinguishable only by its outline.
  `TestTooltipSitsOffThePopupBackground` pins the gap.

With the panel on the chrome colour it matches the toolbars and the titlebar, which paints
`background` regardless of either option: chrome is the panel, the toolbars and the titlebar; the
view colour is the file list, the sidebar and the window body. That is why `off` is the default,
and why `on` is still worth offering — a desktop where everything but the toolbars is the view
colour is a coherent look, just not the shipped one.

Both are products with the palette rather than overlays, because each `colors` file carries the
window background and there is no file to swap that does not also carry the tint —
`variants/colors/app/{palette}-{sidebar}/` and `variants/colors/shell/{palette}-{panel-tint}/`, ten
directories each for five palettes. `TestSidebarMovesOnlyTheWindowBackground` and
`TestPanelTintMovesOnlyTheShell` pin which sections are allowed to move;
`TestTheTwoSchemesTakeSeparateAxes` and `TestPanelTintAndSidebarAreSeparateInstalls` pin that
neither axis reaches the other's file.

Surfaces are written as explicit values per set, not derived by a hue or chroma transform. A
computed shift behaves badly at the lightness of `#141414`, and the near-blacks want hand-tuning.
Six surface roles across five sets is thirty numbers, plus one accent per palette.

## Accent

Accent is not part of the colour scheme format. None of the schemes shipped with Plasma carries an
`AccentColor` key; it lives in `kdeglobals [General] AccentColor`, which the look-and-feel
`defaults` file writes, alongside `accentColorFromWallpaper=false` so KDE cannot override the
choice from the desktop picture.

For applications that is the whole mechanism — one line, and KDE tints selection, focus rings and
checkboxes from it at runtime.

The Plasma shell is the part that was never settled. Because the theme ships its own `colors` file,
the shell reads that file directly rather than resolving through `kdeglobals`, so the accent may
not reach panel artwork on its own. Rather than answer the question, the accent is written into
both: if the shell does resolve `kdeglobals` the second copy changes nothing, and if it does not,
the second copy is the only thing that works.

### What the accent reaches

`Colors:Selection` in both `colors` files, and `AccentColor` in `kdeglobals`. The theme's greyscale
was once deliberate — selection was `143,143,143` and the active task underline drew from
`.ColorScheme-Highlight` rather than from a colour — and an accent that reached only application
focus rings would be close to invisible in a desktop this neutral.

Under the default `neutral` palette that makes the active task underline tan rather than grey. An
`ash` palette once held the grey `143,143,143` selection as a way of keeping the original
no-colour-anywhere look reachable; it was dropped, because a variant whose only content is the
absence of the accent is a menu entry explaining a decision rather than offering one.

## Buttons

The button style axis is not only artwork.

**Order is a KWin setting.** Mac-style buttons sit on the left, which lives in `kwinrc
[org.kde.kdecoration2] ButtonsOnLeft` / `ButtonsOnRight` — that is, in the look-and-feel `defaults`
file, which therefore becomes generated. The shape of the change is `ButtonsOnLeft=XIA` with an
empty `ButtonsOnRight`, against a default of `ButtonsOnRight=IAX`. **The exact letter codes are
unverified** — confirm them once against System Settings -> Window Decorations before relying on
them.

### Metrics as shipped

These were tuned by eye against a real titlebar rather than derived from anything, so they are
worth writing down. `TestTitlebarButtonMetrics` pins them.

| | Symbols | Traffic lights |
| --- | --- | --- |
| Button box | 28 x 28 | 22 x 22 |
| `glyphSize` / `circleRadius` | 13 | 6 |
| **Rendered mark** | 13 x 13 px | 11 px across |
| `nudgeTop` | -1 | -1 |
| `ButtonMarginTop` | 0 | 3 |
| `ButtonWidthMenu` | 20 | 16 |
| Plate | square, no radius | n/a |

Both boxes are square on purpose. Aurorae scales the 24x24 tile to `ButtonWidth x ButtonHeight`, so
a box of 28x26 stretched every symbol 7.7% wider than tall — a circle in a glyph stopped being a
circle. Keeping the box square makes the scale uniform.

**The two size tokens are measured differently, which is a trap.** `glyphSize` is in rendered
pixels: the generator converts it back into tile units so the number in the tokens is the number
you measure on screen. `circleRadius` is still in tile units, so it scales with the button box —
growing the box from 20 to 22 took the circles from 10px to 11px without the token changing.
Worth unifying if the circles are ever tuned as carefully as the glyphs were.

**The application icon is the one button sized on its own.** Aurorae gives every button the same
`ButtonHeight` and only the width is per-type, through `ButtonWidthMenu` — so the icon can be made
smaller sideways and the leftover height is what gives it room above and below. Raising
`ButtonMarginTop` instead would have pushed the close, minimise and maximise buttons down with it.

`TitleEdgeLeft` is 8 against a `TitleEdgeRight` of 6 for the same reason: the icon sits in the
corner of the window and wants more of a margin there than the buttons at the other end do.

**`ButtonMarginTop` is derived**, as `(TitleHeight - ButtonHeight) / 2 + nudgeTop`. Aurorae places a
button that far from the top of the titlebar and leaves the remaining slack below it, so a zero
margin sits every button high. `nudgeTop` is the optical correction on top of that arithmetic, and
it is relative: changing a button height moves the centre, so the nudge has to be revisited to hold
the same edge alignment. The symbols' -1 is what makes their hover plate sit flush against the
titlebar's top border.

The circles stay centred in their own tile so the hit area still matches what is drawn; it is the
button box that moves.

### Maximised windows lay out from different keys

Aurorae positions a maximised titlebar from an entirely separate set of `*Maximized` keys, and
every one of them defaults to zero rather than to its ordinary counterpart. `AuroraeButtonGroup.qml`
computes the button offset as

```qml
maximised ? titleEdgeTopMaximized + buttonMarginTopMaximized
          : titleEdgeTop + padding.top + buttonMarginTop
```

Leaving them unset moved every button the moment a window was maximised: 1px up for the symbols,
4px up for the traffic lights, and 7px outward for both. Nothing in the theme was wrong — the
second branch simply read values nobody had written.

Each maximised key now mirrors its ordinary counterpart, and deliberately does **not** add the
padding the other branch adds. Padding is the frame outside the window: the decoration's origin
sits that far out from the window edge when a window is restored, and exactly on it when maximised.
The ordinary branch adds padding to reach the place the maximised branch already starts from, so
adding it to both puts every button a pixel out — which is what a first attempt at this did.

The edges also feed the titlebar height. `borderTopMaximized` is
`titleEdgeTopMaximized + TitleHeight + titleEdgeBottomMaximized`, so padded edges additionally made
a maximised titlebar 2px taller than a restored one — a second, larger error hiding behind the
first.

These values are **confirmed on a real desktop**, not only reasoned about: the buttons hold their
position across restore and maximise. `TestTitlebarButtonMetrics` asserts them alongside the
ordinary metrics, because this is precisely the kind of thing that is invisible until someone
maximises a window.

The working set, for reference:

```ini
TitleEdgeTop=0                 TitleEdgeTopMaximized=0
TitleEdgeBottom=0              TitleEdgeBottomMaximized=0
TitleEdgeLeft=6                TitleEdgeLeftMaximized=6
TitleEdgeRight=6               TitleEdgeRightMaximized=6
ButtonMarginTop=3              ButtonMarginTopMaximized=3     ; traffic lights; 0 for symbols
PaddingTop=1  PaddingBottom=1  PaddingLeft=1  PaddingRight=1
```

### Applying a decoration change

KWin reads an Aurorae theme **once**, and caches it for the life of the session. Reinstalling puts
new files on disk and changes nothing on screen.

What was confirmed to work is a full restart. Switching to another window decoration in System
Settings and back is the obvious lighter alternative and may be enough, but it has not been shown to
be — a change that appeared not to work here turned out to have been correct on disk the whole time,
and the only thing that had failed was getting KWin to read it.

That is worth remembering before concluding that a decoration edit did not work: check the
installed `VanillaBoxDarkrc` first, and only then doubt the values.

**The interaction model differs.** The Windows-style buttons are monochrome glyphs at rest that
gain a square coloured plate on hover. The Mac style has no glyphs at all: three grey circles at rest,
which take a muted traffic-light colour on hover. This is a per-style treatment, not a shared
pattern with different values.

The traffic-light colours are muted rather than the authentic `#ff5f57`/`#febc2e`/`#28c840`. In a
desktop whose selection is grey and whose accents top out around `#b8776a`, authentic values would
be the most saturated pixels on screen by a wide margin.

**Hover is per-button, and cannot be otherwise.** On macOS, hovering any of the three lights up all
three. Aurorae renders each button from its own SVG with no knowledge of its neighbours, so here
hovering close colours only close. This is a limitation of the format, not a choice.

**The buttons stay on the right,** whichever shape is chosen. Left is the macOS convention and the
only place the traffic-light shape normally appears, so choosing traffic lights here gives the
shape without the placement. It is deliberate:
button order is a `kwinrc` setting rather than an Aurorae file, so writing it would overwrite a
preference the user may have set for reasons of their own — and they can move the buttons in System
Settings at any time without reinstalling. Left is what a macOS switcher, an RTL locale (where KWin
mirrors the layout regardless) or an elementary-style desktop would expect.

Leaving the order alone also means the KWin button letter codes never have to be verified, which is
why that open question is now closed rather than answered.

## Transparency

The toggles are per-surface:

| Toggle | File | Covers |
| --- | --- | --- |
| Panel | `widgets/panel-background.svg` | the panel strip |
| Popups & menus | `dialogs/background.svg` | the application launcher **and** system tray popups |
| Applets | `widgets/background.svg` | plasmoid content areas |

There are three, not the four the `opaque/` tree would suggest. `widgets/tooltip.svg` is opaque in
the base artwork — `opacity.tooltip` is zero, so the root file and its `opaque/` copy are byte for
byte the same — and a switch that cannot change anything is worse than no switch. A fourth toggle
becomes real the moment tooltips are given an opacity, and not before.

The launcher and the tray popups cannot be separated. Both are `PlasmaCore.Dialog` instances and
Plasma ships exactly one `dialogs/background` for all of them; the surfaces are not distinguishable
at the artwork layer by any theme. The option is worded "Popups & menus" so the grouping is honest
in the UI.

`widgets/tasks.svg` remains excluded, for the reason given in the README: its `0.3`/`0.4` values
are white highlights drawn on the panel, not backgrounds.

Overlay granularity changes from directory to file. A whole-directory overlay would need one
directory per combination — eight for three independent toggles. Instead each toggle names the
single file it replaces, drawn from the same `opaque/` tree. `opaque/` and `solid/` continue to be
installed wholesale regardless of any toggle, because Plasma falls back to those prefixes itself
when compositing is off.

An overlay's `from` is relative to the **asset directory**, not to the component's source. The
original transparency option read its overlay out of the tree it had just installed, which was neat
while every overlay lived inside one component. It does not survive a palette, whose two `colors`
files belong to two different components, so overlays now name an asset-relative directory and the
two cases work the same way.

`from` also takes `{option-id}` placeholders — `variants/buttons/{style}-{shadows}` is one — though
the transparency switches no longer need them. They did while corners were a preference: their
source read `variants/containers/{containers}/opaque`, because two overlays land on the same file
there, and a switch that drew its opaque copy from a fixed path would have put rounded corners back
on exactly the surfaces the user made opaque. With one shape left the path is fixed on purpose,
`variants/containers/opaque`, and the placeholder mechanism stays for the axes that still vary.

Only backgrounds have an opaque copy to fall back to, which is why the switches reach the frames
and nothing else: a control has no compositing fallback for them to choose between.

## Tokens

`spec/tokens.json` is the only file edited to add or adjust a variant.

```json
{
  "theme":      { "name":"Vanilla Box Dark", "version":"0.2.0", "…":"…" },
  "foreground": { "text":"#e8e4dd", "textInactive":"#8a8782", "onHighlight":"#1f1f1f" },

  "surfaces": {
    "grey":   { "background":"#292929", "elevated":"#3d3d3d", "view":"#141414", "…":"…" },
    "slate":  { "background":"#272a2f", "…":"…" },
    "forest": { "background":"#252b25", "…":"…" },
    "…":      "…"
  },
  "palettes": {
    "neutral": { "surfaces":"grey",   "accent":"#ae8e6c" },
    "slate":   { "surfaces":"slate",  "accent":"#7d93ad" },
    "…":       "…",
    "forest":  { "surfaces":"forest", "accent":"#4a6d41", "onHighlight":"#e8e4dd" }
  },
  "containerShape": { "rounded": { "panel":10, "popup":8 } },
  "elementShape":   { "rounded": { "button":8 } },
  "decorationShape":{ "rounded": { "titlebar":10 } },
  "buttonStyles": {
    "windows": { "plateRadius":8, "closePlate":"#e0655f", "width":28, "height":26,
                 "closeHover":"0.75", "plainHover":"0.18", "rest":"0.85", "…":"…" }
  },
  "opacity": { "panel":0.85, "popup":0.85, "tooltip":0, "button":0.85 }
}
```

## Layout

Artwork becomes a build product. The SVGs are already mechanically regular — `button.svg` is nine
tiles across four states, `tasks.svg` is five edges by six states by nine tiles at computed offsets
— and nobody should be hand-editing a 210x150 coordinate grid, let alone one per palette.

```
spec/
  tokens.json               the hand-edited variant input
internal/gen/
  main.go                   which files exist at a point in the variant space
  frame.go                  panel and popup frames: nine tiles, cubic corners
  control.go                buttons, inputs and list items: stacked states, arc corners
  aurorae.go                window decoration, titlebar buttons, the two ini files
  colors.go                 KColorScheme ini from a palette and an accent
assets/                     generated; committed
  variants/
    colors/app/<palette>-<sidebar>/ VanillaBoxDark.colors     10 files
    colors/shell/<palette>-<tint>/  the style's colors        10 files
    decoration/<palette>-<shadow>/  decoration.svg            10 files
    defaults/<palette>/             look-and-feel defaults      5 files
    containers/opaque/              the four frames, opaque     4 files
    buttons/<style>-<shadow>/       titlebar set + rc         20 files
```

The emitters are Go rather than text templates. Regeneration has to be byte-for-byte against
artwork that was originally written by hand, and the existing files are not uniformly formatted —
the panel puts its nine tiles on one line where the dialog frames use one per line. Reproducing
that from a template means encoding whitespace in the template, which is worse to read than the
code that decides it.

Artwork no axis touches — `line.svg`, `plasmoidheading.svg`, `tabbar.svg`, `tasks.svg`,
`scrollwidget.svg` — stays hand-maintained, because generating a file that never varies costs a
builder and buys nothing.

The identity files are the exception to that rule, and are generated despite never varying: KDE
wants the same handful of facts in three formats — two `metadata.json` and a `metadata.desktop` —
and the installer wants the version in a fourth. Four hand-maintained copies of one version number
is four chances to disagree, so all four are written from `spec/tokens.json`. `internal/theme/
version.go` is generated Go rather than something read at runtime, so `vanillabox --version` still
answers when the asset directory cannot be found at all.

`assets/theme.json` keeps its own copy, because the manifest stays hand-written. A test asserts the
two agree.

`scrollbar.svg` is the deliberate exception. It is an Inkscape document: 987 lines and 32KB of
editor metadata to express 22 rectangles, unlike every other file in the theme. Its whole
contribution to the theme's corners is one `rx` on the slider, so it stays hand-maintained and takes
no part in any variant. Rewriting it as clean SVG is worthwhile cleanup, but it is not variant work.

### Widgets that exist only to paint nothing

A theme that does not ship a widget falls back to the default theme's copy, so an omission is not
neutral — it inherits Breeze. `widgets/scrollwidget.svg` is here for that reason alone: Breeze's
version paints `border-top`, `border-left`, `border-right` and `border-bottom` in `currentColor` at
full opacity, which is a 1px box drawn around every scroll area in a Plasma popup. Nothing outside
that file can switch it off, so the only way not to have it is to ship a `scrollwidget` whose tiles
are all `fill:none`.

Its 1px edges are kept rather than collapsed to zero, so the insets Plasma reads match what Breeze
reported and nothing reflows. The scrollbar itself is untouched.

Thirty-one other widgets still fall back — `frame`, `listitem`, `slider`, `switch` and the rest —
and each is a place Breeze can show through. They are only worth shipping when one of them is
actually seen to be wrong.

### Three corner idioms

The artwork does not round corners one way, and the differences are in the path data rather than
in appearance, so they cannot be normalised without changing the committed bytes:

| Where | Construction |
| --- | --- |
| Popup and dialog frames | cubic, with a straight run between arc and tile edge (tile 14 > radius 12) |
| Tooltip frame | the same idiom at the small radius (tile 14 > radius 8) |
| Panel frame | cubic, closing directly off the arc (tile 12 == radius 12) |
| Buttons, inputs, list items | `A` arc commands, radius 8 |

Both builders are live, and which one a frame gets is read off its geometry rather than its name.
The panel has been through both: at radius 10 its tile exceeded its radius and it took the popup's
idiom, and at 12 it meets it exactly and closes off the arc again. The mask corners branch on the
same test for the same reason — a frame with no straight run has nowhere to put the step that
[the mask inset](#the-mask-corners-are-a-pixel-smaller) otherwise sits on.

The cubic control points sit at `radius * 0.4478`. That constant is pinned by the artwork's own
rounding rather than chosen: it is the only three-decimal value that yields both the `3.582` the
frames use at radius 8 and the `3.135` the window decoration uses at radius 7 for its inset
border. The usual `1 - 0.5523` gives `3.134` and fails to reproduce the decoration.

### The mask corners are a pixel smaller

Every masked frame draws its `mask-` corners one pixel inside its background corners, and its mask
edges and centre exactly on the background's. The asymmetry is deliberate and it is not ours: it is
Breeze's, arrived at the same way.

A `mask-` prefix is a region, not artwork. KSvg renders that prefix as its own frame and thresholds
it to one bit — `QRegion(QBitmap(alphaMask.mask()))` in `ksvg/framesvg.cpp` — so the mask's corner
comes out a hard staircase where the background's corner is antialiased. Drawn at the same radius
the staircase lands *outside* the artwork's smooth edge, and the pixels between the two are blur
region that no surface covers: a fringe tracing a slightly tighter curve than the corner it belongs
to. Breeze ships a note about it inside its own `dialogs/background.svg`:

> The corners of the mask are 1px smaller because they are not antialiased, whereas the svg corners
> are; if the mask is the same size of the svg, then, some of the corner pixels of the mask will be
> visible even though they should be covered by the svg.

It is a pixel because it covers an antialiasing ramp, and a ramp is a pixel wide whatever the
radius is. `maskCornerInset` is therefore a Go constant rather than a token: it is a property of how
the mask is rasterised, and it does not move when the radii do.

Only the corners take it. Plasma tried shrinking the whole mask first and got a dark outline down
every side with the wallpaper showing through the outermost pixel — see
[libplasma!644](https://invent.kde.org/frameworks/plasma-framework/-/merge_requests/644) — so the
straight runs, edges and centre stay flush with the frame. That leaves a one-pixel step at each end
of the arc, on the straight part of the side where the background is at full coverage and the mask
boundary does not show. Breeze reaches the same shape with zero-area spurs back to the original
endpoints, which is the same compromise expressed in Inkscape; the step is the one difference, and
our tiles can afford it because they exceed their radius and so meet their neighbours on the inner
edges, which the inset never touches.

The panel is masked for the same reason the others are. With no `mask-` prefix at all, `alphaMask()`
returns `frame->cachedBackground` — the translucent artwork itself — so the blur region would be
thresholded out of 85%-opaque pixels rather than off a clean shape.

### The tooltip's outline

`widgets/tooltip.svg` is the only container that carries a border. It is also the only one that
appears over arbitrary content rather than over the desktop or a panel it already contrasts with,
so it is the only one that has to define its own edge.

The background is painted first, over the tile's whole shape, and the outline laid over it as a
ring between that shape and the same shape a pixel in, so the two antialiased curves stay
concentric. The corner builders therefore take an inset, and the idiom is chosen from the frame's
own radius rather than the inset one — an inner path is the same corner drawn a pixel in, and
re-deciding on the reduced radius would hand a frame's inner path to a different template than its
outer one.

A corner's ring is both paths in one `d`, under `fill-rule="evenodd"`, so the inner shape is cut
out rather than drawn. They meet flush along the boundaries a tile shares with its neighbours, and
there the two coincident edges cancel and leave the ring open — the same three sides the edge
strips leave bare, because an outline on a shared seam would draw a line across the middle of the
tooltip.

Both the surface and the outline are tuned low, and they were taken down together over several
passes: the tooltip sits at `surfaces.<set>.tooltip`, three steps below where it started, and the
outline at `opacity.tooltipBorder`, half of the tenth it began at. The outline's number is a token
rather than a literal for the same reason the colour is — it is the only control over how far the
edge separates from the surface, and the two only make sense read against each other.

That pairing is what `TestTheTooltipKeepsAVisibleEdge` exists for. Each step down is defensible on
its own and the failure is cumulative: a darker card keeps improving right up to the step where the
edge stops resolving and the tooltip becomes a shape with no boundary. The test composites the
outline over the surface for every palette and asks whether what comes out is far enough from what
is under it. Nothing else would catch it, because every other test here asks whether the outline is
built correctly rather than whether it can be seen.

Three things about it are easy to get wrong:

- **The outline goes over the surface, not under it.** The window decoration stacks these the other
  way round — border first, background inset over it — and this file was first written from that
  template. It does not carry across. The decoration's border is an opaque literal, so what sits
  underneath it never comes up; this one is a fraction of the text colour, and a fraction of
  something needs the surface underneath it to be a fraction *of*. Painted first, the frame's
  outermost pixel is the text colour at `opacity.tooltipBorder` over whatever is behind the tooltip
  — almost entirely transparent, a gap rather than an edge. Over the surface it resolves to an
  opaque `#0e0e0e` against the default palette's `#030303`.

- **It goes through the stylesheet, not a literal colour.** The container artwork is generated once
  per shape, not per palette — `containers/` has a `rounded` and a `square` tree and no tint axis —
  because Plasma resolves `ColorScheme-*` classes at paint time. A literal border colour would bake
  the default palette's edge into a file all five palettes share. The outline is
  `ColorScheme-Text` at `0.1`, so it follows the tint like everything else.
- **The mask copies stay whole.** `mask-*` is the blur region rather than artwork. An outline drawn
  into it would punch a ring out of the blur instead of showing up as a border, so the mask keeps
  the frame's full outer shape.

The border does not change what the transparency toggles cover. `opacity.tooltip` is still zero, so
the root file and its `opaque/` copy remain byte for byte the same and there is still no fourth
toggle to offer.

#### The empty shadow prefix

`widgets/tooltip.svg` also ships a `shadow-` prefix whose nine tiles are all `fill:none`. It exists
for the same reason `widgets/scrollwidget.svg` does — to stop a fallback — and without it the
tooltip draws two borders.

`org.kde.plasma.components.ToolTip` builds its background from two `FrameSvgItem`s over the same
sheet: one with `prefix: "shadow"`, anchored with *negative* margins so it sits outside, and one
plain. KSvg decides a prefix exists by looking for `<prefix>-center`, and clears the prefix when it
finds none — so a sheet with no shadow tiles renders its own frame a second time, inflated by the
frame's 4px margins. Two flat fills stacked that way are invisible; two outlined ones are a pair of
concentric borders 4px apart.

The tiles are bare rects on the painted tiles' own coordinates. An element is fetched by id, so
what it overlaps in the sheet never comes up — the `mask-` copies already work this way. The
`shadow-hint-*-margin` rects repeat the frame's own, so the prefix reports the insets KSvg was
already deriving from the fallback and the tooltip's geometry does not move.

Only the tooltip needs this. `widgets/button.svg` is the other file a `shadow` prefix is asked for
(`ButtonShadow.qml`), and it has no unprefixed tiles at all, so its fallback finds nothing to draw.
Every other container is a flat fill, where a doubled draw has nothing to show. A border is what
makes the fallback visible, and the tooltip is the only container that carries one.

#### The hover halo

`widgets/button.svg`'s `hover` prefix is anchored the same way, and it caught the theme out.
`ButtonHover.qml` fills the control and then pushes all four edges back out by the prefix's own
margins:

```qml
anchors { fill: parent; leftMargin: -margins.left; topMargin: -margins.top; … }
imagePath: "widgets/button"
prefix: "hover"
```

So `hover` is a halo drawn *around* a raised button, not the button's surface. Breeze says so in
its artwork: `hover-center` there carries `opacity:0.001` and only the border tiles are inked. This
theme painted all nine tiles solid, and with no `hint-*-margin` elements anywhere the margins fell
back to the 6px tile size — so hovering a button produced a filled plate 12px wider and 12px taller
than the button under it. On the lock screen's login button, and on any raised button in a popup,
that read as a container appearing out of nowhere rather than as the button reacting.

The answer is `hover-hint-no-border-padding`. KSvg returns zero from a margin query on a prefix
carrying that element, so there is nothing for `ButtonHover` to push out by and the frame lands on
the button exactly. It is the *margins* that go, not the border: `framesvg.cpp` keeps
`fixedLeftWidth` — the tile's own width, which is what the frame is painted with — separately from
`fixedLeftMargin`, which is only ever a content inset. The nine tiles still draw, so the corners
follow the button's.

Dropping the border tiles instead also stops the overhang, and was the first fix here, but it
squares the corners: with no border there is only the centre, and a centre tile is stretched to the
control's size, so a radius baked into it arrives as an ellipse. A square wash over a rounded
`normal` frame reads as the button changing shape under the pointer.

The hint has to carry the prefix. KSvg looks for it bare as well, and an unprefixed
`hint-no-border-padding` would zero the margins of every state in the sheet — including `normal`'s,
which are a raised button's own padding, so every button in the desktop would lose its content
inset. `TestHoverCannotGrowPastTheButton` pins the prefixed form and the absence of the bare one.

Hover paints only the wash, not the surface under it. `RaisedButtonBackground.qml` keeps the
`normal` frame drawn below, so a wash over it is the button's own background changing colour, which
is all the state has to say.

It washes at `0.15` where `toolbutton-hover` washes at `0.08`, and the two are not meant to match.
A raised button already has a surface, and its hover lightens it; a flat button has none, and its
hover *is* the surface, landing straight on the popup or panel behind it. The same value in both
places would either wash out the flat button or leave the raised one looking untouched.

The states anchored normally — `pressed`, `focus-background`, and both `toolbutton-` states — paint
the control's own rect and need no hint. `docs/plasma-controls.md` has the wider map of which
Plasma widget reads which prefix.

#### The button's surface is translucent

`opacity.button` is `0.85`, and it is the only control fill that is not fully opaque. A button on a
popup sits on a surface that is already `0.85`, so letting a little through keeps it reading as
part of that surface rather than as a card laid on it.

The surface it draws is `elevated`, and the two are one decision rather than two. It shipped opaque
at six units above the window and read as no background at all; the translucency then takes about
15% of whatever lift it has back off, since what shows through is the window itself. So the value
is set by what survives that: `#3d3d3d` on `#292929` at `0.85` lands at `#3a3a3a`, eleven up rather
than six. `TestTheButtonSurfaceReadsAsRaised` pins the composited lift rather than either number,
because lowering `opacity.button` alone would walk the button back into the popup without touching
a colour.

`elevated` now passes `elevatedAlt`, which looks like a ladder out of order and is not one. The two
are separate roles: `elevated` is a control fill, `elevatedAlt` is an edge — the window
decoration's border and the tooltip's alternate row. Nothing composes them, and a fill that has to
be seen at 85% has further to go than a hairline drawn at full strength.

Both states that draw the surface take it: `normal` and `pressed`. A pressed button that went
opaque would read as a different surface rather than a pressed one, which is what
`TestButtonSurfaceIsTranslucent` guards.

Unlike the containers, this needs no `opaque/` copy. Plasma falls back to those prefixes only for
backgrounds, and a control has none — so with compositing off the button simply composites against
the opaque popup behind it and nothing shows through that should not.

Generated assets are committed. The README promises `assets/` ships beside the binary, the tests
install the real shipped artwork rather than a fixture, and a contributor should be able to run the
installer without a generate step. `.gitignore` therefore stays minimal and deliberately does not
list `assets/`.

The tooltip is also the one surface with a colour of its own. It used to take the view colour, on
the reasoning that a card over arbitrary content should sit below the chrome around it — which was
right until the darker-panels option arrived and put the popups a tooltip appears over on exactly
that colour. With that option on, a tooltip over a popup painted the popup's own colour and showed
nothing but its outline. `surfaces.<set>.tooltip` is therefore written out per set, a step below
`view`, and hand-tuned like every other surface: a percentage of `view` collapses the five sets into
each other at that lightness and the tint stops being legible.

## Manifest

`Option` becomes typed. A `select` chooses among named values; a `toggle` stays boolean.

```json
"options": [
  { "id":"palette", "name":"Colour", "kind":"select", "defaultValue":"neutral",
    "values":[ { "id":"neutral", "name":"Neutral" },
               { "id":"slate",   "name":"Slate",
                 "overlay":{ "from":"variants/colors/slate" } } ] },

  { "id":"transparency-panel", "name":"Translucent panel", "kind":"toggle", "default":true,
    "overlayWhenOff": { "from":"plasma/desktoptheme/vanilla-box-dark/opaque",
                        "files":["widgets/panel-background.svg"] } }
]
```

An omitted `kind` means a toggle, so an option written before selects existed still loads. A select
value with no overlay is the one that matches the artwork as generated — `neutral` copies nothing.

`group` names the page an option is asked on and `order` places it within that page, low to high,
with the manifest's own sequence breaking ties. A page is gathered by group name rather than as a
run, because a group can span components — Shape is asked partly by the Plasma style and partly by
the decoration — and without `order` the only way to move one preference within it is to reorder
the components, which moves every other page too.

The order values are listed in is presentational. The cursor opens on the value already selected
rather than on the first line, so `defaultValue` and the listing are free to disagree — and for a
long time they did, every corner choice listing Square while installing Rounded. They agree now,
but nothing enforces it. Tying the cursor to the first row instead would make every listing order a
silent second declaration of the default, and one stray enter on an unread page would change the
answer.

An option still never edits a file. It only chooses which pre-generated bytes get copied, which is
the same guarantee the transparency option makes today.

### There is no component checklist

Every component is marked `required`, so they install without being asked about and the checklist
screen is gone. The run opens on the preferences.

Nothing is conditional either, though the mechanism for it remains. A component can carry
`installedWhen`, naming a preference and the value it has to hold:

```json
{ "id": "extras", "installedWhen": { "option": "tint", "value": "rose" } }
```

The shipped manifest uses it nowhere. It is what let the icon theme install only when it had been
asked for, back when the theme shipped icons, and it stays because it is the only way to add a
genuinely optional component without bringing the checklist screen back. Selection is recomputed
whenever an answer changes rather than decided once when the model is built, since the preference
that decides it can be revisited at any point before the review.

### The theme keeps to its own surfaces

The look-and-feel `defaults` names the colour scheme, the Plasma style, the accent and the window
decoration. It does not name a cursor theme, an icon theme or a splash screen, and the theme ships
none of the three.

It used to do all three. The cursor theme was the clearest mistake: `defaults` named a third-party
set this repo never shipped, so on a machine without it KDE silently fell back and the setting was
simply wrong. The splash line was `Theme=None`, which is a preference about someone else's boot
screen. Icons were at least asked about and defaulted to off — but an icon theme replaces every
icon on the desktop, which is the largest thing an installer like this can do to a machine, and
the honest version of that choice is not to offer it.

What is left is the set of things a *look* actually is. A user who picks this global theme gets
new surfaces, not a new pointer, a new icon set and a different boot screen. Those are theirs.

The review screen names a component whose files are not there — `unavailable, will be skipped` —
and that is the only reason it ever gives. A gap in the asset tree is a broken build, and this is
the only place it gets said; now that nothing is conditional, it is also the only way a component
can fail to install. A component the user had turned down was never named there even when one
could exist, because the answer that turned it down is already listed above it.

### One question at a time

Preferences are asked over pages rather than on one screen: surface colour, shape, transparency.
Shape carries the corner axes, the window buttons and the shadow.
Each option declares its `group` in the manifest, and the pages are the distinct groups in the
order they first become visible.

That order is worth knowing: it follows `visibleOptions`, which walks components, not the order
options sit inside one of them. The palette is declared on the Plasma style but reached first
through the colour scheme's resolved path, so Colour leads. A page whose group appears in two
components — Shape, which spans the Plasma style and the decoration — still renders as one page,
because grouping is by name rather than by run.

A page asking a single choice drops that choice's header row, since the page heading already names
it, and moves its description into the subtitle beside the step counter. Otherwise the screen says
"Colour" twice.

### Choices are lists, not controls

A choice renders as a header with one row per value, `(•)` on the chosen one. The alternative — a
single row cycled with arrow keys — hides every option the user has not already found.

The cost is height: seven preferences with their values is more than a 24-line terminal holds, so
the screen scrolls. Blank lines between groups are rows in the model rather than padding inside
another row, which keeps the scroll window counting screen lines and list rows identically.

The window never begins part-way through a group or ends on a group's header. Unlabelled values and
a header naming nothing are the same mistake, and both are worse than showing one group fewer.

`←`/`→` went with the change. Once values have their own rows, up/down and space do everything, and
a binding that duplicates another is noise in a help line whose whole purpose is to be honest.

### Where a preference is offered

A preference is shown when the current selection actually uses it — either because a selected
component declares it, or because a selected component names it in a resolved path. The palette is
declared once, on the Plasma style, and the colour scheme reads it through
`variants/colors/app/{palette}-{sidebar}/…` — which matters more now that the colour scheme is
required and no longer appears on the checklist at all.

The alternative was repeating every value list on every component that consumes it, which is the
same data in three places and three places for it to drift.

### Resolved files

Two files depend on a combination of axes rather than a single one. Rather than layer overlays and
depend on ordering, they are generated per combination and selected by substituting option ids into
a path:

`source` is relative to the asset directory; `target` is relative to the component's installed
directory, and an empty target means the component's own path — which is what a component that
installs a single file needs.

```json
"resolved": [
  { "source":"variants/colors/{palette}/colors",       "target":"colors" },
  { "source":"variants/defaults/{accent}/defaults",    "target":"contents/defaults" },
  { "source":"variants/decoration/{palette}-{shadows}/decoration.svg",
    "target":"decoration.svg" }
]
```

Resolved files are written on every install, the default combination included. There is no special
case for "this is the one already in the tree", so the path that runs for `neutral`/`sand` is the
same path that runs for everything else.

The `defaults` file is the one place several axes meet, because it is a single KDE-defined file
that happens to carry an accent, a colour scheme name, a Plasma theme name and a button order.
Generating it per combination is preferred over teaching the installer to merge ini fragments,
which would break the guarantee that an option only ever chooses bytes.

## What gets installed

One colour scheme, one Plasma style, one Aurorae theme and one look-and-feel package.
Variants never multiply what lands in `~/.local/share`; they only decide which bytes are copied.
Backups continue to work as described in the README.

## Settled

- **Whether Aurorae substitutes colours at runtime no longer matters.** It was open for a while:
  the five Aurorae SVGs have no `current-color-scheme` block, so if Aurorae did substitute, the
  theme was not taking advantage of it. Nothing now depends on the answer — `decoration.svg` is
  generated per palette with its colours already baked in, and the buttons are painted in
  foregrounds that are held constant across every palette.
- **The maximised layout keys work as described**, verified on a real desktop after a restart.

## Open questions

- **Adding an accent is two edits, not one:** the colour in `spec/tokens.json`, and the value in
  `assets/theme.json` so the installer offers it — now carrying the colour a second time, as the
  swatch the preferences screen draws. The manifest stays hand-written by decision — the README
  promises that adding a component is an edit to it rather than to the code, and generating it
  would buy consistency at the cost of that promise. `TestSwatchesMatchTheAccents` closes the gap
  the copy opens, failing with both values and the instruction to edit them together. If the two
  drift often enough to matter anyway, generating the value lists is the fix.
- **`scrollbar.svg` keeps its rounded slider in the square variant.** Its `rx` is 2px on a 6px
  slider, which is close to invisible, and patching it would mean the generator reading and
  rewriting the one Inkscape document in the tree. Worth doing alongside the rewrite, not before.
