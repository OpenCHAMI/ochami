// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cli

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v4/jwa"
	"github.com/lestrrat-go/jwx/v4/jwt"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/pkg/config"
)

// Helper function to generate a test JWT token.
func generateTestToken(exp time.Time, nbf time.Time, iat time.Time) (string, error) {
	// Generate RSA key for signing
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", err
	}

	// Create token
	token := jwt.New()
	if err := token.Set(jwt.ExpirationKey, exp); err != nil {
		return "", err
	}
	if err := token.Set(jwt.NotBeforeKey, nbf); err != nil {
		return "", err
	}
	if err := token.Set(jwt.IssuedAtKey, iat); err != nil {
		return "", err
	}
	if err := token.Set(jwt.SubjectKey, "test-subject"); err != nil {
		return "", err
	}
	if err := token.Set(jwt.IssuerKey, "test-issuer"); err != nil {
		return "", err
	}

	// Sign token
	signed, err := jwt.Sign(token, jwt.WithKey(jwa.RS256(), privKey))
	if err != nil {
		return "", err
	}

	return string(signed), nil
}

// TestCheckToken_ValidToken verifies that CheckToken accepts a token that is
// currently valid.
func TestCheckToken_ValidToken(t *testing.T) {
	now := time.Now()
	exp := now.Add(1 * time.Hour)
	nbf := now.Add(-1 * time.Hour)
	iat := now.Add(-1 * time.Hour)

	tokenStr, err := generateTestToken(exp, nbf, iat)
	if err != nil {
		t.Fatalf("failed to generate test token: %v", err)
	}

	rt := NewTestRuntime(strings.NewReader(""), io.Discard, io.Discard).WithToken(tokenStr)
	if err := rt.CheckToken(); err != nil {
		t.Fatalf("CheckToken() error = %v", err)
	}

	// We'll just verify the token was generated correctly by parsing it
	// Use WithVerify(false) since we're testing parsing, not signature verification
	parsed, err := jwt.Parse([]byte(tokenStr), jwt.WithVerify(false))
	if err != nil {
		t.Errorf("Token should be valid but parsing failed: %v", err)
	}
	if parsed == nil {
		t.Fatal("parsed token is nil")
	}
	exp, ok := parsed.Expiration()
	if !ok {
		t.Error("Token should have expiration")
	}
	if exp.Before(time.Now()) {
		t.Error("Token should not be expired")
	}
}

// TestCheckToken_ExpiredToken verifies that CheckToken rejects an expired token
// with CodeAuth.
func TestCheckToken_ExpiredToken(t *testing.T) {
	now := time.Now()
	exp := now.Add(-1 * time.Hour) // Expired 1 hour ago
	nbf := now.Add(-2 * time.Hour)
	iat := now.Add(-2 * time.Hour)

	tokenStr, err := generateTestToken(exp, nbf, iat)
	if err != nil {
		t.Fatalf("failed to generate test token: %v", err)
	}

	rt := NewTestRuntime(strings.NewReader(""), io.Discard, io.Discard).WithToken(tokenStr)
	err = rt.CheckToken()
	if err == nil || ExitCode(err) != CodeAuth {
		t.Fatalf("CheckToken() error = %v, want %d (%s)", err, CodeAuth, CodeName(CodeAuth))
	}
	if !errors.Is(err, jwt.TokenExpiredError{}) {
		t.Errorf("Expected TokenExpiredError, got: %v", err)
	}
}

// TestCheckToken_NotYetValid verifies that CheckToken rejects a token whose
// not-before time is in the future with CodeAuth.
func TestCheckToken_NotYetValid(t *testing.T) {
	now := time.Now()
	exp := now.Add(2 * time.Hour)
	nbf := now.Add(1 * time.Hour) // Valid in 1 hour
	iat := now

	tokenStr, err := generateTestToken(exp, nbf, iat)
	if err != nil {
		t.Fatalf("failed to generate test token: %v", err)
	}

	rt := NewTestRuntime(strings.NewReader(""), io.Discard, io.Discard).WithToken(tokenStr)
	err = rt.CheckToken()
	if err == nil || ExitCode(err) != CodeAuth {
		t.Fatalf("CheckToken() error = %v, want %d (%s)", err, CodeAuth, CodeName(CodeAuth))
	}
	if !errors.Is(err, jwt.TokenNotYetValidError{}) {
		t.Errorf("Expected TokenNotYetValidError, got: %v", err)
	}
}

