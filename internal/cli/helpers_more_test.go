// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cli

// helpers_more_test.go covers small internal/cli helpers whose branches the
// existing tests leave uncovered: ioStream.Err, UseCACert (no-op/valid/invalid),
// PrintUsageHandleError, LogHelpHint, ActiveKoanf, and InitLogging's
// format/level/color flag-override arms.

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/openchami/ochami/pkg/client"
)

// TestIOStreamErr verifies Err returns the configured error writer.
func TestIOStreamErr(t *testing.T) {
	var errBuf bytes.Buffer
	restore := SetIOStream(nil, &bytes.Buffer{}, &errBuf)
	defer restore()

	if Ios.Err() != &errBuf {
		t.Errorf("Err() = %v, want the configured error writer", Ios.Err())
	}
}

// writeTestCACert writes a self-signed CA certificate PEM to a temp file and
// returns its path.
func writeTestCACert(t *testing.T) string {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("failed to create cert: %v", err)
	}
	path := filepath.Join(t.TempDir(), "ca.pem")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
	defer f.Close()
	if err := pem.Encode(f, &pem.Block{Type: "CERTIFICATE", Bytes: der}); err != nil {
		t.Fatalf("failed to encode PEM: %v", err)
	}
	return path
}

// TestUseCACert covers the no-op (empty path), valid, and invalid cert arms.
func TestUseCACert(t *testing.T) {
	oc, err := client.NewOchamiClient("test", "https://example.com")
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// No CA path set: no-op, returns nil.
	origPath := CACertPath
	defer func() { CACertPath = origPath }()
	CACertPath = ""
	if err := UseCACert(oc); err != nil {
		t.Errorf("UseCACert with empty path = %v, want nil", err)
	}

	// Valid CA cert.
	CACertPath = writeTestCACert(t)
	if err := UseCACert(oc); err != nil {
		t.Errorf("UseCACert with valid cert = %v, want nil", err)
	}

	// Invalid/nonexistent CA cert => CodePayload.
	CACertPath = filepath.Join(t.TempDir(), "does-not-exist.pem")
	err = UseCACert(oc)
	if err == nil {
		t.Fatal("UseCACert with missing cert = nil, want error")
	}
	if ExitCode(err) != CodePayload {
		t.Errorf("exit code = %d, want %d (CodePayload)", ExitCode(err), CodePayload)
	}

	// Malformed PEM => CodePayload.
	bad := filepath.Join(t.TempDir(), "bad.pem")
	if err := os.WriteFile(bad, []byte("not a pem"), 0o644); err != nil {
		t.Fatalf("failed to write bad pem: %v", err)
	}
	CACertPath = bad
	if err := UseCACert(oc); err == nil || ExitCode(err) != CodePayload {
		t.Errorf("UseCACert with malformed PEM = %v (exit %d), want CodePayload", err, ExitCode(err))
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
}

// TestLogHelpHint exercises the help-hint emitters (which just log).
func TestLogHelpHint(t *testing.T) {
	cmd := &cobra.Command{Use: "demo"}
	LogHelpHint(cmd)
	logHelpHint(cmd)
}

// TestInitLoggingFlagOverrides covers InitLogging's log-format/level/color
// flag-override arms.
func TestInitLoggingFlagOverrides(t *testing.T) {
	cmd := &cobra.Command{Use: "demo"}
	fs := pflag.NewFlagSet("demo", pflag.ContinueOnError)
	fs.String("log-format", "", "")
	fs.String("log-level", "", "")
	fs.String("log-color", "", "")
	cmd.Flags().AddFlagSet(fs)

	if err := cmd.Flags().Set("log-format", "json"); err != nil {
		t.Fatalf("set log-format: %v", err)
	}
	if err := cmd.Flags().Set("log-level", "warning"); err != nil {
		t.Fatalf("set log-level: %v", err)
	}
	if err := cmd.Flags().Set("log-color", "off"); err != nil {
		t.Fatalf("set log-color: %v", err)
	}

	if err := InitLogging(cmd); err != nil {
		t.Fatalf("InitLogging = %v, want nil", err)
	}
	if activeConfig.Log.Format != "json" {
		t.Errorf("Log.Format = %q, want json", activeConfig.Log.Format)
	}
	if activeConfig.Log.Level != "warning" {
		t.Errorf("Log.Level = %q, want warning", activeConfig.Log.Level)
	}
}
