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

	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/knadh/koanf/providers/structs"
	"github.com/knadh/koanf/v2"
)

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
		ko := koanf.NewWithConf(kConfig)
		if err := ko.Load(rawbytes.Provider([]byte("boot-service:\n  api-version: v1beta2\n")), configParser); err != nil {
			t.Fatalf("ko.Load unexpected error = %v", err)
		}
		if ko.String("boot-service.api-version") != "v1beta2" {
			t.Fatalf("BootService.APIVersion = %q, want v1beta2", ko.String("boot-service.api-version"))
		}
	})
}

func TestRemoveFromSlice(t *testing.T) {
	type args struct {
		slice []interface{}
		index int
	}
	tests := []struct {
		name string
		args args
		want []interface{}
	}{
		{
			name: "remove first element",
			args: args{
				slice: []interface{}{
					1,
					2,
					3,
				},
				index: 0,
			},
			want: []interface{}{
				3,
				2,
			},
		},
		{
			name: "remove middle element",
			args: args{
				slice: []interface{}{
					1,
					2,
					3,
					4,
				},
				index: 1,
			},
			want: []interface{}{
				1,
				4,
				3,
			},
		},
		{
			name: "remove last element",
			args: args{
				slice: []interface{}{
					1,
					2,
					3,
				},
				index: 2,
			},
			want: []interface{}{
				1,
				2,
			},
		},
		{
			name: "remove single element",
			args: args{
				slice: []interface{}{
					42,
				},
				index: 0,
			},
			want: []interface{}{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RemoveFromSlice(tt.args.slice, tt.args.index); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RemoveFromSlice() = %v, want %v", got, tt.want)
			}
		})
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
		os.WriteFile(path, []byte("default-cluster: old\n"), 0o644)

		if err := ModifyConfig(path, "default-cluster", "new"); err != nil {
			t.Fatalf("ModifyConfig(): unexpected error: %v", err)
		}

		ko, err := ReadConfig(path)
		if err != nil {
			t.Fatalf("read back failed: %v", err)
		}
		var got Config
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
		os.WriteFile(path, []byte("log:\n  format: pretty\n  level: info"), 0o644)

		if err := ModifyConfig(path, "log.level", "debug"); err != nil {
			t.Fatalf("ModifyConfig(): unexpected error: %v", err)
		}

		ko, err := ReadConfig(path)
		if err != nil {
			t.Fatalf("read back failed: %v", err)
		}
		var got Config
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

	t.Run("permission denied writing file", func(t *testing.T) {
		// Assume non-root context; writing to /root should fail
		err := ModifyConfig("/root/config.yaml", "default-cluster", "x")
		if err == nil {
			t.Fatal("ModifyConfig(): expected permission error, got nil")
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

	t.Run("rename to duplicate cluster name returns error", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "cfg.yaml")

		os.WriteFile(path, []byte(`
default-cluster: ""
clusters:
    - name: a
    - name: b`), 0o644)

		err := ModifyConfigCluster(path, "a", "name", false, "b")
		if err == nil {
			t.Fatalf("ModifyConfigCluster(): expected duplicate-name error, got %v", err)
		}
	})

	t.Run("add new cluster by name", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "cfg.yaml")

		os.WriteFile(path, []byte(`
default-cluster: ""
clusters: null`), 0o644)

		if err := ModifyConfigCluster(path, "c1", "name", false, "c1"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		ko, err := ReadConfig(path)
		if err != nil {
			t.Fatalf("read back failed: %v", err)
		}
		var got Config
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

		os.WriteFile(path, []byte(`
default-cluster: c1
clusters:
    - name: c1`), 0o644)

		if err := ModifyConfigCluster(path, "c1", "name", false, "c2"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		ko, err := ReadConfig(path)
		if err != nil {
			t.Fatalf("read back failed: %v", err)
		}
		var got Config
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

		os.WriteFile(path, []byte(`
default-cluster: ""
clusters: null`), 0o644)

		if err := ModifyConfigCluster(path, "c3", "name", true, "c3"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		ko, err := ReadConfig(path)
		if err != nil {
			t.Fatalf("read back failed: %v", err)
		}
		var got Config
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

		os.WriteFile(path, []byte(`
default-cluster: orig
clusters: []`), 0o644)

		if err := DeleteConfig(path, "default-cluster"); err != nil {
			t.Fatalf("DeleteConfig(): unexpected error: %v", err)
		}

		ko, err := ReadConfig(path)
		if err != nil {
			t.Fatalf("read back failed: %v", err)
		}
		var got Config
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

		os.WriteFile(path, []byte(`
log:
    format: pretty
    level: info`), 0o644)

		if err := DeleteConfig(path, "log.level"); err != nil {
			t.Fatalf("DeleteConfig(): unexpected error: %v", err)
		}

		ko, err := ReadConfig(path)
		if err != nil {
			t.Fatalf("read back failed: %v", err)
		}

		var got Config
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

	t.Run("delete non-existent key leaves config unchanged", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "cfg.yaml")

		initial := Config{
			Timeout:        30 * time.Second,
			DefaultCluster: "x",
			Log: ConfigLog{
				Format: "f",
				Level:  "l",
			},
		}
		os.WriteFile(path, []byte(`
timeout: 30s
default-cluster: x
log:
    format: f
    level: l`), 0o644)

		if err := DeleteConfig(path, "does.not.exist"); err != nil {
			t.Fatalf("DeleteConfig(): unexpected error deleting missing key: %v", err)
		}

		ko, err := ReadConfig(path)
		if err != nil {
			t.Fatalf("read back failed: %v", err)
		}

		var got Config
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
		os.WriteFile(path, []byte(`
clusters:
    - name: c1
      cluster:
        uri: u1`), 0o644)

		err := DeleteConfigCluster(path, "c1", "name")
		if err == nil {
			t.Fatalf("DeleteConfigCluster(): expected cannot unset name error, got %v", err)
		}
	})

	t.Run("cluster not found returns error", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "cfg.yaml")
		os.WriteFile(path, []byte(`
clusters:
  - name: a`), 0o644)

		err := DeleteConfigCluster(path, "b", "cluster.uri")
		if err == nil {
			t.Fatalf("DeleteConfigCluster(): expected not found error, got %v", err)
		}
	})

	t.Run("delete cluster.uri clears only URI", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "cfg.yaml")
		os.WriteFile(path, []byte(`
clusters:
  - name: c1
    cluster:
        uri: u1
        bss:
            uri: b1`), 0o644)

		if err := DeleteConfigCluster(path, "c1", "cluster.uri"); err != nil {
			t.Fatalf("DeleteConfigCluster(): unexpected error: %v", err)
		}
		ko, _ := ReadConfig(path)
		var cl []ConfigCluster
		ko.Unmarshal("clusters", &cl)
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
		os.WriteFile(path, []byte(`
clusters:
  - name: c2
    cluster:
        uri: u2
        bss:
            uri: b2`), 0o644)

		if err := DeleteConfigCluster(path, "c2", "cluster.bss.uri"); err != nil {
			t.Fatalf("DeleteConfigCluster(): unexpected error: %v", err)
		}

		ko, err := ReadConfig(path)
		if err != nil {
			t.Fatalf("read back failed: %v", err)
		}
		var got Config
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
	cfg.Load(structs.Provider(Config{
		DefaultCluster: "def",
		Log: ConfigLog{
			Format: "json",
			Level:  "warn",
		},
		Clusters: []ConfigCluster{
			{Name: "c1"},
			{Name: "c2"},
		},
	}, "koanf"), nil)

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
		// Expect a map[string]interface{} or Config depending on koanf unmarshal,
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
			found = true // unmarshaled directly into Config
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
	// Prepare a sample Config struct and write it to a temp YAML file.
	sample := Config{
		DefaultCluster: "dc",
		Log: ConfigLog{
			Format: "json",
			Level:  "debug",
		},
		Clusters: []ConfigCluster{
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
		// Expect either a map[string]interface{} or the Config struct.
		rv := reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.Map:
			// Map keys should include "default-cluster"
			if !rv.MapIndex(reflect.ValueOf("default-cluster")).IsValid() {
				t.Errorf("returned map missing default-cluster key")
			}
		case reflect.Struct:
			// Struct case: check field
			got := v.(Config)
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
	ko.Load(structs.Provider(Config{
		DefaultCluster: "dc",
		Log: ConfigLog{
			Format: "json",
			Level:  "info",
		},
		Clusters: []ConfigCluster{
			{Name: "c1"},
		},
	}, "koanf"), nil)

	t.Run("nil value returns empty string", func(t *testing.T) {
		s, err := GetConfigString(ko, "does.not.exist", "yaml")
		if err != nil {
			t.Fatalf("GetConfigString(): unexpected error: %v", err)
		}
		if s != "" {
			t.Errorf("got %q, want empty string", s)
		}
	})

	t.Run("string value ignores format", func(t *testing.T) {
		for _, format := range []string{"", "yaml", "json", "json-pretty"} {
			s, err := GetConfigString(ko, "default-cluster", format)
			if err != nil {
				t.Fatalf("GetConfigString(): unexpected error for format %q: %v", format, err)
			}
			if s != "dc" {
				t.Errorf("format %q: got %q, want %q", format, s, "dc")
			}
		}
	})

	t.Run("struct value uses sprint", func(t *testing.T) {
		s, err := GetConfigString(ko, "log", "json")
		if err != nil {
			t.Fatalf("GetConfigString(): unexpected error: %v", err)
		}
		// ConfigLog prints as "{json info}"
		wanted := `{"color":"","format":"json","level":"info"}`
		if s != wanted {
			t.Errorf("got %q, want %q", s, wanted)
		}
	})

	t.Run("unsupported format returns error", func(t *testing.T) {
		_, err := GetConfigString(ko, "clusters", "xml")
		if err == nil {
			t.Fatalf("GetConfigString(): expected unknown format error, got %v", err)
		}
	})

	t.Run("map value marshals to yaml", func(t *testing.T) {
		out, err := GetConfigString(ko, "", "yaml")
		if err != nil {
			t.Fatalf("GetConfigString(): unexpected error: %v", err)
		}
		if !strings.Contains(out, "default-cluster: dc") {
			t.Errorf("yaml output missing default-cluster: %s", out)
		}
	})

	t.Run("map value marshals to json", func(t *testing.T) {
		out, err := GetConfigString(ko, "", "json")
		if err != nil {
			t.Fatalf("GetConfigString(): unexpected error: %v", err)
		}
		if !strings.Contains(out, `"default-cluster":"dc"`) {
			t.Errorf("json output missing default-cluster: %s", out)
		}
	})

	t.Run("map value marshals to pretty json", func(t *testing.T) {
		out, err := GetConfigString(ko, "", "json-pretty")
		if err != nil {
			t.Fatalf("GetConfigString(): unexpected error: %v", err)
		}
		if !strings.Contains(out, "\n\t") {
			t.Errorf("pretty json output not indented: %s", out)
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
		_, err := GetConfigStringFromFile("", "default-cluster", "yaml")
		if err == nil {
			t.Fatalf("GetConfigStringFromFile(): expected read error, got %v", err)
		}
	})

	t.Run("nonexistent file returns error", func(t *testing.T) {
		_, err := GetConfigStringFromFile(filepath.Join(tmp, "nope.yaml"), "default-cluster", "json")
		if err == nil {
			t.Fatalf("GetConfigStringFromFile(): expected read error for missing file, got %v", err)
		}
	})

	t.Run("string value ignores format", func(t *testing.T) {
		for _, fmtName := range []string{"yaml", "json", "json-pretty", ""} {
			out, err := GetConfigStringFromFile(cfgPath, "default-cluster", fmtName)
			if err != nil {
				t.Fatalf("GetConfigStringFromFile(): unexpected error for format %q: %v", fmtName, err)
			}
			if out != "dc" {
				t.Errorf("format %q: got %q, want %q", fmtName, out, "dc")
			}
		}
	})

	t.Run("unsupported format returns error", func(t *testing.T) {
		_, err := GetConfigStringFromFile(cfgPath, "", "xml")
		if err == nil {
			t.Fatalf("GetConfigStringFromFile(): expected unknown format error, got %v", err)
		}
	})

	t.Run("clusters key returns error", func(t *testing.T) {
		_, err := GetConfigStringFromFile(cfgPath, "clusters.c1.name", "json")
		if err == nil {
			t.Fatalf("GetConfigStringFromFile(): expected clusters-prefix error, got %v", err)
		}
	})

	t.Run("whole config yaml output", func(t *testing.T) {
		out, err := GetConfigStringFromFile(cfgPath, "", "yaml")
		if err != nil {
			t.Fatalf("GetConfigStringFromFile(): unexpected error: %v", err)
		}
		if !strings.Contains(out, "default-cluster: dc") || !strings.Contains(out, "log:") {
			t.Errorf("yaml output missing expected fields: %s", out)
		}
	})

	t.Run("whole config json output", func(t *testing.T) {
		out, err := GetConfigStringFromFile(cfgPath, "", "json")
		if err != nil {
			t.Fatalf("GetConfigStringFromFile(): unexpected error: %v", err)
		}
		if !strings.Contains(out, `"default-cluster":"dc"`) {
			t.Errorf("json output missing default-cluster: %s", out)
		}
	})

	t.Run("whole config pretty json output", func(t *testing.T) {
		out, err := GetConfigStringFromFile(cfgPath, "", "json-pretty")
		if err != nil {
			t.Fatalf("GetConfigStringFromFile(): unexpected error: %v", err)
		}
		if !strings.Contains(out, "\n\t") {
			t.Errorf("pretty JSON output not indented: %s", out)
		}
	})
}

func TestGetConfigCluster(t *testing.T) {
	cluster := ConfigCluster{
		Name: "c1",
		Cluster: ConfigClusterConfig{
			URI:       "http://example.com",
			BSS:       ConfigClusterBSS{URI: "/bss"},
			CloudInit: ConfigClusterCloudInit{URI: "/ci"},
			PCS:       ConfigClusterPCS{URI: "/pcs"},
			SMD:       ConfigClusterSMD{URI: "/smd"},
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
	cluster := ConfigCluster{
		Name: "c1",
		Cluster: ConfigClusterConfig{
			URI:       "http://example.com",
			BSS:       ConfigClusterBSS{URI: "/bss"},
			CloudInit: ConfigClusterCloudInit{URI: "/ci"},
		},
	}

	t.Run("nil value returns empty", func(t *testing.T) {
		s, err := GetConfigClusterString(cluster, "does.not.exist", "yaml")
		if err != nil {
			t.Fatalf("GetConfigClusterString(): unexpected error: %v", err)
		}
		if s != "" {
			t.Errorf("got %q, want empty string", s)
		}
	})

	t.Run("string value ignores format", func(t *testing.T) {
		for _, fmtName := range []string{"yaml", "json", "json-pretty", ""} {
			s, err := GetConfigClusterString(cluster, "name", fmtName)
			if err != nil {
				t.Fatalf("GetConfigClusterString(): unexpected error for format %q: %v", fmtName, err)
			}
			if s != "c1" {
				t.Errorf("format %q: got %q, want %q", fmtName, s, "c1")
			}
		}
	})

	t.Run("unsupported format returns error", func(t *testing.T) {
		_, err := GetConfigClusterString(cluster, "", "xml")
		if err == nil {
			t.Fatalf("GetConfigClusterString(): expected unknown format error, got %v", err)
		}
	})

	t.Run("whole cluster YAML output", func(t *testing.T) {
		out, err := GetConfigClusterString(cluster, "", "yaml")
		if err != nil {
			t.Fatalf("GetConfigClusterString(): unexpected error: %v", err)
		}
		if !strings.Contains(out, "name: c1") || !strings.Contains(out, "uri: http://example.com") {
			t.Errorf("yaml output missing expected fields: %s", out)
		}
	})

	t.Run("whole cluster JSON output", func(t *testing.T) {
		out, err := GetConfigClusterString(cluster, "", "json")
		if err != nil {
			t.Fatalf("GetConfigClusterString(): unexpected error: %v", err)
		}
		if !strings.Contains(out, `"name":"c1"`) || !strings.Contains(out, `"uri":"http://example.com"`) {
			t.Errorf("json output missing expected fields: %s", out)
		}
	})

	t.Run("whole cluster pretty JSON output", func(t *testing.T) {
		out, err := GetConfigClusterString(cluster, "", "json-pretty")
		if err != nil {
			t.Fatalf("GetConfigClusterString(): unexpected error: %v", err)
		}
		if !strings.Contains(out, "\n\t") {
			t.Errorf("pretty-json output not indented: %s", out)
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
		err := ko.Load(confmap.Provider(DefaultConfigMap, "."), nil)
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

		// var gotStruct, defStruct Config
		// err = got.Unmarshal("", &gotStruct)
		// if err != nil {
		// 	t.Errorf("ReadConfig(): unable to unmarshal config into struct: %v", err)
		// }

		if !reflect.DeepEqual(got.All(), DefaultConfigMap) {
			t.Errorf("ReadConfig() = %+v, want %+v", got, DefaultConfigMap)
		}
	})
}

func TestWriteConfig(t *testing.T) {
	ko := koanf.NewWithConf(kConfig)
	err := ko.Load(confmap.Provider(DefaultConfigMap, "."), nil)
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
		defer os.RemoveAll(path)

		if err := WriteConfig(path, ko); err != nil {
			t.Fatalf("WriteConfig(): error writing to new file: %v", err)
		}

		ko, err := ReadConfig(path)
		if err != nil {
			t.Fatalf("cannot read written file: %v", err)
		}

		if !reflect.DeepEqual(ko.All(), DefaultConfigMap) {
			t.Errorf("WriteConfig(): unmarshaled config = %+v, want %+v", ko.All(), DefaultConfigMap)
		}
	})

	t.Run("overwrite existing file preserving permissions", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "config.yaml")
		defer os.RemoveAll(path)

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
	})

	t.Run("permission denied", func(t *testing.T) {
		// very likely to fail on non-root test environments
		err := WriteConfig("/root/protected.yaml", ko)
		if err == nil {
			t.Fatal("WriteConfig(): expected permission error, got nil")
		}
	})
}
