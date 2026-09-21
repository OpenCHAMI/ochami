// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package config_test

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/openchami/ochami/pkg/config"
)

// writeSource writes YAML content to a temp file and returns a Source for it.
func writeSource(t *testing.T, name, content string) config.Source {
	t.Helper()
	path := filepath.Join(t.TempDir(), name+".yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write source %s: %v", name, err)
	}
	return config.Source{Name: name, Path: path}
}

// TestLoad_MergePrecedenceAndOrder verifies that later sources win on scalar
// conflicts, cluster configs merge by name, per-cluster defaults are applied,
// and clusters are returned in first-seen order.
func TestLoad_MergePrecedenceAndOrder(t *testing.T) {
	system := writeSource(t, "system", `timeout: 45s
clusters:
  - name: zeta
    cluster:
      uri: https://zeta.example.com
      enable-auth: "false"
  - name: alpha
    cluster:
      uri: https://alpha.example.com
`)
	user := writeSource(t, "user", `timeout: 1m
clusters:
  - name: beta
    cluster:
      uri: https://beta.example.com
  - name: zeta
    cluster:
      bss:
        uri: /custom-bss
`)

	cfg, err := config.Load([]config.Source{system, user})
	if err != nil {
		t.Fatalf("Load(): unexpected error: %v", err)
	}
	if cfg.Timeout != time.Minute {
		t.Errorf("timeout = %s, want 1m", cfg.Timeout)
	}

	wantOrder := []string{"zeta", "alpha", "beta"}
	gotOrder := make([]string, 0, len(cfg.Clusters))
	for _, cluster := range cfg.Clusters {
		gotOrder = append(gotOrder, cluster.Name)
	}
	if !reflect.DeepEqual(gotOrder, wantOrder) {
		t.Errorf("cluster order = %v, want %v", gotOrder, wantOrder)
	}

	zeta, err := cfg.GetCluster("zeta")
	if err != nil {
		t.Fatalf("GetCluster(zeta): unexpected error: %v", err)
	}
	if zeta.Cluster.URI != "https://zeta.example.com" {
		t.Errorf("zeta URI = %q, want system URI", zeta.Cluster.URI)
	}
	if zeta.Cluster.BSS.URI != "/custom-bss" {
		t.Errorf("zeta BSS URI = %q, want user override", zeta.Cluster.BSS.URI)
	}
	if zeta.Cluster.EnableAuth {
		t.Error("zeta enable-auth = true, want coerced false")
	}

	alpha, err := cfg.GetCluster("alpha")
	if err != nil {
		t.Fatalf("GetCluster(alpha): unexpected error: %v", err)
	}
	if !alpha.Cluster.EnableAuth {
		t.Error("alpha enable-auth = false, want true default")
	}
}

// TestLoad_MissingOptionalSources verifies that optional sources that do not
// exist are skipped and defaults still apply.
func TestLoad_MissingOptionalSources(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.yaml")
	cfg, err := config.Load([]config.Source{
		{Name: "system", Path: missing, Optional: true},
		{Name: "user", Path: missing + ".user", Optional: true},
	})
	if err != nil {
		t.Fatalf("Load(): unexpected error: %v", err)
	}
	if cfg.Timeout != config.DefaultTimeout() {
		t.Errorf("timeout = %s, want %s", cfg.Timeout, config.DefaultTimeout())
	}
	if len(cfg.Clusters) != 0 {
		t.Errorf("clusters = %v, want none", cfg.Clusters)
	}
}

// TestLoadDefaults verifies the defaults-only loader returns the built-in
// defaults with no clusters.
func TestLoadDefaults(t *testing.T) {
	cfg, err := config.LoadDefaults()
	if err != nil {
		t.Fatalf("LoadDefaults(): unexpected error: %v", err)
	}
	if cfg.Timeout != config.DefaultTimeout() {
		t.Errorf("timeout = %s, want %s", cfg.Timeout, config.DefaultTimeout())
	}
	if len(cfg.Clusters) != 0 {
		t.Errorf("clusters = %v, want none", cfg.Clusters)
	}
	if cfg.DefaultInputFormat == "" || cfg.DefaultOutputFormat == "" {
		t.Error("default input/output formats should be populated from defaults")
	}
}

