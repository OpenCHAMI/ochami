// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package smd

// smd_errors_test.go unit-tests the SMDClient wrapper methods' error arms:
// input rejected before a request is made, per-item HTTP failures for the
// iterative POST/PUT/PATCH/DELETE helpers, blank-ID/MAC edge cases, and
// single-envelope HTTP failures.

import (
	"errors"
	"net/http"
	"net/http/httptest"
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

	if _, err := sc.GetGroupMembers("", "tok"); err == nil {
		t.Fatal("expected an error for empty group label, got nil")
	}
	if requestMade {
		t.Error("a request was made despite the empty group label")
	}
}

// TestDeleteComponents_PerItemHTTPError verifies that a non-2XX response is
// reported in the per-item error slice (not the function-level error).
func TestDeleteComponents_PerItemHTTPError(t *testing.T) {
	sc, srv := newTestSMD(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	})
	defer srv.Close()

	_, errs, err := sc.DeleteComponents("tok", "x0c0s0b0n0")
	if err != nil {
		t.Fatalf("DeleteComponents func error: %v", err)
	}
	if len(errs) != 1 || errs[0] == nil {
		t.Fatalf("per-item errors = %v, want a single non-nil error", errs)
	}
	if !errors.Is(errs[0], client.UnsuccessfulHTTPError) {
		t.Errorf("errs[0] = %v, want it to wrap client.UnsuccessfulHTTPError", errs[0])
	}
}

// TestGetComponentsAll_UnsuccessfulHTTP verifies a non-2XX response from a
// single-shot getter surfaces as an UnsuccessfulHTTPError.
func TestGetComponentsAll_UnsuccessfulHTTP(t *testing.T) {
	sc, srv := newTestSMD(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	})
	defer srv.Close()

	_, err := sc.GetComponentsAll()
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

	_, errs, err := sc.PutComponents(ComponentSlice{Components: []Component{{ID: ""}}}, "tok")
	if err != nil {
		t.Fatalf("PutComponents: control-flow error = %v", err)
	}
	if len(errs) != 1 || errs[0] == nil {
		t.Errorf("per-item errors = %v, want a single non-nil error for blank ID", errs)
	}
}

