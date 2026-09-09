// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cloud_init

// cloud_init_edge_test.go exercises the per-item edge cases of the mutating
// CloudInitClient helpers: blank names/IDs, empty lists, and per-item HTTP
// failures.

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/openchami/cloud-init/pkg/cistore"

	"github.com/openchami/ochami/pkg/client"
)

// TestPutGroupsEdgeCases verifies validation and request failures remain aligned with group inputs.
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

			results := cic.PutGroups(context.Background(), tc.groups, "tok")
			if len(results) != 1 || (results[0].Err != nil) != tc.wantErr {
				t.Errorf("results = %v, wantErr %v", results, tc.wantErr)
			}
		})
	}
}

// TestPutInstanceInfoEdgeCases verifies validation and request failures for instance updates.
func TestPutInstanceInfoEdgeCases(t *testing.T) {
	t.Run("empty list", func(t *testing.T) {
		cic, srv := newTestCI(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		defer srv.Close()
		if _, err := cic.PutInstanceInfo(context.Background(), nil, "tok"); err == nil {
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

			results, err := cic.PutInstanceInfo(context.Background(), tc.infos, "tok")
			if err != nil {
				t.Fatalf("PutInstanceInfo: control-flow error = %v", err)
			}
			if len(results) != 1 || (results[0].Err != nil) != tc.wantErr {
				t.Errorf("results = %v, wantErr %v", results, tc.wantErr)
			}
		})
	}
}

// TestCloudInitIterativeMixedResults verifies mixed successes and failures preserve batch ordering.
func TestCloudInitIterativeMixedResults(t *testing.T) {
	cases := []struct {
		name       string
		wantMethod string
		call       func(cic *CloudInitClient) (client.BatchResult[client.HTTPEnvelope], error)
	}{
		{"PostGroups", http.MethodPost, func(cic *CloudInitClient) (client.BatchResult[client.HTTPEnvelope], error) {
			return cic.PostGroups(context.Background(), []cistore.GroupData{{Name: "compute"}, {Name: "storage"}}, "tok"), nil
		}},
		{"PutGroups", http.MethodPut, func(cic *CloudInitClient) (client.BatchResult[client.HTTPEnvelope], error) {
			return cic.PutGroups(context.Background(), []cistore.GroupData{{Name: "compute"}, {Name: "storage"}}, "tok"), nil
		}},
		{"PutInstanceInfo", http.MethodPut, func(cic *CloudInitClient) (client.BatchResult[client.HTTPEnvelope], error) {
			return cic.PutInstanceInfo(context.Background(), []cistore.OpenCHAMIInstanceInfo{{ID: "x0c0s0b0n0"}, {ID: "x0c0s0b0n1"}}, "tok")
		}},
		{"DeleteGroups", http.MethodDelete, func(cic *CloudInitClient) (client.BatchResult[client.HTTPEnvelope], error) {
			return cic.DeleteGroups(context.Background(), "tok", "compute", "storage"), nil
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

			results, err := tc.call(cic)
			if err != nil {
				t.Fatalf("%s: control-flow error = %v", tc.name, err)
			}
			if len(results) != 2 {
				t.Fatalf("%s: result length = %d, want 2", tc.name, len(results))
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

// TestCloudInitBatchCancellationPreservesAlignment verifies cancellation produces one error per input.
func TestCloudInitBatchCancellationPreservesAlignment(t *testing.T) {
	requests := 0
	cic, srv := newTestCI(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
	})
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	results := cic.PostGroups(ctx, []cistore.GroupData{{Name: "compute"}, {Name: "storage"}}, "tok")
	if len(results) != 2 {
		t.Fatalf("result length = %d, want 2", len(results))
	}
	for i, result := range results {
		if !errors.Is(result.Err, context.Canceled) {
			t.Errorf("result[%d].Err = %v, want context.Canceled", i, result.Err)
		}
	}
	if requests != 0 {
		t.Errorf("requests = %d, want 0", requests)
	}
}
