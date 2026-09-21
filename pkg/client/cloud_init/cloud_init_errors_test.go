// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cloud_init

// cloud_init_errors_test.go unit-tests the CloudInitClient wrapper methods'
// error arms: input rejected before a request is made, a non-2XX response
// surfacing as an UnsuccessfulHTTPError, and the per-item edge cases of the
// mutating helpers (blank names/IDs, empty lists, and per-item HTTP
// failures). It also covers DecodeCloudConfig's own rejection case (see
// cloud_init_test.go for its success cases).

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/openchami/cloud-init/pkg/cistore"

	"github.com/openchami/ochami/pkg/client"
)

// TestGetNodeData_RequiresID verifies GetNodeData errors (without a request) when
// no IDs are supplied.
func TestGetNodeData_RequiresID(t *testing.T) {
	requestMade := false
	cic, srv := newTestCI(t, func(w http.ResponseWriter, r *http.Request) { requestMade = true })
	defer srv.Close()

	if _, _, err := cic.GetNodeData(CloudInitMetaData, "tok"); err == nil {
		t.Fatal("expected an error when no IDs are supplied, got nil")
	}
	if requestMade {
		t.Error("a request was made despite no IDs being supplied")
	}
}

// TestGetDefaults_UnsuccessfulHTTP verifies a non-2XX response surfaces as an
// UnsuccessfulHTTPError.
func TestGetDefaults_UnsuccessfulHTTP(t *testing.T) {
	cic, srv := newTestCI(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusForbidden)
	})
	defer srv.Close()

	_, err := cic.GetDefaults("tok")
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !errors.Is(err, client.UnsuccessfulHTTPError) {
		t.Errorf("error = %v, want it to wrap client.UnsuccessfulHTTPError", err)
	}
}

// TestDecodeCloudConfig_UnknownEncoding verifies that an unrecognized
// CloudConfigFile.Encoding value is rejected.
func TestDecodeCloudConfig_UnknownEncoding(t *testing.T) {
	if _, err := DecodeCloudConfig(cistore.CloudConfigFile{Content: []byte("x"), Encoding: "rot13"}); err == nil {
		t.Error("expected an error for unknown encoding, got nil")
	}
}

func TestPutGroups_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		groups  []cistore.GroupData
		status  int
		wantErr bool
	}{
		{"blank name", []cistore.GroupData{{Name: "  "}}, http.StatusOK, true},
		{"http error", []cistore.GroupData{{Name: "compute"}}, http.StatusInternalServerError, true},
		{"success", []cistore.GroupData{{Name: "compute"}}, http.StatusOK, false},
	}
	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			cic, srv := newTestCI(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
			})
			defer srv.Close()

			_, errs, err := cic.PutGroups(tc.groups, "tok")
			if err != nil {
				t.Fatalf("PutGroups: control-flow error = %v", err)
			}
			if len(errs) != 1 || (errs[0] != nil) != tc.wantErr {
				t.Errorf("per-item errors = %v, wantErr %v", errs, tc.wantErr)
			}
		})
	}
}

func TestPutInstanceInfo_EdgeCases(t *testing.T) {
	t.Run("empty list", func(t *testing.T) {
		cic, srv := newTestCI(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		defer srv.Close()
		if _, _, err := cic.PutInstanceInfo(nil, "tok"); err == nil {
			t.Fatal("expected control-flow error for empty list, got nil")
		}
	})

	tests := []struct {
		name    string
		infos   []cistore.OpenCHAMIInstanceInfo
		status  int
		wantErr bool
	}{
		{"blank id", []cistore.OpenCHAMIInstanceInfo{{ID: "  "}}, http.StatusOK, true},
		{"http error", []cistore.OpenCHAMIInstanceInfo{{ID: "x0c0s0b0n0"}}, http.StatusInternalServerError, true},
		{"success", []cistore.OpenCHAMIInstanceInfo{{ID: "x0c0s0b0n0"}}, http.StatusOK, false},
	}
	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			cic, srv := newTestCI(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
			})
			defer srv.Close()

			_, errs, err := cic.PutInstanceInfo(tc.infos, "tok")
			if err != nil {
				t.Fatalf("PutInstanceInfo: control-flow error = %v", err)
			}
			if len(errs) != 1 || (errs[0] != nil) != tc.wantErr {
				t.Errorf("per-item errors = %v, wantErr %v", errs, tc.wantErr)
			}
		})
	}
}

func TestCloudInitIterative_MixedResults(t *testing.T) {
	cases := []struct {
		name       string
		wantMethod string
		call       func(cic *CloudInitClient) ([]client.HTTPEnvelope, []error, error)
	}{
		{"PostGroups", http.MethodPost, func(cic *CloudInitClient) ([]client.HTTPEnvelope, []error, error) {
			return cic.PostGroups([]cistore.GroupData{{Name: "compute"}, {Name: "storage"}}, "tok")
		}},
		{"PutGroups", http.MethodPut, func(cic *CloudInitClient) ([]client.HTTPEnvelope, []error, error) {
			return cic.PutGroups([]cistore.GroupData{{Name: "compute"}, {Name: "storage"}}, "tok")
		}},
		{"PutInstanceInfo", http.MethodPut, func(cic *CloudInitClient) ([]client.HTTPEnvelope, []error, error) {
			return cic.PutInstanceInfo([]cistore.OpenCHAMIInstanceInfo{{ID: "x0c0s0b0n0"}, {ID: "x0c0s0b0n1"}}, "tok")
		}},
		{"DeleteGroups", http.MethodDelete, func(cic *CloudInitClient) ([]client.HTTPEnvelope, []error, error) {
			return cic.DeleteGroups("tok", "compute", "storage")
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			requests := 0
			cic, srv := newTestCI(t, func(w http.ResponseWriter, r *http.Request) {
				requests++
				if r.Method != tc.wantMethod {
					t.Errorf("request method = %s, want %s", r.Method, tc.wantMethod)
				}
				if got := r.Header.Get("Authorization"); got != "Bearer tok" {
					t.Errorf("Authorization = %q, want %q", got, "Bearer tok")
				}
				if requests == 1 {
					w.WriteHeader(http.StatusOK)
					return
				}
				http.Error(w, "boom", http.StatusInternalServerError)
			})
			defer srv.Close()

			henvs, errs, err := tc.call(cic)
			if err != nil {
				t.Fatalf("%s: control-flow error = %v", tc.name, err)
			}
			if len(henvs) != 2 || len(errs) != 2 {
				t.Fatalf("%s: result lengths = (%d, %d), want (2, 2)", tc.name, len(henvs), len(errs))
			}
			if errs[0] != nil || henvs[0].StatusCode != http.StatusOK {
				t.Errorf("%s: first result = (status %d, err %v), want success", tc.name, henvs[0].StatusCode, errs[0])
			}
			if !errors.Is(errs[1], client.UnsuccessfulHTTPError) || henvs[1].StatusCode != http.StatusInternalServerError {
				t.Errorf("%s: second result = (status %d, err %v), want HTTP failure", tc.name, henvs[1].StatusCode, errs[1])
			}
		})
	}
}

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

// TestCloudConfigGetters_PreserveMalformedBodies verifies that successful GETs
// return server data unchanged. Parsing remains the caller's responsibility,
// so malformed cloud-config can be diagnosed without losing the response.
func TestCloudConfigGetters_PreserveMalformedBodies(t *testing.T) {
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
