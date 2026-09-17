// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package smd

// smd_edge_test.go exercises the per-item edge cases of the mutating SMDClient
// helpers: blank IDs, MAC-address adaptation, and per-item HTTP failures. These
// complement the happy-path assertions in smd_test.go and smd_more_test.go.

import (
	"errors"
	"net/http"
	"testing"

	"github.com/openchami/schemas/schemas/csm"

	"github.com/openchami/ochami/pkg/client"
)

func TestPutComponentsBlankID(t *testing.T) {
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

func TestPutComponentsHTTPError(t *testing.T) {
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

func TestPatchComponentsNIDHTTPError(t *testing.T) {
	sc, srv := newTestSMD(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	defer srv.Close()

	_, err := sc.PatchComponentsNID(ComponentSlice{Components: []Component{{ID: "x0c0s0b0n0", NID: 1}}}, "tok")
	if err == nil {
		t.Fatal("PatchComponentsNID: expected error on HTTP failure, got nil")
	}
}

// TestIterativePostsPutsPerItemHTTPError verifies the iterative POST/PUT
// helpers record a per-item error when the server returns a failure status.
func TestIterativePostsPutsPerItemHTTPError(t *testing.T) {
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

// TestPostComponentsHTTPError verifies the single-envelope PostComponents
// helper returns an error on HTTP failure.
func TestPostComponentsHTTPError(t *testing.T) {
	sc, srv := newTestSMD(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	defer srv.Close()

	if _, err := sc.PostComponents(ComponentSlice{Components: []Component{{ID: "x0c0s0b0n0"}}}, "tok"); err == nil {
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
