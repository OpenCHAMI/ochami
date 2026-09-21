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
				_, _ = w.Write([]byte("boom")) //nolint:errcheck // test response writes are observed by the client
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

// newTestClient builds an SMDClient pointed at srv.
func newTestClient(t *testing.T, srv *httptest.Server) *SMDClient {
	t.Helper()
	c, err := NewClient(srv.URL, client.WithInsecure(true))
	if err != nil {
		t.Fatalf("failed to create SMD client: %v", err)
	}
	return c
}

// TestGetEthernetInterfaceByIDWithIPs verifies the getIPs=true path targets the
// IPAddresses subpath.
func TestGetEthernetInterfaceByIDWithIPs(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`[]`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if _, err := c.GetEthernetInterfaceByID("decafc0ffeee", "tok", true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(gotPath, "IPAddresses") {
		t.Errorf("path = %q, want it to reference IPAddresses", gotPath)
	}
}

// TestGetEthernetInterfaceByIDHTTPError verifies the GetData error arm.
func TestGetEthernetInterfaceByIDHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if _, err := c.GetEthernetInterfaceByID("decafc0ffeee", "tok", false); err == nil {
		t.Fatal("expected an error, got nil")
	}
}

// TestPostGroupMembersGuards verifies the empty-group and empty-members guard
// clauses.
func TestPostGroupMembersGuards(t *testing.T) {
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

// TestPostGroupMembersPerItemError verifies the per-item POST error arm.
func TestPostGroupMembersPerItemError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()
	c := newTestClient(t, srv)

	_, errs, err := c.PostGroupMembers("tok", "compute", "x0c0s0b0n0")
	if err != nil {
		t.Fatalf("function error = %v, want nil (per-item errors reported separately)", err)
	}
	if len(errs) != 1 || errs[0] == nil {
		t.Errorf("errs = %v, want one non-nil per-item error", errs)
	}
}

// TestDeleteHelpersPerItemError verifies the per-item DELETE error arms of the
// iterative delete helpers.
func TestDeleteHelpersPerItemError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()
	c := newTestClient(t, srv)

	if _, errs, err := c.DeleteComponents("tok", "x0c0s0b0n0"); err != nil || len(errs) != 1 || errs[0] == nil {
		t.Errorf("DeleteComponents per-item error = (err=%v, errs=%v), want one non-nil per-item error", err, errs)
	}
	if _, errs, err := c.DeleteRedfishEndpoints("tok", "x3000c1s7b56"); err != nil || len(errs) != 1 || errs[0] == nil {
		t.Errorf("DeleteRedfishEndpoints per-item error = (err=%v, errs=%v), want one non-nil per-item error", err, errs)
	}
	if _, errs, err := c.DeleteEthernetInterfaces("tok", "decafc0ffeee"); err != nil || len(errs) != 1 || errs[0] == nil {
		t.Errorf("DeleteEthernetInterfaces per-item error = (err=%v, errs=%v), want one non-nil per-item error", err, errs)
	}
	if _, errs, err := c.DeleteGroupMembers("tok", "compute", "x0c0s0b0n0"); err != nil || len(errs) != 1 || errs[0] == nil {
		t.Errorf("DeleteGroupMembers per-item error = (err=%v, errs=%v), want one non-nil per-item error", err, errs)
	}
}

// TestPostHelpersPerItemError verifies the per-item POST error arms of the
// iterative add helpers.
func TestPostHelpersPerItemError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()
	c := newTestClient(t, srv)

	eis := []EthernetInterface{{ComponentID: "x0c0s0b0n0", MACAddress: "de:ad:be:ef:00:00"}}
	if _, errs, err := c.PostEthernetInterfaces(eis, "tok"); err != nil || len(errs) != 1 || errs[0] == nil {
		t.Errorf("PostEthernetInterfaces per-item error = (err=%v, errs=%v), want one non-nil per-item error", err, errs)
	}
}

// TestGetStatusUnknownComponent verifies the unknown-component arm of GetStatus.
func TestGetStatusUnknownComponent(t *testing.T) {
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

// TestGetComponentsByXnameNidError verifies the error arms of the single-item
// component getters.
func TestGetComponentsByXnameNidError(t *testing.T) {
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

// TestPutGroupMembersGuards verifies the empty-group and empty-members guards.
func TestPutGroupMembersGuards(t *testing.T) {
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

// TestDeleteAllHelpersError verifies the error arms of the delete-all helpers.
func TestDeleteAllHelpersError(t *testing.T) {
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
