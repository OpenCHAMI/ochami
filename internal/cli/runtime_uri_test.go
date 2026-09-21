// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cli

import (
	"bytes"
	"context"
	"strings"
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

func TestGetBaseURI_Success(t *testing.T) {
	cfg := config.Config{
		DefaultCluster: "foo",
		Clusters: []config.ConfigCluster{
			{
				Name: "foo",
				Cluster: config.ConfigClusterConfig{
					URI: "https://foo.example.com",
				},
			},
			{
				Name: "bar",
				Cluster: config.ConfigClusterConfig{
					URI: "https://bar.example.com",
				},
			},
		},
	}

	tests := []struct {
		name            string
		setup           func(cmd *cobra.Command)
		service         config.ServiceName
		defaultClus     string
		want            string
		wantErr         bool
		wantErrContains string
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
		{
			// A cluster with no URI configured for any service is found by
			// name (resolveCluster succeeds) but then fails downstream in
			// GetServiceBaseURI with a "no URI configured" style error,
			// rather than a "cluster not found" error.
			name:            "cluster with no configured URI",
			service:         config.ServiceSMD,
			defaultClus:     "empty",
			wantErr:         true,
			wantErrContains: "could not get",
		},
	}
	cfg.Clusters = append(cfg.Clusters, config.ConfigCluster{Name: "empty"})

	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			c := cfg
			c.DefaultCluster = tc.defaultClus
			rt := NewTestRuntime(nil, nil, nil).WithConfig(c)

			cmd := newURICmd()
			if tc.setup != nil {
				tc.setup(cmd)
			}

			got, err := rt.GetBaseURI(cmd, tc.service)
			if (err != nil) != tc.wantErr {
				t.Fatalf("GetBaseURI error = %v, wantErr %v", err, tc.wantErr)
			}
			if tc.wantErr {
				if tc.wantErrContains != "" && !strings.Contains(err.Error(), tc.wantErrContains) {
					t.Errorf("GetBaseURI error = %q, want it to contain %q", err.Error(), tc.wantErrContains)
				}
				return
			}
			if got != tc.want {
				t.Errorf("GetBaseURI = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestGetBaseURI_UnknownServiceWithURIFlag(t *testing.T) {
	rt := NewTestRuntime(nil, nil, nil)
	cmd := newURICmd()
	if err := cmd.Flags().Set("uri", "https://x.example.com"); err != nil {
		t.Fatalf("set uri flag: %v", err)
	}

	if _, err := rt.GetBaseURI(cmd, config.ServiceName("bogus")); err == nil {
		t.Fatal("expected error for unknown service with --uri, got nil")
	}
}

func TestGetAPIVersion_Success(t *testing.T) {
	cfg := config.Config{
		DefaultCluster: "foo",
		Clusters: []config.ConfigCluster{
			{
				Name: "foo",
				Cluster: config.ConfigClusterConfig{
					BootService:     config.ConfigClusterBootService{APIVersion: "v1boot"},
					MetadataService: config.ConfigClusterMetadataService{APIVersion: "v1meta"},
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
			rt := NewTestRuntime(nil, nil, nil).WithConfig(c)

			cmd := newURICmd()
			if tc.setup != nil {
				tc.setup(cmd)
			}

			got, err := rt.GetAPIVersion(cmd, tc.service)
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

// TestGetBaseURI_ExplicitClusterOverridesDefault verifies that an explicit
// --cluster flag takes precedence over the configured default cluster.
func TestGetBaseURI_ExplicitClusterOverridesDefault(t *testing.T) {
	rt := NewTestRuntime(nil, &bytes.Buffer{}, &bytes.Buffer{})
	rt.Config = config.Config{
		DefaultCluster: "default",
		Clusters: []config.ConfigCluster{
			{Name: "default", Cluster: config.ConfigClusterConfig{BSS: config.ConfigClusterBSS{URI: "https://default.example/bss"}}},
			{Name: "chosen", Cluster: config.ConfigClusterConfig{BSS: config.ConfigClusterBSS{URI: "https://chosen.example/bss"}}},
		},
	}

	cmd := newURICmd()
	cmd.SetContext(ContextWithRuntime(context.Background(), rt))
	if err := cmd.Flags().Set("cluster", "chosen"); err != nil {
		t.Fatal(err)
	}

	got, err := rt.GetBaseURI(cmd, config.ServiceBSS)
	if err != nil {
		t.Fatalf("GetBaseURI: %v", err)
	}
	if got != "https://chosen.example/bss" {
		t.Errorf("GetBaseURI = %q, want explicit cluster URI", got)
	}
}

// TestGetAPIVersion_PrecedenceAndErrors verifies --cluster and --api-version
// precedence, and that an explicit --api-version is service-independent.
func TestGetAPIVersion_PrecedenceAndErrors(t *testing.T) {
	rt := NewTestRuntime(nil, &bytes.Buffer{}, &bytes.Buffer{})
	rt.Config = config.Config{
		DefaultCluster: "default",
		Clusters: []config.ConfigCluster{
			{Name: "default", Cluster: config.ConfigClusterConfig{BootService: config.ConfigClusterBootService{APIVersion: "v1"}}},
			{Name: "chosen", Cluster: config.ConfigClusterConfig{BootService: config.ConfigClusterBootService{APIVersion: "v2"}}},
		},
	}
	cmd := newURICmd()
	cmd.SetContext(ContextWithRuntime(context.Background(), rt))
	if err := cmd.Flags().Set("cluster", "chosen"); err != nil {
		t.Fatalf("set cluster flag: %v", err)
	}
	got, err := rt.GetAPIVersion(cmd, config.ServiceBoot)
	if err != nil || got != "v2" {
		t.Fatalf("GetAPIVersion explicit cluster = %q, %v; want v2", got, err)
	}
	if err := cmd.Flags().Set("api-version", "v3"); err != nil {
		t.Fatalf("set api-version flag: %v", err)
	}
	got, err = rt.GetAPIVersion(cmd, config.ServiceBoot)
	if err != nil || got != "v3" {
		t.Fatalf("GetAPIVersion flag = %q, %v; want v3", got, err)
	}
	if _, err := rt.GetAPIVersion(cmd, config.ServiceBSS); err != nil {
		t.Fatalf("explicit api-version should be service-independent: %v", err)
	}
}
