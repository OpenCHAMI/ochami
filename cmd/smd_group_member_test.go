// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// smd_group_member_test.go exercises the success paths of the "smd group
// member" add and delete commands end-to-end against an httptest.Server: one
// request per member, and the delete confirmation prompt. Error arms for every
// member verb are covered in smd_group_member_errors_test.go.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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

	res := runOchami(t, "smd", "group", "member", "add", "--ignore-config", "--uri", srv.URL, "--token", "t",
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
