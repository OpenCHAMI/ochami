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

	"github.com/openchami/ochami/pkg/client"
	"github.com/openchami/ochami/pkg/config"
)

type zeroWriter struct{}

func (zeroWriter) Write([]byte) (int, error) { return 0, nil }

type oversizedWriter struct{}

func (oversizedWriter) Write(p []byte) (int, error) { return len(p) + 1, nil }

type errorWriter struct{ err error }

func (w errorWriter) Write([]byte) (int, error) { return 0, w.err }

type errorReader struct{ err error }

func (r errorReader) Read([]byte) (int, error) { return 0, r.err }
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

func TestIOStreamErrors(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("stream failure")
	t.Run("prompt writer", func(t *testing.T) {
		ios := NewIOStreams(strings.NewReader("y\n"), io.Discard, errorWriter{sentinel})
		if _, err := ios.LoopYesNo("Proceed?"); !errors.Is(err, sentinel) {
			t.Fatalf("LoopYesNo() error = %v, want stream failure", err)
		}
	})
	t.Run("input reader", func(t *testing.T) {
		ios := NewIOStreams(errorReader{sentinel}, io.Discard, io.Discard)
		if _, err := ios.LoopYesNo("Proceed?"); !errors.Is(err, sentinel) {
			t.Fatalf("LoopYesNo() error = %v, want stream failure", err)
		}
	})
	t.Run("zero write", func(t *testing.T) {
		if err := WriteOutput(zeroWriter{}, []byte("data")); !errors.Is(err, io.ErrShortWrite) {
			t.Fatalf("WriteOutput() error = %v, want io.ErrShortWrite", err)
		}
	})
	t.Run("oversized write", func(t *testing.T) {
		if err := WriteOutput(oversizedWriter{}, []byte("data")); !errors.Is(err, io.ErrShortWrite) {
			t.Fatalf("WriteOutput() error = %v, want io.ErrShortWrite", err)
		}
	})
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
		t.Fatalf("CheckToken() error = %v, want CodeAuth", err)
	}
	if !errors.Is(err, jwt.TokenExpiredError()) {
		t.Errorf("Expected TokenExpiredError, got: %v", err)
	}
}

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
		t.Fatalf("CheckToken() error = %v, want CodeAuth", err)
	}
	if !errors.Is(err, jwt.TokenNotYetValidError()) {
		t.Errorf("Expected TokenNotYetValidError, got: %v", err)
	}
}

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

func TestCheckToken_EmptyToken(t *testing.T) {
	rt := NewTestRuntime(strings.NewReader(""), io.Discard, io.Discard)
	if err := rt.CheckToken(); err == nil || ExitCode(err) != CodeAuth {
		t.Fatalf("CheckToken() error = %v, want CodeAuth", err)
	}
}

func TestCheckToken_MalformedToken(t *testing.T) {
	malformedToken := "not.a.valid.jwt.token.at.all"
	rt := NewTestRuntime(strings.NewReader(""), io.Discard, io.Discard).WithToken(malformedToken)
	if err := rt.CheckToken(); err == nil || ExitCode(err) != CodeAuth {
		t.Fatalf("CheckToken() error = %v, want CodeAuth", err)
	}
}

func TestSetToken_FromFlag(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("token", "", "token flag")
	if err := cmd.Flags().Set("token", "test-token-from-flag"); err != nil {
		t.Fatalf("Failed to set flag: %v", err)
	}

	rt := NewTestRuntime(strings.NewReader(""), io.Discard, io.Discard)
	if err := rt.SetTokenFromFlag(cmd); err != nil {
		t.Fatalf("SetToken returned unexpected error: %v", err)
	}

	if rt.Token != "test-token-from-flag" {
		t.Errorf("Token = %q, want %q", rt.Token, "test-token-from-flag")
	}
}

