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

func TestSMDDelete_PayloadEmptyRejected(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    []string
		payload string
		wantMsg string
	}{
		{"group delete", []string{"smd", "group", "delete"}, `[]`, "payload contained no groups to delete"},
		{"compep delete", []string{"smd", "compep", "delete"}, `[]`, "payload contained no component endpoints to delete"},
		{"iface delete", []string{"smd", "iface", "delete"}, `[]`, "payload contained no ethernet interfaces to delete"},
		{"rfe delete", []string{"smd", "rfe", "delete"}, `{"RedfishEndpoints":[]}`, "payload contained no redfish endpoints to delete"},
		{"component delete", []string{"smd", "component", "delete"}, `{"Components":[]}`, "payload contained no components to delete"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var deletes int
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodDelete {
					deletes++
				}
			}))
			defer srv.Close()

			args := append(append([]string{}, tc.args...), "--uri", srv.URL, "--token", "t",
				"--no-confirm", "-d", tc.payload)
			res := runOchamiWithRuntime(t, args...)
			if res.err == nil {
				t.Fatal("expected an error for an empty payload, got nil")
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
