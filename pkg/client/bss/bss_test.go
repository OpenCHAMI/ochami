// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package bss

// bss_test.go unit-tests the BSSClient wrapper methods against an
// httptest.Server, verifying request method/path/query, authorization headers,
// and request body marshaling. Error-arm behavior is covered in
// bss_errors_test.go.

import (
	"context"
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
		w.Write([]byte(`[]`))
	})
	defer srv.Close()

	if _, err := bc.GetBootParams(context.Background(), "name=x0c0s0b0n0", "tok"); err != nil {
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
	if _, err := bc.PostBootParams(context.Background(), bp, "tok"); err != nil {
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
// PUT/PATCH/DELETE bootparameters mutators.
func TestPutPatchDeleteBootParams(t *testing.T) {
	cases := []struct {
		name       string
		call       func(bc *BSSClient) error
		wantMethod string
	}{
		{"put", func(bc *BSSClient) error {
			_, e := bc.PutBootParams(context.Background(), bssTypes.BootParams{}, "tok")
			return e
		}, http.MethodPut},
		{"patch", func(bc *BSSClient) error {
			_, e := bc.PatchBootParams(context.Background(), bssTypes.BootParams{}, "tok")
			return e
		}, http.MethodPatch},
		{"delete", func(bc *BSSClient) error {
			_, e := bc.DeleteBootParams(context.Background(), bssTypes.BootParams{}, "tok")
			return e
		}, http.MethodDelete},
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

// TestBootParamsBatch verifies that each *BootParamsBatch method sends one
// request per BootParams, in order, with the matching HTTP method, and returns
// one result per item, aligned with its input.
func TestBootParamsBatch(t *testing.T) {
	cases := []struct {
		name       string
		call       func(*BSSClient, []bssTypes.BootParams) client.BatchResult[client.HTTPEnvelope]
		wantMethod string
	}{
		{"post", func(bc *BSSClient, bps []bssTypes.BootParams) client.BatchResult[client.HTTPEnvelope] {
			return bc.PostBootParamsBatch(context.Background(), bps, "tok")
		}, http.MethodPost},
		{"put", func(bc *BSSClient, bps []bssTypes.BootParams) client.BatchResult[client.HTTPEnvelope] {
			return bc.PutBootParamsBatch(context.Background(), bps, "tok")
		}, http.MethodPut},
		{"patch", func(bc *BSSClient, bps []bssTypes.BootParams) client.BatchResult[client.HTTPEnvelope] {
			return bc.PatchBootParamsBatch(context.Background(), bps, "tok")
		}, http.MethodPatch},
		{"delete", func(bc *BSSClient, bps []bssTypes.BootParams) client.BatchResult[client.HTTPEnvelope] {
			return bc.DeleteBootParamsBatch(context.Background(), bps, "tok")
		}, http.MethodDelete},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var gotKernels []string
			bc, srv := newTestBSS(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != tc.wantMethod {
					t.Errorf("method = %s, want %s", r.Method, tc.wantMethod)
				}
				var bp bssTypes.BootParams
				if err := json.NewDecoder(r.Body).Decode(&bp); err != nil {
					t.Errorf("decode body: %v", err)
				}
				gotKernels = append(gotKernels, bp.Kernel)
				// Fail the second item to check that results stay aligned
				if bp.Kernel == "k2" {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				w.WriteHeader(http.StatusOK)
			})
			defer srv.Close()

			results := tc.call(bc, []bssTypes.BootParams{{Kernel: "k1"}, {Kernel: "k2"}})
			if len(gotKernels) != 2 || gotKernels[0] != "k1" || gotKernels[1] != "k2" {
				t.Errorf("request kernels = %v, want [k1 k2]", gotKernels)
			}
			if len(results) != 2 || results[0].Err != nil || !errors.Is(results[1].Err, client.UnsuccessfulHTTPError) {
				t.Errorf("results = %v, want success then an unsuccessful HTTP error", results)
			}
		})
	}
}

// TestGetStatus_Components verifies GetStatus maps its component argument to the
// correct BSS /service subpath.
func TestGetStatus_Components(t *testing.T) {
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
				w.Write([]byte(`{}`))
			})
			defer srv.Close()

			if _, err := bc.GetStatus(context.Background(), tc.component); err != nil {
				t.Fatalf("GetStatus(%q): %v", tc.component, err)
			}
			if gotPath != tc.wantPath {
				t.Errorf("path = %q, want %q", gotPath, tc.wantPath)
			}
		})
	}
}

// TestSimpleGetters verifies that GetDumpstate, GetHosts, GetBootScript, and
// GetEndpointHistory each request their own BSS endpoint path.
func TestSimpleGetters(t *testing.T) {
	cases := []struct {
		name     string
		call     func(bc *BSSClient) error
		wantPath string
	}{
		{"dumpstate", func(bc *BSSClient) error { _, e := bc.GetDumpstate(context.Background()); return e }, "/dumpstate"},
		{"hosts", func(bc *BSSClient) error { _, e := bc.GetHosts(context.Background(), ""); return e }, "/hosts"},
		{"bootscript", func(bc *BSSClient) error { _, e := bc.GetBootScript(context.Background(), ""); return e }, "/bootscript"},
		{"endpoint-history", func(bc *BSSClient) error { _, e := bc.GetEndpointHistory(context.Background(), ""); return e }, "/endpoint-history"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var gotPath string
			bc, srv := newTestBSS(t, func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				w.Write([]byte(`[]`))
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
