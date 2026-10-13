// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package config_test

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/openchami/ochami/pkg/config"
)

func writeFileFixture(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "cfg.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

// TestOpenFile verifies that OpenFile requires an existing, readable file and
// exposes its raw (defaults-free) contents.
func TestOpenFile(t *testing.T) {
	t.Run("empty path returns error", func(t *testing.T) {
		if _, err := config.OpenFile(""); err == nil {
			t.Fatal("OpenFile(): expected error for empty path, got nil")
		}
	})

	t.Run("nonexistent file returns error", func(t *testing.T) {
		if _, err := config.OpenFile("/no/such/file.yaml"); err == nil {
			t.Fatal("OpenFile(): expected error for missing file, got nil")
		}
	})

	t.Run("reads existing file", func(t *testing.T) {
		path := writeFileFixture(t, "default-cluster: old\n")
		f, err := config.OpenFile(path)
		if err != nil {
			t.Fatalf("OpenFile(): unexpected error: %v", err)
		}
		if got := f.Get("default-cluster"); got != "old" {
			t.Errorf("Get(default-cluster) = %v, want old", got)
		}
	})
}

// TestCreateFile verifies that CreateFile creates an empty file immediately
// (mirroring os.Create) and that the resulting File is immediately usable.
func TestCreateFile(t *testing.T) {
	t.Run("empty path returns error", func(t *testing.T) {
		if _, err := config.CreateFile(""); err == nil {
			t.Fatal("CreateFile(): expected error for empty path, got nil")
		}
	})

	t.Run("creates a new empty file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "new.yaml")
		f, err := config.CreateFile(path)
		if err != nil {
			t.Fatalf("CreateFile(): unexpected error: %v", err)
		}
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("CreateFile() did not create the file: %v", err)
		}
		if f.Path() != path {
			t.Errorf("Path() = %q, want %q", f.Path(), path)
		}
		if err := f.SetKey("default-cluster", "foo"); err != nil {
			t.Fatalf("SetKey() on created file: %v", err)
		}
	})
}

// TestFile_SetKey verifies that SetKey persists a global key (top-level or
// nested) to disk without disturbing sibling keys, without leaving temporary
// files behind, and that CreateFile rejects a destination that is a directory
// or whose parent directory doesn't exist.
func TestFile_SetKey(t *testing.T) {
	t.Run("modify default-cluster updates config", func(t *testing.T) {
		path := writeFileFixture(t, "default-cluster: old\n")
		f, err := config.OpenFile(path)
		if err != nil {
			t.Fatalf("OpenFile(): %v", err)
		}
		if err := f.SetKey("default-cluster", "new"); err != nil {
			t.Fatalf("SetKey(): unexpected error: %v", err)
		}

		got, err := config.OpenFile(path)
		if err != nil {
			t.Fatalf("read back failed: %v", err)
		}
		if v := got.Get("default-cluster"); v != "new" {
			t.Errorf("default-cluster = %v, want new", v)
		}
	})

	t.Run("modify nested log.level updates config", func(t *testing.T) {
		path := writeFileFixture(t, "log:\n  format: pretty\n  level: info\n")
		f, err := config.OpenFile(path)
		if err != nil {
			t.Fatalf("OpenFile(): %v", err)
		}
		if err := f.SetKey("log.level", "debug"); err != nil {
			t.Fatalf("SetKey(): unexpected error: %v", err)
		}

		got, err := config.OpenFile(path)
		if err != nil {
			t.Fatalf("read back failed: %v", err)
		}
		if v := got.Get("log.level"); v != "debug" {
			t.Errorf("log.level = %v, want debug", v)
		}
		if v := got.Get("log.format"); v != "pretty" {
			t.Errorf("log.format = %v, want unchanged pretty", v)
		}

		matches, err := filepath.Glob(filepath.Join(filepath.Dir(path), ".cfg.yaml.*"))
		if err != nil {
			t.Fatal(err)
		}
		if len(matches) != 0 {
			t.Errorf("temporary files left behind: %v", matches)
		}
	})

	t.Run("destination is a directory", func(t *testing.T) {
		if _, err := config.CreateFile(t.TempDir()); err == nil {
			t.Fatal("CreateFile(): expected directory error, got nil")
		}
	})

	t.Run("nonexistent parent directory", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "missing", "config.yaml")
		if _, err := config.CreateFile(path); err == nil {
			t.Fatal("CreateFile(): expected missing-parent error, got nil")
		}
	})
}

