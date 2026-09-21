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

// TestBootAdd_HTTPError verifies that "boot <type> add" fails with CodeHTTP for
// an unsuccessful HTTP response, for every boot resource type.
func TestBootAdd_HTTPError(t *testing.T) {
	for _, typ := range []string{"config", "node", "bmc"} {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "bad request", http.StatusBadRequest)
			}))
			defer srv.Close()

			res := runOchamiWithRuntime(t, "boot", typ, "add",
				"--ignore-config", "--uri", srv.URL, "--token", "faketoken",
				"-d", bootAddPayload(typ))
			if res.err == nil {
				t.Fatal("expected an error, got nil")
			}
			if res.exitCode != cli.CodeHTTP {
				t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
			}
		})
	}
}
