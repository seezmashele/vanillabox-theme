package theme

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// BackupDir is where a replaced install is kept, relative to the user's data
// directory. It sits outside the theme directories on purpose: Plasma scans
// those, so a backup left beside the real thing would show up in System
// Settings as a second theme.
const BackupDir = "vanillabox/backups"

// Install places one component's files where KDE will find them.
//
// It does not apply anything. Once the files are in place the theme appears in
// System Settings, and it is the user who chooses it there.
//
// choices holds the state of every option, keyed by Option.ID. A toggle that is
// off, and a select value that names one, have their overlay copied over the
// files already written.
func (t *Theme) Install(c Component, choices Choices) error {
	src := t.SourcePath(c)
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("read %s: %w", c.Source, err)
	}

	dst := t.TargetPath(c)

	if err := t.backup(c, dst); err != nil {
		return fmt.Errorf("back up the existing %s: %w", c.Name, err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if err := copyTree(src, dst); err != nil {
		return fmt.Errorf("copy %s: %w", c.Name, err)
	}

	if err := t.overlayOptions(c, dst, choices); err != nil {
		return err
	}

	return t.writeResolved(c, dst, choices)
}

// overlayOptions lays down the overlay each option asks for. A toggle that is
// on, and a select value with no overlay, need nothing done: the files as
// generated are already what they describe.
func (t *Theme) overlayOptions(c Component, dst string, choices Choices) error {
	for _, o := range c.Options {
		overlay := Overlay{}

		switch o.Kind {
		case KindSelect:
			for _, v := range o.Values {
				if v.ID == choices.Values[o.ID] {
					overlay = v.Overlay
				}
			}
		default:
			if !choices.Toggles[o.ID] {
				overlay = o.OverlayWhenOff
			}
		}

		if overlay.Empty() {
			continue
		}
		if err := t.applyOverlay(overlay, dst, choices); err != nil {
			return fmt.Errorf("%s: %w", o.Name, err)
		}
	}

	return nil
}

// applyOverlay copies an overlay's files over an install. Naming no files means
// the whole tree, which is how a variant that replaces everything is written.
//
// From takes the same {option-id} placeholders a resolved path does. That is
// what keeps two overlays from fighting: the opaque copy of a surface has to
// come from the shape currently chosen, or turning transparency off would
// quietly restore the other shape's corners.
func (t *Theme) applyOverlay(o Overlay, dst string, choices Choices) error {
	from, err := choices.expand(o.From)
	if err != nil {
		return err
	}

	root := filepath.Join(t.AssetDir, filepath.FromSlash(from))
	if _, err := os.Stat(root); err != nil {
		return fmt.Errorf("no %q overlay: %w", from, err)
	}

	if len(o.Files) == 0 {
		return copyTree(root, dst)
	}

	for _, name := range o.Files {
		rel := filepath.FromSlash(name)

		src := filepath.Join(root, rel)
		if _, err := os.Stat(src); err != nil {
			return fmt.Errorf("%s has no %s: %w", from, name, err)
		}
		if err := copyTree(src, filepath.Join(dst, rel)); err != nil {
			return err
		}
	}

	return nil
}

// writeResolved copies the files whose content depends on a combination of
// options, substituting the chosen values into the source path.
func (t *Theme) writeResolved(c Component, dst string, choices Choices) error {
	for _, r := range c.Resolved {
		source, err := choices.expand(r.Source)
		if err != nil {
			return fmt.Errorf("%s: %w", c.Name, err)
		}

		src := filepath.Join(t.AssetDir, filepath.FromSlash(source))
		if _, err := os.Stat(src); err != nil {
			return fmt.Errorf("%s: no %s: %w", c.Name, source, err)
		}
		if err := copyTree(src, filepath.Join(dst, filepath.FromSlash(r.Target))); err != nil {
			return fmt.Errorf("%s: %w", c.Name, err)
		}
	}

	return nil
}

// Placeholders lists the option ids a resolved path depends on. It is what lets
// a preference be declared once and still be offered whenever any component
// actually consumes it, rather than only when its declaring component is
// selected.
func Placeholders(path string) []string {
	var ids []string

	for {
		open := strings.IndexByte(path, '{')
		if open < 0 {
			return ids
		}

		end := strings.IndexByte(path[open:], '}')
		if end < 0 {
			return ids
		}
		end += open

		ids = append(ids, path[open+1:end])
		path = path[end+1:]
	}
}

// expand replaces every {option-id} in path with the chosen value. An unknown
// id is an error rather than an empty string: silently resolving to a path that
// does not exist would be reported as a missing file, one step too late to say
// which option was at fault.
func (ch Choices) expand(path string) (string, error) {
	var b strings.Builder

	for {
		open := strings.IndexByte(path, '{')
		if open < 0 {
			b.WriteString(path)

			return b.String(), nil
		}

		end := strings.IndexByte(path[open:], '}')
		if end < 0 {
			return "", fmt.Errorf("unclosed placeholder in %q", path)
		}
		end += open

		id := path[open+1 : end]
		value, ok := ch.Values[id]
		if !ok {
			return "", fmt.Errorf("no option %q to fill %q", id, path)
		}

		b.WriteString(path[:open])
		b.WriteString(value)
		path = path[end+1:]
	}
}

// backup moves an existing install out of the way. Nothing there is not an
// error: most components will not be installed yet.
func (t *Theme) backup(c Component, dst string) error {
	if _, err := os.Lstat(dst); err != nil {
		return nil
	}

	dir := filepath.Join(dataDir(), BackupDir, t.Stamp, c.Target)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	return move(dst, freePath(filepath.Join(dir, filepath.Base(dst))))
}

// freePath returns path, or the first "path-2", "path-3"... that is free.
// Installing twice in one session shares a Stamp, and the second run must not
// land on top of the first run's backup.
func freePath(path string) string {
	if _, err := os.Lstat(path); err != nil {
		return path
	}

	for n := 2; ; n++ {
		candidate := path + "-" + strconv.Itoa(n)
		if _, err := os.Lstat(candidate); err != nil {
			return candidate
		}
	}
}
