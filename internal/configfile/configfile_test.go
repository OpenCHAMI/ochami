// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package configfile

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/openchami/ochami/pkg/config"
)

func mustWriteFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("failed to write test config file %s: %v", path, err)
	}
}

// TestGetConfig verifies that GetConfig returns top-level and nested values,
// nil for an unknown key, and the whole configuration for an empty key, and
// rejects a key under clusters.
func TestGetConfig(t *testing.T) {
	// sample config for testing
	path := filepath.Join(t.TempDir(), "cfg.yaml")
	mustWriteFile(t, path, []byte(`default-cluster: def
log:
  format: json
  level: warn
clusters:
  - name: c1
  - name: c2
`))
	eff, _, err := config.LoadFileEffective(path)
	if err != nil {
		t.Fatalf("failed to load sample config: %v", err)
	}

	t.Run("get default-cluster", func(t *testing.T) {
		v, err := GetConfig(eff, "default-cluster")
		if err != nil {
			t.Fatalf("GetConfig(): unexpected error: %v", err)
		}
		s, ok := v.(string)
		if !ok {
			t.Fatalf("expected string, got %T", v)
		}
		if s != "def" {
			t.Errorf("got %q, want %q", s, "def")
		}
	})

	t.Run("get nested log.level", func(t *testing.T) {
		v, err := GetConfig(eff, "log.level")
		if err != nil {
			t.Fatalf("GetConfig(): unexpected error: %v", err)
		}
		s, ok := v.(string)
		if !ok {
			t.Fatalf("expected string, got %T", v)
		}
		if s != "warn" {
			t.Errorf("got %q, want %q", s, "warn")
		}
	})

	t.Run("get unknown key returns nil", func(t *testing.T) {
		v, err := GetConfig(eff, "does.not.exist")
		if err != nil {
			t.Fatalf("GetConfig(): unexpected error: %v", err)
		}
		if v != nil {
			t.Errorf("got %v, want nil", v)
		}
	})

	t.Run("empty key returns whole config", func(t *testing.T) {
		v, err := GetConfig(eff, "")
		if err != nil {
			t.Fatalf("GetConfig(): unexpected error: %v", err)
		}
		mv := reflect.ValueOf(v)
		found := false
		if mv.Kind() == reflect.Map {
			for _, key := range mv.MapKeys() {
				if key.String() == "default-cluster" {
					found = true
					break
				}
			}
		}
		if !found {
			t.Errorf("returned whole config does not appear to contain default-cluster")
		}
	})

	t.Run("key with clusters prefix returns error", func(t *testing.T) {
		_, err := GetConfig(eff, "clusters.smd")
		if err == nil {
			t.Fatalf("GetConfig(): expected clusters-prefix error, got %v", err)
		}
	})
}

