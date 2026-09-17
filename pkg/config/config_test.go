// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package config

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	kyaml "github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/knadh/koanf/v2"
)

func mustWriteFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("failed to write test config file %s: %v", path, err)
	}
}

func TestConfig_GetCluster(t *testing.T) {
	type args struct {
		name string
	}

	tests := []struct {
		name        string
		cfg         Config
		args        args
		want        ConfigCluster
		wantErr     bool
		wantErrName string // expected cluster name referenced in the not-found error
	}{
		{
			name: "Cluster exists in config",
			cfg: Config{
				Clusters: []ConfigCluster{
					{
						Name: "cluster-a",
						Cluster: ConfigClusterConfig{
							URI: "http://example.com/a",
						},
					},
					{
						Name: "cluster-b",
						Cluster: ConfigClusterConfig{
							URI: "http://example.com/b",
						},
					},
				},
			},
			args: args{name: "cluster-a"},
			want: ConfigCluster{
				Name: "cluster-a",
				Cluster: ConfigClusterConfig{
					URI: "http://example.com/a",
				},
			},
			wantErr: false,
		},
		{
			name: "Cluster does not exist in config",
			cfg: Config{
				Clusters: []ConfigCluster{
					{
						Name: "cluster-a",
						Cluster: ConfigClusterConfig{
							URI: "http://example.com/a",
						},
					},
				},
			},
			args:        args{name: "cluster-x"},
			want:        (ConfigCluster{}),
			wantErr:     true,
			wantErrName: "cluster-x",
		},
		{
			name:        "Empty cluster list",
			cfg:         Config{Clusters: []ConfigCluster{}},
			args:        args{name: "any-cluster"},
			want:        (ConfigCluster{}),
			wantErr:     true,
			wantErrName: "any-cluster",
		},
		{
			name: "Multiple clusters with similar names",
			cfg: Config{
				Clusters: []ConfigCluster{
					{
						Name: "cluster1",
						Cluster: ConfigClusterConfig{
							URI: "http://example.com/1",
						},
					},
					{
						Name: "cluster-1",
						Cluster: ConfigClusterConfig{
							URI: "http://example.com/1-dash",
						},
					},
					{
						Name: "cluster_1",
						Cluster: ConfigClusterConfig{
							URI: "http://example.com/1-underscore",
						},
					},
				},
			},
			args: args{name: "cluster-1"},
			want: ConfigCluster{
				Name: "cluster-1",
				Cluster: ConfigClusterConfig{
					URI: "http://example.com/1-dash",
				},
			},
			wantErr: false,
		},
		{
			name: "Exact match required, case sensitivity test",
			cfg: Config{
				Clusters: []ConfigCluster{
					{
						Name: "ClusterA",
						Cluster: ConfigClusterConfig{
							URI: "http://example.com/case",
						},
					},
					{
						Name: "clustera",
						Cluster: ConfigClusterConfig{
							URI: "http://example.com/lower",
						},
					},
				},
			},
			args: args{name: "ClusterA"},
			want: ConfigCluster{
				Name: "ClusterA",
				Cluster: ConfigClusterConfig{
					URI: "http://example.com/case",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt // capture loop variable
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.cfg.GetCluster(tt.args.name)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("GetCluster(%q) error = nil, want non-nil", tt.args.name)
				}
				// Make sure error is an ErrUnknownCluster and
				// make sure cluster name is contained in it
				var ue ErrUnknownCluster
				if !errors.As(err, &ue) {
					t.Fatalf("GetCluster(%q) error type = %T, want ErrUnknownCluster", tt.args.name, err)
				}
				if !strings.Contains(err.Error(), tt.wantErrName) {
					t.Fatalf("GetCluster(%q) error = %q, want it to mention %q", tt.args.name, err.Error(), tt.wantErrName)
				}
				if !reflect.DeepEqual(got, tt.want) {
					t.Fatalf("GetCluster(%q) got = %#v, want %#v", tt.args.name, got, tt.want)
				}
				return
			}

			if err != nil {
				t.Fatalf("GetCluster(%q) unexpected error: %v", tt.args.name, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("GetCluster(%q) got = %#v, want %#v", tt.args.name, got, tt.want)
			}
		})
	}
}