// TestCheckToken_ExpiringSoon verifies that CheckToken accepts a token that
// expires within the warning window.
func TestCheckToken_ExpiringSoon(t *testing.T) {
	now := time.Now()
	exp := now.Add(10 * time.Minute) // Expires in 10 minutes (< 15 min threshold)
	nbf := now.Add(-1 * time.Hour)
	iat := now.Add(-1 * time.Hour)

	tokenStr, err := generateTestToken(exp, nbf, iat)
	if err != nil {
		t.Fatalf("failed to generate test token: %v", err)
	}

	rt := NewTestRuntime(strings.NewReader(""), io.Discard, io.Discard).WithToken(tokenStr)
	if err := rt.CheckToken(); err != nil {
		t.Fatalf("CheckToken() error = %v, want nil for token in warning window", err)
	}
}

// TestCheckToken_EmptyToken verifies that CheckToken rejects an empty token
// with CodeAuth.
func TestCheckToken_EmptyToken(t *testing.T) {
	rt := NewTestRuntime(strings.NewReader(""), io.Discard, io.Discard)
	if err := rt.CheckToken(); err == nil || ExitCode(err) != CodeAuth {
		t.Fatalf("CheckToken() error = %v, want %d (%s)", err, CodeAuth, CodeName(CodeAuth))
	}
}

// TestCheckToken_MalformedToken verifies that CheckToken rejects a token that
// isn't a valid JWT with CodeAuth.
func TestCheckToken_MalformedToken(t *testing.T) {
	malformedToken := "not.a.valid.jwt.token.at.all"
	rt := NewTestRuntime(strings.NewReader(""), io.Discard, io.Discard).WithToken(malformedToken)
	if err := rt.CheckToken(); err == nil || ExitCode(err) != CodeAuth {
		t.Fatalf("CheckToken() error = %v, want %d (%s)", err, CodeAuth, CodeName(CodeAuth))
	}
}

// TestSetTokenFromFlag verifies that SetTokenFromFlag uses the value of
// --token.
func TestSetTokenFromFlag(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("token", "", "token flag")
	if err := cmd.Flags().Set("token", "test-token-from-flag"); err != nil {
		t.Fatalf("Failed to set flag: %v", err)
	}

	rt := NewTestRuntime(strings.NewReader(""), io.Discard, io.Discard)
	if err := rt.SetTokenFromFlag(cmd); err != nil {
		t.Fatalf("SetTokenFromFlag() error = %v", err)
	}

	if rt.Token != "test-token-from-flag" {
		t.Errorf("Token = %q, want %q", rt.Token, "test-token-from-flag")
	}
}

// TestSetTokenFromEnv_ClusterVariable verifies that SetTokenFromEnv reads the
// token from the default cluster's <CLUSTER>_ACCESS_TOKEN variable, with spaces
// in the cluster name turned into underscores.
func TestSetTokenFromEnv_ClusterVariable(t *testing.T) {
	rt := NewTestRuntime(strings.NewReader(""), io.Discard, io.Discard).
		WithConfig(config.Config{DefaultCluster: "test cluster"}).
		WithEnvironment(EnvironmentFunc(func(key string) (string, bool) {
			return "environment-token", key == "TEST_CLUSTER_ACCESS_TOKEN"
		}))
	cmd := &cobra.Command{}
	if err := rt.SetTokenFromEnv(cmd); err != nil {
		t.Fatalf("SetTokenFromEnv() error = %v", err)
	}
	if rt.Token != "environment-token" {
		t.Fatalf("Token = %q, want environment-token", rt.Token)
	}
}

// tokenTestCmd returns a command with the flags the token helpers inspect.
func tokenTestCmd() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Flags().String("token", "", "")
	cmd.Flags().String("cluster", "", "")
	cmd.Flags().Bool("no-token", false, "")
	cmd.Flags().Bool("show-token", false, "")
	return cmd
}

// TestHandleToken_NoTokenFlag verifies that with --no-token, HandleToken
// succeeds even with no config or token.
func TestHandleToken_NoTokenFlag(t *testing.T) {
	rt := NewTestRuntime(nil, &bytes.Buffer{}, &bytes.Buffer{})
	cmd := tokenTestCmd()
	cmd.SetContext(ContextWithRuntime(context.Background(), rt))
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
	rt := NewTestRuntime(nil, &bytes.Buffer{}, &bytes.Buffer{})
	rt.Config = config.Config{
		DefaultCluster: "foo",
		Clusters: []config.Cluster{
			{Name: "foo", Cluster: config.ClusterConfig{EnableAuth: false}},
		},
	}
	cmd := tokenTestCmd()
	cmd.SetContext(ContextWithRuntime(context.Background(), rt))

	if err := rt.HandleToken(cmd); err != nil {
		t.Fatalf("HandleToken(): unexpected error for auth-disabled cluster: %v", err)
	}
}

