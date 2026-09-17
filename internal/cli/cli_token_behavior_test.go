// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cli

// cli_token_behavior_test.go exercises the token-handling functions by actually
// invoking them (CheckToken and HandleToken now return errors rather than
// calling os.Exit), asserting the CodedError exit codes they resolve to.

import (
	"bytes"
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/pkg/config"
)

// tokenTestCmd returns a command with the flags the token helpers inspect.
func tokenTestCmd() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Flags().String("token", "", "")
	cmd.Flags().String("cluster", "", "")
	cmd.Flags().Bool("no-token", false, "")
	cmd.Flags().Bool("show-token", false, "")
	return cmd
}

func TestCheckToken_Behavior(t *testing.T) {
	now := time.Now()

	valid, err := generateTestToken(now.Add(time.Hour), now.Add(-time.Hour), now.Add(-time.Hour))
	if err != nil {
		t.Fatalf("generate valid token: %v", err)
	}
	expired, err := generateTestToken(now.Add(-time.Hour), now.Add(-2*time.Hour), now.Add(-2*time.Hour))
	if err != nil {
		t.Fatalf("generate expired token: %v", err)
	}
	notYet, err := generateTestToken(now.Add(2*time.Hour), now.Add(time.Hour), now)
	if err != nil {
		t.Fatalf("generate not-yet-valid token: %v", err)
	}

	tests := []struct {
		name     string
		token    string
		wantErr  bool
		wantCode int
	}{
		{"valid", valid, false, CodeSuccess},
		{"empty", "", true, CodeAuth},
		{"malformed", "not-a-jwt", true, CodeAuth},
		{"expired", expired, true, CodeAuth},
		{"not yet valid", notYet, true, CodeAuth},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use runtime-based approach instead of global Token
			rt := NewTestRuntime(nil, &bytes.Buffer{}, &bytes.Buffer{})
			rt.Token = tt.token
			cmd := tokenTestCmd()
			cmd.SetContext(rt.WithContext(context.Background()))
			err := rt.CheckToken()
			if tt.wantErr {
				if err == nil {
					t.Fatal("CheckToken(): expected error, got nil")
				}
				if got := ExitCode(err); got != tt.wantCode {
					t.Errorf("exit code = %d, want %d", got, tt.wantCode)
				}
			} else if err != nil {
				t.Fatalf("CheckToken(): unexpected error: %v", err)
			}
		})
	}
}

// TestHandleToken_NoTokenFlag verifies that --no-token short-circuits token
// handling entirely (no error even with no config).
func TestHandleToken_NoTokenFlag(t *testing.T) {
	// Use runtime-based approach
	rt := NewTestRuntime(nil, &bytes.Buffer{}, &bytes.Buffer{})
	cmd := tokenTestCmd()
	cmd.SetContext(rt.WithContext(context.Background()))
	if err := cmd.Flags().Set("no-token", "true"); err != nil {
		t.Fatalf("set no-token: %v", err)
	}
	if err := rt.HandleToken(cmd); err != nil {
		t.Fatalf("HandleToken(): unexpected error with --no-token: %v", err)
	}
}

// TestHandleToken_AuthDisabledCluster verifies that a cluster with auth disabled
// does not require a token.
func TestHandleToken_AuthDisabledCluster(t *testing.T) {
	// Use runtime-based approach instead of global SetActiveConfig
	rt := NewTestRuntime(nil, &bytes.Buffer{}, &bytes.Buffer{})
	rt.Config = config.Config{
		DefaultCluster: "foo",
		Clusters: []config.ConfigCluster{
			{Name: "foo", Cluster: config.ConfigClusterConfig{EnableAuth: false}},
		},
	}
	cmd := tokenTestCmd()
	cmd.SetContext(rt.WithContext(context.Background()))

	if err := rt.HandleToken(cmd); err != nil {
		t.Fatalf("HandleToken(): unexpected error for auth-disabled cluster: %v", err)
	}
}

// TestHandleToken_UnknownCluster verifies direct callers cannot silently skip
// token handling for a cluster that does not exist.
func TestHandleToken_UnknownCluster(t *testing.T) {
	// Use runtime-based approach instead of global SetActiveConfig
	rt := NewTestRuntime(nil, &bytes.Buffer{}, &bytes.Buffer{})
	rt.Config = config.Config{DefaultCluster: "missing"}
	cmd := tokenTestCmd()
	cmd.SetContext(rt.WithContext(context.Background()))
	err := rt.HandleToken(cmd)
	if err == nil {
		t.Fatal("HandleToken(): expected error for unknown cluster, got nil")
	}
	if got := ExitCode(err); got != CodeConfig {
		t.Errorf("exit code = %d, want CodeConfig (%d)", got, CodeConfig)
	}
}

