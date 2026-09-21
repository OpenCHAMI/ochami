// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// smd_group_member_test.go exercises the "smd group member" commands
// end-to-end against an httptest.Server: add, delete, get, and set, including
// per-item aggregation, confirmation prompts, and network/HTTP error mapping.

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// TestSMDGroupMemberAdd_Multiple verifies "group member add <label> <comp>..."
// issues a POST per component.
func TestSMDGroupMemberAdd_Multiple(t *testing.T) {
	var posts int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			posts++
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "group", "member", "add", "--uri", srv.URL, "--token", "t",
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

	res := runOchamiWithRuntime(t, "smd", "group", "member", "add", "--uri", srv.URL, "--token", "t",
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

	res := runOchamiWithInputAndRuntime(t, "y\n",
		"smd", "group", "member", "delete", "--uri", srv.URL, "--token", "t",
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

	res := runOchamiWithInputAndRuntime(t, "n\n",
		"smd", "group", "member", "delete", "--uri", srv.URL, "--token", "t",
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

	res := runOchamiWithRuntime(t, "smd", "group", "member", "get", "--uri", srv.URL, "--token", "t",
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

	res := runOchamiWithRuntime(t, "smd", "group", "member", "set", "--uri", srv.URL, "--token", "t",
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
	res := runOchamiWithRuntime(t, "smd", "group", "member", "add", "--uri", url, "--token", "t",
		"compute", "x0c0s0b0n0")
	if res.err == nil || res.exitCode == cli.CodeSuccess {
		t.Errorf("member add network: err=%v exit=%d, want a non-success code", res.err, res.exitCode)
	}
	// delete (per-item aggregation surfaces a non-success exit)
	res = runOchamiWithRuntime(t, "smd", "group", "member", "delete", "--uri", url, "--token", "t",
		"--no-confirm", "compute", "x0c0s0b0n0")
	if res.err == nil || res.exitCode == cli.CodeSuccess {
		t.Errorf("member delete network: err=%v exit=%d, want a non-success code", res.err, res.exitCode)
	}
	// get (single request maps transport failure to CodeNetwork)
	res = runOchamiWithRuntime(t, "smd", "group", "member", "get", "--uri", url, "--token", "t", "compute")
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

	res := runOchamiWithRuntime(t, "smd", "group", "member", "delete", "--uri", srv.URL, "--token", "t",
		"--no-confirm", "compute", "x0c0s0b0n0", "x0c0s0b0n1")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if deletes != 2 {
		t.Errorf("DELETE count = %d, want 2", deletes)
	}
}
