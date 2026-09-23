// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// TestSMDGroupMemberAdd_HTTPError verifies an unsuccessful HTTP response
// resolves to CodeHTTP.
func TestSMDGroupMemberAdd_HTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "conflict", http.StatusConflict)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "--ignore-config", "group", "member", "add", "compute", "x0c0s0b0n0",
		"--uri", srv.URL, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestSMDDelete_RejectsEmptyData verifies an explicit empty payload cannot
// turn a requested deletion into a silent no-op.
func TestSMDDelete_RejectsEmptyData(t *testing.T) {
	t.Parallel()

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
			t.Parallel()
			res := runOchamiWithRuntime(t, "smd", "--ignore-config", tc.command, "delete",
				"--uri", "http://127.0.0.1:1", "--token", "t", "--no-confirm", "-d", tc.payload)
			if res.err == nil {
				t.Fatal("expected an error, got nil")
			}
			if res.exitCode != cli.CodeUsage {
				t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeUsage, cli.CodeName(cli.CodeUsage))
			}
		})
	}
}
