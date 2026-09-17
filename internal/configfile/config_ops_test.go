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
	"time"

	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/structs"
	"github.com/knadh/koanf/v2"
	"gopkg.in/yaml.v3"

	"github.com/openchami/ochami/pkg/config"
)

func mustWriteFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("failed to write test config file %s: %v", path, err)
	}
}

func TestModifyConfig(t *testing.T) {
	t.Run("empty path returns error", func(t *testing.T) {
		err := ModifyConfig("", "default-cluster", "new")
		if err == nil {
			t.Fatalf("ModifyConfig(): expected read error, got %v", err)
		}
	})

	t.Run("nonexistent file returns error", func(t *testing.T) {
		err := ModifyConfig("/no/such/file.yaml", "default-cluster", "new")
		if err == nil {
			t.Fatalf("ModifyConfig(): expected file read error, got %v", err)
		}
	})

	t.Run("modify default-cluster updates config", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "cfg.yaml")
		mustWriteFile(t, path, []byte("default-cluster: old\n"))

		if err := ModifyConfig(path, "default-cluster", "new"); err != nil {
			t.Fatalf("ModifyConfig(): unexpected error: %v", err)
		}

		ko, err := ReadConfig(path)
		if err != nil {
			t.Fatalf("read back failed: %v", err)
		}
		var got config.Config
		err = ko.Unmarshal("", &got)
		if err != nil {
			t.Errorf("unable to unmarshal config: %v", err)
		}

		if got.DefaultCluster != "new" {
			t.Errorf("DefaultCluster = %q, want %q", got.DefaultCluster, "new")
		}
	})

	t.Run("modify nested log.level updates config", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "cfg.yaml")
		mustWriteFile(t, path, []byte("log:\n  format: pretty\n  level: info"))

		if err := ModifyConfig(path, "log.level", "debug"); err != nil {
			t.Fatalf("ModifyConfig(): unexpected error: %v", err)
		}

		ko, err := ReadConfig(path)
		if err != nil {
			t.Fatalf("read back failed: %v", err)
		}
		var got config.Config
		err = ko.Unmarshal("", &got)
		if err != nil {
			t.Errorf("unable to unmarshal config: %v", err)
		}

		if got.Log.Level != "debug" {
			t.Errorf("Log.Level = %q, want %q", got.Log.Level, "debug")
		}
		if got.Log.Format != "pretty" {
			t.Errorf("Log.Format = %q, want unchanged %q", got.Log.Format, "pretty")
		}
	})

	t.Run("destination is a directory", func(t *testing.T) {
		err := ModifyConfig(t.TempDir(), "default-cluster", "x")
		if err == nil {
			t.Fatal("ModifyConfig(): expected directory read error, got nil")
		}
	})
}

