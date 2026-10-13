// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cli

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/log"
)

// TestCheckTokenProcess_Validation verifies that CheckToken accepts valid time
// claims without verifying signatures, rejects invalid tokens, and warns about
// imminent expiration. Subprocesses isolate its process exits and global state.
func TestCheckTokenProcess_Validation(t *testing.T) {
	const (
		childEnv  = "OCHAMI_TEST_CHECK_TOKEN_CHILD"
		tokenEnv  = "OCHAMI_TEST_CHECK_TOKEN_VALUE"
		completed = "check-token-complete"
	)
	if os.Getenv(childEnv) == "1" {
		if err := log.Init("info", "basic", "off"); err != nil {
			t.Fatalf("initialize child logger: %v", err)
		}
		Token = os.Getenv(tokenEnv)
		CheckToken(&cobra.Command{})
		if _, err := fmt.Fprintln(os.Stdout, completed); err != nil {
			t.Fatalf("write child completion marker: %v", err)
		}
		return
	}

	now := time.Now()
	signed, err := generateTestToken(now.Add(time.Hour), now.Add(-time.Hour), now.Add(-time.Hour))
	if err != nil {
		t.Fatalf("generate signed token: %v", err)
	}
	// Preserve the signed token's header and payload but replace its signature
	// with a base64url-encoded byte, which cannot be a valid RSA signature.
	parts := strings.Split(signed, ".")
	if len(parts) != 3 {
		t.Fatalf("signed token has %d segments, want 3", len(parts))
	}
	invalidSignature := parts[0] + "." + parts[1] + ".AA"
	compactToken := func(payload string) string {
		return base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`)) + "." +
			base64.RawURLEncoding.EncodeToString([]byte(payload)) + ".AA"
	}
	cases := []struct {
		name     string
		token    string
		exitCode int
		message  string
	}{
		{name: "valid", token: signed},
		{name: "missing_optional_time_claims", token: compactToken(`{"sub":"test-subject"}`)},
		{name: "invalid_signature", token: invalidSignature},
		{name: "expiring_soon", token: compactToken(fmt.Sprintf(`{"exp":%d}`, now.Add(10*time.Minute).Unix())), message: "until token expires"},
		{name: "empty", exitCode: 1, message: "no token set"},
		{name: "malformed", token: "not.a.valid.jwt.token.at.all", exitCode: 1, message: "failed to parse token"},
		{name: "expired", token: compactToken(fmt.Sprintf(`{"exp":%d}`, now.Add(-time.Hour).Unix())), exitCode: 1, message: "token is expired"},
		{name: "future_nbf", token: compactToken(fmt.Sprintf(`{"nbf":%d}`, now.Add(time.Hour).Unix())), exitCode: 1, message: "token is not yet valid (nbf in future)"},
		{name: "future_iat", token: compactToken(fmt.Sprintf(`{"iat":%d}`, now.Add(time.Hour).Unix())), exitCode: 1, message: "failed to parse token"},
		// Write both names manually: jwt.Token.Set replaces an existing claim.
		{name: "duplicate_claim", token: compactToken(fmt.Sprintf(`{"exp":%d,"exp":%d}`, now.Add(time.Hour).Unix(), now.Add(2*time.Hour).Unix())), exitCode: 1, message: "failed to parse token"},
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatalf("locate test executable: %v", err)
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(executable, "-test.run=^TestCheckTokenProcess_Validation$")
			cmd.Env = append(os.Environ(), childEnv+"=1", tokenEnv+"="+tc.token)
			output, err := cmd.CombinedOutput()
			exitCode := 0
			if err != nil {
				var exitErr *exec.ExitError
				if !errors.As(err, &exitErr) {
					t.Fatalf("run token check: %v", err)
				}
				exitCode = exitErr.ExitCode()
			}
			if exitCode != tc.exitCode {
				t.Fatalf("exit status = %d, want %d; output: %s", exitCode, tc.exitCode, output)
			}
			if tc.message != "" && !strings.Contains(string(output), tc.message) {
				t.Errorf("output = %q, want diagnostic containing %q", output, tc.message)
			}
			if got := strings.Contains(string(output), completed); got != (tc.exitCode == 0) {
				t.Errorf("completion marker present = %t, want %t; output: %s", got, tc.exitCode == 0, output)
			}
			if tc.exitCode == 0 && tc.message == "" && strings.Contains(string(output), "until token expires") {
				t.Errorf("unexpected expiry warning: %s", output)
			}
		})
	}
}
