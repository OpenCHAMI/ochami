// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package smd

// smd_test.go unit-tests representative SMDClient wrapper methods against an
// httptest.Server: single-resource GETs, list GETs with query strings, the
// iterative multi-item POST/PUT/DELETE helpers (which return per-item error
// slices), the bulk "*All" deletes, and path construction for sub-resources.
// Error-arm behavior is covered in smd_errors_test.go.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/openchami/schemas/schemas/csm"
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

// TestGetComponents_All verifies GET /State/Components.
func TestGetComponents_All(t *testing.T) {
	var gotMethod, gotPath string
	sc, srv := newTestSMD(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_, _ = w.Write([]byte(`{"Components":[]}`)) //nolint:errcheck // test response writes are observed by the client
	})
	defer srv.Close()

	if _, err := sc.GetComponentsAll(); err != nil {
		t.Fatalf("GetComponentsAll: %v", err)
	}
	if gotMethod != http.MethodGet || gotPath != "/State/Components" {
		t.Errorf("request = %s %s, want GET /State/Components", gotMethod, gotPath)
	}
}

// TestGetComponents_Xname verifies GET /State/Components/{xname} with the auth
// header set.
func TestGetComponents_Xname(t *testing.T) {
	var gotPath, gotAuth string
	sc, srv := newTestSMD(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotAuth = r.URL.Path, r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{}`)) //nolint:errcheck // test response writes are observed by the client
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
				_, _ = w.Write([]byte(`{}`)) //nolint:errcheck // test response writes are observed by the client
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

// TestGetGroupMembers_Success verifies the /groups/{group}/members sub-resource path.
func TestGetGroupMembers_Success(t *testing.T) {
	var gotPath string
	sc, srv := newTestSMD(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`[]`)) //nolint:errcheck // test response writes are observed by the client
	})
	defer srv.Close()

	if _, err := sc.GetGroupMembers("compute", "tok"); err != nil {
		t.Fatalf("GetGroupMembers: %v", err)
	}
	if gotPath != "/groups/compute/members" {
		t.Errorf("path = %q, want /groups/compute/members", gotPath)
	}
}

// TestPostComponents_Success verifies the POST /State/Components request path/method.
func TestPostComponents_Success(t *testing.T) {
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

// TestDeleteComponents_Iterative verifies the iterative DeleteComponents helper
// issues one DELETE per xname and returns a nil per-item error on success.
func TestDeleteComponents_Iterative(t *testing.T) {
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
			if _, err := sc.GetStatus(tc.component); err != nil {
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
		if _, err := sc.GetEthernetInterfaceByID("deadbeef", "tok", false); err != nil {
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
		if _, err := sc.GetEthernetInterfaceByID("deadbeef", "tok", true); err != nil {
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
	if _, err := sc.GetGroupMembership("id=x0", "tok"); err != nil {
		t.Fatalf("GetGroupMembership: %v", err)
	}
	if gotPath != "/memberships" {
		t.Errorf("path = %q, want /memberships", gotPath)
	}
}

// TestPutComponents_Success verifies PutComponents issues PUT /State/Components.
func TestPutComponents_Success(t *testing.T) {
	var gotMethod, gotPath string
	sc, srv := newTestSMD(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()
	_, errs, err := sc.PutComponents(ComponentSlice{Components: []Component{{ID: "x0c0s0b0n0"}}}, "tok")
	if err != nil {
		t.Fatalf("PutComponents: %v", err)
	}
	if len(errs) != 1 || errs[0] != nil {
		t.Errorf("per-item errors = %v, want a single nil", errs)
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
	_, errs, err := sc.PostEthernetInterfaces(eis, "tok")
	if err != nil {
		t.Fatalf("PostEthernetInterfaces: %v", err)
	}
	if len(errs) != 1 || errs[0] != nil {
		t.Errorf("per-item errors = %v, want a single nil", errs)
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
	_, errs, err := sc.PostGroupMembers("tok", "compute", "x0c0s0b0n0")
	if err != nil {
		t.Fatalf("PostGroupMembers: %v", err)
	}
	if len(errs) != 1 || errs[0] != nil {
		t.Errorf("per-item errors = %v, want a single nil", errs)
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
	if _, err := sc.PutGroupMembers("tok", "compute", "x0c0s0b0n0"); err != nil {
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
		{"components", func(sc *SMDClient) error { _, e := sc.DeleteComponentsAll("tok"); return e }, "/State/Components"},
		{"rfe", func(sc *SMDClient) error { _, e := sc.DeleteRedfishEndpointsAll("tok"); return e }, "/Inventory/RedfishEndpoints"},
		{"iface", func(sc *SMDClient) error { _, e := sc.DeleteEthernetInterfacesAll("tok"); return e }, "/Inventory/EthernetInterfaces"},
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

// TestDeleteGroupMembers_Iterative verifies the iterative group-member delete
// issues one DELETE per member under /groups/<group>/members.
func TestDeleteGroupMembers_Iterative(t *testing.T) {
	var paths []string
	sc, srv := newTestSMD(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			paths = append(paths, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()
	_, errs, err := sc.DeleteGroupMembers("tok", "compute", "x0c0s0b0n0", "x0c0s0b0n1")
	if err != nil {
		t.Fatalf("DeleteGroupMembers: %v", err)
	}
	if len(errs) != 2 {
		t.Fatalf("per-item errors length = %d, want 2", len(errs))
	}
	for _, p := range paths {
		if !strings.HasPrefix(p, "/groups/compute/members") {
			t.Errorf("delete path = %q, want under /groups/compute/members", p)
		}
	}
}

// TestAdditionalSMDClientPaths verifies routing for a grab-bag of remaining
// single-resource and bulk-delete SMDClient methods.
func TestAdditionalSMDClientPaths(t *testing.T) {
	tests := []struct {
		name       string
		call       func(*SMDClient) error
		wantMethod string
		wantPath   string
	}{
		{"component by NID", func(c *SMDClient) error { _, err := c.GetComponentsNid(42, "tok"); return err }, http.MethodGet, "/State/Components/ByNID/42"},
		{"component endpoint", func(c *SMDClient) error { _, _, err := c.GetComponentEndpoints("tok", "x0"); return err }, http.MethodGet, "/Inventory/ComponentEndpoints/x0"},
		{"put redfish endpoint", func(c *SMDClient) error {
			_, errs, err := c.PutRedfishEndpoints(RedfishEndpointSlice{RedfishEndpoints: []csm.RedfishEndpoint{{ID: "x0"}}}, "tok")
			if err == nil && len(errs) > 0 {
				err = errs[0]
			}
			return err
		}, http.MethodPut, "/Inventory/RedfishEndpoints/x0"},
		{"put redfish endpoint v2", func(c *SMDClient) error {
			_, errs, err := c.PutRedfishEndpointsV2(RedfishEndpointSliceV2{RedfishEndpoints: []RedfishEndpointV2{{RedfishEndpoint: csm.RedfishEndpoint{ID: "x0"}}}}, "tok")
			if err == nil && len(errs) > 0 {
				err = errs[0]
			}
			return err
		}, http.MethodPut, "/Inventory/RedfishEndpoints/x0"},
		{"patch interface", func(c *SMDClient) error {
			_, errs, err := c.PatchEthernetInterfaces([]EthernetInterface{{ID: "eth0"}}, "tok")
			if err == nil && len(errs) > 0 {
				err = errs[0]
			}
			return err
		}, http.MethodPatch, "/Inventory/EthernetInterfaces/eth0"},
		{"patch group", func(c *SMDClient) error {
			_, errs, err := c.PatchGroups([]Group{{Label: "compute"}}, "tok")
			if err == nil && len(errs) > 0 {
				err = errs[0]
			}
			return err
		}, http.MethodPatch, "/groups/compute"},
		{"delete all component endpoints", func(c *SMDClient) error { _, err := c.DeleteComponentEndpointsAll("tok"); return err }, http.MethodDelete, "/Inventory/ComponentEndpoints"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotMethod, gotPath, gotAuth string
			c, srv := newTestSMD(t, func(w http.ResponseWriter, r *http.Request) {
				gotMethod, gotPath, gotAuth = r.Method, r.URL.Path, r.Header.Get("Authorization")
				w.WriteHeader(http.StatusOK)
			})
			defer srv.Close()
			if err := tc.call(c); err != nil {
				t.Fatalf("call: %v", err)
			}
			if gotMethod != tc.wantMethod || gotPath != tc.wantPath {
				t.Errorf("request = %s %s, want %s %s", gotMethod, gotPath, tc.wantMethod, tc.wantPath)
			}
			if gotAuth != "Bearer tok" {
				t.Errorf("Authorization = %q", gotAuth)
			}
		})
	}
}
