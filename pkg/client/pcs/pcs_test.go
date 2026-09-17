// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package pcs

// pcs_test.go unit-tests the PCSClient wrapper methods against an
// httptest.Server, verifying request routing (including path parameters and
// query strings), the CreateTransition request body, authorization headers, and
// UnsuccessfulHTTPError propagation.

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/openchami/ochami/pkg/client"
)

func newTestPCS(t *testing.T, h http.HandlerFunc) (*PCSClient, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(h)
	pc, err := NewClient(srv.URL)
	if err != nil {
		srv.Close()
		t.Fatalf("NewClient: %v", err)
	}
	return pc, srv
}

// TestReadinessLivenessHealth verifies the three probe endpoints route
// correctly.
func TestReadinessLivenessHealth(t *testing.T) {
	cases := []struct {
		name     string
		call     func(pc *PCSClient) error
		wantPath string
	}{
		{"liveness", func(pc *PCSClient) error { _, e := pc.GetLiveness(context.Background()); return e }, "/liveness"},
		{"readiness", func(pc *PCSClient) error { _, e := pc.GetReadiness(context.Background()); return e }, "/readiness"},
		{"health", func(pc *PCSClient) error { _, e := pc.GetHealth(context.Background()); return e }, "/health"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var gotPath string
			pc, srv := newTestPCS(t, func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				w.WriteHeader(http.StatusNoContent)
			})
			defer srv.Close()
			if err := tc.call(pc); err != nil {
				t.Fatalf("%s: %v", tc.name, err)
			}
			if gotPath != tc.wantPath {
				t.Errorf("path = %q, want %q", gotPath, tc.wantPath)
			}
		})
	}
}

// TestGetTransitions verifies GET /transitions and the auth header.
func TestGetTransitions(t *testing.T) {
	var gotMethod, gotPath, gotAuth string
	pc, srv := newTestPCS(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath, gotAuth = r.Method, r.URL.Path, r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{}`)) //nolint:errcheck // test response writes are observed by the client
	})
	defer srv.Close()

	if _, err := pc.GetTransitions(context.Background(), "tok"); err != nil {
		t.Fatalf("GetTransitions: %v", err)
	}
	if gotMethod != http.MethodGet || gotPath != "/transitions" {
		t.Errorf("request = %s %s, want GET /transitions", gotMethod, gotPath)
	}
	if gotAuth != "Bearer tok" {
		t.Errorf("auth = %q, want Bearer tok", gotAuth)
	}
}

// TestGetTransitionByID verifies GET /transitions/{id}.
func TestGetTransitionByID(t *testing.T) {
	var gotPath string
	pc, srv := newTestPCS(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{}`)) //nolint:errcheck // test response writes are observed by the client
	})
	defer srv.Close()

	if _, err := pc.GetTransition(context.Background(), "abc-123", "tok"); err != nil {
		t.Fatalf("GetTransition: %v", err)
	}
	if gotPath != "/transitions/abc-123" {
		t.Errorf("path = %q, want /transitions/abc-123", gotPath)
	}
}

// TestDeleteTransition verifies DELETE /transitions/{id}.
func TestDeleteTransition(t *testing.T) {
	var gotMethod, gotPath string
	pc, srv := newTestPCS(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_, _ = w.Write([]byte(`{}`)) //nolint:errcheck // test response writes are observed by the client
	})
	defer srv.Close()

	if _, err := pc.DeleteTransition(context.Background(), "abc-123", "tok"); err != nil {
		t.Fatalf("DeleteTransition: %v", err)
	}
	if gotMethod != http.MethodDelete || gotPath != "/transitions/abc-123" {
		t.Errorf("request = %s %s, want DELETE /transitions/abc-123", gotMethod, gotPath)
	}
}

// TestCreateTransition verifies POST /transitions and that the body carries the
// operation and each xname as a location entry.
func TestCreateTransition(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody []byte
	pc, srv := newTestPCS(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		var err error
		gotBody, err = io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{}`)) //nolint:errcheck // test response writes are observed by the client
	})
	defer srv.Close()

	if _, err := pc.CreateTransition(context.Background(), "on", nil, []string{"x0c0s0b0n0", "x0c0s0b0n1"}, "tok"); err != nil {
		t.Fatalf("CreateTransition: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/transitions" {
		t.Errorf("request = %s %s, want POST /transitions", gotMethod, gotPath)
	}
	var body struct {
		Operation string           `json:"operation"`
		Location  []map[string]any `json:"location"`
	}
	if err := json.Unmarshal(gotBody, &body); err != nil {
		t.Fatalf("unmarshal body %q: %v", string(gotBody), err)
	}
	if body.Operation != "on" {
		t.Errorf("operation = %q, want on", body.Operation)
	}
	if len(body.Location) != 2 {
		t.Errorf("location entries = %d, want 2 (body=%q)", len(body.Location), string(gotBody))
	}
}

// TestGetStatusQuery verifies GET /power-status encodes xnames and filters into
// the query string.
func TestGetStatusQuery(t *testing.T) {
	var gotPath, gotQuery string
	pc, srv := newTestPCS(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery = r.URL.Path, r.URL.RawQuery
		_, _ = w.Write([]byte(`{}`)) //nolint:errcheck // test response writes are observed by the client
	})
	defer srv.Close()

	if _, err := pc.GetStatus(context.Background(), []string{"x0c0s0b0"}, "on", "available", "tok"); err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if gotPath != "/power-status" {
		t.Errorf("path = %q, want /power-status", gotPath)
	}
	for _, want := range []string{"xname=x0c0s0b0", "powerStateFilter=on", "managementStateFilter=available"} {
		if !strings.Contains(gotQuery, want) {
			t.Errorf("query = %q, want it to contain %q", gotQuery, want)
		}
	}
}

// TestPCSUnsuccessfulHTTP verifies a non-2XX response surfaces as an
// UnsuccessfulHTTPError.
func TestPCSUnsuccessfulHTTP(t *testing.T) {
	pc, srv := newTestPCS(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusBadGateway)
	})
	defer srv.Close()

	_, err := pc.GetTransitions(context.Background(), "tok")
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !errors.Is(err, client.UnsuccessfulHTTPError) {
		t.Errorf("error = %v, want it to wrap client.UnsuccessfulHTTPError", err)
	}
}

// TestPCSClientPropagatesCancellation verifies caller cancellation reaches the HTTP request.
func TestPCSClientPropagatesCancellation(t *testing.T) {
	requestMade := false
	pc, srv := newTestPCS(t, func(w http.ResponseWriter, r *http.Request) {
		requestMade = true
	})
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := pc.GetHealth(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("GetHealth() error = %v, want context.Canceled", err)
	}
	if requestMade {
		t.Fatal("request was made after context cancellation")
	}
}