func TestModifyConfigCluster(t *testing.T) {
	t.Run("empty path", func(t *testing.T) {
		err := ModifyConfigCluster("", "c1", "name", false, "c1")
		if err == nil {
			t.Fatalf("ModifyConfigCluster(): expected read error, got %v", err)
		}
	})

	t.Run("rename rejects non-string and empty values", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "cfg.yaml")
		mustWriteFile(t, path, []byte("clusters:\n  - name: c1\n"))
		for _, value := range []any{42, ""} {
			if err := ModifyConfigCluster(path, "c1", "name", false, value); err == nil {
				t.Errorf("ModifyConfigCluster(name=%v) error = nil", value)
			}
		}
	})

	t.Run("rename to duplicate cluster name returns error", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "cfg.yaml")

		mustWriteFile(t, path, []byte(`
default-cluster: ""
clusters:
    - name: a
    - name: b`))

		err := ModifyConfigCluster(path, "a", "name", false, "b")
		if err == nil {
			t.Fatalf("ModifyConfigCluster(): expected duplicate-name error, got %v", err)
		}
	})

	t.Run("add new cluster by name", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "cfg.yaml")

		mustWriteFile(t, path, []byte(`
default-cluster: ""
clusters: null`))

		if err := ModifyConfigCluster(path, "c1", "name", false, "c1"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		ko, err := ReadConfig(path)
		if err != nil {
			t.Fatalf("read back failed: %v", err)
		}
		var got config.Config
		err = ko.Unmarshal("", &got)
		if err != nil {
			t.Errorf("unable to unmarshal config: %v", err)
		}

		if len(got.Clusters) != 1 || got.Clusters[0].Name != "c1" {
			t.Errorf("clusters = %+v, want one cluster with Name=c1", got.Clusters)
		}
		if got.DefaultCluster != "" {
			t.Errorf("default cluster = %q, want empty", got.DefaultCluster)
		}
	})

	t.Run("rename existing cluster updates default when it was default", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "cfg.yaml")

		mustWriteFile(t, path, []byte(`
default-cluster: c1
clusters:
    - name: c1`))

		if err := ModifyConfigCluster(path, "c1", "name", false, "c2"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		ko, err := ReadConfig(path)
		if err != nil {
			t.Fatalf("read back failed: %v", err)
		}
		var got config.Config
		err = ko.Unmarshal("", &got)
		if err != nil {
			t.Errorf("unable to unmarshal config: %v", err)
		}

		if got.Clusters[0].Name != "c2" {
			t.Errorf("cluster name = %q, want %q", got.Clusters[0].Name, "c2")
		}
		if got.DefaultCluster != "c2" {
			t.Errorf("default cluster = %q, want %q", got.DefaultCluster, "c2")
		}
	})

	t.Run("add new cluster and set default when default flag true", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "cfg.yaml")

		mustWriteFile(t, path, []byte(`
default-cluster: ""
clusters: null`))

		if err := ModifyConfigCluster(path, "c3", "name", true, "c3"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		ko, err := ReadConfig(path)
		if err != nil {
			t.Fatalf("read back failed: %v", err)
		}
		var got config.Config
		err = ko.Unmarshal("", &got)
		if err != nil {
			t.Errorf("unable to unmarshal config: %v", err)
		}

		if len(got.Clusters) != 1 || got.Clusters[0].Name != "c3" {
			t.Errorf("clusters = %+v, want one cluster with Name=c3", got.Clusters)
		}
		if got.DefaultCluster != "c3" {
			t.Errorf("default cluster = %q, want %q", got.DefaultCluster, "c3")
		}
	})
}

func TestDeleteConfig(t *testing.T) {
	t.Run("empty path returns error", func(t *testing.T) {
		err := DeleteConfig("", "default-cluster")
		if err == nil {
			t.Fatalf("expected read error, got %v", err)
		}
	})

	t.Run("nonexistent file returns error", func(t *testing.T) {
		err := DeleteConfig("/no/such/file.yaml", "default-cluster")
		if err == nil {
			t.Fatalf("DeleteConfig(): expected read error for missing file, got %v", err)
		}
	})

	t.Run("delete top-level key default-cluster", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "cfg.yaml")

		mustWriteFile(t, path, []byte(`
default-cluster: orig
clusters: []`))

		if err := DeleteConfig(path, "default-cluster"); err != nil {
			t.Fatalf("DeleteConfig(): unexpected error: %v", err)
		}

		ko, err := ReadConfig(path)
		if err != nil {
			t.Fatalf("read back failed: %v", err)
		}
		var got config.Config
		err = ko.Unmarshal("", &got)
		if err != nil {
			t.Fatalf("unmarshal failed: %v", err)
		}
		if got.DefaultCluster != "" {
			t.Errorf("DefaultCluster = %q; want empty", got.DefaultCluster)
		}
	})

	t.Run("delete nested key log.level", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "cfg.yaml")

		mustWriteFile(t, path, []byte(`
log:
    format: pretty
    level: info`))

		if err := DeleteConfig(path, "log.level"); err != nil {
			t.Fatalf("DeleteConfig(): unexpected error: %v", err)
		}

		ko, err := ReadConfig(path)
		if err != nil {
			t.Fatalf("read back failed: %v", err)
		}

		var got config.Config
		err = ko.Unmarshal("", &got)

		if err != nil {
			t.Fatalf("unmarshal failed: %v", err)
		}
		if got.Log.Level != "" {
			t.Errorf("Log.Level = %q; want empty", got.Log.Level)
		}
		if got.Log.Format != "pretty" {
			t.Errorf("Log.Format = %q; want unchanged %q", got.Log.Format, "pretty")
		}
	})

	t.Run("delete non-existent key returns error and leaves config unchanged", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "cfg.yaml")

		initial := config.Config{
			Timeout:        30 * time.Second,
			DefaultCluster: "x",
			Log: config.ConfigLog{
				Format: "f",
				Level:  "l",
			},
		}
		mustWriteFile(t, path, []byte(`
timeout: 30s
default-cluster: x
log:
    format: f
    level: l`))

		err := DeleteConfig(path, "does.not.exist")
		if err == nil {
			t.Fatal("DeleteConfig(): expected error deleting missing key, got nil")
		}
		if !strings.Contains(err.Error(), "key 'does.not.exist' does not exist") {
			t.Errorf("DeleteConfig(): error = %q, want missing-key error", err)
		}

		ko, err := ReadConfig(path)
		if err != nil {
			t.Fatalf("read back failed: %v", err)
		}

		var got config.Config
		err = ko.Unmarshal("", &got)
		if err != nil {
			t.Fatalf("unmarshal failed: %v", err)
		}

		if !reflect.DeepEqual(got, initial) {
			t.Errorf("config = %+v; want unchanged %+v", got, initial)
		}
	})

	t.Run("permission denied writing file", func(t *testing.T) {
		// likely to fail on non-root environments
		err := DeleteConfig("/root/config.yaml", "default-cluster")
		if err == nil {
			t.Fatal("DeleteConfig(): expected permission error, got nil")
		}
	})
}

