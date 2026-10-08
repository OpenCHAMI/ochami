// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// smd_group_test.go covers the success paths of the "smd group" verbs (get,
// add, update, delete, membership): query-builder arms, output-format variants,
// flag/payload input variants (including extra arguments alongside -d), and
// the membership filter builder. HTTP and network error mapping and rejected
// input are covered in smd_group_errors_test.go.

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// TestSMDGroupGet_Filters verifies the query builder emits group/tag query
// parameters.
func TestSMDGroupGet_Filters(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    []string
		wantKey string
		wantVal string
	}{
		{"name", []string{"--name", "compute"}, "group", "compute"},
		{"tag", []string{"--tag", "prod"}, "tag", "prod"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotQuery url.Values
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotQuery = r.URL.Query()
				w.Write([]byte(`[]`))
			}))
			defer srv.Close()

			args := append([]string{"--ignore-config", "smd", "group", "get", "--uri", srv.URL, "--token", "t"}, tc.args...)
			res := runOchamiWithRuntime(t, args...)
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
			if got := gotQuery.Get(tc.wantKey); got != tc.wantVal {
				t.Errorf("query %s = %q, want %q", tc.wantKey, got, tc.wantVal)
			}
		})
	}
}

// TestSMDGroupGet_Formats verifies the output-format variants.
func TestSMDGroupGet_Formats(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[{"label":"compute"}]`))
	}))
	defer srv.Close()

	for _, f := range []string{"json", "json-pretty", "yaml"} {
		res := runOchamiWithRuntime(t, "smd", "group", "get", "--uri", srv.URL, "--token", "t", "-F", f)
		if res.err != nil {
			t.Fatalf("format %s: unexpected error: %v (exit %d)", f, res.err, res.exitCode)
		}
		assertFormattedOutput(t, f, res.stdout, "label", "compute")
	}
}

// TestSMDGroupAdd_WithOptionalFlags verifies "add <label>" with
// description/tag/exclusive-group/member issues a POST.
func TestSMDGroupAdd_WithOptionalFlags(t *testing.T) {
	t.Parallel()

	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "group", "add", "--uri", srv.URL, "--token", "t",
		"--description", "The compute group", "--tag", "prod", "--exclusive-group", "excl",
		"--member", "x0c0s0b0n0", "compute")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
}

// TestSMDGroupAdd_ByData verifies "add -d <payload>" issues a POST.
func TestSMDGroupAdd_ByData(t *testing.T) {
	t.Parallel()

	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "group", "add", "--uri", srv.URL, "--token", "t",
		"-d", `[{"label":"compute"}]`)
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
}

// TestSMDGroupUpdate_ByFlags verifies "update <label> --description/--tag" issues
// a PATCH.
func TestSMDGroupUpdate_ByFlags(t *testing.T) {
	t.Parallel()

	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "group", "update", "--uri", srv.URL, "--token", "t",
		"--description", "updated", "--tag", "new", "compute")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodPatch {
		t.Errorf("method = %q, want PATCH", gotMethod)
	}
}

// TestSMDGroupDelete_ByLabels verifies "delete --no-confirm <label>..." issues a
// DELETE per group.
func TestSMDGroupDelete_ByLabels(t *testing.T) {
	t.Parallel()

	var deletes int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deletes++
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "group", "delete", "--uri", srv.URL, "--token", "t",
		"--no-confirm", "compute", "storage")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if deletes != 2 {
		t.Errorf("DELETE count = %d, want 2", deletes)
	}
}

// TestSMDGroupDelete_ByData verifies labels in a payload drive DELETE requests.
func TestSMDGroupDelete_ByData(t *testing.T) {
	t.Parallel()

	var deletes int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deletes++
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "group", "delete", "--uri", srv.URL, "--token", "t",
		"--no-confirm", "-d", `[{"label":"compute"}]`)
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if deletes != 1 {
		t.Errorf("DELETE count = %d, want 1", deletes)
	}
}

// TestSMDGroupMembership_Filters verifies the membership filter builder emits
// slice and scalar query parameters.
func TestSMDGroupMembership_Filters(t *testing.T) {
	t.Parallel()

	var gotQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "group", "membership", "--uri", srv.URL, "--token", "t",
		"--type", "Node", "--arch", "X86", "--nid-start", "1000", "--nid-end", "2000")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotQuery.Get("type") != "Node" {
		t.Errorf("query type = %q, want Node", gotQuery.Get("type"))
	}
	if gotQuery.Get("nid_start") != "1000" {
		t.Errorf("query nid_start = %q, want 1000", gotQuery.Get("nid_start"))
	}
}

// TestSMDGroupAdd_DataWithExtraArgs verifies that "smd group add -d <payload>
// <label>" ignores the extra argument, logs a warning about it, and sends a
// POST.
func TestSMDGroupAdd_DataWithExtraArgs(t *testing.T) {
	t.Parallel()

	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "group", "add", "--uri", srv.URL, "--token", "t",
		"-d", `[{"label":"compute"}]`, "ignored-arg")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.Contains(res.stdout, "raw data passed, ignoring CLI configuration") {
		t.Errorf("output = %q, want a warning that the argument is ignored", res.stdout)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
}

// TestSMDGroupUpdate_DataWithExtraArgs verifies that "update -d <payload>
// <label>" ignores the extra argument and sends a PATCH for the payload's
// group.
func TestSMDGroupUpdate_DataWithExtraArgs(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "group", "update", "--uri", srv.URL, "--token", "t",
		"-d", `[{"label":"compute","description":"d"}]`, "ignored-arg")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodPatch || gotPath != "/groups/compute" {
		t.Errorf("request = %s %s, want PATCH /groups/compute (the payload's group, not the extra argument)", gotMethod, gotPath)
	}
}

// TestSMDGroupDelete_DataWithExtraArgs verifies that "delete --no-confirm -d
// <payload> <label>" ignores the extra argument and sends a DELETE for the
// payload's group.
func TestSMDGroupDelete_DataWithExtraArgs(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "group", "delete", "--uri", srv.URL, "--token", "t",
		"--no-confirm", "-d", `[{"label":"compute"}]`, "ignored-arg")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodDelete || gotPath != "/groups/compute" {
		t.Errorf("request = %s %s, want DELETE /groups/compute (the payload's group, not the extra argument)", gotMethod, gotPath)
	}
}