// TestFile_UnsetKey verifies that UnsetKey removes a global key from disk and
// that unsetting a key that was never set is an error rather than a silent
// no-op.
func TestFile_UnsetKey(t *testing.T) {
	t.Run("delete top-level key", func(t *testing.T) {
		path := writeFileFixture(t, "default-cluster: orig\nclusters: []\n")
		f, err := config.OpenFile(path)
		if err != nil {
			t.Fatalf("OpenFile(): %v", err)
		}
		if err := f.UnsetKey("default-cluster"); err != nil {
			t.Fatalf("UnsetKey(): unexpected error: %v", err)
		}

		got, err := config.OpenFile(path)
		if err != nil {
			t.Fatalf("read back failed: %v", err)
		}
		if got.Get("default-cluster") != nil {
			t.Errorf("default-cluster = %v, want unset", got.Get("default-cluster"))
		}
	})

	t.Run("delete non-existent key returns error", func(t *testing.T) {
		path := writeFileFixture(t, "default-cluster: x\n")
		f, err := config.OpenFile(path)
		if err != nil {
			t.Fatalf("OpenFile(): %v", err)
		}
		err = f.UnsetKey("does.not.exist")
		if err == nil {
			t.Fatal("UnsetKey(): expected error deleting missing key, got nil")
		}
	})
}

// TestFile_Clusters verifies that Clusters returns every cluster in file
// order (not applying defaults), that Cluster looks a single one up by name,
// and that Cluster reports ErrUnknownCluster for a name that isn't present.
func TestFile_Clusters(t *testing.T) {
	path := writeFileFixture(t, `clusters:
  - name: zeta
    cluster:
      uri: https://zeta.example.com
  - name: alpha
    cluster:
      uri: https://alpha.example.com
`)
	f, err := config.OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile(): %v", err)
	}

	clusters, err := f.Clusters()
	if err != nil {
		t.Fatalf("Clusters(): unexpected error: %v", err)
	}
	want := []string{"zeta", "alpha"}
	var got []string
	for _, c := range clusters {
		got = append(got, c.Name)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("cluster order = %v, want %v", got, want)
	}

	cl, err := f.Cluster("zeta")
	if err != nil {
		t.Fatalf("Cluster(zeta): unexpected error: %v", err)
	}
	if cl.Cluster.URI != "https://zeta.example.com" {
		t.Errorf("Cluster(zeta).Cluster.URI = %q, want https://zeta.example.com", cl.Cluster.URI)
	}

	if _, err := f.Cluster("missing"); !errors.As(err, &config.ErrUnknownCluster{}) {
		t.Errorf("Cluster(missing) error = %v, want ErrUnknownCluster", err)
	}
}

