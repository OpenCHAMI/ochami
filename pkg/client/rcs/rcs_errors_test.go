// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package rcs

// rcs_errors_test.go unit-tests the RCSClient's error arms: a non-2XX
// response from the plain HTTP endpoints, a websocket dial failure for the
// console-streaming endpoints, malformed response bodies, and the reachable,
// non-interactive helpers' error-classification branches
// (websocketDialError, headersForToken).

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestListConsoles_HTTPError verifies a non-2XX response is returned as an error.
func TestListConsoles_HTTPError(t *testing.T) {
	c, srv := newTestRCS(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusServiceUnavailable)
	})
	defer srv.Close()

	if _, err := c.ListConsoles("tok"); err == nil {
		t.Fatal("expected an error, got nil")
	}
}

// TestShowConsole_DialError verifies ShowConsole returns an error when the
// websocket dial fails (server rejects the upgrade).
func TestShowConsole_DialError(t *testing.T) {
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

// mkResp builds a minimal *http.Response with the given status and body.
func mkResp(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestWebsocketDialError(t *testing.T) {
	// nil response -> wraps the dial error.
	if err := websocketDialError("n0", nil, io.EOF); err == nil {
		t.Error("nil response = nil, want error")
	}

	// 409 with a message -> returns the message.
	if err := websocketDialError("n0", mkResp(http.StatusConflict, "busy"), nil); err == nil || !strings.Contains(err.Error(), "busy") {
		t.Errorf("409 with message = %v, want it to contain the message", err)
	}

	// 409 without a message -> "already in use" for the node.
	if err := websocketDialError("n0", mkResp(http.StatusConflict, ""), nil); err == nil || !strings.Contains(err.Error(), "already in use") {
		t.Errorf("409 without message = %v, want already-in-use", err)
	}

	// Non-409 with a message.
	if err := websocketDialError("n0", mkResp(http.StatusInternalServerError, "oops"), nil); err == nil || !strings.Contains(err.Error(), "oops") {
		t.Errorf("500 with message = %v, want it to contain the message", err)
	}

	// Non-409 without a message.
	if err := websocketDialError("n0", mkResp(http.StatusBadGateway, ""), nil); err == nil {
		t.Error("502 without message = nil, want error")
	}
}

func TestHeadersForToken(t *testing.T) {
	// Empty token -> headers without Authorization, no error.
	if _, err := headersForToken(""); err != nil {
		t.Errorf("headersForToken(\"\") = %v, want nil", err)
	}
	// Non-empty token -> headers with Authorization, no error.
	h, err := headersForToken("tok")
	if err != nil {
		t.Fatalf("headersForToken(tok) = %v, want nil", err)
	}
	if h == nil {
		t.Error("headersForToken(tok) returned nil headers")
	}
}

func TestGetStatus_HTTPError(t *testing.T) {
	c, srv := newTestRCS(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	})
	defer srv.Close()

	if _, err := c.GetStatus("tok"); err == nil {
		t.Error("GetStatus with HTTP error = nil, want error")
	}
}

func TestGetStatus_MalformedBody(t *testing.T) {
	c, srv := newTestRCS(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`not json`)) //nolint:errcheck // test response writes are observed by the client
	})
	defer srv.Close()

	if _, err := c.GetStatus("tok"); err == nil {
		t.Error("GetStatus with malformed body = nil, want error")
	}
}

func TestListConsolesMalformedBody(t *testing.T) {
	c, srv := newTestRCS(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`not json`)) //nolint:errcheck // test response writes are observed by the client
	})
	defer srv.Close()

	if _, err := c.ListConsoles("tok"); err == nil {
		t.Error("ListConsoles with malformed body = nil, want error")
	}
}
