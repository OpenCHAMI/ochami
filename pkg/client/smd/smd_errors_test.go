// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package smd

// smd_errors_test.go unit-tests the SMDClient wrapper methods' error arms:
// input rejected before a request is made, a non-2XX response reported in the
// iterative DeleteComponents helper's per-item error slice, and a non-2XX
// response from a single-shot getter surfacing as an UnsuccessfulHTTPError.

import (
	"errors"
	"net/http"
	"testing"

	"github.com/openchami/ochami/pkg/client"
)

// TestGetGroupMembers_EmptyLabel verifies GetGroupMembers rejects an empty label
// without making a request.
func TestGetGroupMembers_EmptyLabel(t *testing.T) {
	requestMade := false
	sc, srv := newTestSMD(t, func(w http.ResponseWriter, r *http.Request) { requestMade = true })
	defer srv.Close()

	if _, err := sc.GetGroupMembers("", "tok"); err == nil {
		t.Fatal("expected an error for empty group label, got nil")
	}
	if requestMade {
		t.Error("a request was made despite the empty group label")
	}
}

// TestDeleteComponents_PerItemHTTPError verifies that a non-2XX response is
// reported in the per-item error slice (not the function-level error).
func TestDeleteComponents_PerItemHTTPError(t *testing.T) {
	sc, srv := newTestSMD(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	})
	defer srv.Close()

	_, errs, err := sc.DeleteComponents("tok", "x0c0s0b0n0")
	if err != nil {
		t.Fatalf("DeleteComponents func error: %v", err)
	}
	if len(errs) != 1 || errs[0] == nil {
		t.Fatalf("per-item errors = %v, want a single non-nil error", errs)
	}
	if !errors.Is(errs[0], client.UnsuccessfulHTTPError) {
		t.Errorf("errs[0] = %v, want it to wrap client.UnsuccessfulHTTPError", errs[0])
	}
}

// TestGetComponentsAll_UnsuccessfulHTTP verifies a non-2XX response from a
// single-shot getter surfaces as an UnsuccessfulHTTPError.
func TestGetComponentsAll_UnsuccessfulHTTP(t *testing.T) {
	sc, srv := newTestSMD(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	})
	defer srv.Close()

	_, err := sc.GetComponentsAll()
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !errors.Is(err, client.UnsuccessfulHTTPError) {
		t.Errorf("error = %v, want it to wrap client.UnsuccessfulHTTPError", err)
	}
}
