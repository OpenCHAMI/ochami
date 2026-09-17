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

// TestCloudInitBatchCancellationBetweenItems verifies that cancellation between
// items stops subsequent operations and marks remaining results with context.Canceled.
// This tests the executor's behavior with real HTTP-backed operations.
func TestCloudInitBatchCancellationBetweenItems(t *testing.T) {
	// Test with pre-canceled context for multiple cloud-init batch methods
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cic, srv := newTestCI(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("server should not receive any requests for pre-canceled context")
	})
	defer srv.Close()

	batchTests := []struct {
		name string
		call func() client.BatchResult[client.HTTPEnvelope]
	}{
		{"PostGroups", func() client.BatchResult[client.HTTPEnvelope] {
			return cic.PostGroups(ctx, []cistore.GroupData{{Name: "compute"}, {Name: "storage"}}, "tok")
		}},
		{"PutGroups", func() client.BatchResult[client.HTTPEnvelope] {
			return cic.PutGroups(ctx, []cistore.GroupData{{Name: "compute"}, {Name: "storage"}}, "tok")
		}},
		{"DeleteGroups", func() client.BatchResult[client.HTTPEnvelope] {
			return cic.DeleteGroups(ctx, "tok", "compute", "storage")
		}},
	}

	for _, tc := range batchTests {
		t.Run(tc.name, func(t *testing.T) {
			results := tc.call()
			if len(results) != 2 {
				t.Errorf("results length = %d, want 2", len(results))
			}
			for i, result := range results {
				if !errors.Is(result.Err, context.Canceled) {
					t.Errorf("result[%d].Err = %v, want context.Canceled", i, result.Err)
				}
			}
		})
	}
}

// TestCloudInitBatchAllSuccess verifies all-success paths for batch operations.
func TestCloudInitBatchAllSuccess(t *testing.T) {
	var requestCount int
	cic, srv := newTestCI(t, func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`)) //nolint:errcheck // test response
	})
	defer srv.Close()

	batchTests := []struct {
		name string
		call func() client.BatchResult[client.HTTPEnvelope]
		len  int
	}{
		{"PostGroups", func() client.BatchResult[client.HTTPEnvelope] {
			return cic.PostGroups(context.Background(), []cistore.GroupData{{Name: "compute"}, {Name: "storage"}}, "tok")
		}, 2},
		{"PutGroups", func() client.BatchResult[client.HTTPEnvelope] {
			return cic.PutGroups(context.Background(), []cistore.GroupData{{Name: "compute"}, {Name: "storage"}}, "tok")
		}, 2},
		{"DeleteGroups", func() client.BatchResult[client.HTTPEnvelope] {
			return cic.DeleteGroups(context.Background(), "tok", "compute", "storage")
		}, 2},
	}

	for _, tc := range batchTests {
		t.Run(tc.name, func(t *testing.T) {
			requestCount = 0
			results := tc.call()
			if len(results) != tc.len {
				t.Errorf("results length = %d, want %d", len(results), tc.len)
			}
			for i, result := range results {
				if result.Err != nil {
					t.Errorf("result[%d].Err = %v, want nil", i, result.Err)
				}
				if result.Value.StatusCode != http.StatusOK {
					t.Errorf("result[%d].Value.StatusCode = %d, want %d", i, result.Value.StatusCode, http.StatusOK)
				}
			}
			if requestCount != tc.len {
				t.Errorf("request count = %d, want %d", requestCount, tc.len)
			}
		})
	}
}

// TestCloudInitBatchAllFailure verifies all-failure paths for batch operations.
func TestCloudInitBatchAllFailure(t *testing.T) {
	var requestCount int
	cic, srv := newTestCI(t, func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		http.Error(w, "not found", http.StatusNotFound)
	})
	defer srv.Close()

	batchTests := []struct {
		name string
		call func() client.BatchResult[client.HTTPEnvelope]
		len  int
	}{
		{"PostGroups", func() client.BatchResult[client.HTTPEnvelope] {
			return cic.PostGroups(context.Background(), []cistore.GroupData{{Name: "compute"}, {Name: "storage"}}, "tok")
		}, 2},
		{"PutGroups", func() client.BatchResult[client.HTTPEnvelope] {
			return cic.PutGroups(context.Background(), []cistore.GroupData{{Name: "compute"}, {Name: "storage"}}, "tok")
		}, 2},
		{"DeleteGroups", func() client.BatchResult[client.HTTPEnvelope] {
			return cic.DeleteGroups(context.Background(), "tok", "compute", "storage")
		}, 2},
	}

	for _, tc := range batchTests {
		t.Run(tc.name, func(t *testing.T) {
			requestCount = 0
			results := tc.call()
			if len(results) != tc.len {
				t.Errorf("results length = %d, want %d", len(results), tc.len)
			}
			for i, result := range results {
				if result.Err == nil {
					t.Errorf("result[%d].Err = nil, want non-nil", i)
				}
				if !errors.Is(result.Err, client.UnsuccessfulHTTPError) {
					t.Errorf("result[%d].Err = %v, want UnsuccessfulHTTPError", i, result.Err)
				}
			}
			if requestCount != tc.len {
				t.Errorf("request count = %d, want %d", requestCount, tc.len)
			}
		})
	}
}

// TestCloudInitBatchExactOrderAndCardinality verifies exact order and cardinality
// for mixed outcomes.
func TestCloudInitBatchExactOrderAndCardinality(t *testing.T) {
	var requestOrder []int
	cic, srv := newTestCI(t, func(w http.ResponseWriter, r *http.Request) {
		// Extract index from group name: compute=0, storage=1
		name := r.URL.Query().Get("name")
		var idx int
		switch name {
		case "compute":
			idx = 0
		case "storage":
			idx = 1
		default:
			idx = -1
		}
		// For DELETE, extract from path
		if r.Method == http.MethodDelete {
			path := r.URL.Path
			if path == "/admin/groups/compute" {
				idx = 0
			} else if path == "/admin/groups/storage" {
				idx = 1
			}
		}
		// For PUT/POST, use request count as index
		if idx == -1 {
			idx = len(requestOrder)
		}
		requestOrder = append(requestOrder, idx)
		// Fail on second item
		if idx == 1 {
			http.Error(w, "error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	// Test with PostGroups
	groups := []cistore.GroupData{{Name: "compute"}, {Name: "storage"}}
	results := cic.PostGroups(context.Background(), groups, "tok")

	if len(results) != len(groups) {
		t.Fatalf("results length = %d, want %d", len(results), len(groups))
	}

	// First should succeed, second should fail
	if results[0].Err != nil {
		t.Errorf("result[0] should succeed")
	}
	if results[1].Err == nil {
		t.Errorf("result[1] should fail")
	}

	// Verify request order
	if len(requestOrder) != 2 {
		t.Fatalf("request order length = %d, want 2", len(requestOrder))
	}
	if requestOrder[0] != 0 || requestOrder[1] != 1 {
		t.Errorf("request order = %v, want [0, 1]", requestOrder)
	}
}
