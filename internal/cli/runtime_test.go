// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cli

import (
	"bytes"
	"testing"

	"github.com/openchami/ochami/pkg/format"
)

// TestRuntimeCreation tests that runtime instances can be created
func TestRuntimeCreation(t *testing.T) {
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

// TestTestRuntimeCreation tests that test runtime instances can be created
func TestTestRuntimeCreation(t *testing.T) {
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

// TestRuntimeWithMethods tests the runtime builder methods
func TestRuntimeWithMethods(t *testing.T) {
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

// TestIOStreamsMethods tests the IOStreams methods
func TestIOStreamsMethods(t *testing.T) {
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

// Note: AskToCreate and LoopYesNo methods are delegated to the existing
// ioStream implementation and will be properly integrated in future work.
// For now, the IOStreams provides basic I/O redirection functionality.
