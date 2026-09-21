// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
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

// TestIterativeDeletes_PerItemHTTPError verifies that each iterative delete
// records a per-item error (and an envelope) when the server returns a
// non-success status, while still returning nil for the control-flow error.
func TestIterativeDeletes_PerItemHTTPError(t *testing.T) {
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
			sc, srv := newTestSMD(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte("boom"))
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
			for i, e := range errs {
				if e == nil {
					t.Errorf("%s: per-item error[%d] = nil, want non-nil", tc.name, i)
				}
			}
		})
	}
}

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

// TestIterativePostsPuts_PerItemHTTPError verifies the iterative POST/PUT
// helpers record a per-item error when the server returns a failure status.
func TestIterativePostsPuts_PerItemHTTPError(t *testing.T) {
	cases := []struct {
		name string
		call func(sc *SMDClient) ([]client.HTTPEnvelope, []error, error)
	}{
		{"PostRedfishEndpoints", func(sc *SMDClient) ([]client.HTTPEnvelope, []error, error) {
			return sc.PostRedfishEndpoints(RedfishEndpointSlice{RedfishEndpoints: []csm.RedfishEndpoint{{ID: "x0c0s0b0"}}}, "tok")
		}},
		{"PostRedfishEndpointsV2", func(sc *SMDClient) ([]client.HTTPEnvelope, []error, error) {
			return sc.PostRedfishEndpointsV2(RedfishEndpointSliceV2{RedfishEndpoints: []RedfishEndpointV2{{RedfishEndpoint: csm.RedfishEndpoint{ID: "x0c0s0b0"}}}}, "tok")
		}},
		{"PostEthernetInterfaces", func(sc *SMDClient) ([]client.HTTPEnvelope, []error, error) {
			return sc.PostEthernetInterfaces([]EthernetInterface{{ComponentID: "x0c0s0b0n0", MACAddress: "de:ad:be:ef:00:00"}}, "tok")
		}},
		{"PostGroups", func(sc *SMDClient) ([]client.HTTPEnvelope, []error, error) {
			return sc.PostGroups([]Group{{Label: "compute"}}, "tok")
		}},
		{"PostGroupMembers", func(sc *SMDClient) ([]client.HTTPEnvelope, []error, error) {
			return sc.PostGroupMembers("tok", "compute", "x0c0s0b0n0")
		}},
		{"PutRedfishEndpoints", func(sc *SMDClient) ([]client.HTTPEnvelope, []error, error) {
			return sc.PutRedfishEndpoints(RedfishEndpointSlice{RedfishEndpoints: []csm.RedfishEndpoint{{ID: "x0c0s0b0"}}}, "tok")
		}},
		{"PutRedfishEndpointsV2", func(sc *SMDClient) ([]client.HTTPEnvelope, []error, error) {
			return sc.PutRedfishEndpointsV2(RedfishEndpointSliceV2{RedfishEndpoints: []RedfishEndpointV2{{RedfishEndpoint: csm.RedfishEndpoint{ID: "x0c0s0b0"}}}}, "tok")
		}},
		{"PatchGroups", func(sc *SMDClient) ([]client.HTTPEnvelope, []error, error) {
			return sc.PatchGroups([]Group{{Label: "compute"}}, "tok")
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sc, srv := newTestSMD(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			})
			defer srv.Close()

			_, errs, err := tc.call(sc)
			if err != nil {
				t.Fatalf("%s: control-flow error = %v, want nil", tc.name, err)
			}
			if len(errs) != 1 || errs[0] == nil {
				t.Errorf("%s: per-item errors = %v, want a single non-nil error", tc.name, errs)
			}
		})
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
