// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package rcs

// rcs_errors_test.go unit-tests the RCSClient's error arms: a non-2XX
// response from the plain HTTP endpoints, and a websocket dial failure for the
// console-streaming endpoints.

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
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
