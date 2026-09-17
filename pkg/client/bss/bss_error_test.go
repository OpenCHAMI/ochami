// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package bss

// bss_error_test.go verifies that each BSSClient wrapper propagates an error
// (wrapping client.UnsuccessfulHTTPError) when the server responds with a
// non-success status.

import (
	"context"
	"errors"
	"net/http"
	"testing"

	bssTypes "github.com/openchami/bss/pkg/bssTypes"

	"github.com/openchami/ochami/pkg/client"
)

// TestBSSWrappersHTTPError verifies all BSS wrappers propagate unsuccessful HTTP responses.
func TestBSSWrappersHTTPError(t *testing.T) {
	bp := bssTypes.BootParams{Hosts: []string{"x0c0s0b0n0"}}

	cases := []struct {
		name string
		call func(bc *BSSClient) (client.HTTPEnvelope, error)
	}{
		{"PostBootParams", func(bc *BSSClient) (client.HTTPEnvelope, error) {
			return bc.PostBootParams(context.Background(), bp, "tok")
		}},
		{"PutBootParams", func(bc *BSSClient) (client.HTTPEnvelope, error) {
			return bc.PutBootParams(context.Background(), bp, "tok")
		}},
		{"PatchBootParams", func(bc *BSSClient) (client.HTTPEnvelope, error) {
			return bc.PatchBootParams(context.Background(), bp, "tok")
		}},
		{"DeleteBootParams", func(bc *BSSClient) (client.HTTPEnvelope, error) {
			return bc.DeleteBootParams(context.Background(), bp, "tok")
		}},
		{"GetBootParams", func(bc *BSSClient) (client.HTTPEnvelope, error) {
			return bc.GetBootParams(context.Background(), "", "tok")
		}},
		{"GetBootScript", func(bc *BSSClient) (client.HTTPEnvelope, error) { return bc.GetBootScript(context.Background(), "") }},
		{"GetDumpstate", func(bc *BSSClient) (client.HTTPEnvelope, error) { return bc.GetDumpstate(context.Background()) }},
		{"GetEndpointHistory", func(bc *BSSClient) (client.HTTPEnvelope, error) {
			return bc.GetEndpointHistory(context.Background(), "")
		}},
		{"GetHosts", func(bc *BSSClient) (client.HTTPEnvelope, error) { return bc.GetHosts(context.Background(), "") }},
		{"GetStatus", func(bc *BSSClient) (client.HTTPEnvelope, error) { return bc.GetStatus(context.Background(), "all") }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bc, srv := newTestBSS(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte("boom")) //nolint:errcheck // test response writes are observed by the client
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

// TestGetStatusUnknownComponentError verifies GetStatus errors on an unknown
// component without issuing a request.
func TestGetStatusUnknownComponentError(t *testing.T) {
	bc, srv := newTestBSS(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called for unknown component")
	})
	defer srv.Close()

	if _, err := bc.GetStatus(context.Background(), "bogus"); err == nil {
		t.Fatal("expected error for unknown status component, got nil")
	}
}
