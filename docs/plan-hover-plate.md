# Plan: stop the hover plate growing past the button

Status: **done**, in two passes. The first dropped the hover state's border tiles, which stopped the
overhang but squared its corners. The second — what ships — keeps all nine tiles and adds
`hover-hint-no-border-padding`, so the margins report zero while the tiles still paint: the wash
lands on the button's rect *and* rounds like it. See "Revision" at the end. Background for the
terms used here is in [plasma-controls.md](plasma-controls.md).

## The complaint

Buttons that read as transparent at rest — the lock screen login button, buttons inside system tray
popups — grow a container larger than themselves when hovered. What is wanted is the button's own
background changing colour, and nothing appearing around it. Flat buttons that are transparent at
rest and stay tight on hover (applet header configure, expander arrows) are correct as they are and
must not move.

## Cause

Two things meeting:

1. `ButtonHover.qml` anchors the `hover` frame with negative margins, so Plasma grows it outward
   past the button by that prefix's margins on every side. The prefix is meant to be a halo — Breeze
   paints its `hover-center` at `opacity:0.001`.
2. `controls()` in `internal/gen/main.go` builds `hover` as a solid nine-tile fill
   (`layer{btnBg, "1.0"}, layer{btnHvr, "0.25"}`), and `widgets/button.svg` declares no
   `hint-*-margin` elements anywhere, so its margins fall back to the tile size of 6.

Six-pixel margins on a solid fill means the hover plate is 12px wider and 12px taller than the
button under it.

A second, unrelated reason these buttons read as transparent at rest: `normal` painted
`ColorScheme-ButtonBackground` `#2f2f2f` on a `#292929` popup — six units apart. Not part of the
hover fix, but it is why the plate was the first thing you saw. Addressed in the revision below.

## Scope

Fixed by this change, because they all reach `ButtonHover`: raised `PlasmaComponents3.Button`
(lock screen login, "Configure…" buttons in popups), `ComboBox`, `RoundButton`, `CheckIndicator`.

Untouched, because they anchor normally and already paint the control's own rect: flat toolbuttons
via `toolbutton-hover`, and list, Kickoff and system tray highlights via `widgets/viewitem`.

Note the one place this exceeds the original ask: **dropdowns move too**, because `ComboBox` uses
`ButtonHover`. They have the same oversized plate today, so this fixes them rather than changing
something that was right.

## The change

Emit `hover` as **centre tile only** — no border tiles, no margin hints. With no border elements
FrameSvg reports zero margins, `ButtonHover`'s negative anchors collapse to zero, the frame lands
exactly on the button, and the centre covers precisely its rect.

The centre paints **only the wash**, not the button background again: `surfaceNormal` is already
underneath, so a wash over it is literally the button's background changing colour. Use
`ColorScheme-Text` at `0.08`, the value `toolbutton-hover` already uses, so the two hover
treatments agree.

1. `internal/gen/control.go` — `tileSet` gains a centre-only flag; `geometry()` returns just the
   centre when it is set.
2. `internal/gen/main.go` — `hover` becomes that centre-only set with one wash layer.
3. Tests — pin the invariant with its reason: the `hover` prefix must declare no border tiles,
   because Plasma anchors it with negative margins and any margin it declares becomes overhang.
   Assert that `pressed`, `toolbutton-hover` and `toolbutton-pressed` keep their nine tiles, since
   those are anchored normally and must stay exact.
4. `DESIGN.md` — record the `ButtonHover` quirk under the controls section.

## The trade-off, decided

An exact-bounds hover can only be square-cornered: the centre tile is stretched to the control's
size, so a radius baked into it distorts. `normal` underneath is a 6px rounded rect, so a square
wash over it leaves four slivers of about 1.8px at the corners. At `0.08` they are effectively
invisible, which is the reason for the subtle wash over the current `0.25`.

The alternative — keeping the plate rounded by leaving a small halo — was rejected: a smaller
container is still a container, and the ask was for none.

## Revision: rounded corners, and a translucent surface

The trade-off above was accepted too early. Dropping the border tiles is not the only way to make a
prefix report no margins — `<prefix>-hint-no-border-padding` does it while the tiles stay, and
`framesvg.cpp` stores a tile's own width (`fixedLeftWidth`) separately from the margin it would
otherwise imply (`fixedLeftMargin`), so the frame is still painted with its borders. The hover
state therefore keeps its nine tiles at the element radius and gains the hint. It lands on the
button's rect and rounds like it.

The hint must carry the prefix: KSvg honours a bare `hint-no-border-padding` too, and that one
would zero `normal`'s margins as well — a raised button's own padding.

Separately, `opacity.button` was added to `spec/tokens.json` at `0.85`, applied to the
`ColorScheme-ButtonBackground` layer in both `normal` and `pressed`, so the button lets a little of
what it sits on through. It needs no `opaque/` copy: Plasma falls back to those prefixes only for
backgrounds, and a control has none, so with compositing off it composites against the opaque popup
instead.

Then the colours, once the shape was right. `elevated` was six units above the window in every
palette, which is what made a raised button read as transparent. It is now a third again past
`elevatedAlt` — `#3d3d3d` in neutral, the same move in each tint so the hue relationships hold —
which after the `0.85` composite lands eleven above the window instead of six. A first attempt at
the midpoint (`#343434`) netted three, because the translucency gives most of a small lift straight
back. That reaches applications too, since `elevated` is `[Colors:Button] BackgroundNormal` in both
colour files, which is the point: a button should not be one colour in Dolphin and another in a
popup.

The hover wash went `0.08` → `0.15`. It no longer matches `toolbutton-hover`, deliberately: a
raised button's hover lightens a surface it already has, a flat button's hover *is* its only
surface and lands on the popup behind it.

Tests: `TestHoverCannotGrowPastTheButton` (prefixed hint present, bare hint absent, nine tiles),
`TestHoverCornersFollowTheElementShape` (hover corners arc when the element shape is rounded and
not when it is square), `TestButtonSurfaceIsTranslucent` (both states paint the surface at the
token's value), `TestTheButtonSurfaceReadsAsRaised` (the lift over the window survives the
composite, in every palette). The square-corner note above is kept as the record of why the first
pass was wrong.

## Open afterwards

Whether the system tray complaint is fully answered. Raised buttons inside tray popups are fixed by
this. If what is wrong there is a highlight simply *larger than the icon*, that comes from
`widgets/viewitem` through `PlasmaExtras.Highlight`, which fills its delegate exactly and cannot
overhang — a separate change, to be judged on screen after this lands.
