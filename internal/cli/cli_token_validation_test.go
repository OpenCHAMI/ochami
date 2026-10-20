// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cli

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v4/jwt"
	"github.com/rs/zerolog"
)

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
