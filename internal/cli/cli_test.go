// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cli

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/OpenCHAMI/ochami/internal/config"
	"github.com/spf13/cobra"
)

func TestIOStream_AskToCreate(t *testing.T) {
	t.Run("empty path", func(t *testing.T) {
		t.Parallel()
		inBuf := &bytes.Buffer{}
		outBuf := &bytes.Buffer{}
		errBuf := &bytes.Buffer{}
		ios := newIOStream(inBuf, outBuf, errBuf)

		got, err := ios.AskToCreate("")
		if got != false {
			t.Errorf("AskToCreate(\"\") = %v, want false", got)
		}
		if err == nil || !strings.Contains(err.Error(), "path cannot be empty") {
			t.Errorf("AskToCreate(\"\") error = %v, want non-nil containing “path cannot be empty”", err)
		}
		if outBuf.Len() != 0 {
			t.Errorf("stdout = %q, want empty", outBuf.String())
		}
		if errBuf.Len() != 0 {
			t.Errorf("stderr = %q, want empty", errBuf.String())
		}
	})

	t.Run("existing file", func(t *testing.T) {
		t.Parallel()
		tmp := t.TempDir()
		f := filepath.Join(tmp, "exists")
		if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
			t.Fatalf("setup write: %v", err)
		}

		inBuf := &bytes.Buffer{}
		outBuf := &bytes.Buffer{}
		errBuf := &bytes.Buffer{}
		ios := newIOStream(inBuf, outBuf, errBuf)

		got, err := ios.AskToCreate(f)
		if got != false {
			t.Errorf("AskToCreate(%q) = %v, want false", f, got)
		}
		if !errors.Is(err, FileExistsError) {
			t.Errorf("AskToCreate(%q) error = %v, want FileExistsError", f, err)
		}
		if outBuf.Len() != 0 {
			t.Errorf("stdout = %q, want empty", outBuf.String())
		}
		if errBuf.Len() != 0 {
			t.Errorf("stderr = %q, want empty", errBuf.String())
		}
	})

	t.Run("nonexistent file, user declines", func(t *testing.T) {
		t.Parallel()
		tmp := t.TempDir()
		path := filepath.Join(tmp, "noexist")

		inBuf := bytes.NewBufferString("n\n")
		outBuf := &bytes.Buffer{}
		errBuf := &bytes.Buffer{}
		ios := newIOStream(inBuf, outBuf, errBuf)

		got, err := ios.AskToCreate(path)
		if got != false {
			t.Errorf("AskToCreate(%q) decline = %v, want false", path, got)
		}
		if err != nil {
			t.Errorf("AskToCreate(%q) decline error = %v, want nil", path, err)
		}
		wantPrompt := fmt.Sprintf("%s does not exist. Create it? [yn]:", path)
		if errBuf.String() != wantPrompt {
			t.Errorf("stderr = %q, want %q", errBuf.String(), wantPrompt)
		}
		if outBuf.Len() != 0 {
			t.Errorf("stdout = %q, want empty", outBuf.String())
		}
	})

	t.Run("nonexistent file, user accepts", func(t *testing.T) {
		t.Parallel()
		tmp := t.TempDir()
		path := filepath.Join(tmp, "noexist2")

		inBuf := bytes.NewBufferString("y\n")
		outBuf := &bytes.Buffer{}
		errBuf := &bytes.Buffer{}
		ios := newIOStream(inBuf, outBuf, errBuf)

		got, err := ios.AskToCreate(path)
		if got != true {
			t.Errorf("AskToCreate(%q) accept = %v, want true", path, got)
		}
		if err != nil {
			t.Errorf("AskToCreate(%q) accept error = %v, want nil", path, err)
		}
		wantPrompt := fmt.Sprintf("%s does not exist. Create it? [yn]:", path)
		if errBuf.String() != wantPrompt {
			t.Errorf("stderr = %q, want %q", errBuf.String(), wantPrompt)
		}
		if outBuf.Len() != 0 {
			t.Errorf("stdout = %q, want empty", outBuf.String())
		}
	})
}

func TestIOStream_LoopYesNo(t *testing.T) {
	cases := []struct {
		name      string
		input     string
		want      bool
		wantCount int
	}{
		{
			name:      "yes first try",
			input:     "y\n",
			want:      true,
			wantCount: 1,
		},
		{
			name:      "no first try",
			input:     "n\n",
			want:      false,
			wantCount: 1,
		},
		{
			name:      "invalid then no",
			input:     "maybe\nn\n",
			want:      false,
			wantCount: 2,
		},
	}

	for _, tt := range cases {
		// Create per-iteration copy of test tt so that running
		// tests in parallel does not reuse the same test for
		// each run.
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			inBuf := bytes.NewBufferString(tc.input)
			errBuf := &bytes.Buffer{}
			ios := newIOStream(inBuf, io.Discard, errBuf)

			got, err := ios.LoopYesNo("Proceed?")
			if err != nil {
				t.Fatalf("LoopYesNo() error = %v, want nil", err)
			}
			if got != tc.want {
				t.Errorf("LoopYesNo() = %v, want %v", got, tc.want)
			}

			prompt := "Proceed? [yn]:"
			if count := strings.Count(errBuf.String(), prompt); count != tc.wantCount {
				t.Errorf("prompt count = %d, want %d", count, tc.wantCount)
			}
		})
	}
}