func TestSetToken_FromEnvironment(t *testing.T) {
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

// TestSetToken_NoTokenNoCluster verifies that SetTokenFromEnv returns a
// CodeAuth error when neither --token nor --cluster/default-cluster is
// available to resolve a token from.
func TestSetToken_NoTokenNoCluster(t *testing.T) {
	rt := NewTestRuntime(strings.NewReader(""), io.Discard, io.Discard)
	cmd := &cobra.Command{}
	cmd.Flags().String("cluster", "", "cluster flag")

	err := rt.SetTokenFromEnv(cmd)
	if err == nil {
		t.Fatal("SetTokenFromEnv should have returned an error when no token or cluster is available")
	}
	if ExitCode(err) != CodeAuth {
		t.Errorf("ExitCode = %d, want %d (CodeAuth)", ExitCode(err), CodeAuth)
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
			rt := NewTestRuntime(nil, &bytes.Buffer{}, &bytes.Buffer{})
			rt.Token = tt.token
			cmd := tokenTestCmd()
			cmd.SetContext(ContextWithRuntime(context.Background(), rt))
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
		Clusters: []config.ConfigCluster{
			{Name: "foo", Cluster: config.ConfigClusterConfig{EnableAuth: false}},
		},
	}
	cmd := tokenTestCmd()
	cmd.SetContext(ContextWithRuntime(context.Background(), rt))

	if err := rt.HandleToken(cmd); err != nil {
		t.Fatalf("HandleToken(): unexpected error for auth-disabled cluster: %v", err)
	}
}

// TestHandleToken_UnknownCluster verifies direct callers cannot silently skip
// token handling for a cluster that does not exist.
func TestHandleToken_UnknownCluster(t *testing.T) {
	rt := NewTestRuntime(nil, &bytes.Buffer{}, &bytes.Buffer{})
	rt.Config = config.Config{DefaultCluster: "missing"}
	cmd := tokenTestCmd()
	cmd.SetContext(ContextWithRuntime(context.Background(), rt))
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
	rt := NewTestRuntime(nil, &bytes.Buffer{}, &bytes.Buffer{})
	rt.Config = config.Config{
		DefaultCluster: "secure",
		Clusters: []config.ConfigCluster{
			{Name: "secure", Cluster: config.ConfigClusterConfig{EnableAuth: true}},
		},
	}
	rt.Token = ""
	rt.WithEnvironment(EnvironmentFunc(func(string) (string, bool) { return "", false }))
	cmd := tokenTestCmd()
	cmd.SetContext(ContextWithRuntime(context.Background(), rt))

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

// TestSetToken_MissingEnvVar verifies SetToken errors (CodeAuth) when neither
// --token nor the cluster env var is set.
func TestSetToken_MissingEnvVar(t *testing.T) {
	rt := NewTestRuntime(nil, &bytes.Buffer{}, &bytes.Buffer{})
	rt.Config = config.Config{DefaultCluster: "nope"}
	rt.Token = ""
	rt.WithEnvironment(EnvironmentFunc(func(string) (string, bool) { return "", false }))
	cmd := tokenTestCmd()
	cmd.SetContext(ContextWithRuntime(context.Background(), rt))

	err := rt.SetTokenFromEnv(cmd)
	if err == nil {
		t.Fatal("SetToken(): expected error, got nil")
	}
	if got := ExitCode(err); got != CodeAuth {
		t.Errorf("exit code = %d, want CodeAuth (%d)", got, CodeAuth)
	}
	var ce *CodedError
	if !errors.As(err, &ce) {
		t.Errorf("error %v is not a *CodedError", err)
	}
}

// TestPrintUsageHandleError verifies usage printing succeeds for a normal
// command.
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
		t.Errorf("PrintUsageHandleError failure = %v, want wrapped CodeGeneric error", err)
	}
}

// TestLogHelpHint exercises the help-hint emitters (which just log).
func TestLogHelpHint(t *testing.T) {
	cmd := &cobra.Command{Use: "demo"}
	LogHelpHint(cmd)
}

// TestPatchMethodValue verifies client.PatchMethod's pflag.Value implementation
// (Set/Type) accepts the documented values and rejects everything else.
func TestPatchMethodValue(t *testing.T) {
	var method client.PatchMethod
	for _, value := range []string{"rfc6902", "rfc7386", "keyval"} {
		if err := method.Set(value); err != nil {
			t.Errorf("Set(%q): %v", value, err)
		}
	}
	if err := method.Set("invalid"); err == nil {
		t.Error("Set(invalid) returned nil")
	}
	if method.Type() != "PatchMethod" {
		t.Errorf("Type = %q", method.Type())
	}
}
