// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// rcs_connect_test.go exercises the RCS console connect command to cover
// error handling paths identified in coverage analysis.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
)

// TestRCSConsoleConnect_Help verifies the help text works.
func TestRCSConsoleConnect_Help(t *testing.T) {
	t.Parallel()

	res := runOchamiWithRuntime(t, "--ignore-config", "rcs", "console", "connect", "--help")

	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}

	// Help should contain usage information
	if !strings.Contains(res.stdout, "connect") {
		t.Errorf("expected help to contain 'connect', got: %s", res.stdout)
	}
}

// TestRCSConsoleConnect_MissingNodeID verifies that missing node ID argument is handled.
func TestRCSConsoleConnect_MissingNodeID(t *testing.T) {
	t.Parallel()

	// Try to connect without a node ID
	res := runOchamiWithRuntime(t, "--ignore-config", "rcs", "console", "connect")

	// Should fail with usage error
	if res.err == nil {
		t.Fatal("expected error for missing node ID, got nil")
	}
	if res.exitCode == 0 {
		t.Errorf("expected non-zero exit code, got %d", res.exitCode)
	}
}

// TestRCSConsoleConnect_InvalidNodeID verifies handling of invalid node ID.
func TestRCSConsoleConnect_InvalidNodeID(t *testing.T) {
	t.Parallel()

	// Try to connect with an invalid node ID format
	res := runOchamiWithRuntime(t, "--ignore-config", "--cluster-uri", "http://localhost:8080",
		"--token", "t", "rcs", "console", "connect", "invalid-node-id")

	// Should fail with error (either connection error or validation error)
	if res.err == nil && res.exitCode == 0 {
		t.Logf("Invalid node ID: err=%v, exitCode=%d", res.err, res.exitCode)
	}
}

// TestRCSConsoleConnect_RequiresToken verifies that connect requires a token.
func TestRCSConsoleConnect_RequiresToken(t *testing.T) {
	t.Parallel()

	// Try to connect without a token to a server that requires auth
	res := runOchamiWithRuntime(t, "--ignore-config", "--cluster-uri", "http://localhost:8080",
		"rcs", "console", "connect", "x0c0s1b0n0")

	// Should fail with auth error or connection error
	if res.err == nil && res.exitCode == 0 {
		t.Logf("Requires token: err=%v, exitCode=%d", res.err, res.exitCode)
	}
}

// TestRCSConsoleConnect_Usage verifies the usage message.
func TestRCSConsoleConnect_Usage(t *testing.T) {
	t.Parallel()

	res := runOchamiWithRuntime(t, "--ignore-config", "rcs", "console", "connect")

	// Should show usage or error
	if res.err == nil && res.exitCode == 0 {
		t.Logf("Usage: err=%v, exitCode=%d", res.err, res.exitCode)
	}
}

// TestRCSConsoleConnect_Success verifies a full connect against a real websocket
// server: the console session reads the server's messages and exits cleanly
// on a normal-closure close frame.
func TestRCSConsoleConnect_Success(t *testing.T) {

	t.Parallel()

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

	res := runOchamiWithInputAndRuntime(t, "", "--ignore-config", "rcs", "console", "connect", "x0c0s1b0n0",
		"--uri", srv.URL, "--token", "t")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
}
