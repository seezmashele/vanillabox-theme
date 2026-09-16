// Package theme describes the Vanilla Box theme and the components it is made
// of, and knows how to install them onto a KDE Plasma desktop.
package theme

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ManifestName is the file, relative to the asset directory, that describes the
// theme.
const ManifestName = "theme.json"

// Component is one installable part of the theme, such as the color scheme or
// the window decoration. Everything the installer needs in order to apply it lives here,
// so adding a component is a change to theme.json rather than to code.
type Component struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`

	// Source is the file or directory holding this component, relative to the
	// asset directory.
	Source string `json:"source"`

	// Target is the directory this component is copied into, relative to the
	// user's data directory (~/.local/share).
	Target string `json:"target"`

	// Options are the preferences that change what gets written for this
	// component. They hang off the component rather than the theme so the
	// preferences screen can show only what the current selection affects.
	Options []Option `json:"options"`

	// Resolved are files chosen by a combination of options rather than laid
	// down by a single overlay. They hang off the component so they are written
	// only when it is installed.
	Resolved []Resolved `json:"resolved"`

	// Default reports whether the component starts out selected.
	Default bool `json:"default"`

	// Required components are installed always and are not offered as a choice.
	// The colour scheme is one: every palette is a colour scheme, so asking
	// whether to install one while also asking which one is a contradiction.
	Required bool `json:"required"`

	// InstalledWhen ties this component to a preference declared elsewhere,
	// which is how a component becomes optional without a checklist screen: the
	// question is asked once, among the other preferences, and both the files
	// and whatever depends on them follow the one answer.
	InstalledWhen Condition `json:"installedWhen"`

	// Available reports whether Source actually exists in the asset directory.
	// It is filled in by the availability check, not read from theme.json.
	Available bool `json:"-"`
}

// Condition is a preference and the value it has to hold. A zero Condition is
// no condition at all.
type Condition struct {
	Option string `json:"option"`
	Value  string `json:"value"`
}

// Empty reports whether the condition names nothing to check.
func (c Condition) Empty() bool { return c.Option == "" }

// Wanted reports whether a component should be installed, given the choices.
// It is separate from Available, which is about the asset tree rather than the
// user: a component can be wanted and missing, and the review screen has to
// tell those apart.
func (c Component) Wanted(choices Choices) bool {
	if !c.InstalledWhen.Empty() {
		return choices.Values[c.InstalledWhen.Option] == c.InstalledWhen.Value
	}

	return c.Required || c.Default
}

// OptionKind is how an option is chosen: a switch with two states, or a choice
// among named values.
type OptionKind string

const (
	KindToggle OptionKind = "toggle"
	KindSelect OptionKind = "select"
)

// Overlay is artwork copied over an install to change what was written. From is
// a directory relative to the asset directory; Files names paths inside it, and
// an empty Files means the whole directory.
//
// Naming files rather than always copying a directory is what lets several
// options draw on one overlay tree without needing a directory per combination:
// four independent transparency switches would otherwise want sixteen.
type Overlay struct {
	From  string   `json:"from"`
	Files []string `json:"files"`
}

// Empty reports whether the overlay would copy nothing.
func (o Overlay) Empty() bool { return o.From == "" }

// OptionValue is one choice of a select.
type OptionValue struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`

	// Swatch is the hex colours the preferences screen draws beside this value,
	// as adjacent boxes, for choices whose difference is a colour and cannot be
	// read off a name. It is display only: nothing installed is painted from it,
	// and a value with none simply shows no box.
	//
	// More than one is allowed because a palette is more than one colour, and a
	// single flat box of a dark surface tells the eye very little.
	Swatch []string `json:"swatch"`

	// Overlay is laid over the install when this value is chosen. The value that
	// matches the artwork as generated leaves it empty and copies nothing.
	Overlay Overlay `json:"overlay"`
}

// Option is a user preference that changes what is written for a component.
//
// An option never edits a file. It only decides which already-written bytes get
// copied over the install, which keeps the artwork the only place the theme's
// looks are defined.
type Option struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Kind        OptionKind `json:"kind"`

	// Group is the page this option is asked on. Options sharing a group are
	// shown together, and the pages run in the order their groups first appear.
	// It lives in the manifest rather than the UI so that adding an option means
	// saying where it belongs, not editing a screen.
	Group string `json:"group"`

	// Order places this option within its group, low to high, with the manifest's
	// own order breaking ties. A group is gathered across components — Shape is
	// asked partly by the Plasma style and partly by the decoration — so without
	// it the only way to arrange a page is to reorder the components themselves,
	// which moves every other page too.
	Order int `json:"order"`

	// Default is a toggle's starting state.
	Default bool `json:"default"`

	// OverlayWhenOff is laid over the install when a toggle is switched off.
	OverlayWhenOff Overlay `json:"overlayWhenOff"`

	// Values and DefaultValue belong to a select.
	Values       []OptionValue `json:"values"`
	DefaultValue string        `json:"defaultValue"`
}

// Resolved is a file whose content depends on more than one option, and so
// cannot be expressed as overlays without the order they are laid in mattering.
// Source is relative to the asset directory and may contain {option-id}
// placeholders; Target is relative to the component's installed directory.
type Resolved struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