// TestFile_AddCluster verifies that AddCluster writes a new cluster as a
// whole record and that it refuses to overwrite an existing cluster,
// returning ErrClusterExists instead (callers who want to grow an existing
// cluster incrementally should use SetClusterKey).
func TestFile_AddCluster(t *testing.T) {
	t.Run("adds a new cluster", func(t *testing.T) {
		path := writeFileFixture(t, "clusters: []\n")
		f, err := config.OpenFile(path)
		if err != nil {
			t.Fatalf("OpenFile(): %v", err)
		}
		if err := f.AddCluster("foo", config.ClusterConfig{URI: "https://foo.example.com"}); err != nil {
			t.Fatalf("AddCluster(): unexpected error: %v", err)
		}

		got, err := config.OpenFile(path)
		if err != nil {
			t.Fatalf("read back failed: %v", err)
		}
		cl, err := got.Cluster("foo")
		if err != nil {
			t.Fatalf("Cluster(foo): %v", err)
		}
		if cl.Cluster.URI != "https://foo.example.com" {
			t.Errorf("URI = %q, want https://foo.example.com", cl.Cluster.URI)
		}
	})

	t.Run("duplicate name returns ErrClusterExists", func(t *testing.T) {
		path := writeFileFixture(t, "clusters:\n  - name: foo\n")
		f, err := config.OpenFile(path)
		if err != nil {
			t.Fatalf("OpenFile(): %v", err)
		}
		err = f.AddCluster("foo", config.ClusterConfig{})
		if !errors.As(err, &config.ErrClusterExists{}) {
			t.Errorf("AddCluster(): error = %v, want ErrClusterExists", err)
		}
	})
}

// TestFile_SetClusterKey verifies that SetClusterKey creates the named
// cluster implicitly if it doesn't already exist (matching "ochami config
// cluster set"'s UX) and that it rejects key "name" in favor of the dedicated
// RenameCluster method.
func TestFile_SetClusterKey(t *testing.T) {
	t.Run("creates cluster implicitly", func(t *testing.T) {
		path := writeFileFixture(t, "clusters: null\n")
		f, err := config.OpenFile(path)
		if err != nil {
			t.Fatalf("OpenFile(): %v", err)
		}
		if err := f.SetClusterKey("c1", "cluster.uri", "https://c1.example.com"); err != nil {
			t.Fatalf("SetClusterKey(): unexpected error: %v", err)
		}

		got, err := config.OpenFile(path)
		if err != nil {
			t.Fatalf("read back failed: %v", err)
		}
		cl, err := got.Cluster("c1")
		if err != nil {
			t.Fatalf("Cluster(c1): %v", err)
		}
		if cl.Cluster.URI != "https://c1.example.com" {
			t.Errorf("URI = %q, want https://c1.example.com", cl.Cluster.URI)
		}
	})

	t.Run("key cannot be name", func(t *testing.T) {
		path := writeFileFixture(t, "clusters:\n  - name: c1\n")
		f, err := config.OpenFile(path)
		if err != nil {
			t.Fatalf("OpenFile(): %v", err)
		}
		if err := f.SetClusterKey("c1", "name", "c2"); err == nil {
			t.Fatal("SetClusterKey(): expected error for key \"name\", got nil")
		}
	})
}

// TestFile_UnsetClusterKey verifies that UnsetClusterKey clears only the
// requested key within a cluster (leaving sibling keys intact), rejects key
// "name", and errors when the cluster doesn't exist.
func TestFile_UnsetClusterKey(t *testing.T) {
	t.Run("clears only the requested key", func(t *testing.T) {
		path := writeFileFixture(t, `clusters:
  - name: c1
    cluster:
      uri: u1
      bss:
        uri: b1
`)
		f, err := config.OpenFile(path)
		if err != nil {
			t.Fatalf("OpenFile(): %v", err)
		}
		if err := f.UnsetClusterKey("c1", "cluster.uri"); err != nil {
			t.Fatalf("UnsetClusterKey(): unexpected error: %v", err)
		}

		got, err := config.OpenFile(path)
		if err != nil {
			t.Fatalf("read back failed: %v", err)
		}
		cl, err := got.Cluster("c1")
		if err != nil {
			t.Fatalf("Cluster(c1): %v", err)
		}
		if cl.Cluster.URI != "" {
			t.Errorf("URI = %q, want empty", cl.Cluster.URI)
		}
		if cl.Cluster.BSS.URI != "b1" {
			t.Errorf("BSS.URI = %q, want unchanged b1", cl.Cluster.BSS.URI)
		}
	})

	t.Run("cannot unset name", func(t *testing.T) {
		path := writeFileFixture(t, "clusters:\n  - name: c1\n    cluster:\n      uri: u1\n")
		f, err := config.OpenFile(path)
		if err != nil {
			t.Fatalf("OpenFile(): %v", err)
		}
		if err := f.UnsetClusterKey("c1", "name"); err == nil {
			t.Fatal("UnsetClusterKey(): expected error unsetting name, got nil")
		}
	})

	t.Run("cluster not found returns error", func(t *testing.T) {
		path := writeFileFixture(t, "clusters:\n  - name: a\n")
		f, err := config.OpenFile(path)
		if err != nil {
			t.Fatalf("OpenFile(): %v", err)
		}
		if err := f.UnsetClusterKey("b", "cluster.uri"); err == nil {
			t.Fatal("UnsetClusterKey(): expected not-found error, got nil")
		}
	})
}

