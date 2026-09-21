// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cloud_init

// cloud_init_test.go unit-tests the CloudInitClient wrapper methods against an
// httptest.Server: the simple GET endpoints (version, defaults, api), and the
// iterative multi-item getters (groups, node data) including their per-item
// error slices and path construction. Error-arm behavior is covered in
// cloud_init_errors_test.go.

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestCI(t *testing.T, h http.HandlerFunc) (*CloudInitClient, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(h)
	cic, err := NewClient(srv.URL)
	if err != nil {
		srv.Close()
		t.Fatalf("NewClient: %v", err)
	}
	return cic, srv
}

// TestSimpleGetters verifies the single-shot GET endpoints route correctly.
func TestSimpleGetters(t *testing.T) {
	cases := []struct {
		name     string
		call     func(cic *CloudInitClient) error
		wantPath string
	}{
		{"version", func(cic *CloudInitClient) error { _, e := cic.GetVersion(); return e }, "/version"},
		{"api", func(cic *CloudInitClient) error { _, e := cic.GetAPI(); return e }, "/openapi.json"},
		{"defaults", func(cic *CloudInitClient) error { _, e := cic.GetDefaults("tok"); return e }, "/admin/cluster-defaults"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var gotPath string
			cic, srv := newTestCI(t, func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				_, _ = w.Write([]byte(`{}`))
			})
			defer srv.Close()

			if err := tc.call(cic); err != nil {
				t.Fatalf("%s: %v", tc.name, err)
			}
			if gotPath != tc.wantPath {
				t.Errorf("path = %q, want %q", gotPath, tc.wantPath)
			}
		})
	}
}

// TestGetGroups_All verifies GetGroups with no IDs issues a single GET to the
// groups collection and returns one nil per-item error.
func TestGetGroups_All(t *testing.T) {
	var gotPath string
	cic, srv := newTestCI(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{}`))
	})
	defer srv.Close()

	_, errs, err := cic.GetGroups("tok")
	if err != nil {
		t.Fatalf("GetGroups func error: %v", err)
	}
	if gotPath != "/admin/groups" {
		t.Errorf("path = %q, want /admin/groups", gotPath)
	}
	if len(errs) != 1 || errs[0] != nil {
		t.Errorf("per-item errors = %v, want a single nil", errs)
	}
}

// TestGetGroups_ByID verifies GetGroups with IDs issues one GET per ID to
// /admin/groups/{id}.
func TestGetGroups_ByID(t *testing.T) {
	var paths []string
	cic, srv := newTestCI(t, func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		_, _ = w.Write([]byte(`{}`))
	})
	defer srv.Close()

	_, errs, err := cic.GetGroups("tok", "compute", "storage")
	if err != nil {
		t.Fatalf("GetGroups func error: %v", err)
	}
	if len(errs) != 2 {
		t.Fatalf("per-item errors length = %d, want 2", len(errs))
	}
	want := []string{"/admin/groups/compute", "/admin/groups/storage"}
	if len(paths) != 2 || paths[0] != want[0] || paths[1] != want[1] {
		t.Errorf("paths = %v, want %v", paths, want)
	}
}

// TestGetNodeData_Success verifies GetNodeData builds the impersonation path per ID and
// data type.
func TestGetNodeData_Success(t *testing.T) {
	var gotPath string
	cic, srv := newTestCI(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`data`))
	})
	defer srv.Close()

	_, errs, err := cic.GetNodeData(CloudInitUserData, "tok", "x0c0s0b0n0")
	if err != nil {
		t.Fatalf("GetNodeData func error: %v", err)
	}
	if len(errs) != 1 || errs[0] != nil {
		t.Errorf("per-item errors = %v, want a single nil", errs)
	}
	if gotPath != "/admin/impersonation/x0c0s0b0n0/user-data" {
		t.Errorf("path = %q, want /admin/impersonation/x0c0s0b0n0/user-data", gotPath)
	}
}
