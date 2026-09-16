// Command vanillabox is a terminal installer for the Vanilla Box KDE theme.
package main

//go:generate go run ./internal/gen -root .

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"

	"vanillabox/internal/theme"
	"vanillabox/internal/ui"
)

func main() {
	assetDir := flag.String("assets", "", "directory holding the theme assets")
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println("vanillabox", theme.Version)

		return
	}

	// A theme that will not load is not a reason to exit: the UI has a screen
	// for it that can explain where it looked.
	var model tea.Model
	if t, err := loadTheme(*assetDir); err != nil {
		model = ui.NewError(err)
	} else {
		model = ui.New(t)
	}

	if _, err := tea.NewProgram(model).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "vanillabox:", err)
		os.Exit(1)
	}
}

// loadTheme finds the asset directory and reads the manifest out of it.
func loadTheme(override string) (*theme.Theme, error) {
	candidates := assetCandidates(override)

	for _, dir := range candidates {
		if _, err := os.Stat(filepath.Join(dir, theme.ManifestName)); err == nil {
			return theme.LoadManifest(dir)
		}
	}

	return nil, fmt.Errorf(
		"no %s found. Looked in:\n  %s\n\nPass --assets to point at the theme's asset directory.",
		theme.ManifestName,
		strings.Join(candidates, "\n  "),
	)
}

// assetCandidates lists where the theme assets might be, most explicit first:
// the --assets flag, then $VANILLABOX_ASSETS, then ./assets, then an assets
// directory sitting next to the binary.
func assetCandidates(override string) []string {
	if override != "" {
		return []string{override}
	}

	var candidates []string
	if dir := os.Getenv("VANILLABOX_ASSETS"); dir != "" {
		candidates = append(candidates, dir)
	}

	candidates = append(candidates, "assets")

	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "assets"))
	}

	return candidates
}
