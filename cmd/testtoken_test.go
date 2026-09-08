// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// testtoken_test.go provides a helper for generating a signed JWT with valid
// time-based claims, for commands that call cli.CheckToken (which parses and
// validates the token's exp/nbf/iat before making a request).

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwt"
)

// validToken returns a signed JWT whose claims are valid (expires in the
// future, already valid) so cli.CheckToken accepts it. The signature is not
// verified by the CLI, so an ephemeral RSA key suffices.
func validToken(t *testing.T) string {
	t.Helper()
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	now := time.Now()
	tok := jwt.New()
	if err := tok.Set(jwt.ExpirationKey, now.Add(1*time.Hour)); err != nil {
		t.Fatalf("set expiration claim: %v", err)
	}
	if err := tok.Set(jwt.NotBeforeKey, now.Add(-1*time.Hour)); err != nil {
		t.Fatalf("set not-before claim: %v", err)
	}
	if err := tok.Set(jwt.IssuedAtKey, now.Add(-1*time.Hour)); err != nil {
		t.Fatalf("set issued-at claim: %v", err)
	}
	if err := tok.Set(jwt.SubjectKey, "test-subject"); err != nil {
		t.Fatalf("set subject claim: %v", err)
	}
	signed, err := jwt.Sign(tok, jwt.WithKey(jwa.RS256(), privKey))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}
	return string(signed)
}
