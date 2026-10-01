// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// smd_component_test.go exercises the "smd component" commands end-to-end
// against an httptest.Server. It verifies the outbound request method/path/body
// and that command errors resolve to the correct exit codes defined in
// internal/cli/errors.go. HTTP/network/payload-failure and rejection-path
// cases are covered in smd_component_errors_test.go.

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// TestSMDComponentGet_All verifies that "smd component get" with no selectors
// issues GET /State/Components and prints the server's body on success.
func TestSMDComponentGet_All(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"Components":[{"ID":"x0c0s0b0n0"}]}`))
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "component", "get", "--ignore-config", "--uri", srv.URL)

	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if res.exitCode != cli.CodeSuccess {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeSuccess, cli.CodeName(cli.CodeSuccess))
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

// TestSMDComponentAdd_ViaFlags verifies that "smd component add <xname> <nid>"
// issues POST /State/Components with the component encoded in the body.
func TestSMDComponentAdd_ViaFlags(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		var err error
		gotBody, err = io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "component", "add",
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

// TestSMDComponentDelete_NoConfirm verifies that "smd component delete --no-confirm <xname>"
// issues DELETE /State/Components/<xname> without prompting.
func TestSMDComponentDelete_NoConfirm(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "component", "delete",
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

// TestSMDComponentDelete_ByData verifies IDs in a payload drive DELETE requests.
func TestSMDComponentDelete_ByData(t *testing.T) {
	t.Parallel()

	var deletes int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deletes++
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "component", "delete", "--ignore-config", "--uri", srv.URL,
		"--token", "faketoken", "--no-confirm", "-d", `{"Components":[{"ID":"x3000c1s7b56n0"}]}`)
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if deletes != 1 {
		t.Errorf("DELETE count = %d, want 1", deletes)
	}
}

// TestSMDComponentGet_ByXname verifies that "smd component get --xname"
// requests the component by its xname.
func TestSMDComponentGet_ByXname(t *testing.T) {
	t.Parallel()

	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Write([]byte(`{"ID":"x0c0s0b0n0"}`))
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "component", "get", "--ignore-config", "--uri", srv.URL,
		"--token", validToken(t), "--xname", "x0c0s0b0n0")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.Contains(gotPath, "x0c0s0b0n0") {
		t.Errorf("path = %q, want it to reference the xname", gotPath)
	}
}

// TestSMDComponentGet_ByNID verifies "get --nid" targets the ByNID endpoint.
func TestSMDComponentGet_ByNID(t *testing.T) {
	t.Parallel()

	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Write([]byte(`{"NID":1}`))
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "component", "get", "--ignore-config", "--uri", srv.URL,
		"--token", validToken(t), "--nid", "1")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.Contains(gotPath, "ByNID") {
		t.Errorf("path = %q, want it to reference ByNID", gotPath)
	}
}

// TestSMDComponentGet_Formats verifies output-format variants of get-all.
func TestSMDComponentGet_Formats(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"Components":[{"ID":"x0c0s0b0n0"}]}`))
	}))
	defer srv.Close()

	for _, f := range []string{"json", "json-pretty", "yaml"} {
		res := runOchamiWithRuntime(t, "smd", "component", "get", "--ignore-config", "--uri", srv.URL, "-F", f)
		if res.err != nil {
			t.Fatalf("format %s: unexpected error: %v (exit %d)", f, res.err, res.exitCode)
		}
		assertFormattedOutput(t, f, res.stdout, "ID", "x0c0s0b0n0")
	}
}

// TestSMDComponentDelete_AllConfirm verifies "delete --all" prompts and, on "y",
// issues a DELETE to the collection endpoint.
func TestSMDComponentDelete_AllConfirm(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			gotMethod, gotPath = r.Method, r.URL.Path
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithInputAndRuntime(t, "y\n",
		"smd", "component", "delete", "--ignore-config", "--uri", srv.URL, "--token", "t", "--all")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.Contains(res.stdout, "Really delete ALL COMPONENTS?") {
		t.Errorf("output = %q, want the confirmation prompt", res.stdout)
	}
	if gotMethod != http.MethodDelete || gotPath != "/State/Components" {
		t.Errorf("request = %s %s, want DELETE /State/Components", gotMethod, gotPath)
	}
}

// TestSMDComponentList_EmptyResponse verifies that "smd component list"
// succeeds for an empty component list.
func TestSMDComponentList_EmptyResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"Components":[]}`))
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "--cluster-uri", srv.URL, "--token", "t",
		"smd", "component", "list")

	// Should succeed with empty list
	if res.err != nil {
		t.Fatalf("unexpected error: %v", res.err)
	}
	if res.exitCode != cli.CodeSuccess {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeSuccess, cli.CodeName(cli.CodeSuccess))
	}
}

// TestSMDComponentGet_FormatOutputJSON verifies that --format-output json is
// parsed and applied to "smd component get" output.
func TestSMDComponentGet_FormatOutputJSON(t *testing.T) {
	t.Parallel()

	// Create a test server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"Components":[]}`))
	}))
	defer srv.Close()

	// Test JSON output format with smd component get
	// Note: smd component get requires --nid flag
	res := runOchamiWithRuntime(t, "--ignore-config", "--cluster-uri", srv.URL, "--token", "t",
		"--format-output", "json", "smd", "component", "get", "--nid", "0")
	if res.err != nil {
		t.Fatalf("unexpected error with --format-output json: %v", res.err)
	}

	// Output should contain the component data in JSON format
	if !strings.Contains(res.stdout, "Components") {
		t.Errorf("expected Components in JSON output, got: %s", res.stdout)
	}
}
