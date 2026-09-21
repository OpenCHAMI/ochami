// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cloud_init

// cloud_init_errors_test.go unit-tests the CloudInitClient wrapper methods'
// error arms: input rejected before a request is made, and a non-2XX response
// surfacing as an UnsuccessfulHTTPError.

import (
	"errors"
	"net/http"
	"testing"

	"github.com/openchami/ochami/pkg/client"
)

// TestGetNodeData_RequiresID verifies GetNodeData errors (without a request) when
// no IDs are supplied.
func TestGetNodeData_RequiresID(t *testing.T) {
	requestMade := false
	cic, srv := newTestCI(t, func(w http.ResponseWriter, r *http.Request) { requestMade = true })
	defer srv.Close()

	if _, _, err := cic.GetNodeData(CloudInitMetaData, "tok"); err == nil {
		t.Fatal("expected an error when no IDs are supplied, got nil")
	}
	if requestMade {
		t.Error("a request was made despite no IDs being supplied")
	}
}

// TestGetDefaults_UnsuccessfulHTTP verifies a non-2XX response surfaces as an
// UnsuccessfulHTTPError.
func TestGetDefaults_UnsuccessfulHTTP(t *testing.T) {
	cic, srv := newTestCI(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusForbidden)
	})
	defer srv.Close()

	_, err := cic.GetDefaults("tok")
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !errors.Is(err, client.UnsuccessfulHTTPError) {
		t.Errorf("error = %v, want it to wrap client.UnsuccessfulHTTPError", err)
	}
}