// TestFile_RenameCluster verifies that RenameCluster updates default-cluster
// when the renamed cluster was the default, rejects a name collision with
// ErrClusterExists, and rejects renaming a cluster that doesn't exist with
// ErrUnknownCluster rather than silently creating one under the new name.
func TestFile_RenameCluster(t *testing.T) {
	t.Run("renames and updates default-cluster when it was default", func(t *testing.T) {
		path := writeFileFixture(t, "default-cluster: c1\nclusters:\n  - name: c1\n")
		f, err := config.OpenFile(path)
		if err != nil {
			t.Fatalf("OpenFile(): %v", err)
		}
		if err := f.RenameCluster("c1", "c2"); err != nil {
			t.Fatalf("RenameCluster(): unexpected error: %v", err)
		}

		got, err := config.OpenFile(path)
		if err != nil {
			t.Fatalf("read back failed: %v", err)
		}
		if got.Get("default-cluster") != "c2" {
			t.Errorf("default-cluster = %v, want c2", got.Get("default-cluster"))
		}
		if _, err := got.Cluster("c2"); err != nil {
			t.Errorf("Cluster(c2): %v", err)
		}
	})

	t.Run("rename to empty name returns ErrInvalidConfigVal", func(t *testing.T) {
		path := writeFileFixture(t, "clusters:\n  - name: a\n")
		f, err := config.OpenFile(path)
		if err != nil {
			t.Fatalf("OpenFile(): %v", err)
		}
		err = f.RenameCluster("a", "")
		if !errors.As(err, &config.ErrInvalidConfigVal{}) {
			t.Errorf("RenameCluster(): error = %v, want ErrInvalidConfigVal", err)
		}
	})

	t.Run("rename to duplicate name returns ErrClusterExists", func(t *testing.T) {
		path := writeFileFixture(t, "clusters:\n  - name: a\n  - name: b\n")
		f, err := config.OpenFile(path)
		if err != nil {
			t.Fatalf("OpenFile(): %v", err)
		}
		err = f.RenameCluster("a", "b")
		if !errors.As(err, &config.ErrClusterExists{}) {
			t.Errorf("RenameCluster(): error = %v, want ErrClusterExists", err)
		}
	})

	t.Run("rename of nonexistent cluster returns ErrUnknownCluster", func(t *testing.T) {
		path := writeFileFixture(t, "clusters:\n  - name: a\n")
		f, err := config.OpenFile(path)
		if err != nil {
			t.Fatalf("OpenFile(): %v", err)
		}
		err = f.RenameCluster("doesNotExist", "newname")
		if !errors.As(err, &config.ErrUnknownCluster{}) {
			t.Errorf("RenameCluster(): error = %v, want ErrUnknownCluster", err)
		}
		// And it must not have silently created "newname".
		if _, cerr := f.Cluster("newname"); cerr == nil {
			t.Error("RenameCluster() of a nonexistent cluster must not create a new one")
		}
	})
}