// TestHandleToken_AuthEnabledWithEnvToken verifies that an auth-enabled cluster
// reads its token from the <CLUSTER>_ACCESS_TOKEN environment variable.
func TestHandleToken_AuthEnabledWithEnvToken(t *testing.T) {
	rt := NewTestRuntime(nil, &bytes.Buffer{}, &bytes.Buffer{})
	now := time.Now()
	valid, err := generateTestToken(now.Add(time.Hour), now.Add(-time.Hour), now.Add(-time.Hour))
	if err != nil {
		t.Fatalf("generate valid token: %v", err)
	}

	rt.Config = config.Config{
		DefaultCluster: "my-cluster",
		Clusters: []config.Cluster{
			{Name: "my-cluster", Cluster: config.ClusterConfig{EnableAuth: true}},
		},
	}
	rt.Token = ""
	rt.WithEnvironment(EnvironmentFunc(func(key string) (string, bool) {
		return valid, key == "MY_CLUSTER_ACCESS_TOKEN"
	}))
	cmd := tokenTestCmd()
	cmd.SetContext(ContextWithRuntime(context.Background(), rt))

	if err := rt.HandleToken(cmd); err != nil {
		t.Fatalf("HandleToken(): unexpected error: %v", err)
	}
	if rt.Token != valid {
		t.Errorf("Token not populated from environment variable, got %q, want %q", rt.Token, valid)
	}
}

// TestPrintUsageHandleError verifies that PrintUsageHandleError and the
// PrintUsage adapter print usage for a command, and that PrintUsageHandleError
// returns a failure to print usage as CodeGeneric.
func TestPrintUsageHandleError(t *testing.T) {
	cmd := &cobra.Command{Use: "demo"}
	cmd.SetOut(&bytes.Buffer{})
	if err := PrintUsageHandleError(cmd); err != nil {
		t.Errorf("PrintUsageHandleError = %v, want nil", err)
	}
	// PrintUsage adapter should behave the same.
	if err := PrintUsage(cmd, nil); err != nil {
		t.Errorf("PrintUsage = %v, want nil", err)
	}

	wantErr := errors.New("usage output failed")
	cmd.SetUsageFunc(func(*cobra.Command) error { return wantErr })
	err := PrintUsageHandleError(cmd)
	if err == nil || ExitCode(err) != CodeGeneric || !errors.Is(err, wantErr) {
		t.Errorf("PrintUsageHandleError failure = %v, want wrapped %s error", err, CodeName(CodeGeneric))
	}
}

// TestLogHelpHint verifies the "see '<cmd> --help'" hint is logged.
func TestLogHelpHint(t *testing.T) {
	var stderr bytes.Buffer
	rt := NewTestRuntime(strings.NewReader(""), io.Discard, &stderr)
	cmd := &cobra.Command{Use: "demo"}
	cmd.SetContext(ContextWithRuntime(context.Background(), rt))

	LogHelpHint(cmd)
	if !strings.Contains(stderr.String(), "see 'demo --help' for long command help") {
		t.Errorf("stderr = %q, want the help hint for demo", stderr.String())
	}
}

