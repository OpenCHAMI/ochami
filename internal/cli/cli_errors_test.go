// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cli

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/pkg/config"
)

// TestSetToken_NoTokenNoCluster verifies that SetToken returns a CodeAuth
// error when neither --token nor --cluster/default-cluster is available to
// resolve a token from.
func TestSetToken_NoTokenNoCluster(t *testing.T) {
	// Save original token/active config and restore after test
	originalToken := Token
	originalConfig := ActiveConfig()
	defer func() {
		Token = originalToken
		SetActiveConfig(originalConfig)
	}()
	SetActiveConfig(config.Config{})

	cmd := &cobra.Command{}
	cmd.Flags().String("token", "", "token flag")
	cmd.Flags().String("cluster", "", "cluster flag")

	err := SetToken(cmd)
	if err == nil {
		t.Fatal("SetToken should have returned an error when no token or cluster is available")
	}
	if ExitCode(err) != CodeAuth {
		t.Errorf("ExitCode = %d, want %d (%s)", ExitCode(err), CodeAuth, CodeName(CodeAuth))
	}
}

// TestHandleToken_UnknownCluster verifies direct callers cannot silently skip
// token handling for a cluster that does not exist.
func TestHandleToken_UnknownCluster(t *testing.T) {
	origCfg := ActiveConfig()
	t.Cleanup(func() { SetActiveConfig(origCfg) })

	SetActiveConfig(config.Config{DefaultCluster: "missing"})
	err := HandleToken(tokenTestCmd())
	if err == nil {
		t.Fatal("HandleToken(): expected error for unknown cluster, got nil")
	}
	if got := ExitCode(err); got != CodeConfig {
		t.Errorf("exit code = %d, want %d (%s)", got, CodeConfig, CodeName(CodeConfig))
	}
}

// TestHandleToken_AuthEnabledMissingToken verifies that an auth-enabled cluster
// requires a token and errors when none is available.
func TestHandleToken_AuthEnabledMissingToken(t *testing.T) {
	origCfg := ActiveConfig()
	origToken := Token
	t.Cleanup(func() {
		SetActiveConfig(origCfg)
		Token = origToken
	})

	SetActiveConfig(config.Config{
		DefaultCluster: "secure",
		Clusters: []config.Cluster{
			{Name: "secure", Cluster: config.ClusterConfig{EnableAuth: true}},
		},
	})
	Token = ""
	// Ensure no stray env var satisfies the token lookup.
	const secureEnv = "SECURE_ACCESS_TOKEN"
	if orig, had := os.LookupEnv(secureEnv); had {
		_ = os.Unsetenv(secureEnv)
		t.Cleanup(func() { _ = os.Setenv(secureEnv, orig) })
	}

	err := HandleToken(tokenTestCmd())
	if err == nil {
		t.Fatal("HandleToken(): expected error for missing token, got nil")
	}
	if got := ExitCode(err); got != CodeAuth {
		t.Errorf("exit code = %d, want %d (%s)", got, CodeAuth, CodeName(CodeAuth))
	}
}

// TestSetToken_MissingEnvVar verifies SetToken errors (CodeAuth) when neither
// --token nor the cluster env var is set.
func TestSetToken_MissingEnvVar(t *testing.T) {
	origToken := Token
	origCfg := ActiveConfig()
	t.Cleanup(func() {
		Token = origToken
		SetActiveConfig(origCfg)
	})

	SetActiveConfig(config.Config{DefaultCluster: "nope"})
	Token = ""
	// Ensure the lookup variable is genuinely absent (t.Setenv can only set,
	// not unset, so explicitly unset it and restore afterward).
	const envVar = "NOPE_ACCESS_TOKEN"
	if orig, had := os.LookupEnv(envVar); had {
		_ = os.Unsetenv(envVar)
		t.Cleanup(func() { _ = os.Setenv(envVar, orig) })
	}

	err := SetToken(tokenTestCmd())
	if err == nil {
		t.Fatal("SetToken(): expected error, got nil")
	}
	if got := ExitCode(err); got != CodeAuth {
		t.Errorf("exit code = %d, want %d (%s)", got, CodeAuth, CodeName(CodeAuth))
	}
	// Sanity: the error is a CodedError, matchable by errors.As.
	var ce *CodedError
	if !errors.As(err, &ce) {
		t.Errorf("error %v is not a *CodedError", err)
	}
}

