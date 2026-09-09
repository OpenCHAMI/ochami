// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package smd

// smd_edge_test.go exercises the per-item edge cases of the mutating SMDClient
// helpers: blank IDs, MAC-address adaptation, and per-item HTTP failures. These
// complement the happy-path assertions in smd_test.go and smd_more_test.go.

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/openchami/schemas/schemas/csm"

	"github.com/openchami/ochami/pkg/client"
)

// TestPutComponentsBlankID verifies component updates reject missing identifiers.
func TestPutComponentsBlankID(t *testing.T) {
	sc, srv := newTestSMD(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	results := sc.PutComponents(context.Background(), ComponentSlice{Components: []Component{{ID: ""}}}, "tok")
	if len(results) != 1 || results[0].Err == nil {
		t.Errorf("results = %v, want a single non-nil error for blank ID", results)
	}
}

// TestPutComponentsHTTPError verifies component update HTTP failures are retained per item.
func TestPutComponentsHTTPError(t *testing.T) {
	sc, srv := newTestSMD(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	defer srv.Close()

	results := sc.PutComponents(context.Background(), ComponentSlice{Components: []Component{{ID: "x0c0s0b0n0"}}}, "tok")
	if len(results) != 1 || results[0].Err == nil {
		t.Errorf("results = %v, want a single non-nil error for HTTP failure", results)
	}
}

// TestPatchEthernetInterfacesEdgeCases verifies interface patch validation and HTTP error handling.
func TestPatchEthernetInterfacesEdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		eis        []EthernetInterface
		status     int
		wantErrIdx []bool // true = expect non-nil per-item error
	}{
		{
			name:       "blank ID and blank MAC",
			eis:        []EthernetInterface{{}},
			status:     http.StatusOK,
			wantErrIdx: []bool{true},
		},
		{
			name:       "blank ID adapts from MAC",
			eis:        []EthernetInterface{{MACAddress: "de:ad:be:ef:00:00"}},
			status:     http.StatusOK,
			wantErrIdx: []bool{false},
		},
		{
			name:       "http error",
			eis:        []EthernetInterface{{ID: "deadbeef0000"}},
			status:     http.StatusInternalServerError,
			wantErrIdx: []bool{true},
		},
	}

	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			sc, srv := newTestSMD(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
			})
			defer srv.Close()

			results := sc.PatchEthernetInterfaces(context.Background(), tc.eis, "tok")
			if len(results) != len(tc.wantErrIdx) {
				t.Fatalf("results length = %d, want %d", len(results), len(tc.wantErrIdx))
			}
			for i, wantErr := range tc.wantErrIdx {
				if (results[i].Err != nil) != wantErr {
					t.Errorf("result[%d].Err = %v, wantErr %v", i, results[i].Err, wantErr)
				}
			}
		})
	}
}

