// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package smd

// smd_test.go unit-tests representative SMDClient wrapper methods against an
// httptest.Server: single-resource GETs, list GETs with query strings, the
// iterative multi-item POST/DELETE helpers (which return per-item error
// slices), path construction for sub-resources, and UnsuccessfulHTTPError
// propagation.

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openchami/ochami/pkg/client"
)

func newTestSMD(t *testing.T, h http.HandlerFunc) (*SMDClient, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(h)
	sc, err := NewClient(srv.URL)
	if err != nil {
		srv.Close()
		t.Fatalf("NewClient: %v", err)
	}
	return sc, srv
}

// TestGetComponentsAll verifies GET /State/Components.
func TestGetComponentsAll(t *testing.T) {
	var gotMethod, gotPath string
	sc, srv := newTestSMD(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_, _ = w.Write([]byte(`{"Components":[]}`))
	})
	defer srv.Close()

	if _, err := sc.GetComponentsAll(); err != nil {
		t.Fatalf("GetComponentsAll: %v", err)
	}
	if gotMethod != http.MethodGet || gotPath != "/State/Components" {
		t.Errorf("request = %s %s, want GET /State/Components", gotMethod, gotPath)
	}
}

// TestGetComponentsXname verifies GET /State/Components/{xname} with the auth
// header set.
func TestGetComponentsXname(t *testing.T) {
	var gotPath, gotAuth string
	sc, srv := newTestSMD(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotAuth = r.URL.Path, r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{}`))
	})
	defer srv.Close()

	if _, err := sc.GetComponentsXname("x0c0s0b0n0", "tok"); err != nil {
		t.Fatalf("GetComponentsXname: %v", err)
	}
	if gotPath != "/State/Components/x0c0s0b0n0" {
		t.Errorf("path = %q, want /State/Components/x0c0s0b0n0", gotPath)
	}
	if gotAuth != "Bearer tok" {
		t.Errorf("auth = %q, want Bearer tok", gotAuth)
	}
}

// TestListGettersWithQuery verifies the list endpoints route correctly and
// forward the query string.
func TestListGettersWithQuery(t *testing.T) {
	cases := []struct {
		name      string
		call      func(sc *SMDClient) error
		wantPath  string
		wantQuery string
	}{
		{"groups", func(sc *SMDClient) error { _, e := sc.GetGroups("tag=foo", "tok"); return e }, "/groups", "tag=foo"},
		{"rfe", func(sc *SMDClient) error { _, e := sc.GetRedfishEndpoints("id=x0", "tok"); return e }, "/Inventory/RedfishEndpoints", "id=x0"},
		{"iface", func(sc *SMDClient) error { _, e := sc.GetEthernetInterfaces("MACAddress=de"); return e }, "/Inventory/EthernetInterfaces", "MACAddress=de"},
		{"compep", func(sc *SMDClient) error { _, e := sc.GetComponentEndpointsAll("tok"); return e }, "/Inventory/ComponentEndpoints", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var gotPath, gotQuery string
			sc, srv := newTestSMD(t, func(w http.ResponseWriter, r *http.Request) {
				gotPath, gotQuery = r.URL.Path, r.URL.RawQuery
				_, _ = w.Write([]byte(`{}`))
			})
			defer srv.Close()

			if err := tc.call(sc); err != nil {
				t.Fatalf("%s: %v", tc.name, err)
			}
			if gotPath != tc.wantPath {
				t.Errorf("path = %q, want %q", gotPath, tc.wantPath)
			}
			if gotQuery != tc.wantQuery {
				t.Errorf("query = %q, want %q", gotQuery, tc.wantQuery)
			}
		})
	}
}

// TestGetGroupMembers verifies the /groups/{group}/members sub-resource path.
func TestGetGroupMembers(t *testing.T) {
	var gotPath string
	sc, srv := newTestSMD(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`[]`))
	})
	defer srv.Close()

	if _, err := sc.GetGroupMembers("compute", "tok"); err != nil {
		t.Fatalf("GetGroupMembers: %v", err)
	}
	if gotPath != "/groups/compute/members" {
		t.Errorf("path = %q, want /groups/compute/members", gotPath)
	}
}

// TestGetGroupMembersEmptyLabel verifies GetGroupMembers rejects an empty label
// without making a request.
func TestGetGroupMembersEmptyLabel(t *testing.T) {
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

// TestPostComponents verifies the POST /State/Components request path/method.
func TestPostComponents(t *testing.T) {
	var gotMethod, gotPath string
	sc, srv := newTestSMD(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusCreated)
	})
	defer srv.Close()

	comps := ComponentSlice{Components: []Component{{ID: "x0c0s0b0n0"}}}
	if _, err := sc.PostComponents(comps, "tok"); err != nil {
		t.Fatalf("PostComponents: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/State/Components" {
		t.Errorf("request = %s %s, want POST /State/Components", gotMethod, gotPath)
	}
}

// TestDeleteComponentsIterative verifies the iterative DeleteComponents helper
// issues one DELETE per xname and returns a nil per-item error on success.
func TestDeleteComponentsIterative(t *testing.T) {
	var deletedPaths []string
	sc, srv := newTestSMD(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deletedPaths = append(deletedPaths, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	_, errs, err := sc.DeleteComponents("tok", "x0c0s0b0n0", "x0c0s0b0n1")
	if err != nil {
		t.Fatalf("DeleteComponents func error: %v", err)
	}
	if len(errs) != 2 {
		t.Fatalf("per-item errors length = %d, want 2", len(errs))
	}
	for i, e := range errs {
		if e != nil {
			t.Errorf("errs[%d] = %v, want nil", i, e)
		}
	}
	want := []string{"/State/Components/x0c0s0b0n0", "/State/Components/x0c0s0b0n1"}
	if len(deletedPaths) != 2 || deletedPaths[0] != want[0] || deletedPaths[1] != want[1] {
		t.Errorf("deleted paths = %v, want %v", deletedPaths, want)
	}
}

// TestDeleteComponentsPerItemHTTPError verifies that a non-2XX response is
// reported in the per-item error slice (not the function-level error).
func TestDeleteComponentsPerItemHTTPError(t *testing.T) {
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

// TestSMDUnsuccessfulHTTP verifies a non-2XX response from a single-shot getter
// surfaces as an UnsuccessfulHTTPError.
func TestSMDUnsuccessfulHTTP(t *testing.T) {
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
