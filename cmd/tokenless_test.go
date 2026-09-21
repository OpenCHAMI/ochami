// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// tokenless_test.go checks that commands whose requests carry no access token
// run without one, even when the cluster enables authentication.

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

// TestTokenlessCommands_SkipTokenCheck verifies that each listed command, none
// of which sends an access token, succeeds against a cluster with enable-auth
// set when no token is available, and sends its request without an
// Authorization header.
func TestTokenlessCommands_SkipTokenCheck(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "bss boot script get", args: []string{"bss", "boot", "script", "get", "--xname", "x0c0s0b0n0"}},
		{name: "bss dumpstate", args: []string{"bss", "dumpstate"}},
		{name: "bss history", args: []string{"bss", "history"}},
		{name: "bss hosts get", args: []string{"bss", "hosts", "get"}},
		{name: "bss status", args: []string{"bss", "status"}},
		{name: "bss service status", args: []string{"bss", "service", "status"}},
		{name: "bss service version", args: []string{"bss", "service", "version"}},
		{name: "smd status", args: []string{"smd", "status"}},
		{name: "smd service status", args: []string{"smd", "service", "status"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var requests atomic.Int32
			var sawAuth atomic.Bool
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if r.Header.Get("Authorization") != "" {
					sawAuth.Store(true)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{}`))
			}))
			defer srv.Close()

			cfg := writeTempConfig(t, `default-cluster: demo
clusters:
- name: demo
  cluster:
    uri: `+srv.URL+`
    enable-auth: true
`)
			res := runOchamiWithRuntime(t, append([]string{"--config", cfg}, tc.args...)...)

			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
			if requests.Load() == 0 {
				t.Error("service received no request")
			}
			if sawAuth.Load() {
				t.Error("request carried an Authorization header")
			}
		})
	}
}
