// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// TestBootConfigDelete_NoArgs verifies that "boot config delete" with no UID
// arguments is a usage error (MinimumNArgs(1)).
func TestBootConfigDelete_NoArgs(t *testing.T) {
	t.Parallel()

	res := runOchamiWithRuntime(t, "--ignore-config", "boot", "config", "delete", "--uri", "http://127.0.0.1:0", "--no-confirm")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeUsage {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeUsage, cli.CodeName(cli.CodeUsage))
	}
}

// TestBootConfigAdd_MalformedResponse verifies that "boot config add" fails
// with CodePayload when the service's success response can't be decoded.
func TestBootConfigAdd_MalformedResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"BootConfigParams":[{"ID":`)); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer srv.Close()

	args := []string{"--ignore-config", "--cluster-uri", srv.URL, "--token", "t",
		"boot", "config", "add", "-d", `{}`}
	res := runOchamiWithRuntime(t, args...)

	if res.err == nil {
		t.Fatal("expected malformed response error, got nil")
	}
	if res.exitCode != cli.CodePayload {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodePayload, cli.CodeName(cli.CodePayload))
	}
}

// TestBootConfigPatch_MalformedResponse verifies that "boot config patch" fails
// with CodePayload when the service's success response can't be decoded.
func TestBootConfigPatch_MalformedResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"BootConfigParams":[{"ID":`)); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer srv.Close()

	args := []string{"--ignore-config", "--cluster-uri", srv.URL, "--token", "t",
		"boot", "config", "patch", "x0c0s1b0n0", "-d", `{}`}
	res := runOchamiWithRuntime(t, args...)

	if res.err == nil {
		t.Fatal("expected malformed response error, got nil")
	}
	if res.exitCode != cli.CodePayload {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodePayload, cli.CodeName(cli.CodePayload))
	}
}

// TestBootConfigSet_MalformedResponse verifies that "boot config set" fails
// with CodePayload when the service's success response can't be decoded.
func TestBootConfigSet_MalformedResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"BootConfigParams":[{"ID":`)); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer srv.Close()

	args := []string{"--ignore-config", "--cluster-uri", srv.URL, "--token", "t",
		"boot", "config", "set", "x0c0s1b0n0", "-d", `{}`}
	res := runOchamiWithRuntime(t, args...)

	if res.err == nil {
		t.Fatal("expected malformed response error, got nil")
	}
	if res.exitCode != cli.CodePayload {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodePayload, cli.CodeName(cli.CodePayload))
	}
}
