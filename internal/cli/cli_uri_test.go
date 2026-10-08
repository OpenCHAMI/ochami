// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cli

import (
	"testing"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/pkg/config"
)

// newURICmd returns a cobra command with the flags GetBaseURI and
// GetAPIVersion consult.
func newURICmd() *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("cluster", "", "cluster name")
	cmd.Flags().String("cluster-uri", "", "cluster URI")
	cmd.Flags().String("uri", "", "service URI")
	cmd.Flags().String("api-version", "", "API version")
	return cmd
}

// TestGetBaseURI_Resolution verifies how GetBaseURI resolves a service's base URI
// from the default cluster, --cluster, --cluster-uri, and --uri, and that it
// fails for an unknown cluster.
func TestGetBaseURI_Resolution(t *testing.T) {
	orig := activeConfig
	defer func() { activeConfig = orig }()

	cfg := config.Config{
		DefaultCluster: "foo",
		Clusters: []config.Cluster{
			{
				Name: "foo",
				Cluster: config.ClusterConfig{
					URI: "https://foo.example.com",
				},
			},
			{
				Name: "bar",
				Cluster: config.ClusterConfig{
					URI: "https://bar.example.com",
				},
			},
		},
	}

	tests := []struct {
		name        string
		setup       func(cmd *cobra.Command)
		service     config.ServiceName
		defaultClus string
		want        string
		wantErr     bool
	}{
		{
			name:        "default cluster SMD",
			service:     config.ServiceSMD,
			defaultClus: "foo",
			want:        "https://foo.example.com/hsm/v2",
		},
		{
			name:        "explicit --cluster overrides default",
			service:     config.ServiceSMD,
			defaultClus: "foo",
			setup: func(cmd *cobra.Command) {
				if err := cmd.Flags().Set("cluster", "bar"); err != nil {
					t.Fatalf("set cluster flag: %v", err)
				}
			},
			want: "https://bar.example.com/hsm/v2",
		},
		{
			name:        "unknown --cluster errors",
			service:     config.ServiceSMD,
			defaultClus: "foo",
			setup: func(cmd *cobra.Command) {
				if err := cmd.Flags().Set("cluster", "nope"); err != nil {
					t.Fatalf("set cluster flag: %v", err)
				}
			},
			wantErr: true,
		},
		{
			name:        "unknown default cluster errors",
			service:     config.ServiceSMD,
			defaultClus: "missing",
			wantErr:     true,
		},
		{
			name:        "cluster-uri flag override",
			service:     config.ServiceSMD,
			defaultClus: "",
			setup: func(cmd *cobra.Command) {
				if err := cmd.Flags().Set("cluster-uri", "https://flag.example.com"); err != nil {
					t.Fatalf("set cluster-uri flag: %v", err)
				}
			},
			want: "https://flag.example.com/hsm/v2",
		},
		{
			name:        "uri flag override for SMD",
			service:     config.ServiceSMD,
			defaultClus: "foo",
			setup: func(cmd *cobra.Command) {
				if err := cmd.Flags().Set("uri", "https://svc.example.com/custom"); err != nil {
					t.Fatalf("set uri flag: %v", err)
				}
			},
			want: "https://svc.example.com/custom",
		},
	}

	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			c := cfg
			c.DefaultCluster = tc.defaultClus
			activeConfig = c

			cmd := newURICmd()
			if tc.setup != nil {
				tc.setup(cmd)
			}

			got, err := GetBaseURI(cmd, tc.service)
			if (err != nil) != tc.wantErr {
				t.Fatalf("GetBaseURI error = %v, wantErr %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			if got != tc.want {
				t.Errorf("GetBaseURI = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestGetBaseURI_UnknownServiceWithURIFlag verifies that GetBaseURI rejects an
// unknown service even when --uri is set.
func TestGetBaseURI_UnknownServiceWithURIFlag(t *testing.T) {
	orig := activeConfig
	defer func() { activeConfig = orig }()
	activeConfig = config.Config{}

	cmd := newURICmd()
	if err := cmd.Flags().Set("uri", "https://x.example.com"); err != nil {
		t.Fatalf("set uri flag: %v", err)
	}

	if _, err := GetBaseURI(cmd, config.ServiceName("bogus")); err == nil {
		t.Fatal("expected error for unknown service with --uri, got nil")
	}
}

// TestGetAPIVersion_Resolution verifies that GetAPIVersion resolves each service's
// API version from the default cluster or from --api-version, and fails for a
// service without an API version or an unknown cluster.
func TestGetAPIVersion_Resolution(t *testing.T) {
	orig := activeConfig
	defer func() { activeConfig = orig }()

	cfg := config.Config{
		DefaultCluster: "foo",
		Clusters: []config.Cluster{
			{
				Name: "foo",
				Cluster: config.ClusterConfig{
					BootService:     config.ClusterBootService{APIVersion: "v1boot"},
					MetadataService: config.ClusterMetadataService{APIVersion: "v1meta"},
				},
			},
		},
	}

	tests := []struct {
		name        string
		service     config.ServiceName
		defaultClus string
		setup       func(cmd *cobra.Command)
		want        string
		wantErr     bool
	}{
		{
			name:        "boot service from config",
			service:     config.ServiceBoot,
			defaultClus: "foo",
			want:        "v1boot",
		},
		{
			name:        "metadata service from config",
			service:     config.ServiceMetadata,
			defaultClus: "foo",
			want:        "v1meta",
		},
		{
			name:        "api-version flag override",
			service:     config.ServiceBoot,
			defaultClus: "foo",
			setup: func(cmd *cobra.Command) {
				if err := cmd.Flags().Set("api-version", "v9"); err != nil {
					t.Fatalf("set api-version flag: %v", err)
				}
			},
			want: "v9",
		},
		{
			name:        "unknown service without flag errors",
			service:     config.ServiceSMD,
			defaultClus: "foo",
			wantErr:     true,
		},
		{
			name:        "unknown --cluster errors",
			service:     config.ServiceBoot,
			defaultClus: "foo",
			setup: func(cmd *cobra.Command) {
				if err := cmd.Flags().Set("cluster", "nope"); err != nil {
					t.Fatalf("set cluster flag: %v", err)
				}
			},
			wantErr: true,
		},
		{
			name:        "unknown default cluster errors",
			service:     config.ServiceBoot,
			defaultClus: "missing",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			c := cfg
			c.DefaultCluster = tc.defaultClus
			activeConfig = c

			cmd := newURICmd()
			if tc.setup != nil {
				tc.setup(cmd)
			}

			got, err := GetAPIVersion(cmd, tc.service)
			if (err != nil) != tc.wantErr {
				t.Fatalf("GetAPIVersion error = %v, wantErr %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			if got != tc.want {
				t.Errorf("GetAPIVersion = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestGetBaseURI_ExplicitClusterOverridesDefault verifies that GetBaseURI uses
// the cluster named by --cluster rather than the configured default cluster.
func TestGetBaseURI_ExplicitClusterOverridesDefault(t *testing.T) {
	orig := ActiveConfig()
	t.Cleanup(func() { SetActiveConfig(orig) })
	SetActiveConfig(config.Config{
		DefaultCluster: "default",
		Clusters: []config.Cluster{
			{Name: "default", Cluster: config.ClusterConfig{BSS: config.ClusterBSS{URI: "https://default.example/bss"}}},
			{Name: "chosen", Cluster: config.ClusterConfig{BSS: config.ClusterBSS{URI: "https://chosen.example/bss"}}},
		},
	})
	cmd := newURICmd()
	if err := cmd.Flags().Set("cluster", "chosen"); err != nil {
		t.Fatal(err)
	}

	got, err := GetBaseURI(cmd, config.ServiceBSS)
	if err != nil {
		t.Fatalf("GetBaseURI: %v", err)
	}
	if got != "https://chosen.example/bss" {
		t.Errorf("GetBaseURI = %q, want explicit cluster URI", got)
	}
}

// TestGetAPIVersion_Precedence verifies that GetAPIVersion takes the
// API version from the --cluster cluster, that --api-version overrides it, and
// that --api-version applies to any service.
func TestGetAPIVersion_Precedence(t *testing.T) {
	orig := ActiveConfig()
	t.Cleanup(func() { SetActiveConfig(orig) })
	SetActiveConfig(config.Config{
		DefaultCluster: "default",
		Clusters: []config.Cluster{
			{Name: "default", Cluster: config.ClusterConfig{BootService: config.ClusterBootService{APIVersion: "v1"}}},
			{Name: "chosen", Cluster: config.ClusterConfig{BootService: config.ClusterBootService{APIVersion: "v2"}}},
		},
	})
	cmd := newURICmd()
	if err := cmd.Flags().Set("cluster", "chosen"); err != nil {
		t.Fatalf("set cluster flag: %v", err)
	}
	got, err := GetAPIVersion(cmd, config.ServiceBoot)
	if err != nil || got != "v2" {
		t.Fatalf("GetAPIVersion explicit cluster = %q, %v; want v2", got, err)
	}
	if err := cmd.Flags().Set("api-version", "v3"); err != nil {
		t.Fatalf("set api-version flag: %v", err)
	}
	got, err = GetAPIVersion(cmd, config.ServiceBoot)
	if err != nil || got != "v3" {
		t.Fatalf("GetAPIVersion flag = %q, %v; want v3", got, err)
	}
	if _, err := GetAPIVersion(cmd, config.ServiceBSS); err != nil {
		t.Fatalf("explicit api-version should be service-independent: %v", err)
	}
}