// TestPatchComponentsNIDHTTPError verifies NID patch failures are propagated.
func TestPatchComponentsNIDHTTPError(t *testing.T) {
	sc, srv := newTestSMD(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	defer srv.Close()

	_, err := sc.PatchComponentsNID(context.Background(), ComponentSlice{Components: []Component{{ID: "x0c0s0b0n0", NID: 1}}}, "tok")
	if err == nil {
		t.Fatal("PatchComponentsNID: expected error on HTTP failure, got nil")
	}
}

// TestIterativeWritesPreserveMixedResultAlignment verifies every iterative
// writer keeps its envelope and error slices aligned when one request succeeds
// and the next receives an unsuccessful HTTP response.
func TestIterativeWritesPreserveMixedResultAlignment(t *testing.T) {
	cases := []struct {
		name       string
		wantMethod string
		call       func(sc *SMDClient) (client.BatchResult[client.HTTPEnvelope], error)
	}{
		{"PostRedfishEndpoints", http.MethodPost, func(sc *SMDClient) (client.BatchResult[client.HTTPEnvelope], error) {
			return sc.PostRedfishEndpoints(context.Background(), RedfishEndpointSlice{RedfishEndpoints: []csm.RedfishEndpoint{{ID: "x0c0s0b0"}, {ID: "x0c0s0b1"}}}, "tok"), nil
		}},
		{"PostRedfishEndpointsV2", http.MethodPost, func(sc *SMDClient) (client.BatchResult[client.HTTPEnvelope], error) {
			return sc.PostRedfishEndpointsV2(context.Background(), RedfishEndpointSliceV2{RedfishEndpoints: []RedfishEndpointV2{{RedfishEndpoint: csm.RedfishEndpoint{ID: "x0c0s0b0"}}, {RedfishEndpoint: csm.RedfishEndpoint{ID: "x0c0s0b1"}}}}, "tok"), nil
		}},
		{"PostEthernetInterfaces", http.MethodPost, func(sc *SMDClient) (client.BatchResult[client.HTTPEnvelope], error) {
			return sc.PostEthernetInterfaces(context.Background(), []EthernetInterface{{ComponentID: "x0c0s0b0n0", MACAddress: "de:ad:be:ef:00:00"}, {ComponentID: "x0c0s0b0n1", MACAddress: "de:ad:be:ef:00:01"}}, "tok"), nil
		}},
		{"PostGroups", http.MethodPost, func(sc *SMDClient) (client.BatchResult[client.HTTPEnvelope], error) {
			return sc.PostGroups(context.Background(), []Group{{Label: "compute"}, {Label: "storage"}}, "tok"), nil
		}},
		{"PostGroupMembers", http.MethodPost, func(sc *SMDClient) (client.BatchResult[client.HTTPEnvelope], error) {
			return sc.PostGroupMembers(context.Background(), "tok", "compute", "x0c0s0b0n0", "x0c0s0b0n1")
		}},
		{"PutComponents", http.MethodPut, func(sc *SMDClient) (client.BatchResult[client.HTTPEnvelope], error) {
			return sc.PutComponents(context.Background(), ComponentSlice{Components: []Component{{ID: "x0c0s0b0n0"}, {ID: "x0c0s0b0n1"}}}, "tok"), nil
		}},
		{"PutRedfishEndpoints", http.MethodPut, func(sc *SMDClient) (client.BatchResult[client.HTTPEnvelope], error) {
			return sc.PutRedfishEndpoints(context.Background(), RedfishEndpointSlice{RedfishEndpoints: []csm.RedfishEndpoint{{ID: "x0c0s0b0"}, {ID: "x0c0s0b1"}}}, "tok"), nil
		}},
		{"PutRedfishEndpointsV2", http.MethodPut, func(sc *SMDClient) (client.BatchResult[client.HTTPEnvelope], error) {
			return sc.PutRedfishEndpointsV2(context.Background(), RedfishEndpointSliceV2{RedfishEndpoints: []RedfishEndpointV2{{RedfishEndpoint: csm.RedfishEndpoint{ID: "x0c0s0b0"}}, {RedfishEndpoint: csm.RedfishEndpoint{ID: "x0c0s0b1"}}}}, "tok"), nil
		}},
		{"PatchEthernetInterfaces", http.MethodPatch, func(sc *SMDClient) (client.BatchResult[client.HTTPEnvelope], error) {
			return sc.PatchEthernetInterfaces(context.Background(), []EthernetInterface{{ID: "deadbeef0000"}, {ID: "deadbeef0001"}}, "tok"), nil
		}},
		{"PatchGroups", http.MethodPatch, func(sc *SMDClient) (client.BatchResult[client.HTTPEnvelope], error) {
			return sc.PatchGroups(context.Background(), []Group{{Label: "compute"}, {Label: "storage"}}, "tok"), nil
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			requests := 0
			sc, srv := newTestSMD(t, func(w http.ResponseWriter, r *http.Request) {
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

			results, err := tc.call(sc)
			if err != nil {
				t.Fatalf("%s: control-flow error = %v, want nil", tc.name, err)
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

// TestDeleteGroupMembersGuards verifies group and member identifiers are required for deletion.
func TestDeleteGroupMembersGuards(t *testing.T) {
	requests := 0
	sc, srv := newTestSMD(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	tests := []struct {
		name    string
		group   string
		members []string
	}{
		{name: "blank group", group: "", members: []string{"x0c0s0b0n0"}},
		{name: "no members", group: "compute"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := sc.DeleteGroupMembers(context.Background(), "tok", tc.group, tc.members...); err == nil {
				t.Fatal("DeleteGroupMembers() error = nil, want argument error")
			}
		})
	}
	if requests != 0 {
		t.Errorf("requests = %d, want no requests for rejected arguments", requests)
	}
}

// TestPostComponentsHTTPError verifies the single-envelope PostComponents
// helper returns an error on HTTP failure.
func TestPostComponentsHTTPError(t *testing.T) {
	sc, srv := newTestSMD(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	defer srv.Close()

	if _, err := sc.PostComponents(context.Background(), ComponentSlice{Components: []Component{{ID: "x0c0s0b0n0"}}}, "tok"); err == nil {
		t.Fatal("PostComponents: expected error on HTTP failure, got nil")
	}
}

// TestSingleEnvelopeHTTPErrorWrappers verifies representative SMD helpers
// preserve the unsuccessful-HTTP sentinel while adding operation context.
func TestSingleEnvelopeHTTPErrorWrappers(t *testing.T) {
	cases := []struct {
		name string
		call func(*SMDClient) error
	}{
		{name: "status", call: func(sc *SMDClient) error {
			_, err := sc.GetStatus(context.Background(), "")
			return err
		}},
		{name: "group members", call: func(sc *SMDClient) error {
			_, err := sc.GetGroupMembers(context.Background(), "compute", "tok")
			return err
		}},
		{name: "put group members", call: func(sc *SMDClient) error {
			_, err := sc.PutGroupMembers(context.Background(), "tok", "compute", "x0c0s0b0n0")
			return err
		}},
		{name: "ethernet interface", call: func(sc *SMDClient) error {
			_, err := sc.GetEthernetInterfaceByID(context.Background(), "deadbeef", "tok", false)
			return err
		}},
		{name: "ethernet interface IPs", call: func(sc *SMDClient) error {
			_, err := sc.GetEthernetInterfaceByID(context.Background(), "deadbeef", "tok", true)
			return err
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sc, srv := newTestSMD(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			})
			defer srv.Close()

			err := tc.call(sc)
			if err == nil {
				t.Fatal("call returned nil error on HTTP failure")
			}
			if !errors.Is(err, client.UnsuccessfulHTTPError) {
				t.Errorf("error = %v, want wrapped UnsuccessfulHTTPError", err)
			}
		})
	}
}

// TestSMDBatchCancellationPreservesAlignment verifies cancellation produces one error per input.
func TestSMDBatchCancellationPreservesAlignment(t *testing.T) {
	requests := 0
	sc, srv := newTestSMD(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
	})
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	results := sc.PostGroups(ctx, []Group{{Label: "compute"}, {Label: "storage"}}, "tok")
	if len(results) != 2 {
		t.Fatalf("results length = %d, want 2", len(results))
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
