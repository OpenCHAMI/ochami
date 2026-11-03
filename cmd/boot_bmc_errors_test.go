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

// TestBootBmcAdd_MalformedResponse verifies that "boot bmc add" fails with
// CodePayload when the service's success response can't be decoded.
func TestBootBmcAdd_MalformedResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"BMCs":[{"ID":`)); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "--cluster-uri", srv.URL, "--token", "t",
		"boot", "bmc", "add", "-d", `{}`)

	if res.err == nil {
		t.Fatal("expected malformed response error, got nil")
	}
	if res.exitCode != cli.CodePayload {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodePayload, cli.CodeName(cli.CodePayload))
	}
}

// TestBootBmcSet_MalformedResponse verifies that "boot bmc set" fails with
// CodePayload when the service's success response can't be decoded.
func TestBootBmcSet_MalformedResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"BMCs":[{"ID":`)); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer srv.Close()

	args := []string{"--ignore-config", "--cluster-uri", srv.URL, "--token", "t",
		"boot", "bmc", "set", "x0c0s1b0n0", "-d", `{"xname":"test"}`}
	res := runOchamiWithRuntime(t, args...)

	if res.err == nil {
		t.Fatal("expected malformed response error, got nil")
	}
	if res.exitCode != cli.CodePayload {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodePayload, cli.CodeName(cli.CodePayload))
	}
}

// TestBootBmcSet_EnvelopeMalformedResponse verifies that "boot bmc set -e"
// fails with CodePayload when the service's success response can't be decoded.
func TestBootBmcSet_EnvelopeMalformedResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"BMCs":[{"ID":`)); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer srv.Close()

	args := []string{"--ignore-config", "--cluster-uri", srv.URL, "--token", "t",
		"boot", "bmc", "set", "-e", "x0c0s1b0n0", "-d", `{"spec":{"xname":"test"}}`}
	res := runOchamiWithRuntime(t, args...)

	if res.err == nil {
		t.Fatal("expected malformed response error, got nil")
	}
	if res.exitCode != cli.CodePayload {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodePayload, cli.CodeName(cli.CodePayload))
	}
}

// TestBootBmcPatch_MalformedResponse verifies that "boot bmc patch" fails with
// CodePayload when the service's success response can't be decoded.
func TestBootBmcPatch_MalformedResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"BMCs":`)); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "--cluster-uri", srv.URL, "--token", "t",
		"boot", "bmc", "patch", "x0c0s1b0n0", "-d", `{}`)

	if res.err == nil {
		t.Fatal("expected malformed response error, got nil")
	}
	if res.exitCode != cli.CodePayload {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodePayload, cli.CodeName(cli.CodePayload))
	}
}