// TestGetConfigString verifies that GetConfigString returns an empty string for
// an unknown key and marshals scalar, map, and slice values and the whole
// configuration as YAML, keeping ambiguous strings as strings.
func TestGetConfigString(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cfg.yaml")
	mustWriteFile(t, path, []byte(`default-cluster: dc
log:
  format: json
  level: info
clusters:
  - name: c1
`))
	eff, _, err := config.LoadFileEffective(path)
	if err != nil {
		t.Fatalf("failed to load sample config: %v", err)
	}

	t.Run("nil value returns empty string", func(t *testing.T) {
		s, err := GetConfigString(eff, "does.not.exist")
		if err != nil {
			t.Fatalf("GetConfigString(): unexpected error: %v", err)
		}
		if s != "" {
			t.Errorf("got %q, want empty string", s)
		}
	})

	t.Run("string value marshals to YAML", func(t *testing.T) {
		s, err := GetConfigString(eff, "default-cluster")
		if err != nil {
			t.Fatalf("GetConfigString(): unexpected error: %v", err)
		}
		var got string
		if err := yaml.Unmarshal([]byte(s), &got); err != nil {
			t.Fatalf("failed to parse YAML output %q: %v", s, err)
		}
		if got != "dc" {
			t.Errorf("round-trip value = %q, want %q", got, "dc")
		}
	})

	t.Run("ambiguous strings remain strings in YAML", func(t *testing.T) {
		values := []string{"null", "true", "123", "key: value", "line one\nline two"}
		for _, value := range values {
			t.Run(value, func(t *testing.T) {
				valPath := filepath.Join(t.TempDir(), "cfg.yaml")
				data, err := yaml.Marshal(map[string]string{"default-cluster": value})
				if err != nil {
					t.Fatalf("failed to marshal test value: %v", err)
				}
				mustWriteFile(t, valPath, data)
				valEff, _, err := config.LoadFileEffective(valPath)
				if err != nil {
					t.Fatalf("failed to load test config: %v", err)
				}

				out, err := GetConfigString(valEff, "default-cluster")
				if err != nil {
					t.Fatalf("GetConfigString(): unexpected error: %v", err)
				}
				var got string
				if err := yaml.Unmarshal([]byte(out), &got); err != nil {
					t.Fatalf("failed to parse YAML output %q: %v", out, err)
				}
				if got != value {
					t.Errorf("round-trip value = %q, want %q", got, value)
				}
			})
		}
	})

	t.Run("map value marshals to YAML", func(t *testing.T) {
		s, err := GetConfigString(eff, "log")
		if err != nil {
			t.Fatalf("GetConfigString(): unexpected error: %v", err)
		}
		if !strings.Contains(s, "format: json") || !strings.Contains(s, "level: info") {
			t.Errorf("YAML output missing expected log fields: %s", s)
		}
	})

	t.Run("slice value marshals to YAML", func(t *testing.T) {
		out, err := GetConfigString(eff, "clusters")
		if err != nil {
			t.Fatalf("GetConfigString(): unexpected error: %v", err)
		}
		if !strings.Contains(out, "name: c1") {
			t.Errorf("YAML output missing cluster: %s", out)
		}
	})

	t.Run("whole config marshals to YAML", func(t *testing.T) {
		out, err := GetConfigString(eff, "")
		if err != nil {
			t.Fatalf("GetConfigString(): unexpected error: %v", err)
		}
		if !strings.Contains(out, "default-cluster: dc") {
			t.Errorf("yaml output missing default-cluster: %s", out)
		}
	})
}

// TestGetConfigCluster verifies that GetConfigCluster returns a cluster's name
// and its top-level and nested keys, nil for an unknown key, and the whole
// cluster as a map for an empty key.
func TestGetConfigCluster(t *testing.T) {
	cluster := config.Cluster{
		Name: "c1",
		Cluster: config.ClusterConfig{
			URI:       "http://example.com",
			BSS:       config.ClusterBSS{URI: "/bss"},
			CloudInit: config.ClusterCloudInit{URI: "/ci"},
			PCS:       config.ClusterPCS{URI: "/pcs"},
			SMD:       config.ClusterSMD{URI: "/smd"},
		},
	}

	t.Run("get Name field", func(t *testing.T) {
		v, err := GetConfigCluster(cluster, "name")
		if err != nil {
			t.Fatalf("GetConfigCluster(): unexpected error: %v", err)
		}
		s, ok := v.(string)
		if !ok {
			t.Fatalf("expected string, got %T", v)
		}
		if s != "c1" {
			t.Errorf("got %q, want %q", s, "c1")
		}
	})

	t.Run("get cluster.uri", func(t *testing.T) {
		v, err := GetConfigCluster(cluster, "cluster.uri")
		if err != nil {
			t.Fatalf("GetConfigCluster(): unexpected error: %v", err)
		}
		s, ok := v.(string)
		if !ok {
			t.Fatalf("expected string, got %T", v)
		}
		if s != "http://example.com" {
			t.Errorf("got %q, want %q", s, "http://example.com")
		}
	})

	t.Run("get nested cluster.bss.uri", func(t *testing.T) {
		v, err := GetConfigCluster(cluster, "cluster.bss.uri")
		if err != nil {
			t.Fatalf("GetConfigCluster(): unexpected error: %v", err)
		}
		s, ok := v.(string)
		if !ok {
			t.Fatalf("expected string, got %T", v)
		}
		if s != "/bss" {
			t.Errorf("got %q, want %q", s, "/bss")
		}
	})

	t.Run("unknown key returns nil", func(t *testing.T) {
		v, err := GetConfigCluster(cluster, "does.not.exist")
		if err != nil {
			t.Fatalf("GetConfigCluster(): unexpected error: %v", err)
		}
		if v != nil {
			t.Errorf("got %v, want nil", v)
		}
	})

	t.Run("empty key returns full config as map", func(t *testing.T) {
		v, err := GetConfigCluster(cluster, "")
		if err != nil {
			t.Fatalf("GetConfigCluster(): unexpected error: %v", err)
		}
		m, ok := v.(map[string]interface{})
		if !ok {
			t.Fatalf("expected map[string]interface{}, got %T", v)
		}
		// check top-level fields
		if name, _ := m["name"].(string); name != "c1" {
			t.Errorf("map[\"name\"] = %q, want %q", name, "c1")
		}
		// check nested cluster map
		nested, ok := m["cluster"].(map[string]interface{})
		if !ok {
			t.Fatalf("expected nested map for \"cluster\", got %T", m["cluster"])
		}
		if uri, _ := nested["uri"].(string); uri != "http://example.com" {
			t.Errorf("nested[\"uri\"] = %q, want %q", uri, "http://example.com")
		}
		// BSS URI
		bssMap, ok := nested["bss"].(map[string]interface{})
		if !ok {
			t.Fatalf("expected nested map for \"bss\", got %T", nested["bss"])
		}
		if bssURI, _ := bssMap["uri"].(string); bssURI != "/bss" {
			t.Errorf("nested[\"bss\"].\"uri\" = %q, want %q", bssURI, "/bss")
		}
	})
}

