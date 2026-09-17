// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package smd

// smd_coverage_test.go exercises additional reachable branches of the SMD
// client wrappers: the getIPs variant and error arm of GetEthernetInterfaceByID,
// the argument-guard clauses of the group-member helpers, and per-item HTTP
// error arms of the iterative helpers.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/openchami/ochami/pkg/client"
)

// newTestClient builds an SMDClient pointed at srv.
func newTestClient(t *testing.T, srv *httptest.Server) *SMDClient {
	t.Helper()
	c, err := NewClient(srv.URL, client.WithInsecure(true))
	if err != nil {
		t.Fatalf("failed to create SMD client: %v", err)
	}
	return c
}

// TestGetEthernetInterfaceByIDWithIPs verifies the getIPs=true path targets the
// IPAddresses subpath.
func TestGetEthernetInterfaceByIDWithIPs(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`[]`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if _, err := c.GetEthernetInterfaceByID(context.Background(), "decafc0ffeee", "tok", true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(gotPath, "IPAddresses") {
		t.Errorf("path = %q, want it to reference IPAddresses", gotPath)
	}
}

// TestGetEthernetInterfaceByIDHTTPError verifies the GetData error arm.
func TestGetEthernetInterfaceByIDHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if _, err := c.GetEthernetInterfaceByID(context.Background(), "decafc0ffeee", "tok", false); err == nil {
		t.Fatal("expected an error, got nil")
	}
}

// TestPostGroupMembersGuards verifies the empty-group and empty-members guard
// clauses.
func TestPostGroupMembersGuards(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	c := newTestClient(t, srv)

	if _, err := c.PostGroupMembers(context.Background(), "tok", ""); err == nil {
		t.Error("PostGroupMembers with empty group = nil, want error")
	}
	if _, err := c.PostGroupMembers(context.Background(), "tok", "compute"); err == nil {
		t.Error("PostGroupMembers with no members = nil, want error")
	}
}

// TestPostGroupMembersPerItemError verifies the per-item POST error arm.
func TestPostGroupMembersPerItemError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()
	c := newTestClient(t, srv)

	results, err := c.PostGroupMembers(context.Background(), "tok", "compute", "x0c0s0b0n0")
	if err != nil {
		t.Fatalf("function error = %v, want nil (per-item errors reported separately)", err)
	}
	if len(results) != 1 || results[0].Err == nil {
		t.Errorf("results = %v, want one non-nil per-item error", results)
	}
}

// TestDeleteHelpersPerItemError verifies the per-item DELETE error arms of the
// iterative delete helpers.
func TestDeleteHelpersPerItemError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()
	c := newTestClient(t, srv)

	if results := c.DeleteComponents(context.Background(), "tok", "x0c0s0b0n0"); len(results) != 1 || results[0].Err == nil {
		t.Errorf("DeleteComponents results = %v, want one non-nil per-item error", results)
	}
	if results := c.DeleteRedfishEndpoints(context.Background(), "tok", "x3000c1s7b56"); len(results) != 1 || results[0].Err == nil {
		t.Errorf("DeleteRedfishEndpoints results = %v, want one non-nil per-item error", results)
	}
	if results := c.DeleteEthernetInterfaces(context.Background(), "tok", "decafc0ffeee"); len(results) != 1 || results[0].Err == nil {
		t.Errorf("DeleteEthernetInterfaces results = %v, want one non-nil per-item error", results)
	}
	if results, err := c.DeleteGroupMembers(context.Background(), "tok", "compute", "x0c0s0b0n0"); err != nil || len(results) != 1 || results[0].Err == nil {
		t.Errorf("DeleteGroupMembers result = (err=%v, results=%v), want one non-nil per-item error", err, results)
	}
}

// TestPostHelpersPerItemError verifies the per-item POST error arms of the
// iterative add helpers.
func TestPostHelpersPerItemError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()
	c := newTestClient(t, srv)

	eis := []EthernetInterface{{ComponentID: "x0c0s0b0n0", MACAddress: "de:ad:be:ef:00:00"}}
	if results := c.PostEthernetInterfaces(context.Background(), eis, "tok"); len(results) != 1 || results[0].Err == nil {
		t.Errorf("PostEthernetInterfaces results = %v, want one non-nil per-item error", results)
	}
}

// TestGetStatusUnknownComponent verifies the unknown-component arm of GetStatus.
func TestGetStatusUnknownComponent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	c := newTestClient(t, srv)

	if _, err := c.GetStatus(context.Background(), "bogus"); err == nil {
		t.Error("GetStatus(bogus) = nil, want error")
	}
	// "" and "all" are valid and route to the ready/values endpoints.
	if _, err := c.GetStatus(context.Background(), ""); err != nil {
		t.Errorf("GetStatus(\"\") = %v, want nil", err)
	}
	if _, err := c.GetStatus(context.Background(), "all"); err != nil {
		t.Errorf("GetStatus(all) = %v, want nil", err)
	}
}

// TestGetComponentsByXnameNidError verifies the error arms of the single-item
// component getters.
func TestGetComponentsByXnameNidError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()
	c := newTestClient(t, srv)

	if _, err := c.GetComponentsXname(context.Background(), "x0c0s0b0n0", "tok"); err == nil {
		t.Error("GetComponentsXname error arm = nil, want error")
	}
	if _, err := c.GetComponentsNid(context.Background(), 1, "tok"); err == nil {
		t.Error("GetComponentsNid error arm = nil, want error")
	}
}

// TestPutGroupMembersGuards verifies the empty-group and empty-members guards.
func TestPutGroupMembersGuards(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	c := newTestClient(t, srv)

	if _, err := c.PutGroupMembers(context.Background(), "tok", ""); err == nil {
		t.Error("PutGroupMembers with empty group = nil, want error")
	}
	if _, err := c.PutGroupMembers(context.Background(), "tok", "compute"); err == nil {
		t.Error("PutGroupMembers with no members = nil, want error")
	}
}

// TestDeleteAllHelpersError verifies the error arms of the delete-all helpers.
func TestDeleteAllHelpersError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()
	c := newTestClient(t, srv)

	if _, err := c.DeleteComponentsAll(context.Background(), "tok"); err == nil {
		t.Error("DeleteComponentsAll error arm = nil, want error")
	}
	if _, err := c.DeleteRedfishEndpointsAll(context.Background(), "tok"); err == nil {
		t.Error("DeleteRedfishEndpointsAll error arm = nil, want error")
	}
	if _, err := c.DeleteEthernetInterfacesAll(context.Background(), "tok"); err == nil {
		t.Error("DeleteEthernetInterfacesAll error arm = nil, want error")
	}
	if _, err := c.DeleteComponentEndpointsAll(context.Background(), "tok"); err == nil {
		t.Error("DeleteComponentEndpointsAll error arm = nil, want error")
	}
}
