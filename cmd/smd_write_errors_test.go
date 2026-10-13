// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// TestSMDGroupMemberAdd_HTTPError verifies an unsuccessful HTTP response
// resolves to CodeHTTP.
func TestSMDGroupMemberAdd_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "conflict", http.StatusConflict)
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "group", "member", "add", "compute", "x0c0s0b0n0",
		"--ignore-config", "--uri", srv.URL, "--token", "t")
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
	tests := []struct {
		name    string
		command string
		payload string
		wantMsg string
	}{
		{name: "interface", command: "iface", payload: `[]`, wantMsg: "payload contained no ethernet interfaces to delete"},
		{name: "group", command: "group", payload: `[]`, wantMsg: "payload contained no groups to delete"},
		{name: "redfish endpoint", command: "rfe", payload: `{"RedfishEndpoints":[]}`, wantMsg: "payload contained no redfish endpoints to delete"},
		{name: "component endpoint", command: "compep", payload: `[]`, wantMsg: "payload contained no component endpoints to delete"},
		{name: "component", command: "component", payload: `{"Components":[]}`, wantMsg: "payload contained no components to delete"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var deletes int
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodDelete {
					deletes++
				}
			}))
			defer srv.Close()

			res := runOchami(t, "smd", tc.command, "delete", "--ignore-config",
				"--uri", srv.URL, "--token", "t", "--no-confirm", "-d", tc.payload)
			if res.err == nil {
				t.Fatal("expected an error, got nil")
			}
			if res.exitCode != cli.CodeUsage {
				t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeUsage, cli.CodeName(cli.CodeUsage))
			}
			if !strings.Contains(res.err.Error(), tc.wantMsg) {
				t.Errorf("error = %q, want it to contain %q", res.err.Error(), tc.wantMsg)
			}
			if deletes != 0 {
				t.Errorf("DELETE count = %d, want 0 (no request should be made for an empty payload)", deletes)
			}
		})
	}
}
