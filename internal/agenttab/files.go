package agenttab

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

func makePatch(sourceDir string) (string, func(), error) {
	dir, err := os.MkdirTemp("", "agent-tab-")
	if err != nil {
		return "", func() {}, err
	}
	patch := filepath.Join(dir, "tracked.patch")
	out, err := outputIn(sourceDir, "git", "diff", "--binary", "HEAD")
	if err != nil {
		os.RemoveAll(dir)
		return "", func() {}, err
	}
	if err := os.WriteFile(patch, []byte(out), 0o644); err != nil {
		os.RemoveAll(dir)
		return "", func() {}, err
	}
	return patch, func() { os.RemoveAll(dir) }, nil
}

func copyWorktreeContext(sourceDir, targetDir, patchFile string) error {
	if info, err := os.Stat(patchFile); err == nil && info.Size() > 0 {
		if err := commandIn(targetDir, "git", "apply", "--3way", patchFile).Run(); err != nil {
			return err
		}
	}
	if err := copyUntracked(sourceDir, targetDir); err != nil {
		return err
	}
	if err := symlinkContext(sourceDir, targetDir); err != nil {
		return err
	}
	return nil
}

func copyUntracked(sourceDir, targetDir string) error {
	out, err := outputIn(sourceDir, "git", "ls-files", "--others", "--exclude-standard", "-z")
	if err != nil {
		return err
	}
	for _, rel := range strings.Split(out, "\x00") {
		if rel == "" {
			continue
		}
		if err := copyPath(filepath.Join(sourceDir, rel), filepath.Join(targetDir, rel)); err != nil {
			return err
		}
	}
	return nil
}

func copyPath(src, dst string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		link, err := os.Readlink(src)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		_ = os.Remove(dst)
		return os.Symlink(link, dst)
	}
	if info.IsDir() {
		return os.MkdirAll(dst, info.Mode().Perm())
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode().Perm())
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func symlinkContext(sourceDir, targetDir string) error {
	return filepath.WalkDir(sourceDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		name := d.Name()
		if d.IsDir() && name == ".git" {
			return filepath.SkipDir
		}
		rel, err := filepath.Rel(sourceDir, path)
		if err != nil || rel == "." {
			return nil
		}
		if d.IsDir() && name == "node_modules" {
			if err := symlinkIfMissing(path, filepath.Join(targetDir, rel)); err != nil {
				return err
			}
			return filepath.SkipDir
		}
		if !d.IsDir() && strings.HasPrefix(name, ".env") {
			if err := symlinkIfMissing(path, filepath.Join(targetDir, rel)); err != nil {
				return err
			}
		}
		return nil
	})
}

func symlinkIfMissing(src, dst string) error {
	if _, err := os.Lstat(dst); err == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.Symlink(src, dst)
}