// Choices is what the user picked, keyed by Option.ID.
type Choices struct {
	Toggles map[string]bool
	Values  map[string]string
}

// NewChoices returns an empty set of choices.
func NewChoices() Choices {
	return Choices{Toggles: map[string]bool{}, Values: map[string]string{}}
}

// DefaultChoices is every option in the theme at its declared default.
func (t *Theme) DefaultChoices() Choices {
	ch := NewChoices()

	for _, c := range t.Components {
		for _, o := range c.Options {
			switch o.Kind {
			case KindSelect:
				ch.Values[o.ID] = o.DefaultValue
			default:
				ch.Toggles[o.ID] = o.Default
			}
		}
	}

	return ch
}

// Theme is a whole theme: its identity plus the components it ships.
type Theme struct {
	Name        string      `json:"name"`
	ID          string      `json:"id"`
	Version     string      `json:"version"`
	Author      string      `json:"author"`
	Description string      `json:"description"`
	Components  []Component `json:"components"`

	// AssetDir is the directory the manifest was loaded from.
	AssetDir string `json:"-"`

	// Stamp names this run's backup directory. It is fixed when the theme loads
	// so every component installed in one session backs up to the same place.
	Stamp string `json:"-"`
}

// LoadManifest reads theme.json from assetDir and records, for each component,
// whether its source files are actually present.
func LoadManifest(assetDir string) (*Theme, error) {
	path := filepath.Join(assetDir, ManifestName)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}

	var t Theme
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if err := t.validate(); err != nil {
		return nil, fmt.Errorf("invalid %s: %w", path, err)
	}

	t.AssetDir = assetDir
	t.Stamp = time.Now().Format("20060102-150405")
	t.refreshAvailability()

	return &t, nil
}

func (t *Theme) validate() error {
	if t.Name == "" {
		return fmt.Errorf("theme name is empty")
	}
	if len(t.Components) == 0 {
		return fmt.Errorf("theme declares no components")
	}

	seen := make(map[string]bool, len(t.Components))
	for i, c := range t.Components {
		switch {
		case c.ID == "":
			return fmt.Errorf("component %d has no id", i)
		case seen[c.ID]:
			return fmt.Errorf("duplicate component id %q", c.ID)
		case c.Name == "":
			return fmt.Errorf("component %q has no name", c.ID)
		case c.Source == "":
			return fmt.Errorf("component %q has no source", c.ID)
		case c.Target == "":
			return fmt.Errorf("component %q has no target", c.ID)
		}
		seen[c.ID] = true

		if err := validateOptions(c); err != nil {
			return err
		}
	}

	return nil
}

func validateOptions(c Component) error {
	seen := make(map[string]bool, len(c.Options))

	for i, o := range c.Options {
		switch {
		case o.ID == "":
			return fmt.Errorf("component %q: option %d has no id", c.ID, i)
		case seen[o.ID]:
			return fmt.Errorf("component %q: duplicate option id %q", c.ID, o.ID)
		case o.Name == "":
			return fmt.Errorf("component %q: option %q has no name", c.ID, o.ID)
		}
		seen[o.ID] = true

		if err := validateOption(c, o); err != nil {
			return err
		}
	}

	return nil
}

func validateOption(c Component, o Option) error {
	if o.Kind == KindSelect {
		if len(o.Values) < 2 {
			return fmt.Errorf("component %q: select %q needs at least two values", c.ID, o.ID)
		}

		values := make(map[string]bool, len(o.Values))
		for _, v := range o.Values {
			switch {
			case v.ID == "":
				return fmt.Errorf("component %q: select %q has a value with no id", c.ID, o.ID)
			case values[v.ID]:
				return fmt.Errorf("component %q: select %q has duplicate value %q", c.ID, o.ID, v.ID)
			case v.Name == "":
				return fmt.Errorf("component %q: select %q value %q has no name", c.ID, o.ID, v.ID)
			}
			values[v.ID] = true
		}

		if !values[o.DefaultValue] {
			return fmt.Errorf("component %q: select %q defaults to unknown value %q",
				c.ID, o.ID, o.DefaultValue)
		}

		return nil
	}

	// A toggle that overlays nothing when off is a switch the user can move
	// without changing the install.
	if o.OverlayWhenOff.Empty() {
		return fmt.Errorf("component %q: option %q does nothing when off", c.ID, o.ID)
	}

	return nil
}

// Title is the theme's display name with its version, for the UI header.
func (t *Theme) Title() string {
	if t.Version == "" {
		return t.Name
	}
	return t.Name + " v" + t.Version
}

// SourcePath is where the component's files are read from.
func (t *Theme) SourcePath(c Component) string {
	return filepath.Join(t.AssetDir, c.Source)
}

// TargetPath is where the component's files will be written. Errors resolving
// the home directory are folded into the path itself, since this value is only
// ever displayed or used as a destination that will fail loudly on write.
func (t *Theme) TargetPath(c Component) string {
	return filepath.Join(dataDir(), c.Target, filepath.Base(c.Source))
}

// dataDir is the XDG data directory theme components are installed into.
func dataDir() string {
	if dir := os.Getenv("XDG_DATA_HOME"); dir != "" {
		return dir
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return ".local/share"
	}

	return filepath.Join(home, ".local", "share")
}
