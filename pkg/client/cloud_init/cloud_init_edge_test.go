// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cloud_init

// cloud_init_edge_test.go exercises the per-item edge cases of the mutating
// CloudInitClient helpers: blank names/IDs, empty lists, and per-item HTTP
// failures.

import (
	"errors"
	"net/http"
	"testing"

	"github.com/openchami/cloud-init/pkg/cistore"

	"github.com/openchami/ochami/pkg/client"
)

func TestPutGroupsEdgeCases(t *testing.T) {
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

func TestPutInstanceInfoEdgeCases(t *testing.T) {
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

func TestCloudInitIterativeMixedResults(t *testing.T) {
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
