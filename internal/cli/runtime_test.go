// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cli

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/pkg/client"
	"github.com/openchami/ochami/pkg/config"
	"github.com/openchami/ochami/pkg/format"
)

// TestNewRuntime verifies that NewRuntime returns a runtime with I/O streams
// and JSON input and output formats.
func TestNewRuntime(t *testing.T) {
	// Test production runtime
	rt := NewRuntime()
	if rt == nil {
		t.Fatal("NewRuntime() returned nil")
	}
	if rt.Ios == nil {
		t.Error("Runtime.Ios is nil")
	}
	if rt.FormatInput != format.DataFormatJson {
		t.Errorf("Runtime.FormatInput = %v, want %v", rt.FormatInput, format.DataFormatJson)
	}
	if rt.FormatOutput != format.DataFormatJson {
		t.Errorf("Runtime.FormatOutput = %v, want %v", rt.FormatOutput, format.DataFormatJson)
	}
}

// TestNewTestRuntime verifies that NewTestRuntime returns a runtime whose I/O
// streams are the ones it was given.
func TestNewTestRuntime(t *testing.T) {
	var stdin bytes.Buffer
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	rt := NewTestRuntime(&stdin, &stdout, &stderr)
	if rt == nil {
		t.Fatal("NewTestRuntime() returned nil")
	}
	if rt.Ios == nil {
		t.Error("Runtime.Ios is nil")
	}
	if rt.Ios.In() != &stdin {
		t.Error("Runtime.Ios.In() does not return the provided stdin")
	}
	if rt.Ios.Out() != &stdout {
		t.Error("Runtime.Ios.Out() does not return the provided stdout")
	}
	if rt.Ios.Err() != &stderr {
		t.Error("Runtime.Ios.Err() does not return the provided stderr")
	}
}

// TestRuntime_WithMethods verifies that the With* builder methods set the
// config file, token, formats, CA certificate path, and insecure flag.
func TestRuntime_WithMethods(t *testing.T) {
	rt := NewRuntime().
		WithConfigFile("/path/to/config").
		WithToken("test-token").
		WithFormats(format.DataFormatYaml, format.DataFormatJson).
		WithCACert("/path/to/ca.pem").
		WithInsecure(true)

	if rt.ConfigFile != "/path/to/config" {
		t.Errorf("Runtime.ConfigFile = %v, want %v", rt.ConfigFile, "/path/to/config")
	}
	if rt.Token != "test-token" {
		t.Errorf("Runtime.Token = %v, want %v", rt.Token, "test-token")
	}
	if rt.FormatInput != format.DataFormatYaml {
		t.Errorf("Runtime.FormatInput = %v, want %v", rt.FormatInput, format.DataFormatYaml)
	}
	if rt.FormatOutput != format.DataFormatJson {
		t.Errorf("Runtime.FormatOutput = %v, want %v", rt.FormatOutput, format.DataFormatJson)
	}
	if rt.CACertPath != "/path/to/ca.pem" {
		t.Errorf("Runtime.CACertPath = %v, want %v", rt.CACertPath, "/path/to/ca.pem")
	}
	if !rt.Insecure {
		t.Error("Runtime.Insecure should be true")
	}
}

