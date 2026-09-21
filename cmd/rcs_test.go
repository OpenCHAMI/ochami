// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// rcs_test.go exercises the "rcs console" subcommands end-to-end: "list"
// (plain HTTP), "show" (websocket streamed to captured stdout), and
// "connect" (interactive websocket session). HTTP-failure cases are covered
// in rcs_errors_test.go.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
)

// TestRCSConsoleList_Success verifies "rcs console list" issues GET /consoles and prints
// the returned console list.
func TestRCSConsoleList_Success(t *testing.T) {
	// TODO: Enable t.Parallel() once race conditions are resolved
	// t.Parallel()

	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_, _ = w.Write([]byte(`{"consoles":[{"id":"x0c0s1b0n0","connectionType":"ipmi","connectionHost":"bmc"}]}`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "rcs", "console", "list", "--ignore-config", "--uri", srv.URL, "--token", "t")
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

// TestRCSConsoleShow_Success verifies "rcs console show <node>" connects to the console
// websocket at /consoles/<node>, streams server output to stdout, and returns
// nil on a normal websocket close.
func TestRCSConsoleShow_Success(t *testing.T) {
	// TODO: Enable t.Parallel() once race conditions are resolved
	// t.Parallel()

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

	res := runOchamiWithRuntime(t, "rcs", "console", "show", "--ignore-config", "--uri", srv.URL, "--token", "t",
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

// TestRCSConsole_Connect verifies that a normal websocket close during an
// interactive console session is treated as a clean exit, not an error.
func TestRCSConsole_Connect(t *testing.T) {
	upgrader := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		if err := conn.WriteMessage(websocket.TextMessage, []byte("connected")); err != nil {
			t.Errorf("write console message: %v", err)
			return
		}
		if err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")); err != nil {
			t.Errorf("write close message: %v", err)
		}
	}))
	defer srv.Close()

	res := runOchamiWithInputAndRuntime(t, "", "rcs", "console", "connect", "x0c0s1b0n0",
		"--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
}