// TestHandleToken_AuthEnabledMissingToken verifies that an auth-enabled cluster
// requires a token and errors when none is available.
func TestHandleToken_AuthEnabledMissingToken(t *testing.T) {
	// Use runtime-based approach instead of global SetActiveConfig and Token
	rt := NewTestRuntime(nil, &bytes.Buffer{}, &bytes.Buffer{})
	rt.Config = config.Config{
		DefaultCluster: "secure",
		Clusters: []config.ConfigCluster{
			{Name: "secure", Cluster: config.ConfigClusterConfig{EnableAuth: true}},
		},
	}
	rt.Token = ""
	cmd := tokenTestCmd()
	cmd.SetContext(rt.WithContext(context.Background()))

	// Ensure no stray env var satisfies the token lookup.
	const secureEnv = "SECURE_ACCESS_TOKEN"
	if orig, had := os.LookupEnv(secureEnv); had {
		_ = os.Unsetenv(secureEnv)
		t.Cleanup(func() { _ = os.Setenv(secureEnv, orig) })
	}

	err := rt.HandleToken(cmd)
	if err == nil {
		t.Fatal("HandleToken(): expected error for missing token, got nil")
	}
	if got := ExitCode(err); got != CodeAuth {
		t.Errorf("exit code = %d, want CodeAuth (%d)", got, CodeAuth)
	}
}

// TestHandleToken_AuthEnabledWithEnvToken verifies that an auth-enabled cluster
// reads its token from the <CLUSTER>_ACCESS_TOKEN environment variable.
func TestHandleToken_AuthEnabledWithEnvToken(t *testing.T) {
	// Use runtime-based approach instead of global SetActiveConfig and Token
	rt := NewTestRuntime(nil, &bytes.Buffer{}, &bytes.Buffer{})
	now := time.Now()
	valid, err := generateTestToken(now.Add(time.Hour), now.Add(-time.Hour), now.Add(-time.Hour))
	if err != nil {
		t.Fatalf("generate valid token: %v", err)
	}

	rt.Config = config.Config{
		DefaultCluster: "my-cluster",
		Clusters: []config.ConfigCluster{
			{Name: "my-cluster", Cluster: config.ConfigClusterConfig{EnableAuth: true}},
		},
	}
	rt.Token = ""
	cmd := tokenTestCmd()
	cmd.SetContext(rt.WithContext(context.Background()))
	// Dashes in the cluster name become underscores, uppercased.
	t.Setenv("MY_CLUSTER_ACCESS_TOKEN", valid)

	if err := rt.HandleToken(cmd); err != nil {
		t.Fatalf("HandleToken(): unexpected error: %v", err)
	}
	// Check that the runtime token was populated from environment variable
	// Note: HandleToken with runtime should have set rt.Token
	if rt.Token != valid {
		t.Errorf("Token not populated from environment variable, got %q, want %q", rt.Token, valid)
	}
}

// TestSetToken_MissingEnvVar verifies SetToken errors (CodeAuth) when neither
// --token nor the cluster env var is set.
func TestSetToken_MissingEnvVar(t *testing.T) {
	// Use runtime-based approach instead of global Token and SetActiveConfig
	rt := NewTestRuntime(nil, &bytes.Buffer{}, &bytes.Buffer{})
	rt.Config = config.Config{DefaultCluster: "nope"}
	rt.Token = ""
	cmd := tokenTestCmd()
	cmd.SetContext(rt.WithContext(context.Background()))
	// Ensure the lookup variable is genuinely absent (t.Setenv can only set,
	// not unset, so explicitly unset it and restore afterward).
	const envVar = "NOPE_ACCESS_TOKEN"
	if orig, had := os.LookupEnv(envVar); had {
		_ = os.Unsetenv(envVar)
		t.Cleanup(func() { _ = os.Setenv(envVar, orig) })
	}

	err := rt.SetTokenFromEnv(cmd)
	if err == nil {
		t.Fatal("SetToken(): expected error, got nil")
	}
	if got := ExitCode(err); got != CodeAuth {
		t.Errorf("exit code = %d, want CodeAuth (%d)", got, CodeAuth)
	}
	// Sanity: the error is a CodedError, matchable by errors.As.
	var ce *CodedError
	if !errors.As(err, &ce) {
		t.Errorf("error %v is not a *CodedError", err)
	}
}
