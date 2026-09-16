package theme

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
)

// copyTree copies src onto dst. src may be a single file or a directory, which
// is what lets one component be a .colors file and another a whole package.
//
// Existing files at dst are overwritten and existing ones alongside them are
// left alone, so the same function serves both the initial copy and laying an
// option's overlay over it.
//
// Symlinks are skipped. Nothing in the theme uses them, and following one would
// mean writing wherever it pointed.
func copyTree(src, dst string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}

	if !info.IsDir() {
		return copyFile(src, dst, info.Mode().Perm())
	}

	return filepath.WalkDir(src, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)

		info, err := entry.Info()
		if err != nil {
			return err
		}

		switch {
		case entry.IsDir():
			// The extra owner bits keep us able to write into a directory whose
			// stored mode is read-only.
			return os.MkdirAll(target, info.Mode().Perm()|0o700)

		case !info.Mode().IsRegular():
			return nil

		default:
			return copyFile(path, target, info.Mode().Perm())
		}
	})
}

func copyFile(src, dst string, mode fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	// Truncating rather than removing keeps the write atomic enough for a theme
	// file, without leaving a hole if the copy fails.
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}

	if _, err := io.Copy(out, in); err != nil {
		out.Close()

		return err
	}
	if err := out.Close(); err != nil {
		return err
	}

	// O_CREATE only sets the mode when the file did not already exist.
	return os.Chmod(dst, mode)
}

// move relocates src to dst, falling back to a copy when the two are on
// different filesystems — which they will be if the user's data directory is a
// separate mount from wherever the backup lands.
func move(src, dst string) error {
	err := os.Rename(src, dst)
	if err == nil || !errors.Is(err, syscall.EXDEV) {
		return err
	}

	if err := copyTree(src, dst); err != nil {
		return err
	}

	return os.RemoveAll(src)
}
