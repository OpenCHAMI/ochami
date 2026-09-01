// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
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
	testStdin = strings.NewReader(input)
	return runOchami(t, args...)
}

// TestDeleteConfirmYes verifies that answering "y" to the confirmation prompt
// causes the delete to proceed (a DELETE request is issued) and the command
// exits successfully.
func TestDeleteConfirmYes(t *testing.T) {
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

// TestDeleteConfirmNo verifies that answering "n" aborts the delete: no request
// is issued and the command exits 0 (user-abort is not an error).
func TestDeleteConfirmNo(t *testing.T) {
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

	if res.err != nil {
		t.Fatalf("unexpected error on abort: %v (exit %d)", res.err, res.exitCode)
	}
	if res.exitCode != cli.CodeSuccess {
		t.Errorf("exit code = %d, want %d (CodeSuccess) on user abort", res.exitCode, cli.CodeSuccess)
	}
	if deletes != 0 {
		t.Errorf("DELETE count = %d, want 0 (user declined)", deletes)
	}
	if !strings.Contains(res.stdout, "Really delete?") {
		t.Errorf("stdout = %q, want it to contain the confirmation question", res.stdout)
	}
}

// TestDeleteConfirmYesComponent covers the same confirm-then-delete flow for a
// second command family (smd component) to exercise its prompt branch.
func TestDeleteConfirmYesComponent(t *testing.T) {
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
	if deletes != 1 {
		t.Errorf("DELETE count = %d, want 1", deletes)
	}
}

// TestDeleteConfirmNoBSS covers the abort branch for a bss delete command,
// confirming the pattern holds across services.
func TestDeleteConfirmNoBSS(t *testing.T) {
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

	if res.err != nil {
		t.Fatalf("unexpected error on abort: %v (exit %d)", res.err, res.exitCode)
	}
	if deletes != 0 {
		t.Errorf("DELETE count = %d, want 0 (user declined)", deletes)
	}
}