func TestDeleteConfigCluster(t *testing.T) {
	t.Run("empty path returns error", func(t *testing.T) {
		err := DeleteConfigCluster("", "c1", "cluster.uri")
		if err == nil {
			t.Fatalf("DeleteConfigCluster(): expected read error, got %v", err)
		}
	})

	t.Run("cannot unset name", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "cfg.yaml")
		mustWriteFile(t, path, []byte(`
clusters:
    - name: c1
      cluster:
        uri: u1`))

		err := DeleteConfigCluster(path, "c1", "name")
		if err == nil {
			t.Fatalf("DeleteConfigCluster(): expected cannot unset name error, got %v", err)
		}
	})

	t.Run("cluster not found returns error", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "cfg.yaml")
		mustWriteFile(t, path, []byte(`
clusters:
  - name: a`))

		err := DeleteConfigCluster(path, "b", "cluster.uri")
		if err == nil {
			t.Fatalf("DeleteConfigCluster(): expected not found error, got %v", err)
		}
	})

	t.Run("delete cluster.uri clears only URI", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "cfg.yaml")
		mustWriteFile(t, path, []byte(`
clusters:
  - name: c1
    cluster:
        uri: u1
        bss:
            uri: b1`))

		if err := DeleteConfigCluster(path, "c1", "cluster.uri"); err != nil {
			t.Fatalf("DeleteConfigCluster(): unexpected error: %v", err)
		}
		ko, err := ReadConfig(path)
		if err != nil {
			t.Fatalf("ReadConfig(): %v", err)
		}
		var cl []config.ConfigCluster
		if err := ko.Unmarshal("clusters", &cl); err != nil {
			t.Fatalf("unable to unmarshal clusters: %v", err)
		}
		if cl[0].Cluster.URI != "" {
			t.Errorf("URI = %q; want empty", cl[0].Cluster.URI)
		}
		if cl[0].Cluster.BSS.URI != "b1" {
			t.Errorf("BSS.URI = %q; want unchanged %q", cl[0].Cluster.BSS.URI, "b1")
		}
	})

	t.Run("delete cluster.bss.uri clears only BSS URI", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "cfg.yaml")
		mustWriteFile(t, path, []byte(`
clusters:
  - name: c2
    cluster:
        uri: u2
        bss:
            uri: b2`))

		if err := DeleteConfigCluster(path, "c2", "cluster.bss.uri"); err != nil {
			t.Fatalf("DeleteConfigCluster(): unexpected error: %v", err)
		}

		ko, err := ReadConfig(path)
		if err != nil {
			t.Fatalf("read back failed: %v", err)
		}
		var got config.Config
		err = ko.Unmarshal("", &got)
		if err != nil {
			t.Errorf("unable to unmarshal config: %v", err)
		}

		cl := got.Clusters[0].Cluster
		if cl.BSS.URI != "" {
			t.Errorf("BSS.URI = %q; want empty", cl.BSS.URI)
		}
		if cl.URI != "u2" {
			t.Errorf("URI = %q; want unchanged %q", cl.URI, "u2")
		}
	})

	t.Run("permission denied writing file", func(t *testing.T) {
		// writing to /root should fail under normal test permissions
		err := DeleteConfigCluster("/root/config.yaml", "c1", "cluster.uri")
		if err == nil {
			t.Fatal("DeleteConfigCluster(): expected permission error, got nil")
		}
	})
}

