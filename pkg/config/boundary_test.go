// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/v2"
)

func TestMergeURIConfigIncludesEveryService(t *testing.T) {
	old := ConfigClusterConfig{
		MetadataService: ConfigClusterMetadataService{URI: "/old-metadata"},
		BootService:     ConfigClusterBootService{URI: "/old-boot"},
	}
	got := old.MergeURIConfig(ConfigClusterConfig{
		MetadataService: ConfigClusterMetadataService{URI: "/new-metadata"},
		BootService:     ConfigClusterBootService{URI: "/new-boot"},
	})
	if got.MetadataService.URI != "/new-metadata" || got.BootService.URI != "/new-boot" {
		t.Fatalf("MergeURIConfig() = %#v", got)
	}
}

func TestGetServiceBaseURIBoundaries(t *testing.T) {
	tests := []struct {
		name string
		cfg  ConfigClusterConfig
		svc  ServiceName
		as   any
	}{
		{name: "missing boot URI", svc: ServiceBoot, as: &ErrMissingURI{}},
		{name: "missing metadata URI", svc: ServiceMetadata, as: &ErrMissingURI{}},
		{name: "missing PCS URI", svc: ServicePCS, as: &ErrMissingURI{}},
		{name: "missing SMD URI", svc: ServiceSMD, as: &ErrMissingURI{}},
		{name: "missing RCS URI", svc: ServiceRCS, as: &ErrMissingURI{}},
		{name: "invalid cluster URI", cfg: ConfigClusterConfig{URI: "relative"}, svc: ServiceSMD, as: &ErrInvalidURI{}},
		{name: "malformed service URI", cfg: ConfigClusterConfig{SMD: ConfigClusterSMD{URI: "%zz"}}, svc: ServiceSMD, as: &ErrInvalidServiceURI{}},
		{name: "opaque service URI", cfg: ConfigClusterConfig{SMD: ConfigClusterSMD{URI: "mailto:user@example.com"}}, svc: ServiceSMD, as: &ErrInvalidServiceURI{}},
		{name: "relative service without cluster", cfg: ConfigClusterConfig{SMD: ConfigClusterSMD{URI: "/smd"}}, svc: ServiceSMD, as: &ErrInvalidServiceURI{}},
		{name: "empty service URL", cfg: ConfigClusterConfig{SMD: ConfigClusterSMD{URI: "?query=yes"}}, svc: ServiceSMD, as: &ErrInvalidServiceURI{}},
		{name: "unknown service", cfg: ConfigClusterConfig{URI: "https://cluster.example"}, svc: ServiceName("unknown"), as: &ErrUnknownService{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.cfg.GetServiceBaseURI(tt.svc)
			if err == nil || !errors.As(err, tt.as) {
				t.Fatalf("GetServiceBaseURI() error = %v, want %T", err, tt.as)
			}
		})
	}
}

func TestValidateConfigNullAndShapeBoundaries(t *testing.T) {
	tests := []struct {
		name string
		raw  map[string]any
		want string
	}{
		{name: "global null", raw: map[string]any{"timeout": nil}, want: "non-null"},
		{name: "empty global", raw: map[string]any{"log": map[string]any{"format": ""}}, want: "non-empty"},
		{
			name: "null cluster boolean",
			raw: map[string]any{
				"clusters": []map[string]any{{"name": "demo", "cluster": map[string]any{"enable-auth": nil}}},
			},
			want: "boolean",
		},
		{
			name: "non-map cluster is ignored by semantic validation",
			raw: map[string]any{
				"clusters": []map[string]any{{"name": "demo", "cluster": "bad"}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ko := koanf.NewWithConf(koanfConf)
			if err := ko.Load(confmap.Provider(tt.raw, "."), nil); err != nil {
				t.Fatal(err)
			}
			err := ValidateConfig(ko)
			if tt.want == "" {
				if err != nil {
					t.Fatalf("ValidateConfig() error = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("ValidateConfig() error = %v, want text %q", err, tt.want)
			}
		})
	}
}

func TestLoadClusterShapeBoundaries(t *testing.T) {
	tests := []struct {
		name string
		data string
		want string
	}{
		{name: "missing name", data: "clusters:\n  - cluster: {}\n", want: "missing a name"},
		{name: "empty name", data: "clusters:\n  - name: ''\n    cluster: {}\n", want: "missing a name"},
		{name: "non-map cluster", data: "clusters:\n  - name: demo\n    cluster: bad\n", want: "not a map"},
		{name: "type conflict", data: "log: scalar\n", want: "unable to merge"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.yaml")
			if err := os.WriteFile(path, []byte(tt.data), 0o600); err != nil {
				t.Fatal(err)
			}
			_, err := Load([]Source{{Name: "test", Path: path}})
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Load() error = %v, want text %q", err, tt.want)
			}
		})
	}
}

func TestLoadAcceptsNullClusterConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("clusters:\n  - name: demo\n    cluster:\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load([]Source{{Name: "test", Path: path}})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(cfg.Clusters) != 1 || cfg.Clusters[0].Name != "demo" || !cfg.Clusters[0].Cluster.EnableAuth {
		t.Fatalf("clusters = %#v, want defaulted demo cluster", cfg.Clusters)
	}
}
