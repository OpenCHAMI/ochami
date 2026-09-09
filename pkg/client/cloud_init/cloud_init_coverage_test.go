// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cloud_init

// cloud_init_coverage_test.go covers additional reachable branches of the
// cloud-init client wrappers: the argument-guard clauses of GetNodeGroupData
// and GetNodeData, and the per-item HTTP error arms of the iterative Put/Post
// helpers.

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/openchami/cloud-init/pkg/cistore"

	"github.com/openchami/ochami/pkg/client"
)

// TestGetNodeGroupDataGuards verifies the blank-id and empty-groups guards.
func TestGetNodeGroupDataGuards(t *testing.T) {
	cic, srv := newTestCI(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	if _, err := cic.GetNodeGroupData(context.Background(), "tok", "  "); err == nil {
		t.Error("GetNodeGroupData with blank id = nil, want error")
	}
	if _, err := cic.GetNodeGroupData(context.Background(), "tok", "x0c0s0b0n0"); err == nil {
		t.Error("GetNodeGroupData with no groups = nil, want error")
	}
}

// TestGetNodeDataGuard verifies the empty-ids guard.
func TestGetNodeDataGuard(t *testing.T) {
	cic, srv := newTestCI(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	if _, err := cic.GetNodeData(context.Background(), CloudInitMetaData, "tok"); err == nil {
		t.Error("GetNodeData with no ids = nil, want error")
	}
}

// TestPutGroupsPerItemError verifies the per-item PUT error arm.
func TestPutGroupsPerItemError(t *testing.T) {
	cic, srv := newTestCI(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	})
	defer srv.Close()

	groups := []cistore.GroupData{{Name: "compute"}}
	results := cic.PutGroups(context.Background(), groups, "tok")
	if len(results) != 1 || results[0].Err == nil {
		t.Errorf("results = %v, want one failed result", results)
	}
}

// TestPutInstanceInfoPerItemError verifies the per-item PUT error arm.
func TestPutInstanceInfoPerItemError(t *testing.T) {
	cic, srv := newTestCI(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	})
	defer srv.Close()

	iis := []cistore.OpenCHAMIInstanceInfo{{ID: "x0c0s0b0n0"}}
	results, err := cic.PutInstanceInfo(context.Background(), iis, "tok")
	if err != nil {
		t.Fatalf("function error = %v, want nil", err)
	}
	if len(results) != 1 || results[0].Err == nil {
		t.Errorf("results = %v, want one failed result", results)
	}
}

// TestPostGroupsPerItemError verifies the per-item POST error arm.
func TestPostGroupsPerItemError(t *testing.T) {
	cic, srv := newTestCI(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	})
	defer srv.Close()

	groups := []cistore.GroupData{{Name: "compute"}}
	results := cic.PostGroups(context.Background(), groups, "tok")
	if len(results) != 1 || results[0].Err == nil {
		t.Errorf("results = %v, want one failed result", results)
	}
}

// TestPostDefaultsHTTPError verifies the PostDefaults HTTP error arm.
func TestPostDefaultsHTTPError(t *testing.T) {
	cic, srv := newTestCI(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	})
	defer srv.Close()

	if _, err := cic.PostDefaults(context.Background(), cistore.ClusterDefaults{}, "tok"); err == nil {
		t.Error("PostDefaults with HTTP error = nil, want error")
	}
}

// TestPutPostGroupsSuccess verifies the success arms of the iterative group
// writers against a 200 server.
func TestPutPostGroupsSuccess(t *testing.T) {
	cic, srv := newTestCI(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	groups := []cistore.GroupData{{Name: "compute"}}
	if results := cic.PutGroups(context.Background(), groups, "tok"); results.HasErrors() {
		t.Errorf("PutGroups results = %v, want no errors", results)
	}
	if results := cic.PostGroups(context.Background(), groups, "tok"); results.HasErrors() {
		t.Errorf("PostGroups results = %v, want no errors", results)
	}
}

// TestPutGroupsBlankName verifies the blank-name guard of PutGroups.
func TestPutGroupsBlankName(t *testing.T) {
	cic, srv := newTestCI(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	groups := []cistore.GroupData{{Name: "  "}}
	results := cic.PutGroups(context.Background(), groups, "tok")
	if len(results) != 1 || results[0].Err == nil {
		t.Errorf("results = %v, want one blank-name error", results)
	}
}

// TestDeleteGroupsSuccess verifies the success arm of DeleteGroups.
func TestDeleteGroupsSuccess(t *testing.T) {
	cic, srv := newTestCI(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	if results := cic.DeleteGroups(context.Background(), "tok", "compute"); results.HasErrors() {
		t.Errorf("DeleteGroups results = %v, want no errors", results)
	}
}

// TestGetNodeDataSuccess verifies the success arm of GetNodeData.
func TestGetNodeDataSuccess(t *testing.T) {
	cic, srv := newTestCI(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{}`)) //nolint:errcheck // test response writes are observed by the client
	})
	defer srv.Close()

	if results, err := cic.GetNodeData(context.Background(), CloudInitMetaData, "tok", "x0c0s0b0n0"); err != nil || results.HasErrors() {
		t.Errorf("GetNodeData success = (err=%v, results=%v), want no errors", err, results)
	}
}

// TestCloudConfigGettersPreserveMalformedBodies verifies that successful GETs
// return server data unchanged. Parsing remains the caller's responsibility,
// so malformed cloud-config can be diagnosed without losing the response.
func TestCloudConfigGettersPreserveMalformedBodies(t *testing.T) {
	const malformed = "not-base64!"
	tests := []struct {
		name string
		call func(*CloudInitClient) (client.BatchResult[client.HTTPEnvelope], error)
	}{
		{name: "node data", call: func(cic *CloudInitClient) (client.BatchResult[client.HTTPEnvelope], error) {
			return cic.GetNodeData(context.Background(), CloudInitUserData, "tok", "x0c0s0b0n0")
		}},
		{name: "node group data", call: func(cic *CloudInitClient) (client.BatchResult[client.HTTPEnvelope], error) {
			return cic.GetNodeGroupData(context.Background(), "tok", "x0c0s0b0n0", "compute")
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cic, srv := newTestCI(t, func(w http.ResponseWriter, r *http.Request) {
				if got := r.Header.Get("Authorization"); got != "Bearer tok" {
					t.Errorf("Authorization = %q, want %q", got, "Bearer tok")
				}
				_, _ = w.Write([]byte(malformed)) //nolint:errcheck // test response writes are observed by the client
			})
			defer srv.Close()

			results, err := tc.call(cic)
			if err != nil {
				t.Fatalf("control-flow error = %v", err)
			}
			if len(results) != 1 || results[0].Err != nil {
				t.Fatalf("results = %v, want one successful response", results)
			}
			if got := string(results[0].Value.Body); got != malformed {
				t.Errorf("body = %q, want %q", got, malformed)
			}
			if _, err := DecodeCloudConfig(cistore.CloudConfigFile{Content: []byte(malformed), Encoding: "base64"}); err == nil || !strings.Contains(err.Error(), "base64 decode") {
				t.Errorf("DecodeCloudConfig() error = %v, want contextual base64 error", err)
			}
		})
	}
}
