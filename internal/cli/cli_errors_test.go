// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cli

import (
	"errors"
	"os"
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