func Test_CreateIfNotExists(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "empty path",
			args: args{
				path: "",
			},
			wantErr: true,
		},
		{
			name: "create new file",
			args: args{
				path: "/tmp/newfile",
			},
			wantErr: false,
		},
		{
			name: "already exists",
			args: args{
				path: "/tmp/newfile",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CreateIfNotExists(tt.args.path); (err != nil) != tt.wantErr {
				t.Errorf("CreateIfNotExists() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetBaseURI(t *testing.T) {
	// Save original global config and restore after tests
	origConfig := config.GlobalConfig
	defer func() { config.GlobalConfig = origConfig }()

	tests := []struct {
		name        string
		serviceName config.ServiceName
		setupConfig func()
		setupCmd    func() *cobra.Command
		want        string
		wantErr     bool
		errContains string
	}{
		{
			name:        "no config, no flags returns error",
			serviceName: config.ServiceSMD,
			setupConfig: func() {
				config.GlobalConfig = config.Config{}
			},
			setupCmd: func() *cobra.Command {
				cmd := &cobra.Command{}
				cmd.Flags().String("cluster", "", "cluster name")
				cmd.Flags().String("cluster-uri", "", "cluster URI")
				cmd.Flags().String("uri", "", "service URI")
				return cmd
			},
			wantErr:     true,
			errContains: "could not get",
		},
		{
			name:        "default cluster with service URI",
			serviceName: config.ServiceSMD,
			setupConfig: func() {
				config.GlobalConfig = config.Config{
					DefaultCluster: "test-cluster",
					Clusters: []config.ConfigCluster{
						{
							Name: "test-cluster",
							Cluster: config.ConfigClusterConfig{
								URI: "https://example.com",
								SMD: config.ConfigClusterSMD{
									URI: "",
								},
							},
						},
					},
				}
			},
			setupCmd: func() *cobra.Command {
				cmd := &cobra.Command{}
				cmd.Flags().String("cluster", "", "cluster name")
				cmd.Flags().String("cluster-uri", "", "cluster URI")
				cmd.Flags().String("uri", "", "service URI")
				return cmd
			},
			want:    "https://example.com/hsm/v2",
			wantErr: false,
		},
		{
			name:        "default cluster not found",
			serviceName: config.ServiceSMD,
			setupConfig: func() {
				config.GlobalConfig = config.Config{
					DefaultCluster: "missing-cluster",
					Clusters:       []config.ConfigCluster{},
				}
			},
			setupCmd: func() *cobra.Command {
				cmd := &cobra.Command{}
				cmd.Flags().String("cluster", "", "cluster name")
				cmd.Flags().String("cluster-uri", "", "cluster URI")
				cmd.Flags().String("uri", "", "service URI")
				return cmd
			},
			wantErr:     true,
			errContains: "default cluster missing-cluster not found",
		},
		{
			name:        "--cluster flag does NOT override default cluster (default takes precedence)",
			serviceName: config.ServiceBSS,
			setupConfig: func() {
				config.GlobalConfig = config.Config{
					DefaultCluster: "default-cluster",
					Clusters: []config.ConfigCluster{
						{
							Name: "default-cluster",
							Cluster: config.ConfigClusterConfig{
								URI: "https://default.com",
							},
						},
						{
							Name: "override-cluster",
							Cluster: config.ConfigClusterConfig{
								URI: "https://override.com",
							},
						},
					},
				}
			},
			setupCmd: func() *cobra.Command {
				cmd := &cobra.Command{}
				cmd.Flags().String("cluster", "", "cluster name")
				cmd.Flags().String("cluster-uri", "", "cluster URI")
				cmd.Flags().String("uri", "", "service URI")
				_ = cmd.Flags().Set("cluster", "override-cluster")
				return cmd
			},
			want:    "https://default.com/boot/v1",
			wantErr: false,
		},
		{
			name:        "--cluster flag with nonexistent cluster",
			serviceName: config.ServiceSMD,
			setupConfig: func() {
				config.GlobalConfig = config.Config{
					Clusters: []config.ConfigCluster{
						{
							Name: "exists",
							Cluster: config.ConfigClusterConfig{
								URI: "https://exists.com",
							},
						},
					},
				}
			},
			setupCmd: func() *cobra.Command {
				cmd := &cobra.Command{}
				cmd.Flags().String("cluster", "", "cluster name")
				cmd.Flags().String("cluster-uri", "", "cluster URI")
				cmd.Flags().String("uri", "", "service URI")
				_ = cmd.Flags().Set("cluster", "nonexistent")
				return cmd
			},
			wantErr:     true,
			errContains: "cluster nonexistent not found",
		},
		{
			name:        "--cluster-uri flag overrides config",
			serviceName: config.ServiceSMD,
			setupConfig: func() {
				config.GlobalConfig = config.Config{
					DefaultCluster: "test-cluster",
					Clusters: []config.ConfigCluster{
						{
							Name: "test-cluster",
							Cluster: config.ConfigClusterConfig{
								URI: "https://config.com",
							},
						},
					},
				}
			},
			setupCmd: func() *cobra.Command {
				cmd := &cobra.Command{}
				cmd.Flags().String("cluster", "", "cluster name")
				cmd.Flags().String("cluster-uri", "", "cluster URI")
				cmd.Flags().String("uri", "", "service URI")
				_ = cmd.Flags().Set("cluster-uri", "https://flag.com")
				return cmd
			},
			want:    "https://flag.com/hsm/v2",
			wantErr: false,
		},
		{
			name:        "service --uri flag overrides service config",
			serviceName: config.ServiceBoot,
			setupConfig: func() {
				config.GlobalConfig = config.Config{
					DefaultCluster: "test-cluster",
					Clusters: []config.ConfigCluster{
						{
							Name: "test-cluster",
							Cluster: config.ConfigClusterConfig{
								URI: "https://cluster.com",
								BootService: config.ConfigClusterBootService{
									URI: "https://boot-config.com",
								},
							},
						},
					},
				}
			},
			setupCmd: func() *cobra.Command {
				cmd := &cobra.Command{}
				cmd.Flags().String("cluster", "", "cluster name")
				cmd.Flags().String("cluster-uri", "", "cluster URI")
				cmd.Flags().String("uri", "", "service URI")
				_ = cmd.Flags().Set("uri", "https://boot-flag.com")
				return cmd
			},
			want:    "https://boot-flag.com",
			wantErr: false,
		},
		{
			name:        "boot-service uses default path",
			serviceName: config.ServiceBoot,
			setupConfig: func() {
				config.GlobalConfig = config.Config{
					DefaultCluster: "test-cluster",
					Clusters: []config.ConfigCluster{
						{
							Name: "test-cluster",
							Cluster: config.ConfigClusterConfig{
								URI: "https://example.com",
							},
						},
					},
				}
			},
			setupCmd: func() *cobra.Command {
				cmd := &cobra.Command{}
				cmd.Flags().String("cluster", "", "cluster name")
				cmd.Flags().String("cluster-uri", "", "cluster URI")
				cmd.Flags().String("uri", "", "service URI")
				return cmd
			},
			want:    "https://example.com/boot-service",
			wantErr: false,
		},
		{
			name:        "metadata-service uses default path",
			serviceName: config.ServiceMetadata,
			setupConfig: func() {
				config.GlobalConfig = config.Config{
					DefaultCluster: "test-cluster",
					Clusters: []config.ConfigCluster{
						{
							Name: "test-cluster",
							Cluster: config.ConfigClusterConfig{
								URI: "https://example.com",
							},
						},
					},
				}
			},
			setupCmd: func() *cobra.Command {
				cmd := &cobra.Command{}
				cmd.Flags().String("cluster", "", "cluster name")
				cmd.Flags().String("cluster-uri", "", "cluster URI")
				cmd.Flags().String("uri", "", "service URI")
				return cmd
			},
			want:    "https://example.com/metadata-service",
			wantErr: false,
		},
		{
			name:        "absolute service URI without cluster",
			serviceName: config.ServiceCloudInit,
			setupConfig: func() {
				config.GlobalConfig = config.Config{
					Clusters: []config.ConfigCluster{
						{
							Name: "test",
							Cluster: config.ConfigClusterConfig{
								CloudInit: config.ConfigClusterCloudInit{
									URI: "https://standalone.com/ci",
								},
							},
						},
					},
				}
			},
			setupCmd: func() *cobra.Command {
				cmd := &cobra.Command{}
				cmd.Flags().String("cluster", "", "cluster name")
				cmd.Flags().String("cluster-uri", "", "cluster URI")
				cmd.Flags().String("uri", "", "service URI")
				_ = cmd.Flags().Set("cluster", "test")
				return cmd
			},
			want:    "https://standalone.com/ci",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupConfig()
			cmd := tt.setupCmd()

			got, err := GetBaseURI(cmd, tt.serviceName)
			if tt.wantErr {
				if err == nil {
					t.Errorf("GetBaseURI() error = nil, wantErr = true")
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("GetBaseURI() error = %q, want to contain %q", err.Error(), tt.errContains)
				}
				return
			}
			if err != nil {
				t.Errorf("GetBaseURI() unexpected error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("GetBaseURI() = %q, want %q", got, tt.want)
			}
		})
	}
}
