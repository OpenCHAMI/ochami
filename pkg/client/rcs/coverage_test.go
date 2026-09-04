// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package rcs

// coverage_test.go covers reachable, non-interactive RCS client helpers:
// websocketDialError's response-classification branches, headersForToken, and
// GetStatus success/error arms.

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

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

func TestGetStatusSuccess(t *testing.T) {
	c, srv := newTestRCS(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	defer srv.Close()

	if _, err := c.GetStatus("tok"); err != nil {
		t.Errorf("GetStatus = %v, want nil", err)
	}
}

func TestGetStatusHTTPError(t *testing.T) {
	c, srv := newTestRCS(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	})
	defer srv.Close()

	if _, err := c.GetStatus("tok"); err == nil {
		t.Error("GetStatus with HTTP error = nil, want error")
	}
}

func TestGetStatusMalformedBody(t *testing.T) {
	c, srv := newTestRCS(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`not json`))
	})
	defer srv.Close()

	if _, err := c.GetStatus("tok"); err == nil {
		t.Error("GetStatus with malformed body = nil, want error")
	}
}

func TestListConsolesMalformedBody(t *testing.T) {
	c, srv := newTestRCS(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`not json`))
	})
	defer srv.Close()

	if _, err := c.ListConsoles("tok"); err == nil {
		t.Error("ListConsoles with malformed body = nil, want error")
	}
}
