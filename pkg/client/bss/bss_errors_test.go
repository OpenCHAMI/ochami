// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package bss

// bss_errors_test.go unit-tests the BSSClient wrapper methods' error arms:
// input rejected before a request is made, and a non-2XX response surfacing
// (across every wrapper, not just a representative one) as an
// UnsuccessfulHTTPError.

import (
	"context"
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

	if _, err := bc.GetStatus(context.Background(), "bogus"); err == nil {
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

	_, err := bc.GetBootParams(context.Background(), "", "")
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

// TestBSSWrappersHTTPError verifies every BSSClient wrapper method surfaces a
// non-2XX response as an error wrapping client.UnsuccessfulHTTPError.
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

// TestGetStatus_UnknownComponentError verifies GetStatus errors on an unknown
// component without issuing a request.
func TestGetStatus_UnknownComponentError(t *testing.T) {
	bc, srv := newTestBSS(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called for unknown component")
	})
	defer srv.Close()

	if _, err := bc.GetStatus(context.Background(), "bogus"); err == nil {
		t.Fatal("expected error for unknown status component, got nil")
	}
}

// TestBSSClientPropagatesCancellation verifies caller cancellation reaches the HTTP request.
func TestBSSClientPropagatesCancellation(t *testing.T) {
	requestMade := false
	bc, srv := newTestBSS(t, func(w http.ResponseWriter, r *http.Request) {
		requestMade = true
	})
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := bc.GetBootParams(ctx, "", "tok")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("GetBootParams() error = %v, want context.Canceled", err)
	}
	if requestMade {
		t.Fatal("request was made after context cancellation")
	}
}