// useLoadedConfig loads the config file content through the public loader,
// makes it the active configuration, and restores the previous active
// configuration when t finishes.
func useLoadedConfig(t *testing.T, content string) string {
	t.Helper()
	origConfig, origEffective := activeConfig, activeEffective
	t.Cleanup(func() { activeConfig, activeEffective = origConfig, origEffective })

	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := loadConfigFromFile(path); err != nil {
		t.Fatalf("loadConfigFromFile(): %v", err)
	}
	return path
}

// newClusterFlagsCmd returns a command with the cluster-selection flags
// GetBaseURI and GetAPIVersion read, after parsing args.
func newClusterFlagsCmd(t *testing.T, args ...string) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{}
	cmd.Flags().String("cluster", "", "")
	cmd.Flags().String("cluster-uri", "", "")
	cmd.Flags().String("uri", "", "")
	cmd.Flags().String("api-version", "", "")
	if err := cmd.Flags().Parse(args); err != nil {
		t.Fatalf("parse flags: %v", err)
	}
	return cmd
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

// clusterResolver is GetBaseURI or GetAPIVersion bound to a service.
type clusterResolver func(cmd *cobra.Command) (string, error)

func resolveBaseURI(cmd *cobra.Command) (string, error) { return GetBaseURI(cmd, config.ServiceSMD) }

func resolveAPIVersion(cmd *cobra.Command) (string, error) {
	return GetAPIVersion(cmd, config.ServiceBoot)
}

// requireDuplicateClusterLookupFails fails t unless resolve returns
// ErrDuplicateCluster when the cluster selected through default-cluster or
// --cluster is defined more than once in the config file, including when
// --cluster-uri and --uri are also passed.
func requireDuplicateClusterLookupFails(t *testing.T, resolve clusterResolver) {
	t.Helper()
	path := useLoadedConfig(t, duplicateClusterConfig)
	want := config.ErrDuplicateCluster{Source: "file", Path: path, Name: "demo", Indices: []int{0, 2}}

	for _, args := range [][]string{
		nil,
		{"--cluster", "demo"},
		{"--cluster-uri", "https://flag.example", "--uri", "/smd"},
	} {
		_, err := resolve(newClusterFlagsCmd(t, args...))
		var got config.ErrDuplicateCluster
		if !errors.As(err, &got) || !reflect.DeepEqual(got, want) {
			t.Errorf("with flags %v: error = %v, want %+v", args, err, want)
		}
	}
}

// requireUnknownClusterMessages fails t unless resolve reports a cluster
// missing from the config as "default cluster <name> not found" when it came
// from default-cluster and "cluster <name> not found" when it came from
// --cluster.
func requireUnknownClusterMessages(t *testing.T, resolve clusterResolver) {
	t.Helper()
	useLoadedConfig(t, "default-cluster: missing\nclusters:\n  - name: other\n")
	if _, err := resolve(newClusterFlagsCmd(t)); err == nil || err.Error() != "default cluster missing not found" {
		t.Errorf("default-cluster: error = %v, want %q", err, "default cluster missing not found")
	}

	useLoadedConfig(t, "clusters:\n  - name: other\n")
	if _, err := resolve(newClusterFlagsCmd(t, "--cluster", "missing")); err == nil || err.Error() != "cluster missing not found" {
		t.Errorf("--cluster: error = %v, want %q", err, "cluster missing not found")
	}
}

// TestGetBaseURI_DuplicateCluster verifies that GetBaseURI returns
// ErrDuplicateCluster when the cluster selected through default-cluster or
// --cluster is defined more than once in a config file, even when
// --cluster-uri or --uri is also passed.
func TestGetBaseURI_DuplicateCluster(t *testing.T) {
	requireDuplicateClusterLookupFails(t, resolveBaseURI)
}

// TestGetAPIVersion_DuplicateCluster verifies that GetAPIVersion returns
// ErrDuplicateCluster when the cluster selected through default-cluster or
// --cluster is defined more than once in a config file.
func TestGetAPIVersion_DuplicateCluster(t *testing.T) {
	requireDuplicateClusterLookupFails(t, resolveAPIVersion)
}

// TestGetBaseURI_UnknownCluster verifies that GetBaseURI reports a cluster
// missing from the config as "default cluster <name> not found" when it came
// from default-cluster and "cluster <name> not found" when it came from
// --cluster.
func TestGetBaseURI_UnknownCluster(t *testing.T) {
	requireUnknownClusterMessages(t, resolveBaseURI)
}

// TestGetAPIVersion_UnknownCluster verifies that GetAPIVersion reports a
// cluster missing from the config as "default cluster <name> not found" when
// it came from default-cluster and "cluster <name> not found" when it came
// from --cluster.
func TestGetAPIVersion_UnknownCluster(t *testing.T) {
	requireUnknownClusterMessages(t, resolveAPIVersion)
}
