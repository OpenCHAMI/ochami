// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// services_errors_test.go covers HTTP-failure paths for the pcs, cloud_init,
// and metadata command groups; see services_test.go for the success paths
// these mirror.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// TestPCSStatusShow_Empty verifies that an empty status array resolves to
// CodeGeneric (the "no status found" case).
func TestPCSStatusShow_Empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status":[]}`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	res := runOchami(t, "pcs", "status", "show", "--ignore-config", "--uri", srv.URL, "x3000c0s15b0")

	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeGeneric {
		t.Errorf("exit code = %d, want %d (CodeGeneric)", res.exitCode, cli.CodeGeneric)
	}
}

// TestPCSTransitionStart_InvalidOp verifies that an invalid operation argument
// is a usage error and no request is made.
func TestPCSTransitionStart_InvalidOp(t *testing.T) {
	requestMade := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestMade = true
	}))
	defer srv.Close()

	res := runOchami(t, "pcs", "transition", "start",
		"--ignore-config", "--uri", srv.URL, "--xname", "x0c0s0b0n0",
		"bogus-operation")

	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeUsage {
		t.Errorf("exit code = %d, want %d (CodeUsage)", res.exitCode, cli.CodeUsage)
	}
	if requestMade {
		t.Error("a request was made despite the operation being invalid")
	}
}

// TestCloudInitServiceStatus_NotRunning verifies that when the service is
// unreachable, "cloud-init service status" reports not running and resolves to
// a non-zero exit code.
func TestCloudInitServiceStatus_NotRunning(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close() // connection refused

	res := runOchami(t, "cloud-init", "service", "status", "--ignore-config", "--uri", url)

	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeNetwork {
		t.Errorf("exit code = %d, want %d (CodeNetwork)", res.exitCode, cli.CodeNetwork)
	}
	if !strings.Contains(res.stdout, "cloud-init is not running") {
		t.Errorf("stdout = %q, want it to report not running", res.stdout)
	}
}

// TestMetadataGroupList_HTTPError verifies that an unsuccessful HTTP response
// from the metadata service resolves to a non-success exit code. The metadata
// client wraps an upstream library, so we assert exit-code behavior rather than
// the exact request path.
func TestMetadataGroupList_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer srv.Close()

	res := runOchami(t, "metadata", "group", "list", "--ignore-config", "--uri", srv.URL, "--token", "faketoken")

	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode == cli.CodeSuccess {
		t.Errorf("exit code = %d, want a non-success code", res.exitCode)
	}
}
