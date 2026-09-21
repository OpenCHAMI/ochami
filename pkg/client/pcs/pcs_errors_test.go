// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package pcs

// pcs_errors_test.go unit-tests the PCSClient wrapper methods' error arms: a
// non-2XX response surfacing as an UnsuccessfulHTTPError (across every
// wrapper), caller cancellation reaching the HTTP request, malformed
// transition IDs, and a malformed cluster URI rejected at client
// construction.

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/openchami/ochami/pkg/client"
)

// TestGetTransitions_UnsuccessfulHTTP verifies a non-2XX response surfaces as an
// UnsuccessfulHTTPError.
func TestGetTransitions_UnsuccessfulHTTP(t *testing.T) {
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

// TestPCSWrapperHTTPFailures verifies every PCSClient wrapper method surfaces a
// non-2XX response as an error wrapping client.UnsuccessfulHTTPError.
func TestPCSWrapperHTTPFailures(t *testing.T) {
	tests := []struct {
		name string
		call func(*PCSClient) error
	}{
		{name: "liveness", call: func(c *PCSClient) error { _, err := c.GetLiveness(context.Background()); return err }},
		{name: "readiness", call: func(c *PCSClient) error { _, err := c.GetReadiness(context.Background()); return err }},
		{name: "health", call: func(c *PCSClient) error { _, err := c.GetHealth(context.Background()); return err }},
		{name: "create transition", call: func(c *PCSClient) error {
			_, err := c.CreateTransition(context.Background(), "on", nil, []string{"x0"}, "tok")
			return err
		}},
		{name: "get transition", call: func(c *PCSClient) error { _, err := c.GetTransition(context.Background(), "id", "tok"); return err }},
		{name: "delete transition", call: func(c *PCSClient) error { _, err := c.DeleteTransition(context.Background(), "id", "tok"); return err }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, srv := newTestPCS(t, func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "boom", http.StatusBadGateway)
			})
			defer srv.Close()
			if err := tt.call(c); !errors.Is(err, client.UnsuccessfulHTTPError) {
				t.Fatalf("call error = %v, want UnsuccessfulHTTPError", err)
			}
		})
	}
}

// TestPCSMalformedTransitionIDs verifies malformed transition IDs are rejected.
func TestPCSMalformedTransitionIDs(t *testing.T) {
	c, srv := newTestPCS(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected request: %s", r.URL.Path)
	})
	defer srv.Close()
	if _, err := c.GetTransition(context.Background(), "%zz", ""); err == nil {
		t.Fatal("GetTransition() accepted malformed ID")
	}
	if _, err := c.DeleteTransition(context.Background(), "%zz", ""); err == nil {
		t.Fatal("DeleteTransition() accepted malformed ID")
	}
}

// TestPCSNewClientRejectsMalformedURI verifies NewClient rejects a malformed URI.
func TestPCSNewClientRejectsMalformedURI(t *testing.T) {
	if _, err := NewClient("https://example.com/%zz"); err == nil {
		t.Fatal("NewClient() accepted malformed URI")
	}
}
