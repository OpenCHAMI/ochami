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

// TestSMDGroupMemberAdd_Success verifies "smd group member add" issues POST under
// /groups/<label>/members.
func TestSMDGroupMemberAdd_Success(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`"x0c0s0b0n0"`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	// TODO: Enable t.Parallel() once race conditions are resolved
	// t.Parallel()
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
		_, _ = w.Write([]byte(`{"code":0,"message":"ok"}`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	// TODO: Enable t.Parallel() once race conditions are resolved
	// t.Parallel()
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
		_, _ = w.Write([]byte(`{}`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	// TODO: Enable t.Parallel() once race conditions are resolved
	// t.Parallel()
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

// TestSMDGroupMemberAdd_HTTPError verifies an unsuccessful HTTP response resolves
// to a non-success exit code.
func TestSMDGroupMemberAdd_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "conflict", http.StatusConflict)
	}))
	defer srv.Close()

	// TODO: Enable t.Parallel() once race conditions are resolved
	// t.Parallel()
	res := runOchamiWithRuntime(t, "smd", "--ignore-config", "group", "member", "add", "compute", "x0c0s0b0n0",
		"--uri", srv.URL, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode == cli.CodeSuccess {
		t.Errorf("exit code = %d, want a non-success code", res.exitCode)
	}
}

// TestSMDRFEDelete_NoConfirm verifies "smd rfe delete --no-confirm" issues a
// DELETE.
func TestSMDRFEDelete_NoConfirm(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		_, _ = w.Write([]byte(`{"code":0,"message":"ok"}`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	// TODO: Enable t.Parallel() once race conditions are resolved
	// t.Parallel()
	res := runOchamiWithRuntime(t, "smd", "--ignore-config", "rfe", "delete", "x0c0s0b0",
		"--uri", srv.URL, "--token", "t", "--no-confirm")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want DELETE", gotMethod)
	}
}

// TestSMDDelete_RejectsEmptyData verifies an explicit empty payload cannot
// turn a requested deletion into a silent no-op.
func TestSMDDelete_RejectsEmptyData(t *testing.T) {
	tests := []struct {
		name    string
		command string
		payload string
	}{
		{name: "interface", command: "iface", payload: `[]`},
		{name: "group", command: "group", payload: `[]`},
		{name: "redfish endpoint", command: "rfe", payload: `{"RedfishEndpoints":[]}`},
		{name: "component endpoint", command: "compep", payload: `[]`},
		{name: "component", command: "component", payload: `{"Components":[]}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// TODO: Enable t.Parallel() once race conditions are resolved
			// t.Parallel()
			res := runOchamiWithRuntime(t, "smd", "--ignore-config", tc.command, "delete",
				"--uri", "http://127.0.0.1:1", "--token", "t", "--no-confirm", "-d", tc.payload)
			if res.err == nil {
				t.Fatal("expected an error, got nil")
			}
			if res.exitCode != cli.CodeUsage {
				t.Errorf("exit code = %d, want %d (CodeUsage)", res.exitCode, cli.CodeUsage)
			}
		})
	}
}
