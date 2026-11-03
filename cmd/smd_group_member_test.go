// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// smd_group_member_test.go exercises the success paths of the "smd group
// member" commands (get, add, set, delete) end-to-end against an
// httptest.Server: the request each verb issues, one request per member, and
// the delete confirmation prompt. Error arms for every member verb are covered
// in smd_group_member_errors_test.go.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestSMDGroupMemberAdd_Multiple verifies "group member add <label> <comp>..."
// issues a POST per component.
func TestSMDGroupMemberAdd_Multiple(t *testing.T) {
	t.Parallel()

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

// TestSMDGroupMemberDelete_Confirm verifies "group member delete" prompts and, on
// "y", issues DELETEs.
func TestSMDGroupMemberDelete_Confirm(t *testing.T) {
	t.Parallel()

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
	if !strings.Contains(res.stdout, "Really delete?") {
		t.Errorf("output = %q, want the confirmation prompt", res.stdout)
	}
	if deletes != 1 {
		t.Errorf("DELETE count = %d, want 1", deletes)
	}
}

// TestSMDGroupMemberDelete_Multiple verifies "member delete --no-confirm" issues
// a DELETE per component.
func TestSMDGroupMemberDelete_Multiple(t *testing.T) {
	t.Parallel()

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

// TestSMDGroupMemberGet_Success verifies "smd group member get <label>" issues GET
// /groups/<label>/members.
func TestSMDGroupMemberGet_Success(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "group", "member", "get", "--ignore-config", "--uri", srv.URL, "--token", "t", "compute")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotPath != "/groups/compute/members" {
		t.Errorf("path = %q, want /groups/compute/members", gotPath)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("method = %q, want GET", gotMethod)
	}
}

// TestSMDGroupMemberSet_Success verifies "smd group member set <label> <comp>..." issues
// PUT /groups/<label>/members.
func TestSMDGroupMemberSet_Success(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "group", "member", "set", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"compute", "x0c0s0b0n0")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodPut || gotPath != "/groups/compute/members" {
		t.Errorf("request = %s %s, want PUT /groups/compute/members", gotMethod, gotPath)
	}
}

// TestSMDGroupMemberAdd_Success verifies "smd group member add" issues POST under
// /groups/<label>/members.
func TestSMDGroupMemberAdd_Success(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`"x0c0s0b0n0"`))
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "--ignore-config", "group", "member", "add", "compute", "x0c0s0b0n0",
		"--uri", srv.URL, "--token", "t")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if !strings.Contains(gotPath, "/groups/compute/members") {
		t.Errorf("path = %q, want it under /groups/compute/members", gotPath)
	}
}

// TestSMDGroupMemberDelete_Success verifies "smd group member delete" issues DELETE
// under /groups/<label>/members/<component>.
func TestSMDGroupMemberDelete_Success(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.Write([]byte(`{"code":0,"message":"ok"}`))
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "--ignore-config", "group", "member", "delete", "compute", "x0c0s0b0n0",
		"--uri", srv.URL, "--token", "t", "--no-confirm")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want DELETE", gotMethod)
	}
	if !strings.Contains(gotPath, "/groups/compute/members") {
		t.Errorf("path = %q, want it under /groups/compute/members", gotPath)
	}
}
