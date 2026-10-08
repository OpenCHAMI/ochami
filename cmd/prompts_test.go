// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// prompts_test.go covers the interactive confirmation branch of delete
// commands. Without --no-confirm, a delete command prompts the user via
// cli.Ios.LoopYesNo; these tests inject an in-memory stdin (via
// cli.SetIOStream) to drive the "yes" and "no" answers and assert that a
// confirmed delete issues the request while a declined delete aborts cleanly
// without one.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// runOchamiWithInput runs the CLI with a scripted interactive stdin. The prompt
// text the command writes is captured in the returned cmdResult's stdout (the
// harness routes cli.Ios output into the same capture buffer).
func runOchamiWithInput(t *testing.T, input string, args ...string) cmdResult {
	t.Helper()
	return runOchamiWithStdin(t, strings.NewReader(input), args...)
}

// TestDeleteConfirm_Yes verifies that answering "y" to the confirmation prompt
// causes the delete to proceed (a DELETE request is issued) and the command
// exits successfully.
func TestDeleteConfirm_Yes(t *testing.T) {
	var deletes int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deletes++
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithInput(t, "y\n",
		"smd", "group", "delete", "--ignore-config", "--uri", srv.URL, "--token", "t", "compute")

	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if deletes != 1 {
		t.Errorf("DELETE count = %d, want 1", deletes)
	}
	if !strings.Contains(res.stdout, "Really delete?") {
		t.Errorf("stdout = %q, want it to contain the confirmation question", res.stdout)
	}
}

// TestDeleteConfirm_No verifies that answering "n" to the delete confirmation
// prompt declines the delete: no request is sent and the command exits with
// CodeDeclined.
func TestDeleteConfirm_No(t *testing.T) {
	var deletes int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deletes++
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithInput(t, "n\n",
		"smd", "group", "delete", "--ignore-config", "--uri", srv.URL, "--token", "t", "compute")

	if res.exitCode != cli.CodeDeclined {
		t.Fatalf("result = (err %v, exit %d), want %d (%s)", res.err, res.exitCode, cli.CodeDeclined, cli.CodeName(cli.CodeDeclined))
	}
	if deletes != 0 {
		t.Errorf("DELETE count = %d, want 0 (user declined)", deletes)
	}
	if !strings.Contains(res.stdout, "Really delete?") {
		t.Errorf("stdout = %q, want it to contain the confirmation question", res.stdout)
	}
}

// TestDeleteConfirm_YesComponent verifies that "smd component delete" prompts
// for confirmation and, on "y", sends the DELETE.
func TestDeleteConfirm_YesComponent(t *testing.T) {
	var deletes int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deletes++
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithInput(t, "y\n",
		"smd", "component", "delete", "--ignore-config", "--uri", srv.URL, "--token", "t", "x3000c1s7b56n0")

	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.Contains(res.stdout, "Really delete?") {
		t.Errorf("output = %q, want the confirmation prompt", res.stdout)
	}
	if deletes != 1 {
		t.Errorf("DELETE count = %d, want 1", deletes)
	}
}

// TestDeleteConfirm_NoBSS verifies that answering "n" to the confirmation
// prompt of "bss boot params delete" sends no request and exits with
// CodeDeclined.
func TestDeleteConfirm_NoBSS(t *testing.T) {
	var deletes int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deletes++
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithInput(t, "n\n",
		"bss", "boot", "params", "delete", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"--mac", "de:ad:be:ef:00:00", "--kernel", "https://example.com/vmlinuz")

	if res.exitCode != cli.CodeDeclined {
		t.Fatalf("result = (err %v, exit %d), want %d (%s)", res.err, res.exitCode, cli.CodeDeclined, cli.CodeName(cli.CodeDeclined))
	}
	if deletes != 0 {
		t.Errorf("DELETE count = %d, want 0 (user declined)", deletes)
	}
}