// TestGetConfigClusterString verifies that GetConfigClusterString returns an
// empty string for an unknown key and marshals values and the whole cluster as
// YAML, keeping ambiguous strings as strings.
func TestGetConfigClusterString(t *testing.T) {
	cluster := config.Cluster{
		Name: "c1",
		Cluster: config.ClusterConfig{
			URI:       "http://example.com",
			BSS:       config.ClusterBSS{URI: "/bss"},
			CloudInit: config.ClusterCloudInit{URI: "/ci"},
		},
	}

	t.Run("nil value returns empty", func(t *testing.T) {
		s, err := GetConfigClusterString(cluster, "does.not.exist")
		if err != nil {
			t.Fatalf("GetConfigClusterString(): unexpected error: %v", err)
		}
		if s != "" {
			t.Errorf("got %q, want empty string", s)
		}
	})

	t.Run("string value marshals to YAML", func(t *testing.T) {
		s, err := GetConfigClusterString(cluster, "name")
		if err != nil {
			t.Fatalf("GetConfigClusterString(): unexpected error: %v", err)
		}
		var got string
		if err := yaml.Unmarshal([]byte(s), &got); err != nil {
			t.Fatalf("failed to parse YAML output %q: %v", s, err)
		}
		if got != "c1" {
			t.Errorf("round-trip value = %q, want %q", got, "c1")
		}
	})

	t.Run("ambiguous strings remain strings in YAML", func(t *testing.T) {
		values := []string{"null", "true", "123", "key: value", "line one\nline two"}
		for _, value := range values {
			t.Run(value, func(t *testing.T) {
				cluster := cluster
				cluster.Name = value
				out, err := GetConfigClusterString(cluster, "name")
				if err != nil {
					t.Fatalf("GetConfigClusterString(): unexpected error: %v", err)
				}
				var got string
				if err := yaml.Unmarshal([]byte(out), &got); err != nil {
					t.Fatalf("failed to parse YAML output %q: %v", out, err)
				}
				if got != value {
					t.Errorf("round-trip value = %q, want %q", got, value)
				}
			})
		}
	})

	t.Run("whole cluster YAML output", func(t *testing.T) {
		out, err := GetConfigClusterString(cluster, "")
		if err != nil {
			t.Fatalf("GetConfigClusterString(): unexpected error: %v", err)
		}
		if !strings.Contains(out, "name: c1") || !strings.Contains(out, "uri: http://example.com") {
			t.Errorf("yaml output missing expected fields: %s", out)
		}
	})
}

