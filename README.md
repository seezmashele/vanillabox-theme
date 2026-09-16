# Vanilla Box

A terminal installer for the Vanilla Box Dark KDE Plasma theme, built with
[Bubble Tea](https://charm.land/bubbletea).

It copies the theme's files into `~/.local/share` so KDE can find them, and nothing else. It does
not apply the theme — once the files are in place, "Vanilla Box Dark" appears under **System
Settings → Colors & Themes** and you pick it there. What the installer *does* decide is how the
files are written: see [Preferences](#preferences).

## Build and run

```sh
go build -o vanillabox .
./vanillabox
```

The result is a single static executable with no runtime dependencies. The theme's files are read
from disk rather than embedded, so `assets/` ships alongside the binary.

```sh
CGO_ENABLED=0 go build -o vanillabox .   # explicitly static
```

## Usage

```
vanillabox [--assets DIR] [--version]
```

The asset directory is found from the first of these that contains a `theme.json`:

1. `--assets DIR`
2. `$VANILLABOX_ASSETS`
3. `./assets`
4. `assets/` next to the executable

If none has one, the UI opens on an error screen listing every path it tried.

To see what an install would do without touching your desktop, send it somewhere else:

```sh
XDG_DATA_HOME=/tmp/vbtest ./vanillabox
```

### Keys

| Key | Does |
| --- | --- |
| `↑`/`k`, `↓`/`j` | Move |
| `space` | Choose the value under the cursor, or flip a switch |
| `enter` | Continue, then install |
| `esc` | Back, from the review to the preferences |
| `r` | Start over, from the summary |
| `q` | Quit |

`ctrl+c` always quits, including mid-install.

## The theme

`assets/theme.json` describes the theme and the components it ships. Adding a component is an edit
to that file, not to the code:

```json
{
  "id": "colors",
  "name": "Color scheme",
  "description": "Window and widget colors",
  "source": "color-schemes/VanillaBoxDark.colors",
  "target": "color-schemes",
  "required": true
}
```

`source` is relative to the asset directory; `target` is relative to `~/.local/share`.

Four are marked `required` and install always. The fifth carries `installedWhen`, which ties it to
a preference declared elsewhere — so it is asked about once, among the other preferences, rather
than through a checklist:

| Component | Installs to | When |
| --- | --- | --- |
| Color scheme | `color-schemes/VanillaBoxDark.colors` | always |
| Plasma style | `plasma/desktoptheme/vanilla-box-dark/` | always |
| Window decoration | `aurorae/themes/VanillaBoxDark/` | always |
| Global theme | `plasma/look-and-feel/org.vanillabox.dark/` | always |

Every component installs on every run. A component whose `source` is missing — or is an empty file or directory — is skipped rather than
failing the run, and the review screen lists it as `unavailable, will be skipped`. To see that,
point the installer at a partial tree:

```sh
./vanillabox --assets /path/to/incomplete/assets
```

## Preferences

The preferences are the only thing to decide, and they are asked over three pages:

| Page | Asks |
| --- | --- |
| Surface colour | the palette |
| Shape | window buttons, shadows |
| Transparency | the four switches |

`enter` moves to the next page and, from the last, to the review; `esc` steps back. Every value of
every choice is listed under it, so the alternatives are visible without operating anything. On a
short terminal a page scrolls, marking how much sits above and below.

A value may carry a `"swatch"` — hex colours the screen draws as small filled boxes beside it. The
palettes use it, because "Plum" only tells you which colour if you already know. Each shows the
pair the page is actually about: the panel background and the elevated surface above it. The
colours are display only, and a test checks them against `spec/tokens.json` so a box cannot show
one colour while the install writes another.

They are genuinely subtle. The three panel colours sit within 5–12 units of each other out of a
possible 441, because a tinted dark surface is a dark surface first — that closeness is the design
rather than a rendering problem, and it is why the boxes are two cells wide and come in pairs.

Which page a preference appears on is `"group"` in `theme.json`, not something the UI decides, so
adding an option means saying where it belongs rather than editing a screen. `"order"` arranges a
page: a group is gathered across components — Shape is asked partly by the Plasma style and partly
by the decoration — so without it the only way to move one preference is to reorder the components
and every other page with them.

| Preference | Values | Changes |
| --- | --- | --- |
| Surfaces | Neutral, Slate, Plum | the tint of every surface — and, quietly, the accent that goes with it |
| Sidebar background | Match the window, Match the file list | places panels in Dolphin, Kate and friends — and every other window background |
| Darker panels & popups | Match the toolbars, Darker | the panel strip, the launcher and applet popups |
| Window buttons | Symbols, Traffic lights | close, minimise and maximise |
| Window shadows | Shadows, No shadows | a soft drop shadow under every window |
| Translucent panel | on/off | `widgets/panel-background.svg` |
| Translucent popups & menus | on/off | `dialogs/background.svg` |
| Translucent tooltips | on/off | `widgets/tooltip.svg` |
| Translucent applets | on/off | `widgets/background.svg` |

A choice's default is `"defaultValue"` in `theme.json` rather than whichever value happens to be
listed first, and the two are free to disagree: the cursor opens on the value already selected, not
on the first line, so a choice can list its values in whatever order reads best. Every choice
happens to list the value it installs first. Accepting every prompt gives Neutral, a sidebar on the
window colour, a panel at the toolbar colour, symbol window buttons, and a soft shadow
under every window. The corners are not among the prompts: the theme rounds throughout, titlebars
panels and popups to 10, and tooltips, buttons and inputs to 8.

**Sidebar background** is blunter than its name suggests, and it is the one choice worth reading
before changing. KDE colour schemes have no sidebar role: a places panel paints with the *window*
background, which is why it matches the toolbar rather than the file list next to it. Making it
match the list means moving the window background itself, so every window background moves —
dialogs, settings pages, message boxes. Toolbars and headers stay put, because they read from a
separate role, so a merged panel and list still sit under a strip that reads as chrome.

The default is **Match the window**: KDE's own arrangement, a lighter sidebar beside the file list,
with every other window background left where it is. Choose **Match the file list** if you want the
sidebar and list merged into one dark field and accept the rest of the window backgrounds moving
with it.

**Darker panels & popups** is the same move made on the desktop instead, and it is a separate
question because the desktop has no sidebar to ask about. The panel strip, the launcher and every
applet popup read their background from the Plasma style's own colour file, so they can be darkened
without touching any application, or left alone while every application goes dark. The default
leaves them at the chrome colour, which is where the toolbars and the titlebar already sit: dark
window bodies under a panel that reads as chrome. Choose **Darker** if you would rather the whole
desktop dropped to the file-list colour — the one thing you give up is the tooltip, which is on
that colour deliberately and stops standing off the popups it appears over.

A colour is a surface tint and an accent chosen together, not two separate questions. Neutral is
the quietest of them — grey surfaces under a grey accent, with no colour anywhere. Slate tints the
surfaces and the titlebar a cool navy-grey but keeps the grey accent, and Plum carries its tint
through the surfaces, the selection colour and the titlebar alike.

There is no component checklist. Choosing a colour is already choosing a colour scheme, and the
rest of the theme is what makes that colour mean anything, so every component installs without
being asked about. The theme deliberately keeps to its own surfaces: it never touches your cursor
theme, icon theme or splash screen. The review screen lists every file destination before a byte
is written, and names anything it is leaving out.

Corners are not a preference. The theme rounds, from one small scale rather than a radius picked
per component: 10 for the surfaces — the titlebar, the panel strip, popups and applet backgrounds —
and 8 for what sits on one: tooltips, buttons and inputs.
Two values, shared, is what makes the corners read as one system instead of a set of separately
tuned components — and a shape switch is the thing most likely to break that, since half the
combinations it offers are ones nobody would choose on purpose. The radii live in
`spec/tokens.json` and everything else is generated from them, so the theme is still easy to
re-round; it is just not asked about at install time.

The titlebar rounds its top two corners only. The bottom corners stay square and cannot be anything
else: rounding them needs a bottom border to draw the curve in, and anything narrower than the
radius lets the client window's square corner show through. Breeze gets around it by calling
`KDecoration3::Decoration::setBorderRadius` so KWin clips the client, and no Aurorae plugin exposes
that API — see the comment in `decoration.svg`. Rounded top, square bottom is what every
SVG-themed decoration looks like, and it is the compromise the theme accepts.

Symbols are the default, and they are what KDE users expect a window to be closed with. Traffic
lights are the alternative: grey circles that take a muted colour on hover and never show a symbol.
Either way the buttons stay on the right, where KDE puts them — macOS would put traffic lights on
the left, and you can move them yourself in System Settings → Window Decorations without
reinstalling. Aurorae gives each button its own artwork, so hovering one lights only that one
rather than all three.

An option never edits a file. It only decides which already-written bytes get copied, so the
artwork stays the only place the theme's looks are defined. A switch names an overlay to lay down
when it is off; a choice feeds a `{placeholder}` in a path under `assets/variants/`.

The launcher and the system tray popups cannot be separated: Plasma renders both from one
`dialogs/background`, so one switch covers them and is named for both. Tooltips have their own
switch. It reaches Plasma's tooltips — the panel, the task manager, widgets — and not the ones
inside applications, which the Breeze application style draws and does not make translucent.

`widgets/tasks.svg` is deliberately outside the transparency switches. Its `0.3`/`0.4` values are
white `normal` and `hover` highlights drawn on top of the panel, not backgrounds — forcing them
opaque would give solid white task buttons.

`opaque/` and `solid/` are installed whichever way the switches go: Plasma falls back to those
prefixes on its own when compositing is off.

A preference is shown when the current selection actually uses it, whether the component declares
it or reads it through a variant path. The step is skipped entirely when nothing selected uses any.

## Generated files

Most of `assets/` is written from `spec/tokens.json`:

```sh
go generate ./...
```

Adding a colour is an edit to the tokens — a surface set if it needs a new one, a palette pairing
it with an accent, plus the matching value in `theme.json` so the installer offers it. The version
and the theme's identity live there too, and are written into the three KDE metadata files and the
installer's own const.

`go test ./internal/gen` fails if the committed assets and the spec have drifted apart, and CI
runs `go generate` and fails on a dirty tree. Anything under `assets/variants/` the generator
no longer produces is deleted rather than left behind. See [DESIGN.md](DESIGN.md)
for what is generated and what is not.

## Applying a decoration change

KWin reads an Aurorae theme once per session. Reinstalling writes new files but the titlebar keeps
using what it already loaded, so a change to the window decoration — button sizes, positions, the
titlebar layout — will not appear until KWin reloads. A restart is known to do it.

If a decoration change seems not to have worked, check
`~/.local/share/aurorae/themes/VanillaBoxDark/VanillaBoxDarkrc` before doubting the change itself.

## Backups

Installing over an existing copy moves it to:

```
~/.local/share/vanillabox/backups/<timestamp>/<target>/<name>
```

Backups deliberately land outside the theme directories. Plasma scans those, so a backup left
beside the real thing would show up in System Settings as a second theme.

## Layout

```
main.go                     flags, asset-dir resolution, program startup
spec/tokens.json            the hand-edited input to the generated artwork
internal/theme/
  manifest.go               Theme, Component, Option and Choices, LoadManifest
  availability.go           whether each component's files are actually present
  install.go                copying into place, backups, overlays, resolved files
  copy.go                   copyTree and move
  version.go                generated from spec/tokens.json
internal/gen/               go generate ./... -> assets/
  frame.go                  panel and popup frames
  control.go                buttons, inputs and list items
  aurorae.go                window decoration and titlebar buttons
  colors.go                 colour schemes
  identity.go               the metadata KDE wants in three formats
internal/ui/
  model.go                  root model, screen state machine, install plumbing
  keys.go                   bindings, enabled per screen so help stays honest
  styles.go                 lipgloss styles, adapting to light or dark terminals
  screen_*.go               the five screens plus the error view
assets/                     theme.json and the theme's files
  variants/                 the alternatives each preference picks from
```

## Tests

```sh
go test ./...
```

`internal/ui` drives the whole flow headlessly — running the commands the model returns and feeding
the messages back in, the way the runtime does. `internal/theme` installs the real shipped assets
into a temporary `$XDG_DATA_HOME`, both with transparency on and off, so the artwork and the
manifest are checked against each other rather than against a fixture.
