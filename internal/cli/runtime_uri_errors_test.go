// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cli

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/pkg/config"
)

// TestGetBaseURI_UnknownServiceWithURIFlag verifies that GetBaseURI rejects an
// unknown service even when --uri is set.
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

// TestGetBaseURI_IncludesClusterContextInError verifies that when the default
// cluster has no URI for a service, GetBaseURI's error names the cluster.
func TestGetBaseURI_IncludesClusterContextInError(t *testing.T) {
	rt := NewRuntime().WithConfig(config.Config{
		DefaultCluster: "demo",
		Clusters:       []config.Cluster{{Name: "demo"}},
	})
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("cluster", "", "")
	cmd.Flags().String("cluster-uri", "", "")
	cmd.Flags().String("uri", "", "")
	_, err := rt.GetBaseURI(cmd, config.ServiceSMD)
	if err == nil || !strings.Contains(err.Error(), "cluster demo") {
		t.Fatalf("GetBaseURI() error = %v, want cluster context", err)
	}
}

// duplicateClusterConfig defines cluster "demo" twice, as a manual edit
// might, and makes it the default cluster.
const duplicateClusterConfig = `default-cluster: demo
clusters:
  - name: demo
    cluster:
      uri: https://first.example
  - name: other
    cluster:
      uri: https://other.example
  - name: demo
    cluster:
      uri: https://second.example
`

// requireDuplicateClusterLookupFails fails t unless resolve returns
// ErrDuplicateCluster when the cluster selected through default-cluster or
// --cluster is defined more than once in the loaded config file, including
// when --cluster-uri and --uri are also passed.
func requireDuplicateClusterLookupFails(t *testing.T, resolve func(rt *Runtime, cmd *cobra.Command) (string, error)) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(duplicateClusterConfig), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	want := config.ErrDuplicateCluster{Source: "file", Path: path, Name: "demo", Indices: []int{0, 2}}

	for _, flags := range []map[string]string{
		nil,
		{"cluster": "demo"},
		{"cluster-uri": "https://flag.example", "uri": "/smd"},
	} {
		rt := NewTestRuntime(nil, nil, nil)
		if err := rt.loadConfigFromFile(path); err != nil {
			t.Fatalf("loadConfigFromFile(): %v", err)
		}
		cmd := newURICmd()
		for name, value := range flags {
			if err := cmd.Flags().Set(name, value); err != nil {
				t.Fatalf("set %s flag: %v", name, err)
			}
		}

		_, err := resolve(rt, cmd)
		var got config.ErrDuplicateCluster
		if !errors.As(err, &got) || !reflect.DeepEqual(got, want) {
			t.Errorf("with flags %v: error = %v, want %+v", flags, err, want)
		}
	}
}

// TestGetBaseURI_DuplicateCluster verifies that GetBaseURI returns
// ErrDuplicateCluster when the cluster selected through default-cluster or
// --cluster is defined more than once in a config file, even when
// --cluster-uri or --uri is also passed.
func TestGetBaseURI_DuplicateCluster(t *testing.T) {
	t.Parallel()
	requireDuplicateClusterLookupFails(t, func(rt *Runtime, cmd *cobra.Command) (string, error) {
		return rt.GetBaseURI(cmd, config.ServiceSMD)
	})
}

// TestGetAPIVersion_DuplicateCluster verifies that GetAPIVersion returns
// ErrDuplicateCluster when the cluster selected through default-cluster or
// --cluster is defined more than once in a config file.
func TestGetAPIVersion_DuplicateCluster(t *testing.T) {
	t.Parallel()
	requireDuplicateClusterLookupFails(t, func(rt *Runtime, cmd *cobra.Command) (string, error) {
		return rt.GetAPIVersion(cmd, config.ServiceBoot)
	})
}
