// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cloud_init

// cloud_init_edge_test.go exercises the per-item edge cases of the mutating
// CloudInitClient helpers: blank names/IDs, empty lists, and per-item HTTP
// failures.

import (
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

func TestCloudInitIterativeHTTPErrors(t *testing.T) {
	cases := []struct {
		name string
		call func(cic *CloudInitClient) ([]client.HTTPEnvelope, []error, error)
	}{
		{"PostGroups", func(cic *CloudInitClient) ([]client.HTTPEnvelope, []error, error) {
			return cic.PostGroups([]cistore.GroupData{{Name: "compute"}}, "tok")
		}},
		{"DeleteGroups", func(cic *CloudInitClient) ([]client.HTTPEnvelope, []error, error) {
			return cic.DeleteGroups("tok", "compute", "storage")
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cic, srv := newTestCI(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			})
			defer srv.Close()

			_, errs, err := tc.call(cic)
			if err != nil {
				t.Fatalf("%s: control-flow error = %v", tc.name, err)
			}
			for i, e := range errs {
				if e == nil {
					t.Errorf("%s: per-item error[%d] = nil, want non-nil", tc.name, i)
				}
			}
		})
	}
}
