// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cli

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/pkg/config"
	"github.com/openchami/ochami/pkg/format"
)

// TestIOStream_AskToCreate verifies that AskToCreate rejects an empty path,
// reports an existing file with FileExistsError without prompting, and prompts
// before creating a missing file, creating it only if the user agrees.
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

// TestIOStream_LoopYesNo verifies that LoopYesNo returns the user's yes or no
// answer and prompts again after an invalid one.
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

// Test_CreateIfNotExists verifies that CreateIfNotExists rejects an empty path
// and succeeds for both a missing and an existing file.
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

	// Save original token and restore after test
	originalToken := Token
	defer func() { Token = originalToken }()
	Token = tokenStr

	if err := CheckToken(&cobra.Command{}); err != nil {
		t.Errorf("CheckToken returned unexpected error: %v", err)
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

	// Verify the token is actually expired by trying to parse it
	// Use WithVerify(false) since we're testing expiration validation, not signature
	_, err = jwt.Parse([]byte(tokenStr), jwt.WithVerify(false))
	if err == nil {
		t.Error("Expected token to be expired but parsing succeeded")
	}
	if !errors.Is(err, jwt.TokenExpiredError()) {
		t.Errorf("Expected TokenExpiredError, got: %v", err)
	}

	// Save original token and restore after test
	originalToken := Token
	defer func() { Token = originalToken }()
	Token = tokenStr

	ctErr := CheckToken(&cobra.Command{})
	if ctErr == nil {
		t.Fatal("CheckToken should have returned an error for an expired token")
	}
	if ExitCode(ctErr) != CodeAuth {
		t.Errorf("ExitCode = %d, want %d (%s)", ExitCode(ctErr), CodeAuth, CodeName(CodeAuth))
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

	// Verify the token is not yet valid
	// Use WithVerify(false) since we're testing nbf validation, not signature
	_, err = jwt.Parse([]byte(tokenStr), jwt.WithVerify(false))
	if err == nil {
		t.Error("Expected token to not be valid yet but parsing succeeded")
	}
	if !errors.Is(err, jwt.TokenNotYetValidError()) {
		t.Errorf("Expected TokenNotYetValidError, got: %v", err)
	}

	// Save original token and restore after test
	originalToken := Token
	defer func() { Token = originalToken }()
	Token = tokenStr

	ctErr := CheckToken(&cobra.Command{})
	if ctErr == nil {
		t.Fatal("CheckToken should have returned an error for a not-yet-valid token")
	}
	if ExitCode(ctErr) != CodeAuth {
		t.Errorf("ExitCode = %d, want %d (%s)", ExitCode(ctErr), CodeAuth, CodeName(CodeAuth))
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

	// Verify the token is valid but expiring soon
	// Use WithVerify(false) since we're testing time validation, not signature
	parsed, err := jwt.Parse([]byte(tokenStr), jwt.WithVerify(false))
	if err != nil {
		t.Errorf("Token should be valid: %v", err)
	}

	exp, ok := parsed.Expiration()
	if !ok {
		t.Error("Token should have expiration")
	}
	timeUntilExpiry := exp.Sub(time.Now())
	if timeUntilExpiry.Minutes() > 15 {
		t.Errorf("Token should expire in less than 15 minutes, got: %v", timeUntilExpiry)
	}

	// Save original token and restore after test
	originalToken := Token
	defer func() { Token = originalToken }()
	Token = tokenStr

	// A token expiring soon is still valid, so CheckToken should only warn,
	// not return an error.
	if err := CheckToken(&cobra.Command{}); err != nil {
		t.Errorf("CheckToken returned unexpected error for a token expiring soon: %v", err)
	}
}

// TestCheckToken_EmptyToken verifies that CheckToken rejects an empty token
// with CodeAuth.
func TestCheckToken_EmptyToken(t *testing.T) {
	// Save original token and restore after test
	originalToken := Token
	defer func() { Token = originalToken }()

	Token = ""

	err := CheckToken(&cobra.Command{})
	if err == nil {
		t.Fatal("CheckToken should have returned an error for an empty token")
	}
	if ExitCode(err) != CodeAuth {
		t.Errorf("ExitCode = %d, want %d (%s)", ExitCode(err), CodeAuth, CodeName(CodeAuth))
	}
}

// TestCheckToken_MalformedToken verifies that CheckToken rejects a token that
// isn't a valid JWT with CodeAuth.
func TestCheckToken_MalformedToken(t *testing.T) {
	malformedToken := "not.a.valid.jwt.token.at.all"

	// Try to parse it and verify it fails
	// Use WithVerify(false) to test parsing failure, not signature failure
	_, err := jwt.Parse([]byte(malformedToken), jwt.WithVerify(false))
	if err == nil {
		t.Error("Expected malformed token to fail parsing")
	}

	// Save original token and restore after test
	originalToken := Token
	defer func() { Token = originalToken }()
	Token = malformedToken

	ctErr := CheckToken(&cobra.Command{})
	if ctErr == nil {
		t.Fatal("CheckToken should have returned an error for a malformed token")
	}
	if ExitCode(ctErr) != CodeAuth {
		t.Errorf("ExitCode = %d, want %d (%s)", ExitCode(ctErr), CodeAuth, CodeName(CodeAuth))
	}
}

// TestSetToken_FromFlag verifies that SetToken uses the value of --token.
func TestSetToken_FromFlag(t *testing.T) {
	// Save original token and restore after test
	originalToken := Token
	defer func() { Token = originalToken }()

	cmd := &cobra.Command{}
	cmd.Flags().String("token", "", "token flag")
	if err := cmd.Flags().Set("token", "test-token-from-flag"); err != nil {
		t.Fatalf("Failed to set flag: %v", err)
	}

	if err := SetToken(cmd); err != nil {
		t.Fatalf("SetToken returned unexpected error: %v", err)
	}

	if Token != "test-token-from-flag" {
		t.Errorf("Token = %q, want %q", Token, "test-token-from-flag")
	}
}

// TestSetToken_FromEnvironment verifies that SetToken reads the token from the
// selected cluster's <CLUSTER>_ACCESS_TOKEN environment variable.
func TestSetToken_FromEnvironment(t *testing.T) {
	// Save original token and restore after test
	originalToken := Token
	defer func() { Token = originalToken }()

	cmd := &cobra.Command{}
	cmd.Flags().String("token", "", "token flag")
	cmd.Flags().String("cluster", "", "cluster flag")
	if err := cmd.Flags().Set("cluster", "test-cluster"); err != nil {
		t.Fatalf("Failed to set flag: %v", err)
	}

	t.Setenv("TEST_CLUSTER_ACCESS_TOKEN", "test-token-from-environment")

	if err := SetToken(cmd); err != nil {
		t.Fatalf("SetToken returned unexpected error: %v", err)
	}

	if Token != "test-token-from-environment" {
		t.Errorf("Token = %q, want %q", Token, "test-token-from-environment")
	}
}

// TestInitConfigAndLogging_DeclineCreate verifies that declining to create a
// missing config file resolves to CodeDeclined and leaves the file uncreated.
func TestInitConfigAndLogging_DeclineCreate(t *testing.T) {
	origFile := ConfigFile
	t.Cleanup(func() { ConfigFile = origFile })
	ConfigFile = filepath.Join(t.TempDir(), "config.yaml")
	restore := SetIOStream(strings.NewReader("n\n"), &bytes.Buffer{}, &bytes.Buffer{})
	t.Cleanup(restore)

	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().Bool("ignore-config", false, "")

	err := InitConfigAndLogging(cmd, true)
	if got := ExitCode(err); got != CodeDeclined {
		t.Fatalf("ExitCode(%v) = %d, want %d (%s)", err, got, CodeDeclined, CodeName(CodeDeclined))
	}
	if _, statErr := os.Stat(ConfigFile); !os.IsNotExist(statErr) {
		t.Errorf("stat %s = %v, want not-exist", ConfigFile, statErr)
	}
}

// TestBooleanFlags_UseTheirValue verifies InitConfig and HandleToken consult
// the actual value of --ignore-config/--no-token rather than merely whether
// the flag was passed at all (a flag passed as --ignore-config=false or
// --no-token=false must not be treated the same as omitting it).
func TestBooleanFlags_UseTheirValue(t *testing.T) {
	t.Run("ignore-config false", func(t *testing.T) {
		orig := ConfigFile
		t.Cleanup(func() { ConfigFile = orig })
		ConfigFile = t.TempDir() + "/missing.yaml"
		cmd := &cobra.Command{Use: "test"}
		cmd.Flags().Bool("ignore-config", false, "")
		if err := cmd.Flags().Set("ignore-config", "false"); err != nil {
			t.Fatal(err)
		}
		if err := InitConfig(cmd, false); err == nil {
			t.Fatal("InitConfig unexpectedly ignored a false --ignore-config flag")
		}
	})

	t.Run("no-token false", func(t *testing.T) {
		origCfg, origToken := ActiveConfig(), Token
		t.Cleanup(func() { SetActiveConfig(origCfg); Token = origToken })
		SetActiveConfig(config.Config{
			DefaultCluster: "auth-cluster",
			Clusters:       []config.Cluster{{Name: "auth-cluster", Cluster: config.ClusterConfig{EnableAuth: true}}},
		})
		Token = ""
		_ = os.Unsetenv("AUTH_CLUSTER_ACCESS_TOKEN")
		cmd := &cobra.Command{Use: "test"}
		cmd.Flags().String("cluster", "", "")
		cmd.Flags().Bool("no-token", false, "")
		cmd.Flags().String("token", "", "")
		cmd.Flags().Bool("show-token", false, "")
		if err := cmd.Flags().Set("no-token", "false"); err != nil {
			t.Fatal(err)
		}
		if err := HandleToken(cmd); err == nil || ExitCode(err) != CodeAuth {
			t.Fatalf("HandleToken error = %v, want %d (%s)", err, CodeAuth, CodeName(CodeAuth))
		}
	})
}

// TestPayloadReader_Helpers verifies HandlePayloadStdin/HandlePayloadStdinSlice
// read from the injected IOStream reader rather than the real os.Stdin, and
// surface a CodePayload error for malformed input.
func TestPayloadReader_Helpers(t *testing.T) {
	origFormat := FormatInput
	t.Cleanup(func() { FormatInput = origFormat })
	FormatInput = format.DataFormatJson

	var one map[string]interface{}
	restore := SetIOStream(strings.NewReader(`{"name":"node"}`), &bytes.Buffer{}, &bytes.Buffer{})
	if err := HandlePayloadStdin(&cobra.Command{}, &one); err != nil {
		restore()
		t.Fatalf("HandlePayloadStdin: %v", err)
	}
	restore()
	if one["name"] != "node" {
		t.Errorf("payload = %#v", one)
	}

	var many []map[string]interface{}
	restore = SetIOStream(strings.NewReader(`{"name":"node"}`), &bytes.Buffer{}, &bytes.Buffer{})
	if err := HandlePayloadStdinSlice(&cobra.Command{}, &many); err != nil {
		restore()
		t.Fatalf("HandlePayloadStdinSlice: %v", err)
	}
	restore()
	if len(many) != 1 {
		t.Errorf("slice length = %d, want 1", len(many))
	}

	restore = SetIOStream(strings.NewReader(`{`), &bytes.Buffer{}, &bytes.Buffer{})
	if err := HandlePayloadStdin(&cobra.Command{}, &one); err == nil || ExitCode(err) != CodePayload {
		restore()
		t.Fatalf("invalid payload error = %v, want %d (%s)", err, CodePayload, CodeName(CodePayload))
	}
	restore()
}

// TestGetTimeout_ConfigAndFlag verifies GetTimeout falls back to the active
// config's timeout and honors an explicit --timeout flag override.
func TestGetTimeout_ConfigAndFlag(t *testing.T) {
	orig := ActiveConfig()
	t.Cleanup(func() { SetActiveConfig(orig) })
	SetActiveConfig(config.Config{Timeout: 9 * time.Second})
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().Duration("timeout", 0, "")
	if got := GetTimeout(cmd); got != 9*time.Second {
		t.Errorf("GetTimeout config = %v", got)
	}
	if err := cmd.Flags().Set("timeout", "2s"); err != nil {
		t.Fatal(err)
	}
	if got := GetTimeout(cmd); got != 2*time.Second {
		t.Errorf("GetTimeout flag = %v", got)
	}
}

// TestShellCompletions_ReturnDefaultValues verifies the format/discovery/patch
// shell-completion functions return a non-empty, default-directive value set.
func TestShellCompletions_ReturnDefaultValues(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	for name, fn := range map[string]func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective){
		"format":    CompletionFormatData,
		"discovery": CompletionDiscoveryVersion,
		"patch":     CompletionPatchMethod,
	} {
		values, directive := fn(cmd, nil, "")
		if len(values) == 0 || directive != cobra.ShellCompDirectiveDefault {
			t.Errorf("%s completion = %v, %v", name, values, directive)
		}
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
		Clusters: []config.Cluster{
			{Name: "foo", Cluster: config.ClusterConfig{EnableAuth: false}},
		},
	})

	if err := HandleToken(tokenTestCmd()); err != nil {
		t.Fatalf("HandleToken(): unexpected error for auth-disabled cluster: %v", err)
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
		Clusters: []config.Cluster{
			{Name: "my-cluster", Cluster: config.ClusterConfig{EnableAuth: true}},
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