// TestFile_SetDefaultCluster verifies that SetDefaultCluster sets
// default-cluster for an existing cluster and returns ErrUnknownCluster for
// one that doesn't exist.
func TestFile_SetDefaultCluster(t *testing.T) {
	t.Run("sets default-cluster", func(t *testing.T) {
		path := writeFileFixture(t, "clusters:\n  - name: c1\n")
		f, err := config.OpenFile(path)
		if err != nil {
			t.Fatalf("OpenFile(): %v", err)
		}
		if err := f.SetDefaultCluster("c1"); err != nil {
			t.Fatalf("SetDefaultCluster(): unexpected error: %v", err)
		}

		got, err := config.OpenFile(path)
		if err != nil {
			t.Fatalf("read back failed: %v", err)
		}
		if got.Get("default-cluster") != "c1" {
			t.Errorf("default-cluster = %v, want c1", got.Get("default-cluster"))
		}
	})

	t.Run("unknown cluster returns ErrUnknownCluster", func(t *testing.T) {
		path := writeFileFixture(t, "clusters: []\n")
		f, err := config.OpenFile(path)
		if err != nil {
			t.Fatalf("OpenFile(): %v", err)
		}
		err = f.SetDefaultCluster("missing")
		if !errors.As(err, &config.ErrUnknownCluster{}) {
			t.Errorf("SetDefaultCluster(): error = %v, want ErrUnknownCluster", err)
		}
	})
}

// TestFile_DeleteCluster verifies that DeleteCluster removes the whole
// cluster entry, clears default-cluster only when it pointed at the deleted
// cluster (leaving an unrelated default-cluster untouched), and errors for a
// cluster that doesn't exist.
func TestFile_DeleteCluster(t *testing.T) {
	t.Run("cluster not found returns error", func(t *testing.T) {
		path := writeFileFixture(t, "clusters:\n  - name: a\n")
		f, err := config.OpenFile(path)
		if err != nil {
			t.Fatalf("OpenFile(): %v", err)
		}
		if err := f.DeleteCluster("b"); err == nil {
			t.Fatal("DeleteCluster(): expected not-found error, got nil")
		}
	})

	t.Run("removes cluster and clears default-cluster if it matched", func(t *testing.T) {
		path := writeFileFixture(t, `default-cluster: a
clusters:
  - name: a
    cluster:
      uri: https://a.example
  - name: b
    cluster:
      uri: https://b.example
`)
		f, err := config.OpenFile(path)
		if err != nil {
			t.Fatalf("OpenFile(): %v", err)
		}
		if err := f.DeleteCluster("a"); err != nil {
			t.Fatalf("DeleteCluster(): unexpected error: %v", err)
		}

		got, err := config.OpenFile(path)
		if err != nil {
			t.Fatalf("read back failed: %v", err)
		}
		clusters, err := got.Clusters()
		if err != nil {
			t.Fatalf("Clusters(): %v", err)
		}
		if len(clusters) != 1 || clusters[0].Name != "b" {
			t.Fatalf("clusters = %+v, want only b remaining", clusters)
		}
		if got.Get("default-cluster") != nil {
			t.Error("default-cluster should have been cleared after deleting the cluster it pointed to")
		}
	})

	t.Run("removes cluster without touching an unrelated default-cluster", func(t *testing.T) {
		path := writeFileFixture(t, `default-cluster: b
clusters:
  - name: a
    cluster:
      uri: https://a.example
  - name: b
    cluster:
      uri: https://b.example
`)
		f, err := config.OpenFile(path)
		if err != nil {
			t.Fatalf("OpenFile(): %v", err)
		}
		if err := f.DeleteCluster("a"); err != nil {
			t.Fatalf("DeleteCluster(): unexpected error: %v", err)
		}

		got, err := config.OpenFile(path)
		if err != nil {
			t.Fatalf("read back failed: %v", err)
		}
		if got.Get("default-cluster") != "b" {
			t.Errorf("default-cluster = %v, want unchanged b", got.Get("default-cluster"))
		}
	})
}
