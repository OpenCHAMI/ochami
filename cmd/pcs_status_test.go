// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// pcs_status_test.go exercises the "pcs status" verbs (list, show) end-to-end
// against an httptest.Server, including the list filters. Error arms are
// covered in pcs_status_errors_test.go.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	pcs_status "github.com/openchami/ochami/cmd/pcs/status"
)

// TestPCSStatusList_Success verifies "pcs status list" issues GET /power-status.
func TestPCSStatusList_Success(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.Write([]byte(`{"status":[]}`))
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "pcs", "status", "list", "--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("method = %q, want GET", gotMethod)
	}
	if gotPath != "/power-status" {
		t.Errorf("path = %q, want /power-status", gotPath)
	}
}

// TestPCSStatusList_WithFilters verifies xname and power/mgmt filters are encoded
// in the query string.
func TestPCSStatusList_WithFilters(t *testing.T) {
	t.Parallel()

	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Write([]byte(`{"status":[]}`))
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "pcs", "status", "list", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"--xname", "x0c0s0b0n0", "--power-filter", "on", "--mgmt-filter", "available")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	for _, want := range []string{"xname=x0c0s0b0n0", "powerStateFilter=on", "managementStateFilter=available"} {
		if !strings.Contains(gotQuery, want) {
			t.Errorf("query = %q, want it to contain %q", gotQuery, want)
		}
	}
}

// TestPCSStatusShow_Success verifies "pcs status show <xname>" issues GET /power-status
// and prints the first status entry.
func TestPCSStatusShow_Success(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Write([]byte(`{"status":[{"xname":"x3000c0s15b0","powerState":"on"}]}`))
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "pcs", "status", "show", "--uri", srv.URL, "x3000c0s15b0")

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

// TestPCSStatusList_FlagStateIsLocal verifies one "pcs status list" invocation
// cannot change the default --power-filter of a subsequently constructed
// command (xnames, powerFilter, and mgmtFilter are command-local variables,
// not package-level ones, for this reason).
func TestPCSStatusList_FlagStateIsLocal(t *testing.T) {
	t.Parallel()

	first := pcs_status.NewCmd()
	firstList, _, err := first.Find([]string{"list"})
	if err != nil {
		t.Fatalf("find first list command: %v", err)
	}
	if err := firstList.Flags().Set("power-filter", "on"); err != nil {
		t.Fatalf("set first power-filter: %v", err)
	}

	second := pcs_status.NewCmd()
	secondList, _, err := second.Find([]string{"list"})
	if err != nil {
		t.Fatalf("find second list command: %v", err)
	}
	if got := secondList.Flags().Lookup("power-filter").Value.String(); got != "" {
		t.Errorf("second command power-filter = %q, want default empty", got)
	}
}
