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

// TestSMDGroupGet_Filters verifies the query builder emits group/tag query
// parameters.
func TestSMDGroupGet_Filters(t *testing.T) {
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

// TestSMDGroupGet_Formats verifies the output-format variants.
func TestSMDGroupGet_Formats(t *testing.T) {
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

// TestSMDGroupGet_HTTPError verifies an unsuccessful HTTP response resolves to
// CodeHTTP.
func TestSMDGroupGet_HTTPError(t *testing.T) {
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

// TestSMDGroupGet_NetworkError verifies pointing at a closed port resolves to
// CodeNetwork.
func TestSMDGroupGet_NetworkError(t *testing.T) {
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

// TestSMDGroupAdd_WithOptionalFlags verifies "add <label>" with
// description/tag/exclusive-group/member issues a POST.
func TestSMDGroupAdd_WithOptionalFlags(t *testing.T) {
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

// TestSMDGroupAdd_ByData verifies "add -d <payload>" issues a POST.
func TestSMDGroupAdd_ByData(t *testing.T) {
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

// TestSMDGroupAdd_HTTPError verifies a failing POST resolves to CodeHTTP.
func TestSMDGroupAdd_HTTPError(t *testing.T) {
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

// TestSMDGroupUpdate_ByFlags verifies "update <label> --description/--tag" issues
// a PATCH.
func TestSMDGroupUpdate_ByFlags(t *testing.T) {
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

// TestSMDGroupUpdate_MissingFields verifies "update <label>" with no
// description/tag is a usage error.
func TestSMDGroupUpdate_MissingFields(t *testing.T) {
	res := runOchami(t, "smd", "group", "update", "--ignore-config", "--uri", "http://127.0.0.1:1", "--token", "t",
		"compute")
	if res.err == nil {
		t.Fatal("expected a usage error, got nil")
	}
	if res.exitCode != cli.CodeUsage {
		t.Errorf("exit code = %d, want %d (CodeUsage)", res.exitCode, cli.CodeUsage)
	}
}

// TestSMDGroupUpdate_HTTPError verifies a failing PATCH resolves to CodeHTTP.
func TestSMDGroupUpdate_HTTPError(t *testing.T) {
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

// TestSMDGroupDelete_ByLabels verifies "delete --no-confirm <label>..." issues a
// DELETE per group.
func TestSMDGroupDelete_ByLabels(t *testing.T) {
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

// TestSMDGroupDelete_ByData verifies labels in a payload drive DELETE requests.
func TestSMDGroupDelete_ByData(t *testing.T) {
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

// TestSMDGroupDelete_HTTPError verifies a failing DELETE resolves to CodeHTTP via
// the aggregate.
func TestSMDGroupDelete_HTTPError(t *testing.T) {
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

// TestSMDGroupMembership_Filters verifies the membership filter builder emits
// slice and scalar query parameters.
func TestSMDGroupMembership_Filters(t *testing.T) {
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

// TestSMDGroupMembership_HTTPError verifies an unsuccessful HTTP response
// resolves to CodeHTTP.
func TestSMDGroupMembership_HTTPError(t *testing.T) {
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

// TestSMDGroupAdd_DataWithExtraArgs verifies "add -d <payload> <label>" ignores
// the extra argument (warning arm) and still POSTs.
func TestSMDGroupAdd_DataWithExtraArgs(t *testing.T) {
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

// TestSMDGroupAdd_BadData verifies "add -d <malformed>" resolves to CodePayload.
func TestSMDGroupAdd_BadData(t *testing.T) {
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

// TestSMDGroupUpdate_DataWithExtraArgs verifies "update -d <payload> <label>"
// ignores the extra argument and PATCHes.
func TestSMDGroupUpdate_DataWithExtraArgs(t *testing.T) {
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

// TestSMDGroupDelete_DataWithExtraArgs verifies "delete --no-confirm -d <payload>
// <label>" ignores the extra argument.
func TestSMDGroupDelete_DataWithExtraArgs(t *testing.T) {
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

// TestSMDGroupAdd_NetworkError verifies a closed port surfaces the CodeHTTP/
// network aggregate for group add.
func TestSMDGroupAdd_NetworkError(t *testing.T) {
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
func TestSMDGroupMemberAdd_Multiple(t *testing.T) {
	var posts int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			posts++
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "group", "member", "add", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"compute", "x0c0s0b0n0", "x0c0s0b0n1")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if posts != 2 {
		t.Errorf("POST count = %d, want 2", posts)
	}
}

// TestSMDGroupMemberAdd_HTTPErrorAggregate verifies a failing member add resolves
// to CodeHTTP via the aggregate.
func TestSMDGroupMemberAdd_HTTPErrorAggregate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "group", "member", "add", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"compute", "x0c0s0b0n0")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestSMDGroupMemberDelete_Confirm verifies "group member delete" prompts and, on
// "y", issues DELETEs.
func TestSMDGroupMemberDelete_Confirm(t *testing.T) {
	var deletes int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deletes++
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithInput(t, "y\n",
		"smd", "group", "member", "delete", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"compute", "x0c0s0b0n0")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if deletes != 1 {
		t.Errorf("DELETE count = %d, want 1", deletes)
	}
}

// TestSMDGroupMemberDelete_Abort verifies answering "n" aborts without a request.
func TestSMDGroupMemberDelete_Abort(t *testing.T) {
	var deletes int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deletes++
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithInput(t, "n\n",
		"smd", "group", "member", "delete", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"compute", "x0c0s0b0n0")
	if res.err != nil {
		t.Fatalf("unexpected error on abort: %v (exit %d)", res.err, res.exitCode)
	}
	if deletes != 0 {
		t.Errorf("DELETE count = %d, want 0", deletes)
	}
}

// TestSMDGroupMemberGet_HTTPError verifies a failing member get resolves to
// CodeHTTP.
func TestSMDGroupMemberGet_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "group", "member", "get", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"compute")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestSMDGroupMemberSet_HTTPError verifies a failing member set resolves to
// CodeHTTP.
func TestSMDGroupMemberSet_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "group", "member", "set", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"compute", "x0c0s0b0n0")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestSMDGroupMemberNetworkErrors verifies member verbs resolve a closed port to
// CodeNetwork.
func TestSMDGroupMemberNetworkErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	// add (per-item aggregation surfaces a non-success exit)
	res := runOchami(t, "smd", "group", "member", "add", "--ignore-config", "--uri", url, "--token", "t",
		"compute", "x0c0s0b0n0")
	if res.err == nil || res.exitCode == cli.CodeSuccess {
		t.Errorf("member add network: err=%v exit=%d, want a non-success code", res.err, res.exitCode)
	}
	// delete (per-item aggregation surfaces a non-success exit)
	res = runOchami(t, "smd", "group", "member", "delete", "--ignore-config", "--uri", url, "--token", "t",
		"--no-confirm", "compute", "x0c0s0b0n0")
	if res.err == nil || res.exitCode == cli.CodeSuccess {
		t.Errorf("member delete network: err=%v exit=%d, want a non-success code", res.err, res.exitCode)
	}
	// get (single request maps transport failure to CodeNetwork)
	res = runOchami(t, "smd", "group", "member", "get", "--ignore-config", "--uri", url, "--token", "t", "compute")
	if res.err == nil || res.exitCode != cli.CodeNetwork {
		t.Errorf("member get network: err=%v exit=%d, want CodeNetwork", res.err, res.exitCode)
	}
}

// TestSMDGroupMemberDelete_Multiple verifies "member delete --no-confirm" issues
// a DELETE per component.
func TestSMDGroupMemberDelete_Multiple(t *testing.T) {
	var deletes int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deletes++
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "group", "member", "delete", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"--no-confirm", "compute", "x0c0s0b0n0", "x0c0s0b0n1")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if deletes != 2 {
		t.Errorf("DELETE count = %d, want 2", deletes)
	}
}