func TestConfigClusterConfig_MergeURIConfig(t *testing.T) {
	type fields struct {
		URI       string
		BSS       ConfigClusterBSS
		CloudInit ConfigClusterCloudInit
		PCS       ConfigClusterPCS
		SMD       ConfigClusterSMD
		RCS       ConfigClusterRCS
	}
	type args struct {
		c ConfigClusterConfig
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   ConfigClusterConfig
	}{
		{
			name: "empty old and empty new",
			fields: fields{
				URI: "",
				BSS: ConfigClusterBSS{
					URI: "",
				},
				CloudInit: ConfigClusterCloudInit{
					URI: "",
				},
				PCS: ConfigClusterPCS{
					URI: "",
				},
				SMD: ConfigClusterSMD{
					URI: "",
				},
				RCS: ConfigClusterRCS{
					URI: "",
				},
			},
			args: args{
				c: ConfigClusterConfig{
					URI: "",
					BSS: ConfigClusterBSS{
						URI: "",
					},
					CloudInit: ConfigClusterCloudInit{
						URI: "",
					},
					PCS: ConfigClusterPCS{
						URI: "",
					},
					SMD: ConfigClusterSMD{
						URI: "",
					},
				},
			},
			want: ConfigClusterConfig{
				URI: "",
				BSS: ConfigClusterBSS{
					URI: "",
				},
				CloudInit: ConfigClusterCloudInit{
					URI: "",
				},
				PCS: ConfigClusterPCS{
					URI: "",
				},
				SMD: ConfigClusterSMD{
					URI: "",
				},
				RCS: ConfigClusterRCS{
					URI: "",
				},
			},
		},
		{
			name: "empty old and new all fields",
			fields: fields{
				URI: "",
				BSS: ConfigClusterBSS{
					URI: "",
				},
				CloudInit: ConfigClusterCloudInit{
					URI: "",
				},
				PCS: ConfigClusterPCS{
					URI: "",
				},
				SMD: ConfigClusterSMD{
					URI: "",
				},
				RCS: ConfigClusterRCS{
					URI: "",
				},
			},
			args: args{
				c: ConfigClusterConfig{
					URI: "newUri",
					BSS: ConfigClusterBSS{
						URI: "newBss",
					},
					CloudInit: ConfigClusterCloudInit{
						URI: "newCi",
					},
					PCS: ConfigClusterPCS{
						URI: "newPcs",
					},
					SMD: ConfigClusterSMD{
						URI: "newSmd",
					},
					RCS: ConfigClusterRCS{
						URI: "newRcs",
					},
				},
			},
			want: ConfigClusterConfig{
				URI: "newUri",
				BSS: ConfigClusterBSS{
					URI: "newBss",
				},
				CloudInit: ConfigClusterCloudInit{
					URI: "newCi",
				},
				PCS: ConfigClusterPCS{
					URI: "newPcs",
				},
				SMD: ConfigClusterSMD{
					URI: "newSmd",
				},
				RCS: ConfigClusterRCS{
					URI: "newRcs",
				},
			},
		},
		{
			name: "old all fields and empty new",
			fields: fields{
				URI: "oldUri",
				BSS: ConfigClusterBSS{
					URI: "oldBss",
				},
				CloudInit: ConfigClusterCloudInit{
					URI: "oldCi",
				},
				PCS: ConfigClusterPCS{
					URI: "oldPcs",
				},
				SMD: ConfigClusterSMD{
					URI: "oldSmd",
				},
				RCS: ConfigClusterRCS{
					URI: "oldRcs",
				},
			},
			args: args{
				c: ConfigClusterConfig{
					URI: "",
					BSS: ConfigClusterBSS{
						URI: "",
					},
					CloudInit: ConfigClusterCloudInit{
						URI: "",
					},
					PCS: ConfigClusterPCS{
						URI: "",
					},
					SMD: ConfigClusterSMD{
						URI: "",
					},
					RCS: ConfigClusterRCS{
						URI: "",
					},
				},
			},
			want: ConfigClusterConfig{
				URI: "oldUri",
				BSS: ConfigClusterBSS{
					URI: "oldBss",
				},
				CloudInit: ConfigClusterCloudInit{
					URI: "oldCi",
				},
				PCS: ConfigClusterPCS{
					URI: "oldPcs",
				},
				SMD: ConfigClusterSMD{
					URI: "oldSmd",
				},
				RCS: ConfigClusterRCS{
					URI: "oldRcs",
				},
			},
		},
		{
			name: "partial override",
			fields: fields{
				URI: "oldUri",
				BSS: ConfigClusterBSS{
					URI: "oldBss",
				},
				CloudInit: ConfigClusterCloudInit{
					URI: "oldCi",
				},
				PCS: ConfigClusterPCS{
					URI: "oldPcs",
				},
				SMD: ConfigClusterSMD{
					URI: "oldSmd",
				},
			},
			args: args{
				c: ConfigClusterConfig{
					URI: "newUri",
					BSS: ConfigClusterBSS{
						URI: "",
					},
					CloudInit: ConfigClusterCloudInit{
						URI: "newCi",
					},
					PCS: ConfigClusterPCS{
						URI: "",
					},
					SMD: ConfigClusterSMD{
						URI: "newSmd",
					},
				},
			},
			want: ConfigClusterConfig{
				URI: "newUri",
				BSS: ConfigClusterBSS{
					URI: "oldBss",
				},
				CloudInit: ConfigClusterCloudInit{
					URI: "newCi",
				},
				PCS: ConfigClusterPCS{
					URI: "oldPcs",
				},
				SMD: ConfigClusterSMD{
					URI: "newSmd",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ccc := &ConfigClusterConfig{
				URI:       tt.fields.URI,
				BSS:       tt.fields.BSS,
				CloudInit: tt.fields.CloudInit,
				PCS:       tt.fields.PCS,
				SMD:       tt.fields.SMD,
				RCS:       tt.fields.RCS,
			}
			if got := ccc.MergeURIConfig(tt.args.c); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ConfigClusterConfig.MergeURIConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfigClusterConfig_GetServiceBaseURI(t *testing.T) {
	type fields struct {
		URI       string
		BSS       ConfigClusterBSS
		CloudInit ConfigClusterCloudInit
		PCS       ConfigClusterPCS
		SMD       ConfigClusterSMD
		RCS       ConfigClusterRCS
	}
	type args struct {
		svcName ServiceName
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "missing cluster and service URI",
			fields: fields{
				URI: "",
				BSS: ConfigClusterBSS{
					URI: "",
				},
				CloudInit: ConfigClusterCloudInit{
					URI: "",
				},
				PCS: ConfigClusterPCS{
					URI: "",
				},
				SMD: ConfigClusterSMD{
					URI: "",
				},
				RCS: ConfigClusterRCS{
					URI: "",
				},
			},
			args: args{
				svcName: ServiceBSS,
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "absolute service URI without cluster",
			fields: fields{
				URI: "",
				BSS: ConfigClusterBSS{
					URI: "https://service.example.com/bss",
				},
				CloudInit: ConfigClusterCloudInit{
					URI: "",
				},
				PCS: ConfigClusterPCS{
					URI: "",
				},
				SMD: ConfigClusterSMD{
					URI: "",
				},
				RCS: ConfigClusterRCS{
					URI: "",
				},
			},
			args: args{
				svcName: ServiceBSS,
			},
			want:    "https://service.example.com/bss",
			wantErr: false,
		},
		{
			name: "relative service URI without cluster",
			fields: fields{
				URI: "",
				BSS: ConfigClusterBSS{
					URI: "/bss",
				},
				CloudInit: ConfigClusterCloudInit{
					URI: "",
				},
				PCS: ConfigClusterPCS{
					URI: "",
				},
				SMD: ConfigClusterSMD{
					URI: "",
				},
				RCS: ConfigClusterRCS{
					URI: "",
				},
			},
			args: args{
				svcName: ServiceBSS,
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "default service path with cluster",
			fields: fields{
				URI: "https://cluster.local/api",
				BSS: ConfigClusterBSS{
					URI: "",
				},
				CloudInit: ConfigClusterCloudInit{
					URI: "",
				},
				PCS: ConfigClusterPCS{
					URI: "",
				},
				SMD: ConfigClusterSMD{
					URI: "",
				},
			},
			args: args{
				svcName: ServiceBSS,
			},
			want:    "https://cluster.local/api" + DefaultBasePathBSS,
			wantErr: false,
		},
		{
			name: "absolute service override with cluster",
			fields: fields{
				URI: "https://cluster.local/api",
				BSS: ConfigClusterBSS{
					URI: "https://override.example.com/bss",
				},
				CloudInit: ConfigClusterCloudInit{
					URI: "",
				},
				PCS: ConfigClusterPCS{
					URI: "",
				},
				SMD: ConfigClusterSMD{
					URI: "",
				},
			},
			args: args{
				svcName: ServiceBSS,
			},
			want:    "https://override.example.com/bss",
			wantErr: false,
		},
		{
			name: "invalid cluster URI",
			fields: fields{
				URI: "://bad_uri",
				BSS: ConfigClusterBSS{
					URI: "",
				},
				CloudInit: ConfigClusterCloudInit{
					URI: "",
				},
				PCS: ConfigClusterPCS{
					URI: "",
				},
				SMD: ConfigClusterSMD{
					URI: "",
				},
			},
			args: args{
				svcName: ServiceBSS,
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "unknown service",
			fields: fields{
				URI: "https://cluster.local",
				BSS: ConfigClusterBSS{
					URI: "",
				},
				CloudInit: ConfigClusterCloudInit{
					URI: "",
				},
				PCS: ConfigClusterPCS{
					URI: "",
				},
				SMD: ConfigClusterSMD{
					URI: "",
				},
			},
			args: args{
				svcName: ServiceName("unknown"),
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ccc := &ConfigClusterConfig{
				URI:       tt.fields.URI,
				BSS:       tt.fields.BSS,
				CloudInit: tt.fields.CloudInit,
				PCS:       tt.fields.PCS,
				SMD:       tt.fields.SMD,
				RCS:       tt.fields.RCS,
			}
			got, err := ccc.GetServiceBaseURI(tt.args.svcName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ConfigClusterConfig.GetServiceBaseURI() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ConfigClusterConfig.GetServiceBaseURI() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfigClusterConfig_BootServiceBaseURIAndMerge(t *testing.T) {
	t.Run("default boot-service path with cluster", func(t *testing.T) {
		ccc := ConfigClusterConfig{URI: "https://cluster.local/api"}
		got, err := ccc.GetServiceBaseURI(ServiceBoot)
		if err != nil {
			t.Fatalf("GetServiceBaseURI(ServiceBoot) unexpected error = %v", err)
		}
		want := "https://cluster.local/api" + DefaultBasePathBootService
		if got != want {
			t.Fatalf("GetServiceBaseURI(ServiceBoot) = %q, want %q", got, want)
		}
	})

	t.Run("absolute boot-service override", func(t *testing.T) {
		ccc := ConfigClusterConfig{BootService: ConfigClusterBootService{URI: "https://boot.example.com/boot-service"}}
		got, err := ccc.GetServiceBaseURI(ServiceBoot)
		if err != nil {
			t.Fatalf("GetServiceBaseURI(ServiceBoot) unexpected error = %v", err)
		}
		want := "https://boot.example.com/boot-service"
		if got != want {
			t.Fatalf("GetServiceBaseURI(ServiceBoot) = %q, want %q", got, want)
		}
	})

	t.Run("relative boot-service override with cluster", func(t *testing.T) {
		ccc := ConfigClusterConfig{URI: "https://cluster.local/api", BootService: ConfigClusterBootService{URI: "/custom-boot"}}
		got, err := ccc.GetServiceBaseURI(ServiceBoot)
		if err != nil {
			t.Fatalf("GetServiceBaseURI(ServiceBoot) unexpected error = %v", err)
		}
		want := "https://cluster.local/api/custom-boot"
		if got != want {
			t.Fatalf("GetServiceBaseURI(ServiceBoot) = %q, want %q", got, want)
		}
	})

	t.Run("merge boot-service URI", func(t *testing.T) {
		old := ConfigClusterConfig{URI: "https://cluster.local", BootService: ConfigClusterBootService{URI: "/old-boot"}}
		newCfg := ConfigClusterConfig{BootService: ConfigClusterBootService{URI: "/new-boot"}}
		got := old.MergeURIConfig(newCfg)
		if got.BootService.URI != "/new-boot" {
			t.Fatalf("BootService.URI = %q, want /new-boot", got.BootService.URI)
		}
	})

	t.Run("boot-service api version unmarshals", func(t *testing.T) {
		ko := koanf.NewWithConf(koanfConf)
		if err := ko.Load(rawbytes.Provider([]byte("boot-service:\n  api-version: v1beta2\n")), kyaml.Parser()); err != nil {
			t.Fatalf("ko.Load unexpected error = %v", err)
		}
		if ko.String("boot-service.api-version") != "v1beta2" {
			t.Fatalf("BootService.APIVersion = %q, want v1beta2", ko.String("boot-service.api-version"))
		}
	})
}

func TestGetUserConfigPath(t *testing.T) {
	tmpHome := t.TempDir()
	// Override HOME for this test.
	oldHome, had := os.LookupEnv("HOME")
	os.Setenv("HOME", tmpHome)
	if had {
		defer os.Setenv("HOME", oldHome)
	} else {
		defer os.Unsetenv("HOME")
	}

	p, err := UserConfigPath()
	if err != nil {
		t.Fatalf("getUserConfigPath returned error: %v", err)
	}
	want := filepath.Join(tmpHome, ".config", "ochami", "config.yaml")
	if p != want {
		t.Fatalf("path = %s, want %s", p, want)
	}
}

// TestGetDefaultTimeout simply ensures the helper returns the parsed default.

func TestGetDefaultTimeout(t *testing.T) {
	got := DefaultTimeout()
	want, _ := time.ParseDuration(DefaultGlobalMap()["timeout"].(string))
	if got != want {
		t.Fatalf("GetDefaultTimeout = %s, want %s", got, want)
	}
}

// TestReadConfigWithDefaultsAppliesClusterDefaults checks that a cluster that
// omits enable‑auth receives the default value (true).
