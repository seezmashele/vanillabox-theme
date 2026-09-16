# What Plasma paints with what

A map from the thing you see on screen to the artwork that draws it, so a complaint about "the
button in the system tray" can be turned into a prefix in an SVG without guessing.

Everything here was read off an installed desktop rather than recalled: **Plasma 6.7.4**,
`kf6-ksvg 6.28.0`, Fedora 44. Paths are from that machine. Where a claim is inferred rather than
read, it says so.

## Where the QML lives

| What | Where |
| --- | --- |
| Plasma controls (Button, ComboBox, …) | `/usr/lib64/qt6/qml/org/kde/plasma/components/` |
| Their backgrounds | `.../components/private/` — `RaisedButtonBackground.qml`, `FlatButtonBackground.qml`, `ButtonHover.qml`, `ButtonShadow.qml`, `ButtonFocus.qml` |
| List/grid highlight | `/usr/lib64/qt6/qml/org/kde/plasma/extras/Highlight.qml` |
| Lock screen | `/usr/share/plasma/shells/org.kde.plasma.desktop/contents/lockscreen/` |
| Applets | `/usr/lib64/qt6/plugins/plasma/applets/*.so` — QML is compiled in, but `strings` on the `.so` still shows the source |
| Breeze's own artwork | `/usr/share/plasma/desktoptheme/default/widgets/*.svgz` — `zcat` it and read the element ids |

## The controls

| On screen | QML path | Artwork it reads |
| --- | --- | --- |
| Ordinary button — lock screen login, "Configure…" in a popup | `PlasmaComponents3.Button` (not flat) → `RaisedButtonBackground` | `widgets/button`: `normal` always, `hover` on top when hovered, `pressed`, `focus-background` |
| Flat button — applet header configure and pin, tray expander arrow | `PlasmaComponents3.Button` with `flat: true` → `FlatButtonBackground` | `widgets/button`: `toolbutton-hover`, `toolbutton-pressed`, `toolbutton-focus` — nothing at rest |
| Dropdown | `ComboBox` | `widgets/button` `normal`/`pressed`, and **`ButtonHover`** — a dropdown is a raised button for hover purposes |
| Round button, checkbox indicator | `RoundButton`, `CheckIndicator` | also `ButtonHover` |
| Text field | `TextField`, editable `ComboBox` | `widgets/lineedit`: `base`, `hover`, `focus` |
| List row, Kickoff entry, system tray grid item | `PlasmaExtras.Highlight` | `widgets/viewitem`: `normal`, `hover`, `selected`, `selected+hover` |
| Panel strip | the shell | `widgets/panel-background` |
| Launcher, tray and applet popups | `KSvg.FrameSvgItem` in the dialog | `dialogs/background`, `widgets/background` |
| Tooltip | the tooltip dialog | `widgets/tooltip`, resolved against `[Colors:Tooltip]` |
| Task manager buttons | the task manager applet | `widgets/tasks` — white washes over the panel, not a background of its own |

Verified for this list: the lock screen login button is `PlasmaComponents3.Button` with no `flat`
(`lockscreen/MainBlock.qml:112`), so it is raised. The system tray's popup highlight is
`highlight: PlasmaExtras.Highlight {}` (`CurrentItemHighLight.qml`, read out of
`org.kde.plasma.systemtray.so`), and the applet references no `imagePath` of its own, so every
hover it draws comes through `widgets/viewitem`.

## The trap: `hover` is a halo, not a surface

`ButtonHover.qml` anchors the frame with **negative** margins:

```qml
anchors {
    fill: parent
    leftMargin: -margins.left
    topMargin: -margins.top
    rightMargin: -margins.right
    bottomMargin: -margins.bottom
}
imagePath: "widgets/button"
prefix: "hover"
```

So the `hover` frame is deliberately grown outward past the button by whatever margins that prefix
declares, on every side. It is meant to be a glow **around** the control. Breeze says the same
thing in its artwork: in `button.svgz`, `hover-center` carries `opacity:0.001` — it paints nothing
in the middle, and only the border tiles are inked.

A theme that paints `hover` as a solid nine-tile fill therefore gets a plate larger than the button
it belongs to, by twice its margins in each direction. Nothing else reads that prefix's margins —
`RaisedButtonBackground` takes its padding from `normal`, `pressed` and `focus` — so a `hover`
prefix that reports no margins at all is both safe and the only way to make the plate land on the
button exactly.

The way to report none while still painting a border is `<prefix>-hint-no-border-padding`. From
`framesvg.cpp`:

```cpp
frame->noBorderPadding = (q->hasElement(hintNoBorderPadding)
                          || q->hasElement(createName(u"hint-no-border-padding")));
// …and every margin and inset query then answers:
if (d->frame->noBorderPadding) {
    return .0;
}
```

Note the second `hasElement`: KSvg accepts the hint **bare as well as prefixed**, and the bare form
applies to every prefix in the sheet. On `widgets/button` that would zero `normal`'s margins too,
which are a raised button's own padding, so always write the prefixed name.

The border still paints, because the tile's own size and the margin it implies are stored
separately — `fixedLeftWidth` from the `left` element versus `fixedLeftMargin` from
`hint-left-margin`. The frame is drawn with the widths; only the content inset uses the margins.
That separation is what lets a hover state land on the button's rect and still round its corners
the way the button does.

`pressed`, `focus-background`, `toolbutton-hover` and `toolbutton-pressed` are all anchored
normally (`anchors.fill: parent`), so they paint the control's own rect and want their full nine
tiles.

## How margins are decided

KSvg's `FrameSvg` takes a prefix's margins from `<prefix>-hint-{left,right,top,bottom}-margin`
elements when they exist, and otherwise from the size of the border tiles themselves. Breeze
declares them explicitly — `normal` is 6, `toolbutton-hover` is 4. This theme declares them too,
pinned at 6 (`pad` in `internal/gen/main.go`), on every prefix but two: the button's `hover`, whose
halo must report nothing and gets `hover-hint-no-border-padding` instead, and the view item's
`normal`, a placeholder whose tile of 3 is already its margin.

They are declared because the tile is not free to be the margin any more. A corner tile has to be
at least the corner radius to draw it in, so `fullTop` is 8 to hold a radius-8 corner — and without
the hints, that 8 would become every control's padding and make every button bigger.

Margins are a control's padding as well as a frame's border, so raising them makes the control
bigger, not just its artwork thicker.

Insets are a third quantity. KSvg reads `<prefix>-hint-<edge>-inset` into `insetLeftMargin` and
friends, exposed to QML as `FrameSvgItem.inset` — Breeze uses them on the artwork that carries a
shadow region outside the surface proper (`dialogs/background`, `widgets/background`,
`panel-background`, `tooltip`, `scrollbar`), and on no control. They are reported, not applied: a
consumer that wants to sit inside them has to read them. `ButtonHover` reads only `margins`, so
insets are no lever on it.

## Reading Breeze's artwork

The `.svgz` files are gzipped SVG. To see which prefixes and hints a widget really has:

```sh
zcat /usr/share/plasma/desktoptheme/default/widgets/button.svgz > /tmp/breeze-button.svg
grep -o 'id="[^"]*"' /tmp/breeze-button.svg | sort -u
```

Element ids are the whole interface: `<prefix>-{topleft,top,…,center}` are the nine tiles,
`<prefix>-hint-*-margin` sets the margins, `hint-tile-center` tiles the middle instead of
stretching it, and `<prefix>-hint-compose-over-border` changes how the centre composites.
