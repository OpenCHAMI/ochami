// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package bss

// bss_errors_test.go unit-tests the BSSClient wrapper methods' error arms:
// input rejected before a request is made, and a non-2XX response surfacing as
// an UnsuccessfulHTTPError.

import (
	"errors"
	"net/http"
	"testing"

	"github.com/openchami/ochami/pkg/client"
)

// TestGetStatus_UnknownComponent verifies GetStatus rejects an unknown component
// without making a request.
func TestGetStatus_UnknownComponent(t *testing.T) {
	requestMade := false
	bc, srv := newTestBSS(t, func(w http.ResponseWriter, r *http.Request) { requestMade = true })
	defer srv.Close()

	if _, err := bc.GetStatus("bogus"); err == nil {
		t.Fatal("expected an error for unknown component, got nil")
	}
	if requestMade {
		t.Error("a request was made for an unknown component")
	}
}

// TestGetBootParams_UnsuccessfulHTTP verifies a non-2XX response surfaces as an
// UnsuccessfulHTTPError so callers can map it to the CodeHTTP exit code.
func TestGetBootParams_UnsuccessfulHTTP(t *testing.T) {
	bc, srv := newTestBSS(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusInternalServerError)
	})
	defer srv.Close()

	_, err := bc.GetBootParams("", "")
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !isUnsuccessfulHTTP(err) {
		t.Errorf("error = %v, want it to wrap client.UnsuccessfulHTTPError", err)
	}
}

// isUnsuccessfulHTTP reports whether err wraps client.UnsuccessfulHTTPError.
func isUnsuccessfulHTTP(err error) bool {
	return err != nil && errors.Is(err, client.UnsuccessfulHTTPError)
}
