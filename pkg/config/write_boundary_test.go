// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// symlinkOrSkip creates a symbolic link, skipping the test on platforms or
// accounts that can't create one (e.g. Windows without the privilege).
func symlinkOrSkip(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("cannot create symlink: %v", err)
	}
}

// TestFileSave_SymlinkedConfig verifies that saving a config file reached
// through a symbolic link updates the link's target, keeps the target's mode,
// and leaves the link in place.
func TestFileSave_SymlinkedConfig(t *testing.T) {
	t.Parallel()

	targetDir, linkDir := t.TempDir(), t.TempDir()
	target := filepath.Join(targetDir, "config.yaml")
	if err := os.WriteFile(target, []byte("default-cluster: old\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(linkDir, "config.yaml")
	symlinkOrSkip(t, target, link)

	f, err := OpenFile(link)
	if err != nil {
		t.Fatalf("OpenFile(): %v", err)
	}
	if err := f.SetKey("default-cluster", "new"); err != nil {
		t.Fatalf("SetKey(): %v", err)
	}

	linfo, err := os.Lstat(link)
	if err != nil {
		t.Fatal(err)
	}
	if linfo.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("%s mode = %v after save, want it to remain a symlink", link, linfo.Mode())
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "default-cluster: new") {
		t.Errorf("link target contents = %q, want default-cluster: new", got)
	}
	tinfo, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if tinfo.Mode().Perm() != 0o600 {
		t.Errorf("link target mode = %v, want 0600 preserved", tinfo.Mode().Perm())
	}
}

// TestFileSave_SymlinkedParentDirectory verifies that saving a config file
// opened through a symbolically linked directory, whose relative link target
// climbs out with "..", updates the file that was read rather than the one a
// lexical reading of the path names.
func TestFileSave_SymlinkedParentDirectory(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "real", "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	read := filepath.Join(root, "real", "config.yaml")
	if err := os.WriteFile(read, []byte("default-cluster: old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	symlinkOrSkip(t, "../config.yaml", filepath.Join(root, "real", "sub", "config.yaml"))
	symlinkOrSkip(t, filepath.Join("real", "sub"), filepath.Join(root, "alias"))

	f, err := OpenFile(filepath.Join(root, "alias", "config.yaml"))
	if err != nil {
		t.Fatalf("OpenFile(): %v", err)
	}
	if err := f.SetKey("default-cluster", "new"); err != nil {
		t.Fatalf("SetKey(): %v", err)
	}

	got, err := os.ReadFile(read)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "default-cluster: new") {
		t.Errorf("%s contents = %q, want default-cluster: new", read, got)
	}
	if _, err := os.Lstat(filepath.Join(root, "config.yaml")); err == nil {
		t.Errorf("SetKey() created %s instead of updating %s", filepath.Join(root, "config.yaml"), read)
	}
}

// realTempDir returns a new temporary directory with any symbolic links in
// its path resolved (e.g. macOS's /var -> /private/var), so it can be compared
// with resolveSymlinks results.
func realTempDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

// TestResolveSymlinks_RelativeChain verifies that resolveSymlinks follows a
// chain of relative links to the final file.
func TestResolveSymlinks_RelativeChain(t *testing.T) {
	t.Parallel()

	dir := realTempDir(t)
	target := filepath.Join(dir, "real.yaml")
	if err := os.WriteFile(target, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	symlinkOrSkip(t, "real.yaml", filepath.Join(dir, "middle.yaml"))
	symlinkOrSkip(t, "middle.yaml", filepath.Join(dir, "config.yaml"))

	got, err := resolveSymlinks(filepath.Join(dir, "config.yaml"))
	if err != nil {
		t.Fatalf("resolveSymlinks(): %v", err)
	}
	if got != target {
		t.Errorf("resolveSymlinks() = %q, want %q", got, target)
	}
}

// TestResolveSymlinks_TargetThroughSymlinkedDirectory verifies that
// resolveSymlinks follows a symbolically linked directory inside a relative
// link target before applying the ".." that follows it.
func TestResolveSymlinks_TargetThroughSymlinkedDirectory(t *testing.T) {
	t.Parallel()

	dir := realTempDir(t)
	if err := os.MkdirAll(filepath.Join(dir, "a", "b"), 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, "a", "real.yaml")
	if err := os.WriteFile(target, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	symlinkOrSkip(t, filepath.Join("a", "b"), filepath.Join(dir, "linkdir"))
	symlinkOrSkip(t, "linkdir/../real.yaml", filepath.Join(dir, "config.yaml"))

	got, err := resolveSymlinks(filepath.Join(dir, "config.yaml"))
	if err != nil {
		t.Fatalf("resolveSymlinks(): %v", err)
	}
	if got != target {
		t.Errorf("resolveSymlinks() = %q, want %q", got, target)
	}
}

// TestResolveSymlinks_DanglingLink verifies that resolveSymlinks returns the
// target of a link whose target doesn't exist yet.
func TestResolveSymlinks_DanglingLink(t *testing.T) {
	t.Parallel()

	dir := realTempDir(t)
	target := filepath.Join(dir, "missing.yaml")
	symlinkOrSkip(t, target, filepath.Join(dir, "config.yaml"))

	got, err := resolveSymlinks(filepath.Join(dir, "config.yaml"))
	if err != nil {
		t.Fatalf("resolveSymlinks(): %v", err)
	}
	if got != target {
		t.Errorf("resolveSymlinks() = %q, want %q", got, target)
	}
}

// TestResolveSymlinks_Cycle verifies that resolveSymlinks rejects a cycle of
// symbolic links.
func TestResolveSymlinks_Cycle(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	a, b := filepath.Join(dir, "a.yaml"), filepath.Join(dir, "b.yaml")
	symlinkOrSkip(t, b, a)
	symlinkOrSkip(t, a, b)

	if _, err := resolveSymlinks(a); err == nil {
		t.Fatal("resolveSymlinks() on a link cycle = nil error, want error")
	}
}