func TestGetConfig(t *testing.T) {
	// sample config for testing
	cfg := koanf.NewWithConf(kConfig)
	if err := cfg.Load(structs.Provider(config.Config{
		DefaultCluster: "def",
		Log: config.ConfigLog{
			Format: "json",
			Level:  "warn",
		},
		Clusters: []config.ConfigCluster{
			{Name: "c1"},
			{Name: "c2"},
		},
	}, "koanf"), nil); err != nil {
		t.Fatalf("failed to load sample config: %v", err)
	}

	t.Run("get default-cluster", func(t *testing.T) {
		v, err := GetConfig(cfg, "default-cluster")
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
		v, err := GetConfig(cfg, "log.level")
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
		v, err := GetConfig(cfg, "does.not.exist")
		if err != nil {
			t.Fatalf("GetConfig(): unexpected error: %v", err)
		}
		if v != nil {
			t.Errorf("got %v, want nil", v)
		}
	})

	t.Run("empty key returns whole config", func(t *testing.T) {
		v, err := GetConfig(cfg, "")
		if err != nil {
			t.Fatalf("GetConfig(): unexpected error: %v", err)
		}
		// Expect a map[string]interface{} or config.Config depending on koanf unmarshal,
		// but at minimum verify that default-cluster value appears in v via reflection.
		mv := reflect.ValueOf(v)
		found := false
		switch mv.Kind() {
		case reflect.Map:
			for _, key := range mv.MapKeys() {
				if key.String() == "default-cluster" {
					found = true
					break
				}
			}
		case reflect.Struct:
			found = true // unmarshaled directly into config.Config
		}
		if !found {
			t.Errorf("returned whole config does not appear to contain default-cluster")
		}
	})

	t.Run("key with clusters prefix returns error", func(t *testing.T) {
		_, err := GetConfig(cfg, "clusters.smd")
		if err == nil {
			t.Fatalf("GetConfig(): expected clusters-prefix error, got %v", err)
		}
	})
}

func TestGetConfigFromFile(t *testing.T) {
	// Prepare a sample config.Config struct and write it to a temp YAML file.
	sample := config.Config{
		DefaultCluster: "dc",
		Log: config.ConfigLog{
			Format: "json",
			Level:  "debug",
		},
		Clusters: []config.ConfigCluster{
			{Name: "c1"},
		},
	}

	data := []byte(`
default-cluster: dc
log:
    format: json
    level: debug
clusters:
  - name: c1`)

	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "cfg.yaml")
	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		t.Fatalf("failed to write sample config to file: %v", err)
	}

	t.Run("empty path returns error", func(t *testing.T) {
		_, err := GetConfigFromFile("", "default-cluster")
		if err == nil {
			t.Fatalf("GetConfigFromFile(): expected read error, got %v", err)
		}
	})

	t.Run("nonexistent file returns error", func(t *testing.T) {
		_, err := GetConfigFromFile(filepath.Join(tmp, "nope.yaml"), "default-cluster")
		if err == nil {
			t.Fatalf("GetConfigFromFile(): expected read error for missing file, got %v", err)
		}
	})

	t.Run("get top-level default-cluster", func(t *testing.T) {
		v, err := GetConfigFromFile(configPath, "default-cluster")
		if err != nil {
			t.Fatalf("GetConfigFromFile(): unexpected error: %v", err)
		}
		s, ok := v.(string)
		if !ok || s != "dc" {
			t.Errorf("got %v (type %T), want %q", v, v, "dc")
		}
	})

	t.Run("get nested log.level", func(t *testing.T) {
		v, err := GetConfigFromFile(configPath, "log.level")
		if err != nil {
			t.Fatalf("GetConfigFromFile(): unexpected error: %v", err)
		}
		s, ok := v.(string)
		if !ok || s != "debug" {
			t.Errorf("got %v (type %T), want %q", v, v, "debug")
		}
	})

	t.Run("unknown key returns nil", func(t *testing.T) {
		v, err := GetConfigFromFile(configPath, "does.not.exist")
		if err != nil {
			t.Fatalf("GetConfigFromFile(): unexpected error: %v", err)
		}
		if v != nil {
			t.Errorf("got %v, want nil for unknown key", v)
		}
	})

	t.Run("empty key returns whole config", func(t *testing.T) {
		v, err := GetConfigFromFile(configPath, "")
		if err != nil {
			t.Fatalf("GetConfigFromFile(): unexpected error: %v", err)
		}
		// Expect either a map[string]interface{} or the config.Config struct.
		rv := reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.Map:
			// Map keys should include "default-cluster"
			if !rv.MapIndex(reflect.ValueOf("default-cluster")).IsValid() {
				t.Errorf("returned map missing default-cluster key")
			}
		case reflect.Struct:
			// Struct case: check field
			got := v.(config.Config)
			if got.DefaultCluster != "dc" {
				t.Errorf("got %+v, want %+v", got, sample)
			}
		default:
			t.Errorf("unexpected type %T for whole config", v)
		}
	})

	t.Run("clusters.* key returns error", func(t *testing.T) {
		_, err := GetConfigFromFile(configPath, "clusters.c1.name")
		if err == nil {
			t.Fatalf("GetConfigFromFile(): expected clusters-prefix error, got %v", err)
		}
	})
}

