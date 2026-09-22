// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestPCSStatusShow_Success verifies "pcs status show <xname>" issues GET /power-status
// and prints the first status entry.
func TestPCSStatusShow_Success(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Write([]byte(`{"status":[{"xname":"x3000c0s15b0","powerState":"on"}]}`))
	}))
	defer srv.Close()

	res := runOchami(t, "pcs", "status", "show", "--ignore-config", "--uri", srv.URL, "x3000c0s15b0")

	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotPath != "/power-status" {
		t.Errorf("path = %q, want /power-status", gotPath)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("method = %q, want GET", gotMethod)
	}
	if !strings.Contains(res.stdout, "x3000c0s15b0") {
		t.Errorf("stdout = %q, want it to contain the xname", res.stdout)
	}
}
