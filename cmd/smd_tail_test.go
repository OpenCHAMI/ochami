// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// smd_tail_test.go covers additional branch families of "smd component" (get by
// selector, formats, delete --all/confirm) and "smd group member" (add, delete,
// get, set with per-item aggregation and confirmation).

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// TestSMDComponentGetByXname verifies "get --xname" targets the by-xname
// endpoint. The command validates the token, so a real JWT is supplied.
func TestSMDComponentGetByXname(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"ID":"x0c0s0b0n0"}`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "component", "get", "--ignore-config", "--uri", srv.URL,
		"--token", validToken(t), "--xname", "x0c0s0b0n0")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.Contains(gotPath, "x0c0s0b0n0") {
		t.Errorf("path = %q, want it to reference the xname", gotPath)
	}
}

// TestSMDComponentGetByNID verifies "get --nid" targets the ByNID endpoint.
func TestSMDComponentGetByNID(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"NID":1}`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "component", "get", "--ignore-config", "--uri", srv.URL,
		"--token", validToken(t), "--nid", "1")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.Contains(gotPath, "ByNID") {
		t.Errorf("path = %q, want it to reference ByNID", gotPath)
	}
}

// TestSMDComponentGetFormats verifies output-format variants of get-all.
func TestSMDComponentGetFormats(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"Components":[{"ID":"x0c0s0b0n0"}]}`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	for _, f := range []string{"json", "json-pretty", "yaml"} {
		res := runOchami(t, "smd", "component", "get", "--ignore-config", "--uri", srv.URL, "-F", f)
		if res.err != nil {
			t.Fatalf("format %s: unexpected error: %v (exit %d)", f, res.err, res.exitCode)
		}
	}
}

// TestSMDComponentDeleteAllConfirm verifies "delete --all" prompts and, on "y",
// issues a DELETE to the collection endpoint.
func TestSMDComponentDeleteAllConfirm(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			gotMethod, gotPath = r.Method, r.URL.Path
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithInput(t, "y\n",
		"smd", "component", "delete", "--ignore-config", "--uri", srv.URL, "--token", "t", "--all")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodDelete || gotPath != "/State/Components" {
		t.Errorf("request = %s %s, want DELETE /State/Components", gotMethod, gotPath)
	}
}

// TestSMDComponentDeleteAbort verifies answering "n" aborts without a request.
func TestSMDComponentDeleteAbort(t *testing.T) {
	var deletes int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deletes++
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithInput(t, "n\n",
		"smd", "component", "delete", "--ignore-config", "--uri", srv.URL, "--token", "t", "x3000c1s7b56n0")
	if res.err != nil {
		t.Fatalf("unexpected error on abort: %v (exit %d)", res.err, res.exitCode)
	}
	if deletes != 0 {
		t.Errorf("DELETE count = %d, want 0", deletes)
	}
}

// TestSMDGroupMemberAddMultiple verifies "group member add <label> <comp>..."
// issues a POST per component.
func TestSMDGroupMemberAddMultiple(t *testing.T) {
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

// TestSMDGroupMemberAddHTTPErrorAggregate verifies a failing member add resolves
// to CodeHTTP via the aggregate.
func TestSMDGroupMemberAddHTTPErrorAggregate(t *testing.T) {
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

// TestSMDGroupMemberDeleteConfirm verifies "group member delete" prompts and, on
// "y", issues DELETEs.
func TestSMDGroupMemberDeleteConfirm(t *testing.T) {
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

// TestSMDGroupMemberDeleteAbort verifies answering "n" aborts without a request.
func TestSMDGroupMemberDeleteAbort(t *testing.T) {
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

// TestSMDGroupMemberGetHTTPError verifies a failing member get resolves to
// CodeHTTP.
func TestSMDGroupMemberGetHTTPError(t *testing.T) {
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

// TestSMDGroupMemberSetHTTPError verifies a failing member set resolves to
// CodeHTTP.
func TestSMDGroupMemberSetHTTPError(t *testing.T) {
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

// TestSMDGroupMemberDeleteMultiple verifies "member delete --no-confirm" issues
// a DELETE per component.
func TestSMDGroupMemberDeleteMultiple(t *testing.T) {
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
