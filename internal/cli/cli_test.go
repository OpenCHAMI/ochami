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
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/pkg/config"
)

// TestIOStreams_LoopYesNo verifies that LoopYesNo returns the user's yes or no
// answer and prompts again after an invalid one.
func TestIOStreams_LoopYesNo(t *testing.T) {
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
			ios := NewIOStreams(inBuf, io.Discard, errBuf)

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

// Helper function to generate a test JWT token
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
	if !errors.Is(err, jwt.TokenExpiredError()) {
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
	if !errors.Is(err, jwt.TokenNotYetValidError()) {
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

// TestHandleToken_NoTokenFlag verifies that --no-token short-circuits token
// handling entirely (no error even with no config).
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
