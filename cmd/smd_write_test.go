// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// smd_write_test.go covers SMD write commands end-to-end: group member
// add/delete, group update, and Redfish endpoint delete. Error arms are
// covered in smd_write_errors_test.go.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestSMDGroupMemberAdd_Success verifies "smd group member add" issues POST under
// /groups/<label>/members.
func TestSMDGroupMemberAdd_Success(t *testing.T) {
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

// TestSMDGroupUpdate_Success verifies "smd group update" issues PATCH under /groups.
func TestSMDGroupUpdate_Success(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "--ignore-config", "group", "update", "compute",
		"--description", "compute nodes",
		"--uri", srv.URL, "--token", "t")
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

// TestSMDRFEDelete_NoConfirm verifies "smd rfe delete --no-confirm" issues a
// DELETE.
func TestSMDRFEDelete_NoConfirm(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.Write([]byte(`{"code":0,"message":"ok"}`))
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "--ignore-config", "rfe", "delete", "x0c0s0b0",
		"--uri", srv.URL, "--token", "t", "--no-confirm")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want DELETE", gotMethod)
	}
}
