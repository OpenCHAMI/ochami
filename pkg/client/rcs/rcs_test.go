// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package rcs

// rcs_test.go unit-tests the RCSClient. The plain HTTP endpoints (GetStatus,
// ListConsoles) are tested against an httptest.Server. The websocket-based
// ShowConsole and ConnectConsole are tested against a gorilla/websocket echo
// server, using in-memory io.Reader/io.Writer for stdin/stdout (buffered input
// mode, since the readers are not real terminals).

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func newTestRCS(t *testing.T, h http.HandlerFunc) (*RCSClient, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(h)
	c, err := NewClient(srv.URL)
	if err != nil {
		srv.Close()
		t.Fatalf("NewClient: %v", err)
	}
	return c, srv
}

// TestGetStatus verifies GetStatus issues GET /health and unmarshals the
// response.
func TestGetStatus(t *testing.T) {
	var gotPath string
	c, srv := newTestRCS(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"consoles":"3","hardwareupdate":"2026-01-01"}`)) //nolint:errcheck // test response writes are observed by the client
	})
	defer srv.Close()

	resp, err := c.GetStatus(context.Background(), "tok")
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if gotPath != "/health" {
		t.Errorf("path = %q, want /health", gotPath)
	}
	if resp.NumberConsoles != "3" {
		t.Errorf("NumberConsoles = %q, want 3", resp.NumberConsoles)
	}
}

// TestListConsoles verifies ListConsoles issues GET /consoles and unmarshals the
// console list.
func TestListConsoles(t *testing.T) {
	var gotPath string
	c, srv := newTestRCS(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"consoles":[{"id":"x0c0s1b0n0","connectionType":"ipmi","connectionHost":"bmc"}]}`)) //nolint:errcheck // test response writes are observed by the client
	})
	defer srv.Close()

	consoles, err := c.ListConsoles(context.Background(), "tok")
	if err != nil {
		t.Fatalf("ListConsoles: %v", err)
	}
	if gotPath != "/consoles" {
		t.Errorf("path = %q, want /consoles", gotPath)
	}
	if len(consoles) != 1 || consoles[0].ID != "x0c0s1b0n0" {
		t.Errorf("consoles = %+v, want a single console x0c0s1b0n0", consoles)
	}
}

// TestListConsolesHTTPError verifies a non-2XX response is returned as an error.
func TestListConsolesHTTPError(t *testing.T) {
	c, srv := newTestRCS(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusServiceUnavailable)
	})
	defer srv.Close()

	if _, err := c.ListConsoles(context.Background(), "tok"); err == nil {
		t.Fatal("expected an error, got nil")
	}
}

// wsUpgrader upgrades test HTTP connections to websockets.
var wsUpgrader = websocket.Upgrader{}

// TestShowConsoleStreamsOutput verifies ShowConsole connects to
// /consoles/{nodeID}, streams server messages to the output writer, and returns
// nil on a normal websocket close.
func TestShowConsoleStreamsOutput(t *testing.T) {
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery = r.URL.Path, r.URL.RawQuery
		conn, err := wsUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		if err := conn.WriteMessage(websocket.TextMessage, []byte("hello ")); err != nil {
			t.Errorf("write first websocket message: %v", err)
			return
		}
		if err := conn.WriteMessage(websocket.TextMessage, []byte("console")); err != nil {
			t.Errorf("write second websocket message: %v", err)
			return
		}
		// Close normally so ShowConsole returns nil.
		if err := conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")); err != nil {
			t.Errorf("write websocket close message: %v", err)
		}
	}))
	defer srv.Close()

	c, err := NewClient(srv.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	var out bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := c.ShowConsole(ctx, "x0c0s1b0n0", true, 100, "tok", &out); err != nil {
		t.Fatalf("ShowConsole: %v", err)
	}
	if gotPath != "/consoles/x0c0s1b0n0" {
		t.Errorf("path = %q, want /consoles/x0c0s1b0n0", gotPath)
	}
	if !strings.Contains(gotQuery, "mode=tail") || !strings.Contains(gotQuery, "follow=true") || !strings.Contains(gotQuery, "lines=100") {
		t.Errorf("query = %q, want it to contain mode=tail, follow=true, lines=100", gotQuery)
	}
	if out.String() != "hello console" {
		t.Errorf("output = %q, want %q", out.String(), "hello console")
	}
}

// TestShowConsoleDialError verifies ShowConsole returns an error when the
// websocket dial fails (server rejects the upgrade).
func TestShowConsoleDialError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Do not upgrade; respond with an error status so the dial fails.
		http.Error(w, "in use", http.StatusConflict)
	}))
	defer srv.Close()

	c, err := NewClient(srv.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := c.ShowConsole(ctx, "x0c0s1b0n0", false, 10, "tok", &bytes.Buffer{}); err == nil {
		t.Fatal("expected a dial error, got nil")
	}
}

// TestConnectConsoleEchoesInput verifies ConnectConsole forwards buffered stdin
// to the server and streams the server's echo back to stdout. The session ends
// when the context is cancelled.
func TestConnectConsoleEchoesInput(t *testing.T) {
	var serverGotMu sync.Mutex
	var serverGot []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := wsUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			mt, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if mt == websocket.TextMessage {
				serverGotMu.Lock()
				serverGot = append(serverGot, msg...)
				serverGotMu.Unlock()
				// Echo back so the client's stdout stream receives data.
				if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
					return
				}
			}
		}
	}))
	defer srv.Close()

	c, err := NewClient(srv.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	// A bytes.Reader is not an *os.File, so ConnectConsole uses buffered input
	// mode (no raw terminal handling), which is exactly what we want for a
	// deterministic test.
	stdin := strings.NewReader("uptime\n")
	var stdout syncBuffer

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// ConnectConsole blocks until the context is done (buffered stdin reaches
	// EOF but that does not end the session), so run it and then wait for the
	// echoed output to arrive before the context expires.
	errCh := make(chan error, 1)
	go func() { errCh <- c.ConnectConsole(ctx, "x0c0s1b0n0", "tok", stdin, &stdout) }()

	// Wait for the echo to make it back to stdout.
	deadline := time.After(2 * time.Second)
	for {
		if strings.Contains(stdout.String(), "uptime") {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for echoed output; stdout=%q", stdout.String())
		case <-time.After(10 * time.Millisecond):
		}
	}

	cancel()
	<-errCh // ConnectConsole should return after context cancellation.

	serverGotMu.Lock()
	got := string(serverGot)
	serverGotMu.Unlock()
	if !strings.Contains(got, "uptime") {
		t.Errorf("server received = %q, want it to contain the forwarded input", got)
	}
}

// syncBuffer is a goroutine-safe bytes.Buffer for use as a test stdout, since
// ConnectConsole writes to stdout from a separate goroutine while the test
// reads it.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}