// TestPutComponents_HTTPError verifies component update HTTP failures are
// retained per item.
func TestPutComponents_HTTPError(t *testing.T) {
	sc, srv := newTestSMD(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	defer srv.Close()

	_, errs, err := sc.PutComponents(ComponentSlice{Components: []Component{{ID: "x0c0s0b0n0"}}}, "tok")
	if err != nil {
		t.Fatalf("PutComponents: control-flow error = %v", err)
	}
	if len(errs) != 1 || errs[0] == nil {
		t.Errorf("per-item errors = %v, want a single non-nil error for HTTP failure", errs)
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

			_, errs, err := sc.PatchEthernetInterfaces(tc.eis, "tok")
			if err != nil {
				t.Fatalf("PatchEthernetInterfaces: control-flow error = %v", err)
			}
			if len(errs) != len(tc.wantErrIdx) {
				t.Fatalf("per-item errors length = %d, want %d", len(errs), len(tc.wantErrIdx))
			}
			for i, wantErr := range tc.wantErrIdx {
				if (errs[i] != nil) != wantErr {
					t.Errorf("per-item error[%d] = %v, wantErr %v", i, errs[i], wantErr)
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

	_, err := sc.PatchComponentsNID(ComponentSlice{Components: []Component{{ID: "x0c0s0b0n0", NID: 1}}}, "tok")
	if err == nil {
		t.Fatal("PatchComponentsNID: expected error on HTTP failure, got nil")
	}
}

// TestIterativeWrites_PreserveMixedResultAlignment verifies every iterative
// writer keeps its envelope and error slices aligned when one request
// succeeds and the next receives an unsuccessful HTTP response.
func TestIterativeWrites_PreserveMixedResultAlignment(t *testing.T) {
	cases := []struct {
		name       string
		wantMethod string
		call       func(sc *SMDClient) ([]client.HTTPEnvelope, []error, error)
	}{
		{"PostRedfishEndpoints", http.MethodPost, func(sc *SMDClient) ([]client.HTTPEnvelope, []error, error) {
			return sc.PostRedfishEndpoints(RedfishEndpointSlice{RedfishEndpoints: []csm.RedfishEndpoint{{ID: "x0c0s0b0"}, {ID: "x0c0s0b1"}}}, "tok")
		}},
		{"PostRedfishEndpointsV2", http.MethodPost, func(sc *SMDClient) ([]client.HTTPEnvelope, []error, error) {
			return sc.PostRedfishEndpointsV2(RedfishEndpointSliceV2{RedfishEndpoints: []RedfishEndpointV2{{RedfishEndpoint: csm.RedfishEndpoint{ID: "x0c0s0b0"}}, {RedfishEndpoint: csm.RedfishEndpoint{ID: "x0c0s0b1"}}}}, "tok")
		}},
		{"PostEthernetInterfaces", http.MethodPost, func(sc *SMDClient) ([]client.HTTPEnvelope, []error, error) {
			return sc.PostEthernetInterfaces([]EthernetInterface{{ComponentID: "x0c0s0b0n0", MACAddress: "de:ad:be:ef:00:00"}, {ComponentID: "x0c0s0b0n1", MACAddress: "de:ad:be:ef:00:01"}}, "tok")
		}},
		{"PostGroups", http.MethodPost, func(sc *SMDClient) ([]client.HTTPEnvelope, []error, error) {
			return sc.PostGroups([]Group{{Label: "compute"}, {Label: "storage"}}, "tok")
		}},
		{"PostGroupMembers", http.MethodPost, func(sc *SMDClient) ([]client.HTTPEnvelope, []error, error) {
			return sc.PostGroupMembers("tok", "compute", "x0c0s0b0n0", "x0c0s0b0n1")
		}},
		{"PutComponents", http.MethodPut, func(sc *SMDClient) ([]client.HTTPEnvelope, []error, error) {
			return sc.PutComponents(ComponentSlice{Components: []Component{{ID: "x0c0s0b0n0"}, {ID: "x0c0s0b0n1"}}}, "tok")
		}},
		{"PutRedfishEndpoints", http.MethodPut, func(sc *SMDClient) ([]client.HTTPEnvelope, []error, error) {
			return sc.PutRedfishEndpoints(RedfishEndpointSlice{RedfishEndpoints: []csm.RedfishEndpoint{{ID: "x0c0s0b0"}, {ID: "x0c0s0b1"}}}, "tok")
		}},
		{"PutRedfishEndpointsV2", http.MethodPut, func(sc *SMDClient) ([]client.HTTPEnvelope, []error, error) {
			return sc.PutRedfishEndpointsV2(RedfishEndpointSliceV2{RedfishEndpoints: []RedfishEndpointV2{{RedfishEndpoint: csm.RedfishEndpoint{ID: "x0c0s0b0"}}, {RedfishEndpoint: csm.RedfishEndpoint{ID: "x0c0s0b1"}}}}, "tok")
		}},
		{"PatchEthernetInterfaces", http.MethodPatch, func(sc *SMDClient) ([]client.HTTPEnvelope, []error, error) {
			return sc.PatchEthernetInterfaces([]EthernetInterface{{ID: "deadbeef0000"}, {ID: "deadbeef0001"}}, "tok")
		}},
		{"PatchGroups", http.MethodPatch, func(sc *SMDClient) ([]client.HTTPEnvelope, []error, error) {
			return sc.PatchGroups([]Group{{Label: "compute"}, {Label: "storage"}}, "tok")
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

			henvs, errs, err := tc.call(sc)
			if err != nil {
				t.Fatalf("%s: control-flow error = %v, want nil", tc.name, err)
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

// TestDeleteGroupMembers_Guards verifies DeleteGroupMembers rejects a blank
// group or an empty member list without making a request.
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
			if _, _, err := sc.DeleteGroupMembers("tok", tc.group, tc.members...); err == nil {
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

	if _, err := sc.PostComponents(ComponentSlice{Components: []Component{{ID: "x0c0s0b0n0"}}}, "tok"); err == nil {
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
			_, err := sc.GetStatus("")
			return err
		}},
		{name: "group members", call: func(sc *SMDClient) error {
			_, err := sc.GetGroupMembers("compute", "tok")
			return err
		}},
		{name: "put group members", call: func(sc *SMDClient) error {
			_, err := sc.PutGroupMembers("tok", "compute", "x0c0s0b0n0")
			return err
		}},
		{name: "ethernet interface", call: func(sc *SMDClient) error {
			_, err := sc.GetEthernetInterfaceByID("deadbeef", "tok", false)
			return err
		}},
		{name: "ethernet interface IPs", call: func(sc *SMDClient) error {
			_, err := sc.GetEthernetInterfaceByID("deadbeef", "tok", true)
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

// TestIterativeDeletes_PreserveMixedResultAlignment verifies that each
// iterative delete keeps the success and failure results in input order.
func TestIterativeDeletes_PreserveMixedResultAlignment(t *testing.T) {
	cases := []struct {
		name string
		call func(sc *SMDClient) ([]client.HTTPEnvelope, []error, error)
	}{
		{"components", func(sc *SMDClient) ([]client.HTTPEnvelope, []error, error) {
			return sc.DeleteComponents("tok", "x0c0s0b0n0", "x0c0s0b0n1")
		}},
		{"rfe", func(sc *SMDClient) ([]client.HTTPEnvelope, []error, error) {
			return sc.DeleteRedfishEndpoints("tok", "x0c0s0b0", "x0c0s0b1")
		}},
		{"iface", func(sc *SMDClient) ([]client.HTTPEnvelope, []error, error) {
			return sc.DeleteEthernetInterfaces("tok", "de:ad:be:ef:00:01", "de:ad:be:ef:00:02")
		}},
		{"compendpoints", func(sc *SMDClient) ([]client.HTTPEnvelope, []error, error) {
			return sc.DeleteComponentEndpoints("tok", "x0c0s0b0n0", "x0c0s0b0n1")
		}},
		{"groups", func(sc *SMDClient) ([]client.HTTPEnvelope, []error, error) {
			return sc.DeleteGroups("tok", "compute", "storage")
		}},
		{"groupmembers", func(sc *SMDClient) ([]client.HTTPEnvelope, []error, error) {
			return sc.DeleteGroupMembers("tok", "compute", "x0c0s0b0n0", "x0c0s0b0n1")
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

			henvs, errs, err := tc.call(sc)
			if err != nil {
				t.Fatalf("%s: control-flow error = %v, want nil", tc.name, err)
			}
			if len(errs) != 2 {
				t.Fatalf("%s: per-item errors length = %d, want 2", tc.name, len(errs))
			}
			if len(henvs) != 2 {
				t.Fatalf("%s: envelopes length = %d, want 2", tc.name, len(henvs))
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

// TestSMDClient_RejectsBlankRequiredFields verifies the iterative helpers
// report a per-item error (without a control-flow error) when a required
// field is left blank.
func TestSMDClient_RejectsBlankRequiredFields(t *testing.T) {
	c, srv := newTestSMD(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	defer srv.Close()

	_, errs, err := c.PutRedfishEndpoints(RedfishEndpointSlice{RedfishEndpoints: []csm.RedfishEndpoint{{}}}, "")
	if err != nil || len(errs) != 1 || errs[0] == nil {
		t.Fatalf("blank RFE errors = %v, %v", errs, err)
	}
	_, errs, err = c.PutRedfishEndpointsV2(RedfishEndpointSliceV2{RedfishEndpoints: []RedfishEndpointV2{{}}}, "")
	if err != nil || len(errs) != 1 || errs[0] == nil {
		t.Fatalf("blank RFE v2 errors = %v, %v", errs, err)
	}
	_, errs, err = c.PatchEthernetInterfaces([]EthernetInterface{{}}, "")
	if err != nil || len(errs) != 1 || errs[0] == nil {
		t.Fatalf("blank interface errors = %v, %v", errs, err)
	}
	_, errs, err = c.PatchGroups([]Group{{}}, "")
	if err != nil || len(errs) != 1 || errs[0] == nil {
		t.Fatalf("blank group errors = %v, %v", errs, err)
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

	if _, _, err := c.PostGroupMembers("tok", ""); err == nil {
		t.Error("PostGroupMembers with empty group = nil, want error")
	}
	if _, _, err := c.PostGroupMembers("tok", "compute"); err == nil {
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

	if _, err := c.GetStatus("bogus"); err == nil {
		t.Error("GetStatus(bogus) = nil, want error")
	}
	// "" and "all" are valid and route to the ready/values endpoints.
	if _, err := c.GetStatus(""); err != nil {
		t.Errorf("GetStatus(\"\") = %v, want nil", err)
	}
	if _, err := c.GetStatus("all"); err != nil {
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

	if _, err := c.GetComponentsXname("x0c0s0b0n0", "tok"); err == nil {
		t.Error("GetComponentsXname error arm = nil, want error")
	}
	if _, err := c.GetComponentsNid(1, "tok"); err == nil {
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

	if _, err := c.PutGroupMembers("tok", ""); err == nil {
		t.Error("PutGroupMembers with empty group = nil, want error")
	}
	if _, err := c.PutGroupMembers("tok", "compute"); err == nil {
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

	if _, err := c.DeleteComponentsAll("tok"); err == nil {
		t.Error("DeleteComponentsAll error arm = nil, want error")
	}
	if _, err := c.DeleteRedfishEndpointsAll("tok"); err == nil {
		t.Error("DeleteRedfishEndpointsAll error arm = nil, want error")
	}
	if _, err := c.DeleteEthernetInterfacesAll("tok"); err == nil {
		t.Error("DeleteEthernetInterfacesAll error arm = nil, want error")
	}
	if _, err := c.DeleteComponentEndpointsAll("tok"); err == nil {
		t.Error("DeleteComponentEndpointsAll error arm = nil, want error")
	}
}
