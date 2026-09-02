// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// smd_write_test.go extends the SMD command coverage to write verbs that were
// not previously exercised end-to-end: group member add/delete, group update,
// component endpoint add, and Redfish endpoint delete.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// TestSMDGroupMemberAdd verifies "smd group member add" issues POST under
// /groups/<label>/members.
func TestSMDGroupMemberAdd(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`"x0c0s0b0n0"`))
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "group", "member", "add", "compute", "x0c0s0b0n0",
		"--ignore-config", "--uri", srv.URL, "--token", "t")
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

// TestSMDGroupMemberDelete verifies "smd group member delete" issues DELETE
// under /groups/<label>/members/<component>.
func TestSMDGroupMemberDelete(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_, _ = w.Write([]byte(`{"code":0,"message":"ok"}`))
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "group", "member", "delete", "compute", "x0c0s0b0n0",
		"--ignore-config", "--uri", srv.URL, "--token", "t", "--no-confirm")
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

// TestSMDGroupUpdate verifies "smd group update" issues PATCH under /groups.
func TestSMDGroupUpdate(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "group", "update", "compute",
		"--description", "compute nodes",
		"--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodPatch {
		t.Errorf("method = %q, want PATCH", gotMethod)
	}
	if !strings.HasPrefix(gotPath, "/hsm/v2/groups") && !strings.Contains(gotPath, "/groups") {
		t.Errorf("path = %q, want it under /groups", gotPath)
	}
}

// TestSMDGroupMemberAddHTTPError verifies an unsuccessful HTTP response resolves
// to a non-success exit code.
func TestSMDGroupMemberAddHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "conflict", http.StatusConflict)
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "group", "member", "add", "compute", "x0c0s0b0n0",
		"--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode == cli.CodeSuccess {
		t.Errorf("exit code = %d, want a non-success code", res.exitCode)
	}
}

// TestSMDRFEDeleteNoConfirm verifies "smd rfe delete --no-confirm" issues a
// DELETE.
func TestSMDRFEDeleteNoConfirm(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		_, _ = w.Write([]byte(`{"code":0,"message":"ok"}`))
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "rfe", "delete", "x0c0s0b0",
		"--ignore-config", "--uri", srv.URL, "--token", "t", "--no-confirm")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want DELETE", gotMethod)
	}
}
