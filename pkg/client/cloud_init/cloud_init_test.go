// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cloud_init

// cloud_init_test.go unit-tests the CloudInitClient wrapper methods against an
// httptest.Server: the simple GET endpoints (version, defaults, api), the
// iterative multi-item getters and setters (groups, node data, instance
// info) including their per-item error slices and path construction, and the
// pure helpers (CIGroupDataMapToSlice, DecodeCloudConfig). Error-arm behavior
// is covered in cloud_init_errors_test.go.

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/openchami/cloud-init/pkg/cistore"
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
				_, _ = w.Write([]byte(`{}`)) //nolint:errcheck // test response writes are observed by the client
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
		_, _ = w.Write([]byte(`{}`)) //nolint:errcheck // test response writes are observed by the client
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
		_, _ = w.Write([]byte(`{}`)) //nolint:errcheck // test response writes are observed by the client
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
		_, _ = w.Write([]byte(`data`)) //nolint:errcheck // test response writes are observed by the client
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

// TestPostDefaults_Success verifies PostDefaults issues POST /admin/cluster-defaults.
func TestPostDefaults_Success(t *testing.T) {
	var gotMethod, gotPath string
	cic, srv := newTestCI(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()
	if _, err := cic.PostDefaults(cistore.ClusterDefaults{ClusterName: "demo"}, "tok"); err != nil {
		t.Fatalf("PostDefaults: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/admin/cluster-defaults" {
		t.Errorf("request = %s %s, want POST /admin/cluster-defaults", gotMethod, gotPath)
	}
}

// TestPostGroups verifies the iterative PostGroups issues POST /admin/groups and
// returns a nil per-item error on success.
func TestPostGroups(t *testing.T) {
	var gotMethod, gotPath string
	cic, srv := newTestCI(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusCreated)
	})
	defer srv.Close()
	_, errs, err := cic.PostGroups([]cistore.GroupData{{Name: "compute"}}, "tok")
	if err != nil {
		t.Fatalf("PostGroups: %v", err)
	}
	if len(errs) != 1 || errs[0] != nil {
		t.Errorf("per-item errors = %v, want a single nil", errs)
	}
	if gotMethod != http.MethodPost || gotPath != "/admin/groups" {
		t.Errorf("request = %s %s, want POST /admin/groups", gotMethod, gotPath)
	}
}

// TestPutGroups_Success verifies the iterative PutGroups targets /admin/groups/<name>.
func TestPutGroups_Success(t *testing.T) {
	var gotMethod, gotPath string
	cic, srv := newTestCI(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()
	_, errs, err := cic.PutGroups([]cistore.GroupData{{Name: "compute"}}, "tok")
	if err != nil {
		t.Fatalf("PutGroups: %v", err)
	}
	if len(errs) != 1 || errs[0] != nil {
		t.Errorf("per-item errors = %v, want a single nil", errs)
	}
	if gotMethod != http.MethodPut || !strings.HasPrefix(gotPath, "/admin/groups/compute") {
		t.Errorf("request = %s %s, want PUT /admin/groups/compute", gotMethod, gotPath)
	}
}

// TestPutInstanceInfo_Success verifies the iterative PutInstanceInfo targets
// /admin/instance-info/<id>.
func TestPutInstanceInfo_Success(t *testing.T) {
	var gotMethod, gotPath string
	cic, srv := newTestCI(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()
	_, errs, err := cic.PutInstanceInfo([]cistore.OpenCHAMIInstanceInfo{{ID: "x0c0s0b0n0"}}, "tok")
	if err != nil {
		t.Fatalf("PutInstanceInfo: %v", err)
	}
	if len(errs) != 1 || errs[0] != nil {
		t.Errorf("per-item errors = %v, want a single nil", errs)
	}
	if gotMethod != http.MethodPut || !strings.HasPrefix(gotPath, "/admin/instance-info/x0c0s0b0n0") {
		t.Errorf("request = %s %s, want PUT /admin/instance-info/x0c0s0b0n0", gotMethod, gotPath)
	}
}

// TestDeleteGroups verifies the iterative DeleteGroups issues one DELETE per
// group under /admin/groups.
func TestDeleteGroups(t *testing.T) {
	var paths []string
	cic, srv := newTestCI(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			paths = append(paths, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()
	_, errs, err := cic.DeleteGroups("tok", "compute", "storage")
	if err != nil {
		t.Fatalf("DeleteGroups: %v", err)
	}
	if len(errs) != 2 {
		t.Fatalf("per-item errors length = %d, want 2", len(errs))
	}
	want := []string{"/admin/groups/compute", "/admin/groups/storage"}
	if len(paths) != 2 || paths[0] != want[0] || paths[1] != want[1] {
		t.Errorf("delete paths = %v, want %v", paths, want)
	}
}

// TestCIGroupDataMapToSlice verifies the map-to-slice conversion returns every
// group value.
func TestCIGroupDataMapToSlice(t *testing.T) {
	m := map[string]cistore.GroupData{
		"compute": {Name: "compute"},
		"storage": {Name: "storage"},
	}
	got := CIGroupDataMapToSlice(m)
	if len(got) != 2 {
		t.Fatalf("slice length = %d, want 2", len(got))
	}
	names := map[string]bool{}
	for _, g := range got {
		names[g.Name] = true
	}
	if !names["compute"] || !names["storage"] {
		t.Errorf("slice = %+v, want it to contain both groups", got)
	}
}

// TestDecodeCloudConfig verifies plain content passes through and base64
// content is decoded. The unknown-encoding rejection case is covered by
// TestDecodeCloudConfig_UnknownEncoding in cloud_init_errors_test.go.
func TestDecodeCloudConfig(t *testing.T) {
	t.Run("plain", func(t *testing.T) {
		out, err := DecodeCloudConfig(cistore.CloudConfigFile{Content: []byte("#cloud-config"), Encoding: "plain"})
		if err != nil {
			t.Fatalf("plain: %v", err)
		}
		if string(out) != "#cloud-config" {
			t.Errorf("out = %q, want the plain content", out)
		}
	})
	t.Run("base64", func(t *testing.T) {
		enc := base64.StdEncoding.EncodeToString([]byte("#cloud-config"))
		out, err := DecodeCloudConfig(cistore.CloudConfigFile{Content: []byte(enc), Encoding: "base64"})
		if err != nil {
			t.Fatalf("base64: %v", err)
		}
		if string(out) != "#cloud-config" {
			t.Errorf("out = %q, want the decoded content", out)
		}
	})
}
