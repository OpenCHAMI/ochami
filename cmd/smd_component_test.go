// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// smd_component_test.go exercises the "smd component" commands end-to-end
// against an httptest.Server. It verifies the outbound request method/path/body
// and that command errors resolve to the correct exit codes defined in
// internal/cli/errors.go.

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// TestSMDComponentGetAll verifies that "smd component get" with no selectors
// issues GET /State/Components and prints the server's body on success.
func TestSMDComponentGetAll(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"Components":[{"ID":"x0c0s0b0n0"}]}`))
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "component", "get", "--ignore-config", "--uri", srv.URL)

	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if res.exitCode != cli.CodeSuccess {
		t.Errorf("exit code = %d, want %d", res.exitCode, cli.CodeSuccess)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("request method = %q, want GET", gotMethod)
	}
	if gotPath != "/State/Components" {
		t.Errorf("request path = %q, want /State/Components", gotPath)
	}
	if !strings.Contains(res.stdout, "x0c0s0b0n0") {
		t.Errorf("stdout = %q, want it to contain the component ID", res.stdout)
	}
}

// TestSMDComponentGetAllHTTPError verifies that an unsuccessful HTTP response
// resolves to the CodeHTTP exit code.
func TestSMDComponentGetAllHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "component", "get", "--ignore-config", "--uri", srv.URL)

	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestSMDComponentGetNetworkError verifies that a transport-level failure
// (server closed, connection refused) resolves to the CodeNetwork exit code.
func TestSMDComponentGetNetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close() // close immediately so the connection is refused

	res := runOchami(t, "smd", "component", "get", "--ignore-config", "--uri", url)

	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeNetwork {
		t.Errorf("exit code = %d, want %d (CodeNetwork)", res.exitCode, cli.CodeNetwork)
	}
}

// TestSMDComponentAddViaFlags verifies that "smd component add <xname> <nid>"
// issues POST /State/Components with the component encoded in the body.
func TestSMDComponentAddViaFlags(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "component", "add",
		"--ignore-config", "--uri", srv.URL,
		"--token", "faketoken",
		"x3000c1s7b56n0", "56")

	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("request method = %q, want POST", gotMethod)
	}
	if gotPath != "/State/Components" {
		t.Errorf("request path = %q, want /State/Components", gotPath)
	}
	// Body should be a ComponentSlice with our xname.
	var payload struct {
		Components []map[string]any `json:"Components"`
	}
	if err := json.Unmarshal(gotBody, &payload); err != nil {
		t.Fatalf("failed to unmarshal request body %q: %v", string(gotBody), err)
	}
	if len(payload.Components) != 1 {
		t.Fatalf("Components length = %d, want 1 (body=%q)", len(payload.Components), string(gotBody))
	}
	if id, _ := payload.Components[0]["ID"].(string); id != "x3000c1s7b56n0" {
		t.Errorf("Components[0].ID = %v, want x3000c1s7b56n0", payload.Components[0]["ID"])
	}
}

// TestSMDComponentAddBadPayload verifies that malformed -d payload data
// resolves to the CodePayload exit code before any request is made.
func TestSMDComponentAddBadPayload(t *testing.T) {
	requestMade := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestMade = true
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "component", "add",
		"--ignore-config", "--uri", srv.URL,
		"--token", "faketoken",
		"-d", "{this is not valid json")

	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodePayload {
		t.Errorf("exit code = %d, want %d (CodePayload)", res.exitCode, cli.CodePayload)
	}
	if requestMade {
		t.Error("a request was made despite the payload being invalid")
	}
}

// TestSMDComponentAddMissingArgs verifies that invoking add without -d and
// without the required positional arguments is a usage error (CodeUsage).
func TestSMDComponentAddMissingArgs(t *testing.T) {
	res := runOchami(t, "smd", "component", "add", "--ignore-config", "--uri", "http://127.0.0.1:0")

	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeUsage {
		t.Errorf("exit code = %d, want %d (CodeUsage)", res.exitCode, cli.CodeUsage)
	}
}

// TestSMDComponentDeleteNoConfirm verifies that "smd component delete --no-confirm <xname>"
// issues DELETE /State/Components/<xname> without prompting.
func TestSMDComponentDeleteNoConfirm(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "component", "delete",
		"--ignore-config", "--uri", srv.URL,
		"--token", "faketoken",
		"--no-confirm",
		"x3000c1s7b56n0")

	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("request method = %q, want DELETE", gotMethod)
	}
	if gotPath != "/State/Components/x3000c1s7b56n0" {
		t.Errorf("request path = %q, want /State/Components/x3000c1s7b56n0", gotPath)
	}
}

// TestSMDComponentDeleteAllPartialFailure verifies that a per-item unsuccessful
// HTTP response during multi-item deletion resolves to CodeHTTP (the
// "completed with errors" aggregate).
func TestSMDComponentDeletePartialFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Fail every delete so the aggregate reports errors.
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "component", "delete",
		"--ignore-config", "--uri", srv.URL,
		"--token", "faketoken",
		"--no-confirm",
		"x3000c1s7b56n0")

	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}
