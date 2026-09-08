// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package bss

// bss_test.go unit-tests the BSSClient wrapper methods against an
// httptest.Server, verifying request method/path/query, authorization headers,
// request body marshaling, and error handling (including UnsuccessfulHTTPError
// on non-2XX responses).

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	bssTypes "github.com/openchami/bss/pkg/bssTypes"

	"github.com/openchami/ochami/pkg/client"
)

// newTestBSS starts an httptest.Server with the given handler and returns a
// BSSClient pointed at it.
func newTestBSS(t *testing.T, h http.HandlerFunc) (*BSSClient, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(h)
	bc, err := NewClient(srv.URL)
	if err != nil {
		srv.Close()
		t.Fatalf("NewClient: %v", err)
	}
	return bc, srv
}

// TestGetBootParams verifies the GET /bootparameters request, that the query
// string is forwarded, and that the auth header is set when a token is given.
func TestGetBootParams(t *testing.T) {
	var gotMethod, gotPath, gotQuery, gotAuth string
	bc, srv := newTestBSS(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath, gotQuery = r.Method, r.URL.Path, r.URL.RawQuery
		gotAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`[]`)) //nolint:errcheck // test response writes are observed by the client
	})
	defer srv.Close()

	if _, err := bc.GetBootParams("name=x0c0s0b0n0", "tok"); err != nil {
		t.Fatalf("GetBootParams: %v", err)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("method = %q, want GET", gotMethod)
	}
	if gotPath != "/bootparameters" {
		t.Errorf("path = %q, want /bootparameters", gotPath)
	}
	if gotQuery != "name=x0c0s0b0n0" {
		t.Errorf("query = %q, want name=x0c0s0b0n0", gotQuery)
	}
	if gotAuth != "Bearer tok" {
		t.Errorf("auth = %q, want Bearer tok", gotAuth)
	}
}

// TestPostBootParams verifies the POST /bootparameters request marshals the
// BootParams into the body.
func TestPostBootParams(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody []byte
	bc, srv := newTestBSS(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		var err error
		gotBody, err = io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
		}
		w.WriteHeader(http.StatusCreated)
	})
	defer srv.Close()

	bp := bssTypes.BootParams{Kernel: "https://example/vmlinuz", Macs: []string{"de:ad:be:ef:00:00"}}
	if _, err := bc.PostBootParams(bp, "tok"); err != nil {
		t.Fatalf("PostBootParams: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/bootparameters" {
		t.Errorf("request = %s %s, want POST /bootparameters", gotMethod, gotPath)
	}
	var sent map[string]any
	if err := json.Unmarshal(gotBody, &sent); err != nil {
		t.Fatalf("unmarshal body %q: %v", string(gotBody), err)
	}
	if sent["kernel"] != "https://example/vmlinuz" {
		t.Errorf("body kernel = %v, want the kernel URL", sent["kernel"])
	}
}

// TestPutPatchDeleteBootParams verifies the HTTP method used by each of the
// remaining bootparameters mutators.
func TestPutPatchDeleteBootParams(t *testing.T) {
	cases := []struct {
		name       string
		call       func(bc *BSSClient) error
		wantMethod string
	}{
		{"put", func(bc *BSSClient) error { _, e := bc.PutBootParams(bssTypes.BootParams{}, "tok"); return e }, http.MethodPut},
		{"patch", func(bc *BSSClient) error { _, e := bc.PatchBootParams(bssTypes.BootParams{}, "tok"); return e }, http.MethodPatch},
		{"delete", func(bc *BSSClient) error { _, e := bc.DeleteBootParams(bssTypes.BootParams{}, "tok"); return e }, http.MethodDelete},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var gotMethod, gotPath string
			bc, srv := newTestBSS(t, func(w http.ResponseWriter, r *http.Request) {
				gotMethod, gotPath = r.Method, r.URL.Path
				w.WriteHeader(http.StatusOK)
			})
			defer srv.Close()

			if err := tc.call(bc); err != nil {
				t.Fatalf("%s: %v", tc.name, err)
			}
			if gotMethod != tc.wantMethod {
				t.Errorf("method = %q, want %q", gotMethod, tc.wantMethod)
			}
			if gotPath != "/bootparameters" {
				t.Errorf("path = %q, want /bootparameters", gotPath)
			}
		})
	}
}

// TestGetStatusComponents verifies GetStatus maps its component argument to the
// correct BSS /service subpath.
func TestGetStatusComponents(t *testing.T) {
	cases := []struct {
		component string
		wantPath  string
	}{
		{"", "/service/status"},
		{"all", "/service/status/all"},
		{"storage", "/service/storage/status"},
		{"smd", "/service/hsm"},
		{"version", "/service/version"},
	}
	for _, tc := range cases {
		t.Run(tc.component, func(t *testing.T) {
			var gotPath string
			bc, srv := newTestBSS(t, func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				_, _ = w.Write([]byte(`{}`)) //nolint:errcheck // test response writes are observed by the client
			})
			defer srv.Close()

			if _, err := bc.GetStatus(tc.component); err != nil {
				t.Fatalf("GetStatus(%q): %v", tc.component, err)
			}
			if gotPath != tc.wantPath {
				t.Errorf("path = %q, want %q", gotPath, tc.wantPath)
			}
		})
	}
}

// TestGetStatusUnknownComponent verifies GetStatus rejects an unknown component
// without making a request.
func TestGetStatusUnknownComponent(t *testing.T) {
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

// TestSimpleGetters verifies the remaining simple GET endpoints route correctly.
func TestSimpleGetters(t *testing.T) {
	cases := []struct {
		name     string
		call     func(bc *BSSClient) error
		wantPath string
	}{
		{"dumpstate", func(bc *BSSClient) error { _, e := bc.GetDumpstate(); return e }, "/dumpstate"},
		{"hosts", func(bc *BSSClient) error { _, e := bc.GetHosts(""); return e }, "/hosts"},
		{"bootscript", func(bc *BSSClient) error { _, e := bc.GetBootScript(""); return e }, "/bootscript"},
		{"endpoint-history", func(bc *BSSClient) error { _, e := bc.GetEndpointHistory(""); return e }, "/endpoint-history"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var gotPath string
			bc, srv := newTestBSS(t, func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				_, _ = w.Write([]byte(`[]`)) //nolint:errcheck // test response writes are observed by the client
			})
			defer srv.Close()

			if err := tc.call(bc); err != nil {
				t.Fatalf("%s: %v", tc.name, err)
			}
			if gotPath != tc.wantPath {
				t.Errorf("path = %q, want %q", gotPath, tc.wantPath)
			}
		})
	}
}

// TestUnsuccessfulHTTPError verifies a non-2XX response surfaces as an
// UnsuccessfulHTTPError so callers can map it to the CodeHTTP exit code.
func TestUnsuccessfulHTTPError(t *testing.T) {
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
