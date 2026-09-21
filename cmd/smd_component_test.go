// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
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
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"Components":[{"ID":"x0c0s0b0n0"}]}`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "component", "get", "--ignore-config", "--uri", srv.URL)

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

// TestSMDComponentAdd_ViaFlags verifies that "smd component add <xname> <nid>"
// issues POST /State/Components with the component encoded in the body.
func TestSMDComponentAdd_ViaFlags(t *testing.T) {
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

func TestSMDComponentGet_ByXname(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"ID":"x0c0s0b0n0"}`)) //nolint:errcheck // test response writes are observed by the client
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
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"NID":1}`)) //nolint:errcheck // test response writes are observed by the client
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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"Components":[{"ID":"x0c0s0b0n0"}]}`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	for _, f := range []string{"json", "json-pretty", "yaml"} {
		res := runOchamiWithRuntime(t, "smd", "component", "get", "--ignore-config", "--uri", srv.URL, "-F", f)
		if res.err != nil {
			t.Fatalf("format %s: unexpected error: %v (exit %d)", f, res.err, res.exitCode)
		}
	}
}

// TestSMDComponentDelete_AllConfirm verifies "delete --all" prompts and, on "y",
// issues a DELETE to the collection endpoint.
func TestSMDComponentDelete_AllConfirm(t *testing.T) {
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
	if gotMethod != http.MethodDelete || gotPath != "/State/Components" {
		t.Errorf("request = %s %s, want DELETE /State/Components", gotMethod, gotPath)
	}
}

// TestSMDComponentDelete_Abort verifies answering "n" aborts without a request.
func TestSMDComponentDelete_Abort(t *testing.T) {
	var deletes int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deletes++
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithInputAndRuntime(t, "n\n",
		"smd", "component", "delete", "--ignore-config", "--uri", srv.URL, "--token", "t", "x3000c1s7b56n0")
	if res.err != nil {
		t.Fatalf("unexpected error on abort: %v (exit %d)", res.err, res.exitCode)
	}
	if deletes != 0 {
		t.Errorf("DELETE count = %d, want 0", deletes)
	}
}
