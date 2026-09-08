// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cloud_init

// cloud_init_coverage_test.go covers additional reachable branches of the
// cloud-init client wrappers: the argument-guard clauses of GetNodeGroupData
// and GetNodeData, and the per-item HTTP error arms of the iterative Put/Post
// helpers.

import (
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

	if _, _, err := cic.GetNodeGroupData("tok", "  "); err == nil {
		t.Error("GetNodeGroupData with blank id = nil, want error")
	}
	if _, _, err := cic.GetNodeGroupData("tok", "x0c0s0b0n0"); err == nil {
		t.Error("GetNodeGroupData with no groups = nil, want error")
	}
}

// TestGetNodeDataGuard verifies the empty-ids guard.
func TestGetNodeDataGuard(t *testing.T) {
	cic, srv := newTestCI(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	if _, _, err := cic.GetNodeData(CloudInitMetaData, "tok"); err == nil {
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
	_, errs, err := cic.PutGroups(groups, "tok")
	if err != nil {
		t.Fatalf("function error = %v, want nil", err)
	}
	if len(errs) != 1 || errs[0] == nil {
		t.Errorf("errs = %v, want one non-nil per-item error", errs)
	}
}

// TestPutInstanceInfoPerItemError verifies the per-item PUT error arm.
func TestPutInstanceInfoPerItemError(t *testing.T) {
	cic, srv := newTestCI(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	})
	defer srv.Close()

	iis := []cistore.OpenCHAMIInstanceInfo{{ID: "x0c0s0b0n0"}}
	_, errs, err := cic.PutInstanceInfo(iis, "tok")
	if err != nil {
		t.Fatalf("function error = %v, want nil", err)
	}
	if len(errs) != 1 || errs[0] == nil {
		t.Errorf("errs = %v, want one non-nil per-item error", errs)
	}
}

// TestPostGroupsPerItemError verifies the per-item POST error arm.
func TestPostGroupsPerItemError(t *testing.T) {
	cic, srv := newTestCI(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	})
	defer srv.Close()

	groups := []cistore.GroupData{{Name: "compute"}}
	_, errs, err := cic.PostGroups(groups, "tok")
	if err != nil {
		t.Fatalf("function error = %v, want nil", err)
	}
	if len(errs) != 1 || errs[0] == nil {
		t.Errorf("errs = %v, want one non-nil per-item error", errs)
	}
}

// TestPostDefaultsHTTPError verifies the PostDefaults HTTP error arm.
func TestPostDefaultsHTTPError(t *testing.T) {
	cic, srv := newTestCI(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	})
	defer srv.Close()

	if _, err := cic.PostDefaults(cistore.ClusterDefaults{}, "tok"); err == nil {
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
	if _, errs, err := cic.PutGroups(groups, "tok"); err != nil || (len(errs) == 1 && errs[0] != nil) {
		t.Errorf("PutGroups success = (err=%v, errs=%v), want no errors", err, errs)
	}
	if _, errs, err := cic.PostGroups(groups, "tok"); err != nil || (len(errs) == 1 && errs[0] != nil) {
		t.Errorf("PostGroups success = (err=%v, errs=%v), want no errors", err, errs)
	}
}

// TestPutGroupsBlankName verifies the blank-name guard of PutGroups.
func TestPutGroupsBlankName(t *testing.T) {
	cic, srv := newTestCI(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	groups := []cistore.GroupData{{Name: "  "}}
	_, errs, err := cic.PutGroups(groups, "tok")
	if err != nil {
		t.Fatalf("PutGroups func error = %v, want nil", err)
	}
	if len(errs) != 1 || errs[0] == nil {
		t.Errorf("errs = %v, want one non-nil blank-name error", errs)
	}
}

// TestDeleteGroupsSuccess verifies the success arm of DeleteGroups.
func TestDeleteGroupsSuccess(t *testing.T) {
	cic, srv := newTestCI(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	if _, errs, err := cic.DeleteGroups("tok", "compute"); err != nil || (len(errs) == 1 && errs[0] != nil) {
		t.Errorf("DeleteGroups success = (err=%v, errs=%v), want no errors", err, errs)
	}
}

// TestGetNodeDataSuccess verifies the success arm of GetNodeData.
func TestGetNodeDataSuccess(t *testing.T) {
	cic, srv := newTestCI(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{}`)) //nolint:errcheck // test response writes are observed by the client
	})
	defer srv.Close()

	if _, errs, err := cic.GetNodeData(CloudInitMetaData, "tok", "x0c0s0b0n0"); err != nil || (len(errs) == 1 && errs[0] != nil) {
		t.Errorf("GetNodeData success = (err=%v, errs=%v), want no errors", err, errs)
	}
}

// TestCloudConfigGettersPreserveMalformedBodies verifies that successful GETs
// return server data unchanged. Parsing remains the caller's responsibility,
// so malformed cloud-config can be diagnosed without losing the response.
func TestCloudConfigGettersPreserveMalformedBodies(t *testing.T) {
	const malformed = "not-base64!"
	tests := []struct {
		name string
		call func(*CloudInitClient) ([]client.HTTPEnvelope, []error, error)
	}{
		{name: "node data", call: func(cic *CloudInitClient) ([]client.HTTPEnvelope, []error, error) {
			return cic.GetNodeData(CloudInitUserData, "tok", "x0c0s0b0n0")
		}},
		{name: "node group data", call: func(cic *CloudInitClient) ([]client.HTTPEnvelope, []error, error) {
			return cic.GetNodeGroupData("tok", "x0c0s0b0n0", "compute")
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

			henvs, errs, err := tc.call(cic)
			if err != nil {
				t.Fatalf("control-flow error = %v", err)
			}
			if len(henvs) != 1 || len(errs) != 1 || errs[0] != nil {
				t.Fatalf("results = (%v, %v), want one successful response", henvs, errs)
			}
			if got := string(henvs[0].Body); got != malformed {
				t.Errorf("body = %q, want %q", got, malformed)
			}
			if _, err := DecodeCloudConfig(cistore.CloudConfigFile{Content: []byte(malformed), Encoding: "base64"}); err == nil || !strings.Contains(err.Error(), "base64 decode") {
				t.Errorf("DecodeCloudConfig() error = %v, want contextual base64 error", err)
			}
		})
	}
}
