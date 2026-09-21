// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package pcs

// pcs_errors_test.go unit-tests the PCSClient wrapper methods' error arm: a
// non-2XX response surfacing as an UnsuccessfulHTTPError.

import (
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

	_, err := pc.GetTransitions("tok")
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !errors.Is(err, client.UnsuccessfulHTTPError) {
		t.Errorf("error = %v, want it to wrap client.UnsuccessfulHTTPError", err)
	}
}
