// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package smd

// smd_errors_test.go unit-tests the SMDClient wrapper methods' error arms:
// input rejected before a request is made, blank-ID/MAC edge cases and
// per-item HTTP failures for the iterative POST/PUT/PATCH/DELETE helpers,
// single-envelope HTTP failures, and batch semantics under cancellation
// (order, cardinality, and alignment).

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/openchami/schemas/schemas/csm"

	"github.com/openchami/ochami/pkg/client"
)

// TestGetGroupMembers_EmptyLabel verifies GetGroupMembers rejects an empty label
// without making a request.
func TestGetGroupMembers_EmptyLabel(t *testing.T) {
	requestMade := false
	sc, srv := newTestSMD(t, func(w http.ResponseWriter, r *http.Request) { requestMade = true })
	defer srv.Close()

	if _, err := sc.GetGroupMembers(context.Background(), "", "tok"); err == nil {
		t.Fatal("expected an error for empty group label, got nil")
	}
	if requestMade {
		t.Error("a request was made despite the empty group label")
	}
}

// TestGetComponentsAll_UnsuccessfulHTTP verifies a non-2XX response from a
// single-shot getter surfaces as an UnsuccessfulHTTPError.
func TestGetComponentsAll_UnsuccessfulHTTP(t *testing.T) {
	sc, srv := newTestSMD(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	})
	defer srv.Close()

	_, err := sc.GetComponentsAll(context.Background())
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !errors.Is(err, client.UnsuccessfulHTTPError) {
		t.Errorf("error = %v, want it to wrap client.UnsuccessfulHTTPError", err)
	}
}

