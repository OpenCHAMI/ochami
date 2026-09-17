// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package smd

// smd_more_test.go extends the SMDClient wrapper coverage to the mutating
// endpoints (POST/PUT/PATCH), the bulk "*All" deletes, and the remaining
// getters, complementing smd_test.go. Each test asserts the HTTP method and
// path against an httptest.Server.

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/openchami/ochami/pkg/client"
)

// TestGetStatus verifies GetStatus routes to the SMD /service readiness/values
// endpoints depending on the requested component.
func TestGetStatus(t *testing.T) {
	cases := []struct {
		name      string
		component string
		wantPath  string
	}{
		{"ready", "", "/service/ready"},
		{"all", "all", "/service/values"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var gotPath string
			sc, srv := newTestSMD(t, func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				_, _ = w.Write([]byte(`{}`)) //nolint:errcheck // test response writes are observed by the client
			})
			defer srv.Close()
			if _, err := sc.GetStatus(context.Background(), tc.component); err != nil {
				t.Fatalf("GetStatus(%q): %v", tc.component, err)
			}
			if gotPath != tc.wantPath {
				t.Errorf("path = %q, want %q", gotPath, tc.wantPath)
			}
		})
	}
}

// TestGetEthernetInterfaceByID verifies the by-ID getter routes to
// /Inventory/EthernetInterfaces/<id> (and appends /IPAddresses when requested).
func TestGetEthernetInterfaceByID(t *testing.T) {
	t.Run("plain", func(t *testing.T) {
		var gotPath string
		sc, srv := newTestSMD(t, func(w http.ResponseWriter, r *http.Request) {
			gotPath = r.URL.Path
			_, _ = w.Write([]byte(`{}`)) //nolint:errcheck // test response writes are observed by the client
		})
		defer srv.Close()
		if _, err := sc.GetEthernetInterfaceByID(context.Background(), "deadbeef", "tok", false); err != nil {
			t.Fatalf("GetEthernetInterfaceByID: %v", err)
		}
		if gotPath != "/Inventory/EthernetInterfaces/deadbeef" {
			t.Errorf("path = %q, want /Inventory/EthernetInterfaces/deadbeef", gotPath)
		}
	})
	t.Run("with-ips", func(t *testing.T) {
		var gotPath string
		sc, srv := newTestSMD(t, func(w http.ResponseWriter, r *http.Request) {
			gotPath = r.URL.Path
			_, _ = w.Write([]byte(`{}`)) //nolint:errcheck // test response writes are observed by the client
		})
		defer srv.Close()
		if _, err := sc.GetEthernetInterfaceByID(context.Background(), "deadbeef", "tok", true); err != nil {
			t.Fatalf("GetEthernetInterfaceByID: %v", err)
		}
		if gotPath != "/Inventory/EthernetInterfaces/deadbeef/IPAddresses" {
			t.Errorf("path = %q, want .../IPAddresses", gotPath)
		}
	})
}

