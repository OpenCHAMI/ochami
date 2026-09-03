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

func TestGetBaseURI(t *testing.T) {
	orig := activeConfig
	defer func() { activeConfig = orig }()

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
				_ = cmd.Flags().Set("cluster", "bar")
			},
			want: "https://bar.example.com/hsm/v2",
		},
		{
			name:        "unknown --cluster errors",
			service:     config.ServiceSMD,
			defaultClus: "foo",
			setup: func(cmd *cobra.Command) {
				_ = cmd.Flags().Set("cluster", "nope")
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
				_ = cmd.Flags().Set("cluster-uri", "https://flag.example.com")
			},
			want: "https://flag.example.com/hsm/v2",
		},
		{
			name:        "uri flag override for SMD",
			service:     config.ServiceSMD,
			defaultClus: "foo",
			setup: func(cmd *cobra.Command) {
				_ = cmd.Flags().Set("uri", "https://svc.example.com/custom")
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

func TestGetBaseURI_UnknownServiceWithURIFlag(t *testing.T) {
	orig := activeConfig
	defer func() { activeConfig = orig }()
	activeConfig = config.Config{}

	cmd := newURICmd()
	_ = cmd.Flags().Set("uri", "https://x.example.com")

	if _, err := GetBaseURI(cmd, config.ServiceName("bogus")); err == nil {
		t.Fatal("expected error for unknown service with --uri, got nil")
	}
}

func TestGetAPIVersion(t *testing.T) {
	orig := activeConfig
	defer func() { activeConfig = orig }()

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
				_ = cmd.Flags().Set("api-version", "v9")
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
				_ = cmd.Flags().Set("cluster", "nope")
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
