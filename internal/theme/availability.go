package theme

import "os"

// refreshAvailability marks each component according to whether the files it
// needs are actually present in the asset directory. Unavailable components are
// shown but cannot be selected, so a partial asset tree degrades to a clear
// message instead of a failed install.
func (t *Theme) refreshAvailability() {
	for i := range t.Components {
		t.Components[i].Available = t.sourceExists(t.Components[i])
	}
}

// sourceExists reports whether a component's source is a readable file, or a
// directory with something in it. An empty directory counts as missing: it is
// what a fresh checkout leaves behind, and installing it would be a no-op.
func (t *Theme) sourceExists(c Component) bool {
	path := t.SourcePath(c)

	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	if !info.IsDir() {
		return info.Size() > 0
	}

	entries, err := os.ReadDir(path)

	return err == nil && len(entries) > 0
}

// Available returns the components whose source files are present.
func (t *Theme) Available() []Component {
	var available []Component
	for _, c := range t.Components {
		if c.Available {
			available = append(available, c)
		}
	}

	return available
}