func TestGetConfigString(t *testing.T) {
	ko := koanf.NewWithConf(kConfig)
	if err := ko.Load(structs.Provider(config.Config{
		DefaultCluster: "dc",
		Log: config.ConfigLog{
			Format: "json",
			Level:  "info",
		},
		Clusters: []config.ConfigCluster{
			{Name: "c1"},
		},
	}, "koanf"), nil); err != nil {
		t.Fatalf("failed to load sample config: %v", err)
	}

	t.Run("nil value returns empty string", func(t *testing.T) {
		s, err := GetConfigString(ko, "does.not.exist")
		if err != nil {
			t.Fatalf("GetConfigString(): unexpected error: %v", err)
		}
		if s != "" {
			t.Errorf("got %q, want empty string", s)
		}
	})

	t.Run("string value marshals to YAML", func(t *testing.T) {
		s, err := GetConfigString(ko, "default-cluster")
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
				if err := ko.Set("default-cluster", value); err != nil {
					t.Fatalf("failed to set test value: %v", err)
				}
				out, err := GetConfigString(ko, "default-cluster")
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
		if err := ko.Set("default-cluster", "dc"); err != nil {
			t.Fatalf("failed to restore test config: %v", err)
		}
	})

	t.Run("map value marshals to YAML", func(t *testing.T) {
		s, err := GetConfigString(ko, "log")
		if err != nil {
			t.Fatalf("GetConfigString(): unexpected error: %v", err)
		}
		if !strings.Contains(s, "format: json") || !strings.Contains(s, "level: info") {
			t.Errorf("YAML output missing expected log fields: %s", s)
		}
	})

	t.Run("slice value marshals to YAML", func(t *testing.T) {
		out, err := GetConfigString(ko, "clusters")
		if err != nil {
			t.Fatalf("GetConfigString(): unexpected error: %v", err)
		}
		if !strings.Contains(out, "name: c1") {
			t.Errorf("YAML output missing cluster: %s", out)
		}
	})

	t.Run("whole config marshals to YAML", func(t *testing.T) {
		out, err := GetConfigString(ko, "")
		if err != nil {
			t.Fatalf("GetConfigString(): unexpected error: %v", err)
		}
		if !strings.Contains(out, "default-cluster: dc") {
			t.Errorf("yaml output missing default-cluster: %s", out)
		}
	})
}

