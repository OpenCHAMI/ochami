// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// rcs_test.go exercises the non-interactive "rcs console" subcommands end-to-end:
// "list" (plain HTTP) and "show" (websocket streamed to captured stdout). The
// interactive "connect" command requires terminal/stdin injection and is
// covered after the refactor on the follow-up branch.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
)

// TestRCSConsoleList verifies "rcs console list" issues GET /consoles and prints
// the returned console list.
func TestRCSConsoleList(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_, _ = w.Write([]byte(`{"consoles":[{"id":"x0c0s1b0n0","connectionType":"ipmi","connectionHost":"bmc"}]}`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	res := runOchami(t, "rcs", "console", "list", "--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodGet || gotPath != "/consoles" {
		t.Errorf("request = %s %s, want GET /consoles", gotMethod, gotPath)
	}
	if !strings.Contains(res.stdout, "x0c0s1b0n0") {
		t.Errorf("stdout = %q, want it to contain the console ID", res.stdout)
	}
}

// TestRCSConsoleListHTTPError verifies an unsuccessful HTTP response resolves to
// a non-zero exit code.
func TestRCSConsoleListHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	res := runOchami(t, "rcs", "console", "list", "--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode == 0 {
		t.Errorf("exit code = %d, want non-zero", res.exitCode)
	}
}

// TestRCSConsoleShow verifies "rcs console show <node>" connects to the console
// websocket at /consoles/<node>, streams server output to stdout, and returns
// nil on a normal websocket close.
func TestRCSConsoleShow(t *testing.T) {
	var gotPath string
	upgrader := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		if err := conn.WriteMessage(websocket.TextMessage, []byte("console output line")); err != nil {
			t.Errorf("write console message: %v", err)
			return
		}
		if err := conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")); err != nil {
			t.Errorf("write close message: %v", err)
		}
	}))
	defer srv.Close()

	res := runOchami(t, "rcs", "console", "show", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"x0c0s1b0n0")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotPath != "/consoles/x0c0s1b0n0" {
		t.Errorf("path = %q, want /consoles/x0c0s1b0n0", gotPath)
	}
	if !strings.Contains(res.stdout, "console output line") {
		t.Errorf("stdout = %q, want it to contain the streamed console output", res.stdout)
	}
}