// TestIOStreams_Accessors verifies that In, Out, and Err return the streams
// NewIOStreams was given.
func TestIOStreams_Accessors(t *testing.T) {
	var stdin bytes.Buffer
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	stdin.WriteString("test input")

	ios := NewIOStreams(&stdin, &stdout, &stderr)

	// Test In()
	if ios.In() != &stdin {
		t.Error("IOStreams.In() does not return the provided stdin")
	}

	// Test Out()
	if ios.Out() != &stdout {
		t.Error("IOStreams.Out() does not return the provided stdout")
	}

	// Test Err()
	if ios.Err() != &stderr {
		t.Error("IOStreams.Err() does not return the provided stderr")
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

// TestUseCACert verifies that UseCACert does nothing without a CA path, accepts
// a valid CA certificate, and fails with CodePayload for a missing file or one
// without a valid PEM certificate.
func TestUseCACert(t *testing.T) {
	rt := NewTestRuntime(nil, &bytes.Buffer{}, &bytes.Buffer{})

	oc, err := client.NewOchamiClient("test", "https://example.com")
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// No CA path set: no-op, returns nil.
	rt.CACertPath = ""
	if err := rt.UseCACert(oc); err != nil {
		t.Errorf("UseCACert with empty path = %v, want nil", err)
	}

	// Valid CA cert.
	rt.CACertPath = writeTestCACert(t)
	if err := rt.UseCACert(oc); err != nil {
		t.Errorf("UseCACert with valid cert = %v, want nil", err)
	}

	// Invalid/nonexistent CA cert => CodePayload.
	rt.CACertPath = filepath.Join(t.TempDir(), "does-not-exist.pem")
	err = rt.UseCACert(oc)
	if err == nil {
		t.Fatal("UseCACert with missing cert = nil, want error")
	}
	if ExitCode(err) != CodePayload {
		t.Errorf("exit code = %d, want %d (%s)", ExitCode(err), CodePayload, CodeName(CodePayload))
	}

	// Malformed PEM => CodePayload.
	bad := filepath.Join(t.TempDir(), "bad.pem")
	if err := os.WriteFile(bad, []byte("not a pem"), 0o644); err != nil {
		t.Fatalf("failed to write bad pem: %v", err)
	}
	rt.CACertPath = bad
	if err := rt.UseCACert(oc); err == nil || ExitCode(err) != CodePayload {
		t.Errorf("UseCACert with malformed PEM = %v (exit %d), want %d (%s)", err, ExitCode(err), CodePayload, CodeName(CodePayload))
	}
}

// TestRuntimeInitConfig_DeclineCreate verifies that declining to create a
// missing config file resolves to CodeDeclined and leaves the file uncreated.
func TestRuntimeInitConfig_DeclineCreate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	rt := NewTestRuntime(strings.NewReader("n\n"), &bytes.Buffer{}, &bytes.Buffer{}).WithConfigFile(path)
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().Bool("ignore-config", false, "")

	err := rt.InitConfig(cmd, true)
	if got := ExitCode(err); got != CodeDeclined {
		t.Fatalf("ExitCode(%v) = %d, want %d (%s)", err, got, CodeDeclined, CodeName(CodeDeclined))
	}
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Errorf("stat %s = %v, want not-exist", path, statErr)
	}
}

// TestBooleanFlags_UseTheirValue verifies InitConfig and HandleToken consult
// the actual value of --ignore-config/--no-token rather than merely whether
// the flag was passed at all (a flag passed as --ignore-config=false or
// --no-token=false must not be treated the same as omitting it).
func TestBooleanFlags_UseTheirValue(t *testing.T) {
	t.Run("ignore-config false", func(t *testing.T) {
		rt := NewTestRuntime(nil, &bytes.Buffer{}, &bytes.Buffer{})
		rt.ConfigFile = t.TempDir() + "/missing.yaml"
		cmd := &cobra.Command{Use: "test"}
		cmd.SetContext(ContextWithRuntime(context.Background(), rt))
		cmd.Flags().Bool("ignore-config", false, "")
		if err := cmd.Flags().Set("ignore-config", "false"); err != nil {
			t.Fatalf("set ignore-config flag: %v", err)
		}
		if err := rt.InitConfig(cmd, false); err == nil {
			t.Fatal("InitConfig unexpectedly ignored a false --ignore-config flag")
		}
	})

	t.Run("no-token false", func(t *testing.T) {
		rt := NewTestRuntime(nil, &bytes.Buffer{}, &bytes.Buffer{})
		rt.Config = config.Config{
			DefaultCluster: "auth-cluster",
			Clusters:       []config.Cluster{{Name: "auth-cluster", Cluster: config.ClusterConfig{EnableAuth: true}}},
		}
		rt.Token = ""
		cmd := &cobra.Command{Use: "test"}
		cmd.Flags().String("cluster", "", "")
		cmd.Flags().Bool("no-token", false, "")
		cmd.Flags().String("token", "", "")
		cmd.Flags().Bool("show-token", false, "")
		cmd.SetContext(ContextWithRuntime(context.Background(), rt))
		if err := cmd.Flags().Set("no-token", "false"); err != nil {
			t.Fatalf("set no-token flag: %v", err)
		}
		if err := rt.HandleToken(cmd); err == nil || ExitCode(err) != CodeAuth {
			t.Fatalf("HandleToken error = %v, want %d (%s)", err, CodeAuth, CodeName(CodeAuth))
		}
	})
}

// TestGetTimeout_ConfigAndFlag verifies GetTimeout falls back to the
// configured timeout and honors an explicit --timeout flag override.
func TestGetTimeout_ConfigAndFlag(t *testing.T) {
	rt := NewTestRuntime(nil, &bytes.Buffer{}, &bytes.Buffer{})
	rt.Config = config.Config{Timeout: 9 * time.Second}
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().Duration("timeout", 0, "")
	cmd.SetContext(ContextWithRuntime(context.Background(), rt))
	if got := rt.GetTimeout(cmd); got != 9*time.Second {
		t.Errorf("GetTimeout config = %v", got)
	}
	if err := cmd.Flags().Set("timeout", "2s"); err != nil {
		t.Fatalf("set timeout flag: %v", err)
	}
	if got := rt.GetTimeout(cmd); got != 2*time.Second {
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
