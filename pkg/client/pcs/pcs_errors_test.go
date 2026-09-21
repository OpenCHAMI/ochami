// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package pcs

// pcs_errors_test.go unit-tests the PCSClient wrapper methods' error arms: a
// non-2XX response surfacing as an UnsuccessfulHTTPError, and caller
// cancellation reaching the HTTP request.

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
