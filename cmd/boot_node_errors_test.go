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

// TestBootNodeAdd_MalformedResponse verifies that "boot node add" fails with
// CodePayload when the service's success response can't be decoded.
func TestBootNodeAdd_MalformedResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"Nodes":[{"ID":`)); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer srv.Close()

	args := []string{"--ignore-config", "--cluster-uri", srv.URL, "--token", "t",
		"boot", "node", "add", "-d", `{}`}
	res := runOchamiWithRuntime(t, args...)

	if res.err == nil {
		t.Fatal("expected malformed response error, got nil")
	}
	if res.exitCode != cli.CodePayload {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodePayload, cli.CodeName(cli.CodePayload))
	}
}

// TestBootNodePatch_MalformedResponse verifies that "boot node patch" fails
// with CodePayload when the service's success response can't be decoded.
func TestBootNodePatch_MalformedResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"Nodes":[{"ID":`)); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer srv.Close()

	args := []string{"--ignore-config", "--cluster-uri", srv.URL, "--token", "t",
		"boot", "node", "patch", "x0c0s1b0n0", "-d", `{}`}
	res := runOchamiWithRuntime(t, args...)

	if res.err == nil {
		t.Fatal("expected malformed response error, got nil")
	}
	if res.exitCode != cli.CodePayload {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodePayload, cli.CodeName(cli.CodePayload))
	}
}

// TestBootNodeSet_MalformedResponse verifies that "boot node set" fails with
// CodePayload when the service's success response can't be decoded.
func TestBootNodeSet_MalformedResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"Nodes":[{"ID":`)); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer srv.Close()

	args := []string{"--ignore-config", "--cluster-uri", srv.URL, "--token", "t",
		"boot", "node", "set", "x0c0s1b0n0", "-d", `{}`}
	res := runOchamiWithRuntime(t, args...)

	if res.err == nil {
		t.Fatal("expected malformed response error, got nil")
	}
	if res.exitCode != cli.CodePayload {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodePayload, cli.CodeName(cli.CodePayload))
	}
}
