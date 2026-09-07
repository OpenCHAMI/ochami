// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// smd_group_test.go covers the branch families of the "smd group" verbs (get,
// add, update, delete, membership): query-builder arms, output-format variants,
// HTTP and network error mapping, flag/payload input variants, the delete
// confirmation and per-item aggregation, and the membership filter builder.

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// TestSMDGroupGetFilters verifies the query builder emits group/tag query
// parameters.
func TestSMDGroupGetFilters(t *testing.T) {
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
				_, _ = w.Write([]byte(`[]`))
			}))
			defer srv.Close()

			args := append([]string{"smd", "group", "get", "--ignore-config", "--uri", srv.URL, "--token", "t"}, tc.args...)
			res := runOchami(t, args...)
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
			if got := gotQuery.Get(tc.wantKey); got != tc.wantVal {
				t.Errorf("query %s = %q, want %q", tc.wantKey, got, tc.wantVal)
			}
		})
	}
}

// TestSMDGroupGetFormats verifies the output-format variants.
func TestSMDGroupGetFormats(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"label":"compute"}]`))
	}))
	defer srv.Close()

	for _, f := range []string{"json", "json-pretty", "yaml"} {
		res := runOchami(t, "smd", "group", "get", "--ignore-config", "--uri", srv.URL, "--token", "t", "-F", f)
		if res.err != nil {
			t.Fatalf("format %s: unexpected error: %v (exit %d)", f, res.err, res.exitCode)
		}
		if !strings.Contains(res.stdout, "compute") {
			t.Errorf("format %s: stdout = %q, want it to contain the label", f, res.stdout)
		}
	}
}

// TestSMDGroupGetHTTPError verifies an unsuccessful HTTP response resolves to
// CodeHTTP.
func TestSMDGroupGetHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "group", "get", "--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestSMDGroupGetNetworkError verifies pointing at a closed port resolves to
// CodeNetwork.
func TestSMDGroupGetNetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	res := runOchami(t, "smd", "group", "get", "--ignore-config", "--uri", url, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeNetwork {
		t.Errorf("exit code = %d, want %d (CodeNetwork)", res.exitCode, cli.CodeNetwork)
	}
}

// TestSMDGroupAddWithOptionalFlags verifies "add <label>" with
// description/tag/exclusive-group/member issues a POST.
func TestSMDGroupAddWithOptionalFlags(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "group", "add", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"--description", "The compute group", "--tag", "prod", "--exclusive-group", "excl",
		"--member", "x0c0s0b0n0", "compute")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
}

// TestSMDGroupAddByData verifies "add -d <payload>" issues a POST.
func TestSMDGroupAddByData(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "group", "add", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"-d", `[{"label":"compute"}]`)
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
}

// TestSMDGroupAddHTTPError verifies a failing POST resolves to CodeHTTP.
func TestSMDGroupAddHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad", http.StatusBadRequest)
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "group", "add", "--ignore-config", "--uri", srv.URL, "--token", "t", "compute")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestSMDGroupUpdateByFlags verifies "update <label> --description/--tag" issues
// a PATCH.
func TestSMDGroupUpdateByFlags(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "group", "update", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"--description", "updated", "--tag", "new", "compute")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodPatch {
		t.Errorf("method = %q, want PATCH", gotMethod)
	}
}

// TestSMDGroupUpdateMissingFields verifies "update <label>" with no
// description/tag is a usage error.
func TestSMDGroupUpdateMissingFields(t *testing.T) {
	res := runOchami(t, "smd", "group", "update", "--ignore-config", "--uri", "http://127.0.0.1:1", "--token", "t",
		"compute")
	if res.err == nil {
		t.Fatal("expected a usage error, got nil")
	}
	if res.exitCode != cli.CodeUsage {
		t.Errorf("exit code = %d, want %d (CodeUsage)", res.exitCode, cli.CodeUsage)
	}
}

// TestSMDGroupUpdateHTTPError verifies a failing PATCH resolves to CodeHTTP.
func TestSMDGroupUpdateHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad", http.StatusBadRequest)
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "group", "update", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"--description", "updated", "compute")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestSMDGroupDeleteByLabels verifies "delete --no-confirm <label>..." issues a
// DELETE per group.
func TestSMDGroupDeleteByLabels(t *testing.T) {
	var deletes int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deletes++
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "group", "delete", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"--no-confirm", "compute", "storage")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if deletes != 2 {
		t.Errorf("DELETE count = %d, want 2", deletes)
	}
}

// TestSMDGroupDeleteByData verifies labels in a payload drive DELETE requests.
func TestSMDGroupDeleteByData(t *testing.T) {
	var deletes int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deletes++
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "group", "delete", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"--no-confirm", "-d", `[{"label":"compute"}]`)
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if deletes != 1 {
		t.Errorf("DELETE count = %d, want 1", deletes)
	}
}

// TestSMDGroupDeleteHTTPError verifies a failing DELETE resolves to CodeHTTP via
// the aggregate.
func TestSMDGroupDeleteHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "group", "delete", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"--no-confirm", "compute")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestSMDGroupMembershipFilters verifies the membership filter builder emits
// slice and scalar query parameters.
func TestSMDGroupMembershipFilters(t *testing.T) {
	var gotQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "group", "membership", "--ignore-config", "--uri", srv.URL, "--token", "t",
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

// TestSMDGroupMembershipHTTPError verifies an unsuccessful HTTP response
// resolves to CodeHTTP.
func TestSMDGroupMembershipHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "group", "membership", "--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestSMDGroupAddDataWithExtraArgs verifies "add -d <payload> <label>" ignores
// the extra argument (warning arm) and still POSTs.
func TestSMDGroupAddDataWithExtraArgs(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "group", "add", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"-d", `[{"label":"compute"}]`, "ignored-arg")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
}

// TestSMDGroupAddBadData verifies "add -d <malformed>" resolves to CodePayload.
func TestSMDGroupAddBadData(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "group", "add", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"-d", `not json`)
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodePayload {
		t.Errorf("exit code = %d, want %d (CodePayload)", res.exitCode, cli.CodePayload)
	}
}

// TestSMDGroupUpdateDataWithExtraArgs verifies "update -d <payload> <label>"
// ignores the extra argument and PATCHes.
func TestSMDGroupUpdateDataWithExtraArgs(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "group", "update", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"-d", `[{"label":"compute","description":"d"}]`, "ignored-arg")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodPatch {
		t.Errorf("method = %q, want PATCH", gotMethod)
	}
}

// TestSMDGroupDeleteDataWithExtraArgs verifies "delete --no-confirm -d <payload>
// <label>" ignores the extra argument.
func TestSMDGroupDeleteDataWithExtraArgs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "group", "delete", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"--no-confirm", "-d", `[{"label":"compute"}]`, "ignored-arg")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
}

// TestSMDGroupAddNetworkError verifies a closed port surfaces the CodeHTTP/
// network aggregate for group add.
func TestSMDGroupAddNetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	res := runOchami(t, "smd", "group", "add", "--ignore-config", "--uri", url, "--token", "t", "compute")
	if res.err == nil {
		t.Fatal("expected a network error, got nil")
	}
	if res.exitCode == cli.CodeSuccess {
		t.Errorf("exit code = %d, want a non-success code", res.exitCode)
	}
}