// TestPutComponents_BlankID verifies component updates reject missing
// identifiers.
func TestPutComponents_BlankID(t *testing.T) {
	sc, srv := newTestSMD(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	results := sc.PutComponents(context.Background(), ComponentSlice{Components: []Component{{ID: ""}}}, "tok")
	if len(results) != 1 || !errors.Is(results[0].Err, client.InvalidArgumentError) {
		t.Errorf("results = %v, want a single client.InvalidArgumentError for blank ID", results)
	}
}

// TestPutComponents_HTTPError verifies component update HTTP failures are
// retained per item.
func TestPutComponents_HTTPError(t *testing.T) {
	sc, srv := newTestSMD(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	defer srv.Close()

	results := sc.PutComponents(context.Background(), ComponentSlice{Components: []Component{{ID: "x0c0s0b0n0"}}}, "tok")
	if len(results) != 1 || results[0].Err == nil {
		t.Errorf("results = %v, want a single non-nil error for HTTP failure", results)
	}
}

// TestPatchEthernetInterfaces_EdgeCases verifies that PatchEthernetInterfaces
// derives a missing ID from the MAC address and reports an interface with
// neither, or an unsuccessful response, as that item's error.
func TestPatchEthernetInterfaces_EdgeCases(t *testing.T) {
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

// TestPatchComponentsNID_HTTPError verifies NID patch failures are propagated.
func TestPatchComponentsNID_HTTPError(t *testing.T) {
	sc, srv := newTestSMD(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	defer srv.Close()

	_, err := sc.PatchComponentsNID(context.Background(), ComponentSlice{Components: []Component{{ID: "x0c0s0b0n0", NID: 1}}}, "tok")
	if err == nil {
		t.Fatal("PatchComponentsNID: expected error on HTTP failure, got nil")
	}
}

// TestIterativeWrites_PreserveMixedResultAlignment verifies every iterative
// writer returns one result per input item, in order, when one request
// succeeds and the next receives an unsuccessful HTTP response.
func TestIterativeWrites_PreserveMixedResultAlignment(t *testing.T) {
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

// TestDeleteGroupMembers_Guards verifies group and member identifiers are required for deletion.
func TestDeleteGroupMembers_Guards(t *testing.T) {
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

// TestPostComponents_HTTPError verifies the single-envelope PostComponents
// helper returns an error on HTTP failure.
func TestPostComponents_HTTPError(t *testing.T) {
	sc, srv := newTestSMD(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	defer srv.Close()

	if _, err := sc.PostComponents(context.Background(), ComponentSlice{Components: []Component{{ID: "x0c0s0b0n0"}}}, "tok"); err == nil {
		t.Fatal("PostComponents: expected error on HTTP failure, got nil")
	}
}

// TestSingleEnvelopeWrappers_HTTPError verifies representative SMD helpers
// preserve the unsuccessful-HTTP sentinel while adding operation context.
func TestSingleEnvelopeWrappers_HTTPError(t *testing.T) {
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

// TestSMDBatch_CancellationPreservesAlignment verifies that batch methods
// called with an already-canceled context send no requests and return one
// context.Canceled result per input.
func TestSMDBatch_CancellationPreservesAlignment(t *testing.T) {
	requests := 0
	sc, srv := newTestSMD(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
	})
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	tests := []struct {
		name string
		call func() client.BatchResult[client.HTTPEnvelope]
	}{
		{"PostGroups", func() client.BatchResult[client.HTTPEnvelope] {
			return sc.PostGroups(ctx, []Group{{Label: "compute"}, {Label: "storage"}}, "tok")
		}},
		{"GetComponentEndpoints", func() client.BatchResult[client.HTTPEnvelope] {
			return sc.GetComponentEndpoints(ctx, "", "x0c0s0b0n0", "x0c0s0b0n1")
		}},
		{"DeleteComponents", func() client.BatchResult[client.HTTPEnvelope] {
			return sc.DeleteComponents(ctx, "", "x0c0s0b0n0", "x0c0s0b0n1")
		}},
		{"PutComponents", func() client.BatchResult[client.HTTPEnvelope] {
			return sc.PutComponents(ctx, ComponentSlice{Components: []Component{{ID: "x0c0s0b0n0", Type: "Node"}, {ID: "x0c0s0b0n1", Type: "Node"}}}, "")
		}},
		{"PostRedfishEndpoints", func() client.BatchResult[client.HTTPEnvelope] {
			return sc.PostRedfishEndpoints(ctx, RedfishEndpointSlice{RedfishEndpoints: []csm.RedfishEndpoint{{ID: "rfe0"}, {ID: "rfe1"}}}, "")
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			results := tc.call()
			if len(results) != 2 {
				t.Fatalf("results length = %d, want 2", len(results))
			}
			for i, result := range results {
				if !errors.Is(result.Err, context.Canceled) {
					t.Errorf("result[%d].Err = %v, want context.Canceled", i, result.Err)
				}
			}
		})
	}
	if requests != 0 {
		t.Errorf("requests = %d, want 0", requests)
	}
}

type smdRoundTripFunc func(*http.Request) (*http.Response, error)

func (f smdRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// TestSMDBatch_CancellationBetweenItems verifies that cancellation after the first
// item completes stops subsequent operations and marks remaining results with
// context.Canceled. This test uses the same pattern as client.RunBatch's own
// tests in pkg/client/batch_test.go: cancel from within the first operation's
// callback.
func TestSMDBatch_CancellationBetweenItems(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	requests := 0
	c, err := NewClient("https://example.com")
	if err != nil {
		t.Fatal(err)
	}
	c.Client = &http.Client{Transport: smdRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		requests++
		cancel()
		return &http.Response{
			Status: "200 OK", StatusCode: http.StatusOK, Header: make(http.Header),
			Body: io.NopCloser(strings.NewReader(`{}`)), Request: r,
		}, nil
	})}

	results := c.PutComponents(ctx, ComponentSlice{Components: []Component{{ID: "x0"}, {ID: "x1"}, {ID: "x2"}}}, "tok")
	if len(results) != 3 || results[0].Err != nil || !errors.Is(results[1].Err, context.Canceled) || !errors.Is(results[2].Err, context.Canceled) {
		t.Fatalf("results = %#v, want success followed by aligned cancellation errors", results)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
}

// TestSMDBatch_AllFailure verifies that all items in a batch can fail and results
// maintain exact cardinality and order.
func TestSMDBatch_AllFailure(t *testing.T) {
	var requestPaths []string
	sc, srv := newTestSMD(t, func(w http.ResponseWriter, r *http.Request) {
		requestPaths = append(requestPaths, r.URL.Path)
		http.Error(w, "not found", http.StatusNotFound)
	})
	defer srv.Close()

	xnames := []string{"x0c0s0b0n0", "x0c0s0b0n1", "x0c0s0b0n2"}
	results := sc.DeleteComponents(context.Background(), "", xnames...)

	if len(results) != len(xnames) {
		t.Fatalf("results length = %d, want %d", len(results), len(xnames))
	}

	// All should fail
	for i, result := range results {
		if result.Err == nil {
			t.Errorf("result[%d].Err = nil, want non-nil", i)
		}
		if !errors.Is(result.Err, client.UnsuccessfulHTTPError) {
			t.Errorf("result[%d].Err = %v, want UnsuccessfulHTTPError", i, result.Err)
		}
	}

	// Verify exact request order
	wantPaths := []string{
		"/State/Components/x0c0s0b0n0",
		"/State/Components/x0c0s0b0n1",
		"/State/Components/x0c0s0b0n2",
	}
	if len(requestPaths) != len(wantPaths) {
		t.Fatalf("request paths length = %d, want %d", len(requestPaths), len(wantPaths))
	}
	for i, want := range wantPaths {
		if requestPaths[i] != want {
			t.Errorf("requestPaths[%d] = %q, want %q", i, requestPaths[i], want)
		}
	}
}

// TestSMDBatch_ExactOrderAndCardinality verifies that results maintain exact
// input order and cardinality for mixed success/failure outcomes.
func TestSMDBatch_ExactOrderAndCardinality(t *testing.T) {
	var requestOrder []int
	sc, srv := newTestSMD(t, func(w http.ResponseWriter, r *http.Request) {
		// Extract index from path: /State/Components/x0c0s0b0n{N}
		path := r.URL.Path
		idx := path[len(path)-1] - '0' // Simple extraction for this test
		requestOrder = append(requestOrder, int(idx))
		// Fail on items 1 and 3 (0-indexed)
		if idx == 1 || idx == 3 {
			http.Error(w, "error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	// Use 4 items: 0, 1, 2, 3
	xnames := []string{"x0c0s0b0n0", "x0c0s0b0n1", "x0c0s0b0n2", "x0c0s0b0n3"}
	results := sc.GetComponentEndpoints(context.Background(), "", xnames...)

	if len(results) != len(xnames) {
		t.Fatalf("results length = %d, want %d", len(results), len(xnames))
	}

	// Verify exact order: 0 succeeds, 1 fails, 2 succeeds, 3 fails
	if results[0].Err != nil {
		t.Errorf("result[0] should succeed")
	}
	if results[1].Err == nil {
		t.Errorf("result[1] should fail")
	}
	if results[2].Err != nil {
		t.Errorf("result[2] should succeed")
	}
	if results[3].Err == nil {
		t.Errorf("result[3] should fail")
	}

	// Verify request order matches input order
	wantOrder := []int{0, 1, 2, 3}
	if len(requestOrder) != len(wantOrder) {
		t.Fatalf("request order length = %d, want %d", len(requestOrder), len(wantOrder))
	}
	for i, want := range wantOrder {
		if requestOrder[i] != want {
			t.Errorf("requestOrder[%d] = %d, want %d", i, requestOrder[i], want)
		}
	}
}

// TestPutRedfishEndpoints_BlankID verifies PUT batch rejects blank RFE IDs.
func TestPutRedfishEndpoints_BlankID(t *testing.T) {
	requestMade := false
	sc, srv := newTestSMD(t, func(w http.ResponseWriter, r *http.Request) {
		requestMade = true
	})
	defer srv.Close()

	rfes := []csm.RedfishEndpoint{
		{ID: "rfe0"},
		{ID: ""}, // Blank ID
		{ID: "rfe2"},
	}
	results := sc.PutRedfishEndpoints(context.Background(), RedfishEndpointSlice{RedfishEndpoints: rfes}, "")

	if len(results) != len(rfes) {
		t.Fatalf("results length = %d, want %d", len(results), len(rfes))
	}

	// A blank ID fails local validation before any request is issued for that
	// item; the executor is not short-circuited by a single item's failure, so
	// items after it are still attempted.
	if results[0].Err != nil {
		t.Errorf("result[0].Err = %v, want nil", results[0].Err)
	}
	if !errors.Is(results[1].Err, client.InvalidArgumentError) {
		t.Errorf("result[1].Err = %v, want client.InvalidArgumentError for blank ID", results[1].Err)
	}
	if results[2].Err != nil {
		t.Errorf("result[2].Err = %v, want nil (executor continues past item-local failures)", results[2].Err)
	}

	// The first and third items should have reached the server; the second
	// fails validation locally and never issues a request.
	if !requestMade {
		t.Error("no requests were made")
	}
}

// TestIterativeDeletes_PreserveMixedResultAlignment verifies that each
// iterative delete keeps the success and failure results in input order.
func TestIterativeDeletes_PreserveMixedResultAlignment(t *testing.T) {
	cases := []struct {
		name string
		call func(sc *SMDClient) (client.BatchResult[client.HTTPEnvelope], error)
	}{
		{"components", func(sc *SMDClient) (client.BatchResult[client.HTTPEnvelope], error) {
			return sc.DeleteComponents(context.Background(), "tok", "x0c0s0b0n0", "x0c0s0b0n1"), nil
		}},
		{"rfe", func(sc *SMDClient) (client.BatchResult[client.HTTPEnvelope], error) {
			return sc.DeleteRedfishEndpoints(context.Background(), "tok", "x0c0s0b0", "x0c0s0b1"), nil
		}},
		{"iface", func(sc *SMDClient) (client.BatchResult[client.HTTPEnvelope], error) {
			return sc.DeleteEthernetInterfaces(context.Background(), "tok", "de:ad:be:ef:00:01", "de:ad:be:ef:00:02"), nil
		}},
		{"compendpoints", func(sc *SMDClient) (client.BatchResult[client.HTTPEnvelope], error) {
			return sc.DeleteComponentEndpoints(context.Background(), "tok", "x0c0s0b0n0", "x0c0s0b0n1"), nil
		}},
		{"groups", func(sc *SMDClient) (client.BatchResult[client.HTTPEnvelope], error) {
			return sc.DeleteGroups(context.Background(), "tok", "compute", "storage"), nil
		}},
		{"groupmembers", func(sc *SMDClient) (client.BatchResult[client.HTTPEnvelope], error) {
			return sc.DeleteGroupMembers(context.Background(), "tok", "compute", "x0c0s0b0n0", "x0c0s0b0n1")
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			requests := 0
			sc, srv := newTestSMD(t, func(w http.ResponseWriter, r *http.Request) {
				requests++
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
				t.Fatalf("%s: results length = %d, want 2", tc.name, len(results))
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

// TestSMDClient_RejectsBlankRequiredFields verifies the iterative helpers
// report a per-item error in their BatchResult when a required field is left
// blank.
func TestSMDClient_RejectsBlankRequiredFields(t *testing.T) {
	c, srv := newTestSMD(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	defer srv.Close()

	results := c.PutRedfishEndpoints(context.Background(), RedfishEndpointSlice{RedfishEndpoints: []csm.RedfishEndpoint{{}}}, "")
	if len(results) != 1 || results[0].Err == nil {
		t.Fatalf("blank RFE results = %v", results)
	}
	results = c.PutRedfishEndpointsV2(context.Background(), RedfishEndpointSliceV2{RedfishEndpoints: []RedfishEndpointV2{{}}}, "")
	if len(results) != 1 || results[0].Err == nil {
		t.Fatalf("blank RFE v2 results = %v", results)
	}
	results = c.PatchEthernetInterfaces(context.Background(), []EthernetInterface{{}}, "")
	if len(results) != 1 || results[0].Err == nil {
		t.Fatalf("blank interface results = %v", results)
	}
	results = c.PatchGroups(context.Background(), []Group{{}}, "")
	if len(results) != 1 || results[0].Err == nil {
		t.Fatalf("blank group results = %v", results)
	}
}

// newTestClient builds an SMDClient pointed at srv.
func newTestClient(t *testing.T, srv *httptest.Server) *SMDClient {
	t.Helper()
	c, err := NewClient(srv.URL, client.WithInsecure(true))
	if err != nil {
		t.Fatalf("failed to create SMD client: %v", err)
	}
	return c
}

// TestPostGroupMembers_Guards verifies the empty-group and empty-members guard
// clauses.
func TestPostGroupMembers_Guards(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	c := newTestClient(t, srv)

	if _, err := c.PostGroupMembers(context.Background(), "tok", ""); err == nil {
		t.Error("PostGroupMembers with empty group = nil, want error")
	}
	if _, err := c.PostGroupMembers(context.Background(), "tok", "compute"); err == nil {
		t.Error("PostGroupMembers with no members = nil, want error")
	}
}

// TestGetStatus_UnknownComponent verifies that GetStatus rejects an unknown
// component and accepts "" and "all".
func TestGetStatus_UnknownComponent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	c := newTestClient(t, srv)

	if _, err := c.GetStatus(context.Background(), "bogus"); err == nil {
		t.Error("GetStatus(bogus) = nil, want error")
	}
	// "" and "all" are valid and route to the ready/values endpoints.
	if _, err := c.GetStatus(context.Background(), ""); err != nil {
		t.Errorf("GetStatus(\"\") = %v, want nil", err)
	}
	if _, err := c.GetStatus(context.Background(), "all"); err != nil {
		t.Errorf("GetStatus(all) = %v, want nil", err)
	}
}

// TestGetComponentsXnameNid_HTTPError verifies that GetComponentsXname and
// GetComponentsNid return an error for an unsuccessful response.
func TestGetComponentsXnameNid_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()
	c := newTestClient(t, srv)

	if _, err := c.GetComponentsXname(context.Background(), "x0c0s0b0n0", "tok"); err == nil {
		t.Error("GetComponentsXname error arm = nil, want error")
	}
	if _, err := c.GetComponentsNid(context.Background(), 1, "tok"); err == nil {
		t.Error("GetComponentsNid error arm = nil, want error")
	}
}

// TestPutGroupMembers_Guards verifies the empty-group and empty-members guards.
func TestPutGroupMembers_Guards(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	c := newTestClient(t, srv)

	if _, err := c.PutGroupMembers(context.Background(), "tok", ""); err == nil {
		t.Error("PutGroupMembers with empty group = nil, want error")
	}
	if _, err := c.PutGroupMembers(context.Background(), "tok", "compute"); err == nil {
		t.Error("PutGroupMembers with no members = nil, want error")
	}
}

// TestDeleteAllHelpers_HTTPError verifies that each delete-all helper returns
// an error for an unsuccessful response.
func TestDeleteAllHelpers_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()
	c := newTestClient(t, srv)

	if _, err := c.DeleteComponentsAll(context.Background(), "tok"); err == nil {
		t.Error("DeleteComponentsAll error arm = nil, want error")
	}
	if _, err := c.DeleteRedfishEndpointsAll(context.Background(), "tok"); err == nil {
		t.Error("DeleteRedfishEndpointsAll error arm = nil, want error")
	}
	if _, err := c.DeleteEthernetInterfacesAll(context.Background(), "tok"); err == nil {
		t.Error("DeleteEthernetInterfacesAll error arm = nil, want error")
	}
	if _, err := c.DeleteComponentEndpointsAll(context.Background(), "tok"); err == nil {
		t.Error("DeleteComponentEndpointsAll error arm = nil, want error")
	}
}