func TestGetConfigStringFromFile(t *testing.T) {
	// Prepare a sample config and write it to a temp file
	data := []byte(`
default-cluster: dc
log:
    format: json
    level: debug
clusters:
  - name: c1`)

	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, "cfg.yaml")
	if err := os.WriteFile(cfgPath, data, 0o644); err != nil {
		t.Fatalf("failed to write sample config: %v", err)
	}

	t.Run("empty path returns error", func(t *testing.T) {
		_, err := GetConfigStringFromFile("", "default-cluster")
		if err == nil {
			t.Fatalf("GetConfigStringFromFile(): expected read error, got %v", err)
		}
	})

	t.Run("nonexistent file returns error", func(t *testing.T) {
		_, err := GetConfigStringFromFile(filepath.Join(tmp, "nope.yaml"), "default-cluster")
		if err == nil {
			t.Fatalf("GetConfigStringFromFile(): expected read error for missing file, got %v", err)
		}
	})

	t.Run("string value marshals to YAML", func(t *testing.T) {
		out, err := GetConfigStringFromFile(cfgPath, "default-cluster")
		if err != nil {
			t.Fatalf("GetConfigStringFromFile(): unexpected error: %v", err)
		}
		var got string
		if err := yaml.Unmarshal([]byte(out), &got); err != nil {
			t.Fatalf("failed to parse YAML output %q: %v", out, err)
		}
		if got != "dc" {
			t.Errorf("round-trip value = %q, want %q", got, "dc")
		}
	})

	t.Run("clusters key returns error", func(t *testing.T) {
		_, err := GetConfigStringFromFile(cfgPath, "clusters.c1.name")
		if err == nil {
			t.Fatalf("GetConfigStringFromFile(): expected clusters-prefix error, got %v", err)
		}
	})

	t.Run("whole config YAML output", func(t *testing.T) {
		out, err := GetConfigStringFromFile(cfgPath, "")
		if err != nil {
			t.Fatalf("GetConfigStringFromFile(): unexpected error: %v", err)
		}
		if !strings.Contains(out, "default-cluster: dc") || !strings.Contains(out, "log:") {
			t.Errorf("yaml output missing expected fields: %s", out)
		}
	})
}

func TestGetConfigCluster(t *testing.T) {
	cluster := config.ConfigCluster{
		Name: "c1",
		Cluster: config.ConfigClusterConfig{
			URI:       "http://example.com",
			BSS:       config.ConfigClusterBSS{URI: "/bss"},
			CloudInit: config.ConfigClusterCloudInit{URI: "/ci"},
			PCS:       config.ConfigClusterPCS{URI: "/pcs"},
			SMD:       config.ConfigClusterSMD{URI: "/smd"},
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

func TestGetConfigClusterString(t *testing.T) {
	cluster := config.ConfigCluster{
		Name: "c1",
		Cluster: config.ConfigClusterConfig{
			URI:       "http://example.com",
			BSS:       config.ConfigClusterBSS{URI: "/bss"},
			CloudInit: config.ConfigClusterCloudInit{URI: "/ci"},
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

func TestReadConfig(t *testing.T) {
	t.Run("empty path", func(t *testing.T) {
		_, err := ReadConfig("")
		if err == nil {
			t.Fatal("ReadConfig(): expected error for empty path, got nil")
		}
	})

	t.Run("nonexistent file", func(t *testing.T) {
		_, err := ReadConfig("/no/such/config.yaml")
		if err == nil {
			t.Fatal("ReadConfig(): expected error for missing file, got nil")
		}
	})

	t.Run("invalid yaml", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "bad.yaml")
		if err := os.WriteFile(path, []byte("not: valid: :::"), 0o644); err != nil {
			t.Fatalf("setup failed: %v", err)
		}
		_, err := ReadConfig(path)
		if err == nil {
			t.Fatal("ReadConfig(): expected error for invalid YAML, got nil")
		}
	})

	t.Run("valid yaml", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "good.yaml")

		// Use default config
		ko := koanf.NewWithConf(kConfig)
		err := ko.Load(confmap.Provider(config.DefaultGlobalMap(), "."), nil)
		if err != nil {
			t.Fatalf("failed to load default config: %v", err)
		}

		data, err := ko.Marshal(configParser)
		if err != nil {
			t.Fatalf("unable to marshal default config: %v", err)
		}

		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatalf("failed to write config file: %v", err)
		}

		got, err := ReadConfig(path)
		if err != nil {
			t.Fatalf("ReadConfig(): unexpected error reading valid config: %v", err)
		}

		// var gotStruct, defStruct config.Config
		// err = got.Unmarshal("", &gotStruct)
		// if err != nil {
		// 	t.Errorf("ReadConfig(): unable to unmarshal config into struct: %v", err)
		// }

		if !reflect.DeepEqual(got.All(), config.DefaultGlobalMap()) {
			t.Errorf("ReadConfig() = %+v, want %+v", got, config.DefaultGlobalMap())
		}
	})
}

func TestWriteConfig(t *testing.T) {
	ko := koanf.NewWithConf(kConfig)
	err := ko.Load(confmap.Provider(config.DefaultGlobalMap(), "."), nil)
	if err != nil {
		t.Fatalf("WriteConfig(): failed to load default config")
		return
	}
	t.Run("empty path", func(t *testing.T) {
		err := WriteConfig("", ko)
		if err == nil {
			t.Fatal("WriteConfig(): expected error for empty path, got nil")
		}
	})

	t.Run("new file", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "config.yaml")
		defer os.RemoveAll(path) //nolint:errcheck // best-effort cleanup; t.TempDir also removes it

		if err := WriteConfig(path, ko); err != nil {
			t.Fatalf("WriteConfig(): error writing to new file: %v", err)
		}

		ko, err := ReadConfig(path)
		if err != nil {
			t.Fatalf("cannot read written file: %v", err)
		}

		if !reflect.DeepEqual(ko.All(), config.DefaultGlobalMap()) {
			t.Errorf("WriteConfig(): unmarshaled config = %+v, want %+v", ko.All(), config.DefaultGlobalMap())
		}
	})

	t.Run("overwrite existing file preserving permissions", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "config.yaml")
		defer os.RemoveAll(path) //nolint:errcheck // best-effort cleanup; t.TempDir also removes it

		// create an existing file with a restrictive mode
		if err := os.WriteFile(path, []byte("old"), 0o600); err != nil {
			t.Fatalf("failed to write initial file: %v", err)
		}

		if err := WriteConfig(path, ko); err != nil {
			t.Fatalf("WriteConfig(): unable to overwrite file %s: %v", path, err)
		}

		fi, err := os.Stat(path)
		if err != nil {
			t.Fatalf("failed to stat written file %s: %v", path, err)
		}
		if perm := fi.Mode().Perm(); perm != 0o600 {
			t.Errorf("WriteConfig(): file mode = %o, want 0600", perm)
		}
		matches, err := filepath.Glob(filepath.Join(tmp, ".config.yaml.*"))
		if err != nil {
			t.Fatal(err)
		}
		if len(matches) != 0 {
			t.Errorf("temporary files left behind: %v", matches)
		}
	})

	t.Run("nonexistent parent directory", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "missing", "config.yaml")
		err := WriteConfig(path, ko)
		if err == nil {
			t.Fatal("WriteConfig(): expected missing-parent error, got nil")
		}
	})

	t.Run("destination is a directory", func(t *testing.T) {
		err := WriteConfig(t.TempDir(), ko)
		if err == nil {
			t.Fatal("WriteConfig(): expected directory error, got nil")
		}
	})
}

