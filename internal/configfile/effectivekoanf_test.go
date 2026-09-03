// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package configfile

// effectivekoanf_test.go exercises EffectiveKoanf's layered loading: default
// values, multi-file merge, missing-file skipping, and error paths.

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEffectiveKoanf(t *testing.T) {
	writeCfg := func(t *testing.T, name, contents string) string {
		t.Helper()
		path := filepath.Join(t.TempDir(), name)
		if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		return path
	}

	t.Run("defaults only (no paths)", func(t *testing.T) {
		ko, err := EffectiveKoanf()
		if err != nil {
			t.Fatalf("EffectiveKoanf: %v", err)
		}
		if ko == nil {
			t.Fatal("EffectiveKoanf returned nil koanf")
		}
	})

	t.Run("missing file is skipped", func(t *testing.T) {
		ko, err := EffectiveKoanf(filepath.Join(t.TempDir(), "does_not_exist.yaml"))
		if err != nil {
			t.Fatalf("EffectiveKoanf with missing file: %v", err)
		}
		if ko == nil {
			t.Fatal("EffectiveKoanf returned nil koanf")
		}
	})

	t.Run("single valid file with cluster", func(t *testing.T) {
		path := writeCfg(t, "cfg.yaml", `
default-cluster: foo
clusters:
    - name: foo
      cluster:
        uri: https://foo.example.com
`)
		ko, err := EffectiveKoanf(path)
		if err != nil {
			t.Fatalf("EffectiveKoanf: %v", err)
		}
		if got := ko.String("default-cluster"); got != "foo" {
			t.Errorf("default-cluster = %q, want foo", got)
		}
	})

	t.Run("later file overrides earlier", func(t *testing.T) {
		p1 := writeCfg(t, "a.yaml", "default-cluster: aaa\n")
		p2 := writeCfg(t, "b.yaml", "default-cluster: bbb\n")
		ko, err := EffectiveKoanf(p1, p2)
		if err != nil {
			t.Fatalf("EffectiveKoanf: %v", err)
		}
		if got := ko.String("default-cluster"); got != "bbb" {
			t.Errorf("default-cluster = %q, want bbb (later file wins)", got)
		}
	})

	t.Run("cluster missing name errors", func(t *testing.T) {
		path := writeCfg(t, "cfg.yaml", `
clusters:
    - cluster:
        uri: https://foo.example.com
`)
		if _, err := EffectiveKoanf(path); err == nil {
			t.Fatal("expected error for cluster missing name, got nil")
		}
	})

	t.Run("malformed yaml errors", func(t *testing.T) {
		path := writeCfg(t, "cfg.yaml", "clusters: [unterminated\n")
		if _, err := EffectiveKoanf(path); err == nil {
			t.Fatal("expected error for malformed YAML, got nil")
		}
	})
}
