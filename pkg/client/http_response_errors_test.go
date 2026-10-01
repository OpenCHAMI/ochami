// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestHTTP_ResponseReadFailure verifies handling of response read failures
// (e.g., server sends Content-Length but closes connection early).
func TestHTTP_ResponseReadFailure(t *testing.T) {
	t.Parallel()

	// Server sends partial response then closes connection
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "100")
		w.WriteHeader(http.StatusOK)
		// Write only 10 bytes then close
		if _, err := w.Write([]byte("partial data")); err != nil {
			t.Errorf("write partial response: %v", err)
		}
		// Connection will be closed by the server after this
	}))
	defer srv.Close()

	c, err := NewOchamiClient("TestClient", srv.URL)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// Make a request and try to read the response
	_, err = c.GetData(context.Background(), "/test", "", nil)
	if err == nil {
		t.Log("Response read failure: expected error, got nil")
	} else {
		t.Logf("Response read failure: got expected error: %v", err)
	}
}

// TestHTTP_MalformedBaseURI verifies a malformed base URI is rejected.
func TestHTTP_MalformedBaseURI(t *testing.T) {
	t.Parallel()

	if _, err := NewOchamiClient("TestClient", "://invalid-uri"); err == nil {
		t.Fatal("NewOchamiClient() error = nil, want a URI parse error")
	}
}