func TestReadConfigWithDefaults(t *testing.T) {
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

		ko, err := ReadConfigWithDefaults(path)
		if err != nil {
			t.Fatalf("ReadConfigWithDefaults(): unexpected error: %v", err)
		}

		// Global default should be present even though not in the file.
		if got := ko.String("timeout"); got != config.DefaultGlobalMap()["timeout"] {
			t.Errorf("timeout = %q, want %q", got, config.DefaultGlobalMap()["timeout"])
		}

		// Cluster default (enable-auth: true) should be applied.
		var clusters []config.ConfigCluster
		if err := ko.Unmarshal("clusters", &clusters); err != nil {
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
			ko, err := ReadConfigWithDefaults(path)
			if err != nil {
				t.Fatalf("ReadConfigWithDefaults(): unexpected error: %v", err)
			}
			var clusters []config.ConfigCluster
			if err := ko.Unmarshal("clusters", &clusters); err != nil {
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

// TestLoadGlobalConfigMerged ensures that the merged loader applies defaults and
// respects user‑config precedence. The system config path is constant and may not
// exist on the test runner, which is fine – the loader skips missing files.

func TestReadConfigWithDefaultsAppliesClusterDefaults(t *testing.T) {
	cfg := []byte(`clusters:
  - name: bar
    cluster:
      uri: https://bar.example.com
`)
	path := filepath.Join(t.TempDir(), "cfg.yaml")
	if err := os.WriteFile(path, cfg, 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}
	ko, err := ReadConfigWithDefaults(path)
	if err != nil {
		t.Fatalf("ReadConfigWithDefaults error: %v", err)
	}
	var clusters []config.ConfigCluster
	if err := ko.Unmarshal("clusters", &clusters); err != nil {
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
	// Verify that the overall koanf still contains the default log.format.
	if v := ko.String("log.format"); v != config.DefaultGlobalMap()["log.format"] {
		t.Errorf("log.format = %q, want %q", v, config.DefaultGlobalMap()["log.format"])
	}
	// Ensure the slice ordering is deterministic (single element).
	if order := reflect.TypeOf(clusters); order.Kind() != reflect.Slice {
		t.Errorf("clusters not slice: %v", order)
	}
}
