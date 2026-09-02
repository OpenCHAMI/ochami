// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cli

// cli_token_behavior_test.go exercises the token-handling functions by actually
// invoking them (CheckToken and HandleToken now return errors rather than
// calling os.Exit), asserting the CodedError exit codes they resolve to.

import (
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

	origToken := Token
	t.Cleanup(func() { Token = origToken })

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Token = tt.token
			err := CheckToken(tokenTestCmd())
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
	cmd := tokenTestCmd()
	if err := cmd.Flags().Set("no-token", "true"); err != nil {
		t.Fatalf("set no-token: %v", err)
	}
	if err := HandleToken(cmd); err != nil {
		t.Fatalf("HandleToken(): unexpected error with --no-token: %v", err)
	}
}

// TestHandleToken_AuthDisabledCluster verifies that a cluster with auth disabled
// does not require a token.
func TestHandleToken_AuthDisabledCluster(t *testing.T) {
	origCfg := ActiveConfig()
	t.Cleanup(func() { SetActiveConfig(origCfg) })

	SetActiveConfig(config.Config{
		DefaultCluster: "foo",
		Clusters: []config.ConfigCluster{
			{Name: "foo", Cluster: config.ConfigClusterConfig{EnableAuth: false}},
		},
	})

	if err := HandleToken(tokenTestCmd()); err != nil {
		t.Fatalf("HandleToken(): unexpected error for auth-disabled cluster: %v", err)
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
		Clusters: []config.ConfigCluster{
			{Name: "secure", Cluster: config.ConfigClusterConfig{EnableAuth: true}},
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
		t.Errorf("exit code = %d, want CodeAuth (%d)", got, CodeAuth)
	}
}

// TestHandleToken_AuthEnabledWithEnvToken verifies that an auth-enabled cluster
// reads its token from the <CLUSTER>_ACCESS_TOKEN environment variable.
func TestHandleToken_AuthEnabledWithEnvToken(t *testing.T) {
	origCfg := ActiveConfig()
	origToken := Token
	t.Cleanup(func() {
		SetActiveConfig(origCfg)
		Token = origToken
	})

	now := time.Now()
	valid, err := generateTestToken(now.Add(time.Hour), now.Add(-time.Hour), now.Add(-time.Hour))
	if err != nil {
		t.Fatalf("generate valid token: %v", err)
	}

	SetActiveConfig(config.Config{
		DefaultCluster: "my-cluster",
		Clusters: []config.ConfigCluster{
			{Name: "my-cluster", Cluster: config.ConfigClusterConfig{EnableAuth: true}},
		},
	})
	Token = ""
	// Dashes in the cluster name become underscores, uppercased.
	t.Setenv("MY_CLUSTER_ACCESS_TOKEN", valid)

	if err := HandleToken(tokenTestCmd()); err != nil {
		t.Fatalf("HandleToken(): unexpected error: %v", err)
	}
	if Token != valid {
		t.Errorf("Token not populated from environment variable")
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
		t.Errorf("exit code = %d, want CodeAuth (%d)", got, CodeAuth)
	}
	// Sanity: the error is a CodedError, matchable by errors.As.
	var ce *CodedError
	if !errors.As(err, &ce) {
		t.Errorf("error %v is not a *CodedError", err)
	}
}