// TestCheckToken_ParsingAndTimeClaims verifies that local validation accepts
// optional time claims and invalid signatures but rejects duplicate claims.
func TestCheckToken_ParsingAndTimeClaims(t *testing.T) {
	t.Parallel()
	now := time.Now().Unix()
	for _, tc := range []struct {
		name       string
		payload    string
		wantReject bool
		wantCause  error
		wantWarn   bool
	}{
		{
			name:    "missing_optional_time_claims",
			payload: `{"sub":"test-subject"}`,
		},
		{
			name:    "invalid_signature_checked_by_service",
			payload: fmt.Sprintf(`{"exp":%d,"nbf":%d,"iat":%d}`, now+3600, now-60, now-60),
		},
		{
			name:       "future_issued_at",
			payload:    fmt.Sprintf(`{"iat":%d}`, now+3600),
			wantReject: true,
			wantCause:  jwt.InvalidIssuedAtError{},
		},
		{
			name:       "duplicate_expiration",
			payload:    fmt.Sprintf(`{"exp":%d,"exp":%d}`, now+7200, now+3600),
			wantReject: true,
		},
		{
			name:       "duplicate_custom_claim",
			payload:    `{"scope":"admin","scope":"audit"}`,
			wantReject: true,
		},
		{
			name:     "expiry_warning",
			payload:  fmt.Sprintf(`{"exp":%d}`, now+600),
			wantWarn: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Construct raw payloads so duplicate names survive serialization.
			// The signature is deliberately invalid: only services verify it.
			token := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`)) + "." +
				base64.RawURLEncoding.EncodeToString([]byte(tc.payload)) + ".c2lnbmF0dXJl"
			var logs bytes.Buffer
			rt := NewTestRuntime(strings.NewReader(""), io.Discard, io.Discard).
				WithToken(token).WithLogger(zerolog.New(&logs))
			err := rt.CheckToken()
			if tc.wantReject {
				if err == nil || ExitCode(err) != CodeAuth {
					t.Fatalf("CheckToken() = %v, want authentication error", err)
				}
				if tc.wantCause != nil && !errors.Is(err, tc.wantCause) {
					t.Errorf("CheckToken() = %v, want cause %T", err, tc.wantCause)
				}
			} else if err != nil {
				t.Fatalf("CheckToken() = %v, want success", err)
			}
			if rt.Token != token {
				t.Error("CheckToken modified the bearer token")
			}
			if warned := strings.Contains(logs.String(), "until token expires"); warned != tc.wantWarn {
				t.Errorf("expiry warning = %t, want %t", warned, tc.wantWarn)
			}
		})
	}
}

// TestSetTokenFromEnv_PerRuntimeEnvironment verifies that each runtime
// resolves its cluster's token from its own environment.
func TestSetTokenFromEnv_PerRuntimeEnvironment(t *testing.T) {
	t.Parallel()

	lookup := func(values map[string]string) EnvironmentFunc {
		return func(key string) (string, bool) {
			value, ok := values[key]
			return value, ok
		}
	}
	rtA := NewTestRuntime(nil, io.Discard, io.Discard).WithEnvironment(lookup(map[string]string{"CLUSTER_A_ACCESS_TOKEN": "token-a"}))
	rtB := NewTestRuntime(nil, io.Discard, io.Discard).WithEnvironment(lookup(map[string]string{"CLUSTER_B_ACCESS_TOKEN": "token-b"}))
	rtA.Config.DefaultCluster = "cluster-a"
	rtB.Config.DefaultCluster = "cluster-b"

	if err := rtA.SetTokenFromEnv(tokenTestCmd()); err != nil {
		t.Fatalf("runtime A SetTokenFromEnv() error = %v", err)
	}
	if err := rtB.SetTokenFromEnv(tokenTestCmd()); err != nil {
		t.Fatalf("runtime B SetTokenFromEnv() error = %v", err)
	}
	if rtA.Token != "token-a" || rtB.Token != "token-b" {
		t.Fatalf("tokens = %q, %q; want token-a, token-b", rtA.Token, rtB.Token)
	}
}

// TestHandleToken_FlagPrecedesRuntimeEnvironment verifies that HandleToken uses
// --token even when the cluster's token environment variable is also set.
func TestHandleToken_FlagPrecedesRuntimeEnvironment(t *testing.T) {
	t.Parallel()

	now := time.Now()
	flagToken, err := generateTestToken(now.Add(time.Hour), now.Add(-time.Hour), now.Add(-time.Hour))
	if err != nil {
		t.Fatalf("generate flag token: %v", err)
	}
	envToken, err := generateTestToken(now.Add(2*time.Hour), now.Add(-time.Hour), now.Add(-time.Hour))
	if err != nil {
		t.Fatalf("generate environment token: %v", err)
	}
	rt := NewTestRuntime(nil, io.Discard, io.Discard).WithEnvironment(EnvironmentFunc(func(key string) (string, bool) {
		return envToken, key == "MY_CLUSTER_ACCESS_TOKEN"
	}))
	rt.Config = config.Config{
		DefaultCluster: "my-cluster",
		Clusters: []config.Cluster{{
			Name:    "my-cluster",
			Cluster: config.ClusterConfig{EnableAuth: true},
		}},
	}
	cmd := tokenTestCmd()
	if err := cmd.Flags().Set("token", flagToken); err != nil {
		t.Fatalf("set token flag: %v", err)
	}
	if err := rt.HandleToken(cmd); err != nil {
		t.Fatalf("HandleToken() error = %v", err)
	}
	if rt.Token != flagToken {
		t.Fatal("HandleToken() did not preserve explicit token precedence")
	}
}
