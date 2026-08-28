// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package config

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/rawbytes"
)

func yamlConfigLoader(name, data string) configLoader {
	return configLoader{
		name:     name,
		provider: rawbytes.Provider([]byte(data)),
		parser:   configParser,
	}
}

func defaultConfigLoader() configLoader {
	return configLoader{
		name:     "default",
		provider: confmap.Provider(DefaultConfigMap, "."),
	}
}

func TestLoadConfigSources_MergePrecedenceAndOrder(t *testing.T) {
	sources := []configLoader{
		defaultConfigLoader(),
		yamlConfigLoader("system", `timeout: 45s
clusters:
  - name: zeta
    cluster:
      uri: https://zeta.example.com
      enable-auth: "false"
  - name: alpha
    cluster:
      uri: https://alpha.example.com
`),
		yamlConfigLoader("user", `timeout: 1m
clusters:
  - name: beta
    cluster:
      uri: https://beta.example.com
  - name: zeta
    cluster:
      bss:
        uri: /custom-bss
`),
	}

	ko, cfg, err := loadConfigSources(sources)
	if err != nil {
		t.Fatalf("loadConfigSources(): unexpected error: %v", err)
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

	var koClusters []ConfigCluster
	if err := ko.Unmarshal("clusters", &koClusters); err != nil {
		t.Fatalf("unmarshal effective clusters: %v", err)
	}
	if !reflect.DeepEqual(koClusters, cfg.Clusters) {
		t.Errorf("koanf clusters and Config clusters differ: %#v != %#v", koClusters, cfg.Clusters)
	}
}

func TestLoadConfigSources_MissingOptionalSources(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.yaml")
	ko, cfg, err := loadConfigSources([]configLoader{
		defaultConfigLoader(),
		{name: "system", provider: file.Provider(missing), parser: configParser},
		{name: "user", provider: file.Provider(missing + ".user"), parser: configParser},
	})
	if err != nil {
		t.Fatalf("loadConfigSources(): unexpected error: %v", err)
	}
	if cfg.Timeout != GetDefaultTimeout() {
		t.Errorf("timeout = %s, want %s", cfg.Timeout, GetDefaultTimeout())
	}
	if len(cfg.Clusters) != 0 {
		t.Errorf("clusters = %v, want none", cfg.Clusters)
	}
	if ko.String("log.format") != DefaultConfigMap["log.format"] {
		t.Errorf("log.format = %q, want default %q", ko.String("log.format"), DefaultConfigMap["log.format"])
	}
}

func TestLoadConfigSources_SourceErrors(t *testing.T) {
	tests := []struct {
		name       string
		sourceName string
		data       string
		want       string
	}{
		{
			name:       "malformed YAML",
			sourceName: "system",
			data:       "clusters: [\n",
			want:       "unable to load config 'system'",
		},
		{
			name:       "null global",
			sourceName: "user",
			data:       "timeout:\n",
			want:       "invalid config 'user'",
		},
		{
			name:       "strict type mismatch",
			sourceName: "user",
			data:       "timeout: 30\n",
			want:       "unable to merge config 'user'",
		},
		{
			name:       "invalid cluster boolean",
			sourceName: "system",
			data:       "clusters:\n  - name: foo\n    cluster:\n      enable-auth: maybe\n",
			want:       "unable to merge cluster 'foo' from config 'system'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := loadConfigSources([]configLoader{
				defaultConfigLoader(),
				yamlConfigLoader(tt.sourceName, tt.data),
			})
			if err == nil {
				t.Fatal("loadConfigSources(): expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error = %q, want substring %q", err, tt.want)
			}
		})
	}
}

func TestLoadGlobalConfigDefaultOnly_ReplacesStaleGlobals(t *testing.T) {
	originalConfig := GlobalConfig
	originalKoanf := GlobalKoanf
	t.Cleanup(func() {
		GlobalConfig = originalConfig
		GlobalKoanf = originalKoanf
	})

	GlobalConfig = Config{
		DefaultCluster: "stale",
		Timeout:        time.Hour,
		Clusters: []ConfigCluster{
			{Name: "stale"},
		},
	}
	GlobalKoanf = nil

	if err := LoadGlobalConfigDefaultOnly(); err != nil {
		t.Fatalf("LoadGlobalConfigDefaultOnly(): unexpected error: %v", err)
	}
	if GlobalConfig.Timeout != GetDefaultTimeout() {
		t.Errorf("timeout = %s, want %s", GlobalConfig.Timeout, GetDefaultTimeout())
	}
	if GlobalConfig.DefaultCluster != "" {
		t.Errorf("default cluster = %q, want empty", GlobalConfig.DefaultCluster)
	}
	if len(GlobalConfig.Clusters) != 0 {
		t.Errorf("clusters = %v, want none", GlobalConfig.Clusters)
	}
	if GlobalKoanf == nil {
		t.Fatal("GlobalKoanf = nil, want effective defaults")
	}
}