// TestGetGroupMembership verifies GetGroupMembership routes to /memberships.
func TestGetGroupMembership(t *testing.T) {
	var gotPath string
	sc, srv := newTestSMD(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{}`)) //nolint:errcheck // test response writes are observed by the client
	})
	defer srv.Close()
	if _, err := sc.GetGroupMembership(context.Background(), "id=x0", "tok"); err != nil {
		t.Fatalf("GetGroupMembership: %v", err)
	}
	if gotPath != "/memberships" {
		t.Errorf("path = %q, want /memberships", gotPath)
	}
}

// TestPutComponents verifies PutComponents issues PUT /State/Components.
func TestPutComponents(t *testing.T) {
	var gotMethod, gotPath string
	sc, srv := newTestSMD(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()
	results := sc.PutComponents(context.Background(), ComponentSlice{Components: []Component{{ID: "x0c0s0b0n0"}}}, "tok")
	if len(results) != 1 || results[0].Err != nil {
		t.Errorf("results = %v, want a single success", results)
	}
	if gotMethod != http.MethodPut || !strings.HasPrefix(gotPath, "/State/Components") {
		t.Errorf("request = %s %s, want PUT under /State/Components", gotMethod, gotPath)
	}
}

// TestPostEthernetInterfaces verifies the iterative POST helper issues POST
// /Inventory/EthernetInterfaces and returns a nil per-item error on success.
func TestPostEthernetInterfaces(t *testing.T) {
	var gotMethod, gotPath string
	sc, srv := newTestSMD(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusCreated)
	})
	defer srv.Close()
	eis := []EthernetInterface{{ComponentID: "x0c0s0b0n0", MACAddress: "de:ad:be:ef:00:00"}}
	results := sc.PostEthernetInterfaces(context.Background(), eis, "tok")
	if len(results) != 1 || results[0].Err != nil {
		t.Errorf("results = %v, want a single success", results)
	}
	if gotMethod != http.MethodPost || gotPath != "/Inventory/EthernetInterfaces" {
		t.Errorf("request = %s %s, want POST /Inventory/EthernetInterfaces", gotMethod, gotPath)
	}
}

// TestPostGroupMembers verifies the iterative POST helper targets
// /groups/<group>/members.
func TestPostGroupMembers(t *testing.T) {
	var gotMethod, gotPath string
	sc, srv := newTestSMD(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusCreated)
	})
	defer srv.Close()
	results, err := sc.PostGroupMembers(context.Background(), "tok", "compute", "x0c0s0b0n0")
	if err != nil {
		t.Fatalf("PostGroupMembers: %v", err)
	}
	if len(results) != 1 || results[0].Err != nil {
		t.Errorf("results = %v, want a single success", results)
	}
	if gotMethod != http.MethodPost || gotPath != "/groups/compute/members" {
		t.Errorf("request = %s %s, want POST /groups/compute/members", gotMethod, gotPath)
	}
}

// TestPutGroupMembers verifies PutGroupMembers issues PUT
// /groups/<group>/members.
func TestPutGroupMembers(t *testing.T) {
	var gotMethod, gotPath string
	sc, srv := newTestSMD(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()
	if _, err := sc.PutGroupMembers(context.Background(), "tok", "compute", "x0c0s0b0n0"); err != nil {
		t.Fatalf("PutGroupMembers: %v", err)
	}
	if gotMethod != http.MethodPut || gotPath != "/groups/compute/members" {
		t.Errorf("request = %s %s, want PUT /groups/compute/members", gotMethod, gotPath)
	}
}

// TestBulkDeletes verifies the "*All" delete helpers issue DELETE to their
// respective collection endpoints.
func TestBulkDeletes(t *testing.T) {
	cases := []struct {
		name     string
		call     func(sc *SMDClient) error
		wantPath string
	}{
		{"components", func(sc *SMDClient) error { _, e := sc.DeleteComponentsAll(context.Background(), "tok"); return e }, "/State/Components"},
		{"rfe", func(sc *SMDClient) error { _, e := sc.DeleteRedfishEndpointsAll(context.Background(), "tok"); return e }, "/Inventory/RedfishEndpoints"},
		{"iface", func(sc *SMDClient) error {
			_, e := sc.DeleteEthernetInterfacesAll(context.Background(), "tok")
			return e
		}, "/Inventory/EthernetInterfaces"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var gotMethod, gotPath string
			sc, srv := newTestSMD(t, func(w http.ResponseWriter, r *http.Request) {
				gotMethod, gotPath = r.Method, r.URL.Path
				w.WriteHeader(http.StatusOK)
			})
			defer srv.Close()
			if err := tc.call(sc); err != nil {
				t.Fatalf("%s: %v", tc.name, err)
			}
			if gotMethod != http.MethodDelete || !strings.HasPrefix(gotPath, tc.wantPath) {
				t.Errorf("request = %s %s, want DELETE %s", gotMethod, gotPath, tc.wantPath)
			}
		})
	}
}

// TestIterativeDeletesPreserveMixedResultAlignment verifies that each iterative
// delete keeps the success and failure results in input order.
func TestIterativeDeletesPreserveMixedResultAlignment(t *testing.T) {
	cases := []struct {
		name string
		call func(sc *SMDClient) (client.BatchResult[client.HTTPEnvelope], error)
	}{
		{"components", func(sc *SMDClient) (client.BatchResult[client.HTTPEnvelope], error) {
			return sc.DeleteComponents(context.Background(), "tok", "x0c0s0b0n0", "x0c0s0b0n1"), nil
		}},
		{"rfe", func(sc *SMDClient) (client.BatchResult[client.HTTPEnvelope], error) {
			return sc.DeleteRedfishEndpoints(context.Background(), "tok", "x0c0s0b0", "x0c0s0b1"), nil
		}},
		{"iface", func(sc *SMDClient) (client.BatchResult[client.HTTPEnvelope], error) {
			return sc.DeleteEthernetInterfaces(context.Background(), "tok", "de:ad:be:ef:00:01", "de:ad:be:ef:00:02"), nil
		}},
		{"compendpoints", func(sc *SMDClient) (client.BatchResult[client.HTTPEnvelope], error) {
			return sc.DeleteComponentEndpoints(context.Background(), "tok", "x0c0s0b0n0", "x0c0s0b0n1"), nil
		}},
		{"groups", func(sc *SMDClient) (client.BatchResult[client.HTTPEnvelope], error) {
			return sc.DeleteGroups(context.Background(), "tok", "compute", "storage"), nil
		}},
		{"groupmembers", func(sc *SMDClient) (client.BatchResult[client.HTTPEnvelope], error) {
			return sc.DeleteGroupMembers(context.Background(), "tok", "compute", "x0c0s0b0n0", "x0c0s0b0n1")
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			requests := 0
			sc, srv := newTestSMD(t, func(w http.ResponseWriter, r *http.Request) {
				requests++
				if requests == 1 {
					w.WriteHeader(http.StatusOK)
					return
				}
				http.Error(w, "boom", http.StatusInternalServerError)
			})
			defer srv.Close()

			results, err := tc.call(sc)
			if err != nil {
				t.Fatalf("%s: control-flow error = %v, want nil", tc.name, err)
			}
			if len(results) != 2 {
				t.Fatalf("%s: results length = %d, want 2", tc.name, len(results))
			}
			if results[0].Err != nil || results[0].Value.StatusCode != http.StatusOK {
				t.Errorf("%s: first result = (status %d, err %v), want success", tc.name, results[0].Value.StatusCode, results[0].Err)
			}
			if !errors.Is(results[1].Err, client.UnsuccessfulHTTPError) || results[1].Value.StatusCode != http.StatusInternalServerError {
				t.Errorf("%s: second result = (status %d, err %v), want HTTP failure", tc.name, results[1].Value.StatusCode, results[1].Err)
			}
		})
	}
}

// TestDeleteGroupMembersIterative verifies the iterative group-member delete
// issues one DELETE per member under /groups/<group>/members.
func TestDeleteGroupMembersIterative(t *testing.T) {
	var paths []string
	sc, srv := newTestSMD(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			paths = append(paths, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()
	results, err := sc.DeleteGroupMembers(context.Background(), "tok", "compute", "x0c0s0b0n0", "x0c0s0b0n1")
	if err != nil {
		t.Fatalf("DeleteGroupMembers: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("results length = %d, want 2", len(results))
	}
	for _, p := range paths {
		if !strings.HasPrefix(p, "/groups/compute/members") {
			t.Errorf("delete path = %q, want under /groups/compute/members", p)
		}
	}
}