// TestReadConfigWithDefaults_Success verifies that ReadConfigWithDefaults
// rejects an empty path, applies the global and per-cluster defaults, and keeps
// the clusters in file order.
func TestReadConfigWithDefaults_Success(t *testing.T) {
	t.Run("empty path", func(t *testing.T) {
		if _, err := ReadConfigWithDefaults(""); err == nil {
			t.Fatal("ReadConfigWithDefaults(): expected error for empty path, got nil")
		}
	})

	t.Run("applies global and cluster defaults", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "config.yaml")
		content := `default-cluster: foo
clusters:
  - name: foo
    cluster:
      uri: https://foo.example.com
`
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("setup failed: %v", err)
		}

		eff, err := ReadConfigWithDefaults(path)
		if err != nil {
			t.Fatalf("ReadConfigWithDefaults(): unexpected error: %v", err)
		}

		// Global default should be present even though not in the file.
		if got, _ := eff.Get("timeout").(string); got != config.DefaultGlobalMap()["timeout"] {
			t.Errorf("timeout = %q, want %q", got, config.DefaultGlobalMap()["timeout"])
		}

		// Cluster default (enable-auth: true) should be applied.
		var clusters []config.Cluster
		if err := eff.Unmarshal("clusters", &clusters); err != nil {
			t.Fatalf("unmarshal clusters: %v", err)
		}
		if len(clusters) != 1 {
			t.Fatalf("got %d clusters, want 1", len(clusters))
		}
		if !clusters[0].Cluster.EnableAuth {
			t.Errorf("clusters[0].Cluster.EnableAuth = false, want true")
		}
		if got := clusters[0].Cluster.URI; got != "https://foo.example.com" {
			t.Errorf("clusters[0].Cluster.URI = %q, want https://foo.example.com", got)
		}
	})

	t.Run("preserves cluster order", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "config.yaml")
		content := `clusters:
  - name: zeta
    cluster:
      uri: https://zeta.example.com
  - name: alpha
    cluster:
      uri: https://alpha.example.com
  - name: mu
    cluster:
      uri: https://mu.example.com
`
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("setup failed: %v", err)
		}

		want := []string{"zeta", "alpha", "mu"}
		// Run multiple times to guard against map-order nondeterminism.
		for i := 0; i < 5; i++ {
			eff, err := ReadConfigWithDefaults(path)
			if err != nil {
				t.Fatalf("ReadConfigWithDefaults(): unexpected error: %v", err)
			}
			var clusters []config.Cluster
			if err := eff.Unmarshal("clusters", &clusters); err != nil {
				t.Fatalf("unmarshal clusters: %v", err)
			}
			var got []string
			for _, c := range clusters {
				got = append(got, c.Name)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("iteration %d: cluster order = %v, want %v", i, got, want)
			}
		}
	})
}

// TestReadConfigWithDefaults_AppliesClusterDefaults verifies that
// ReadConfigWithDefaults applies the per-cluster defaults while keeping the
// cluster's own values, and keeps the global defaults.
func TestReadConfigWithDefaults_AppliesClusterDefaults(t *testing.T) {
	cfg := []byte(`clusters:
  - name: bar
    cluster:
      uri: https://bar.example.com
`)
	path := filepath.Join(t.TempDir(), "cfg.yaml")
	if err := os.WriteFile(path, cfg, 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}
	eff, err := ReadConfigWithDefaults(path)
	if err != nil {
		t.Fatalf("ReadConfigWithDefaults error: %v", err)
	}
	var clusters []config.Cluster
	if err := eff.Unmarshal("clusters", &clusters); err != nil {
		t.Fatalf("unmarshal clusters: %v", err)
	}
	if len(clusters) != 1 {
		t.Fatalf("expected 1 cluster, got %d", len(clusters))
	}
	if !clusters[0].Cluster.EnableAuth {
		t.Errorf("enable‑auth default not applied; got false, want true")
	}
	// Ensure the URI is preserved.
	if clusters[0].Cluster.URI != "https://bar.example.com" {
		t.Errorf("uri mismatch: %s", clusters[0].Cluster.URI)
	}
	// Verify that the overall effective config still contains the default log.format.
	if v, _ := eff.Get("log.format").(string); v != config.DefaultGlobalMap()["log.format"] {
		t.Errorf("log.format = %q, want %q", v, config.DefaultGlobalMap()["log.format"])
	}
	// Ensure the slice ordering is deterministic (single element).
	if order := reflect.TypeOf(clusters); order.Kind() != reflect.Slice {
		t.Errorf("clusters not slice: %v", order)
	}
}