// TestLoad_IsIndependent verifies that separate Load calls do not share or leak
// state between one another.
func TestLoad_IsIndependent(t *testing.T) {
	a := writeSource(t, "a", "default-cluster: aaa\nclusters:\n  - name: aaa\n    cluster:\n      uri: https://a\n")
	b := writeSource(t, "b", "default-cluster: bbb\nclusters:\n  - name: bbb\n    cluster:\n      uri: https://b\n")

	cfgA, err := config.Load([]config.Source{a})
	if err != nil {
		t.Fatalf("Load(a): %v", err)
	}
	cfgB, err := config.Load([]config.Source{b})
	if err != nil {
		t.Fatalf("Load(b): %v", err)
	}

	if cfgA.DefaultCluster != "aaa" {
		t.Errorf("cfgA default-cluster = %q, want aaa", cfgA.DefaultCluster)
	}
	if cfgB.DefaultCluster != "bbb" {
		t.Errorf("cfgB default-cluster = %q, want bbb", cfgB.DefaultCluster)
	}
	if len(cfgA.Clusters) != 1 || cfgA.Clusters[0].Name != "aaa" {
		t.Errorf("cfgA clusters = %v, want single aaa", cfgA.Clusters)
	}
	if len(cfgB.Clusters) != 1 || cfgB.Clusters[0].Name != "bbb" {
		t.Errorf("cfgB clusters = %v, want single bbb", cfgB.Clusters)
	}
}

// TestLoadFile verifies the single-file loader applies defaults and requires the
// file to exist.
func TestLoadFile(t *testing.T) {
	t.Run("applies defaults", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.yaml")
		if err := os.WriteFile(path, []byte("default-cluster: foo\nclusters:\n  - name: foo\n    cluster:\n      uri: https://foo\n"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
		cfg, err := config.LoadFile(path)
		if err != nil {
			t.Fatalf("LoadFile(): unexpected error: %v", err)
		}
		if cfg.Timeout != config.DefaultTimeout() {
			t.Errorf("timeout = %s, want default %s", cfg.Timeout, config.DefaultTimeout())
		}
		foo, err := cfg.GetCluster("foo")
		if err != nil {
			t.Fatalf("GetCluster(foo): %v", err)
		}
		if !foo.Cluster.EnableAuth {
			t.Error("foo enable-auth = false, want true default")
		}
	})

	t.Run("missing file errors", func(t *testing.T) {
		_, err := config.LoadFile(filepath.Join(t.TempDir(), "nope.yaml"))
		if err == nil {
			t.Fatal("LoadFile(): expected error for missing file, got nil")
		}
	})
}

// recordingLogger implements config.Logger, recording every message it
// receives.
type recordingLogger struct {
	messages []string
}

func (r *recordingLogger) Logf(format string, args ...any) {
	r.messages = append(r.messages, fmt.Sprintf(format, args...))
}

// TestLoadKoanf_TracesOnlyWithLogger verifies that a WithLogger option
// receives the source and final key/value trace, and that omitting it incurs
// no logging calls at all (LoadKoanf must not panic or otherwise require a
// logger).
func TestLoadKoanf_TracesOnlyWithLogger(t *testing.T) {
	source := writeSource(t, "user", "timeout: 45s\n")

	t.Run("with logger", func(t *testing.T) {
		var rl recordingLogger
		_, _, err := config.LoadKoanf([]config.Source{
			{Name: "default", Map: config.DefaultGlobalMap()},
			source,
		}, config.WithLogger(&rl))
		if err != nil {
			t.Fatalf("LoadKoanf(): unexpected error: %v", err)
		}
		if len(rl.messages) == 0 {
			t.Fatal("LoadKoanf() with WithLogger produced no trace messages")
		}
		joined := strings.Join(rl.messages, "\n")
		if !strings.Contains(joined, "final config:") {
			t.Errorf("trace messages missing final config dump: %v", rl.messages)
		}
	})

	t.Run("without logger", func(t *testing.T) {
		if _, _, err := config.LoadKoanf([]config.Source{
			{Name: "default", Map: config.DefaultGlobalMap()},
			source,
		}); err != nil {
			t.Fatalf("LoadKoanf() without a logger: unexpected error: %v", err)
		}
	})
}
