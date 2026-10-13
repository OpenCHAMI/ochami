// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package bss

// bss_errors_test.go unit-tests the BSSClient wrapper methods' error arms:
// input rejected before a request is made, and a non-2XX response surfacing
// (across every wrapper, not just a representative one) as an
// UnsuccessfulHTTPError.

import (
	"errors"
	"net/http"
	"testing"

	bssTypes "github.com/openchami/bss/pkg/bssTypes"

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

// TestBSSWrappers_HTTPError verifies every BSSClient wrapper method surfaces a
// non-2XX response as an error wrapping client.UnsuccessfulHTTPError.
func TestBSSWrappers_HTTPError(t *testing.T) {
	bp := bssTypes.BootParams{Hosts: []string{"x0c0s0b0n0"}}

	cases := []struct {
		name string
		call func(bc *BSSClient) (client.HTTPEnvelope, error)
	}{
		{"PostBootParams", func(bc *BSSClient) (client.HTTPEnvelope, error) { return bc.PostBootParams(bp, "tok") }},
		{"PutBootParams", func(bc *BSSClient) (client.HTTPEnvelope, error) { return bc.PutBootParams(bp, "tok") }},
		{"PatchBootParams", func(bc *BSSClient) (client.HTTPEnvelope, error) { return bc.PatchBootParams(bp, "tok") }},
		{"DeleteBootParams", func(bc *BSSClient) (client.HTTPEnvelope, error) { return bc.DeleteBootParams(bp, "tok") }},
		{"GetBootParams", func(bc *BSSClient) (client.HTTPEnvelope, error) { return bc.GetBootParams("", "tok") }},
		{"GetBootScript", func(bc *BSSClient) (client.HTTPEnvelope, error) { return bc.GetBootScript("") }},
		{"GetDumpstate", func(bc *BSSClient) (client.HTTPEnvelope, error) { return bc.GetDumpstate() }},
		{"GetEndpointHistory", func(bc *BSSClient) (client.HTTPEnvelope, error) { return bc.GetEndpointHistory("") }},
		{"GetHosts", func(bc *BSSClient) (client.HTTPEnvelope, error) { return bc.GetHosts("") }},
		{"GetStatus", func(bc *BSSClient) (client.HTTPEnvelope, error) { return bc.GetStatus("all") }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bc, srv := newTestBSS(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("boom"))
			})
			defer srv.Close()

			_, err := tc.call(bc)
			if err == nil {
				t.Fatalf("%s: expected error on HTTP failure, got nil", tc.name)
			}
			if !errors.Is(err, client.UnsuccessfulHTTPError) {
				t.Errorf("%s: error = %v, want Is(UnsuccessfulHTTPError)", tc.name, err)
			}
		})
	}
}
