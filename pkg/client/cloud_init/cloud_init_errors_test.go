// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cloud_init

// cloud_init_errors_test.go unit-tests the CloudInitClient wrapper methods'
// error arms: input rejected before a request is made, a non-2XX response
// surfacing as an UnsuccessfulHTTPError, and the per-item edge cases of the
// mutating helpers (blank names/IDs, empty lists, and per-item HTTP
// failures). It also covers DecodeCloudConfig's own rejection case (see
// cloud_init_test.go for its success cases).

import (
	"errors"
	"net/http"
	"testing"

	"github.com/openchami/cloud-init/pkg/cistore"

	"github.com/openchami/ochami/pkg/client"
)

// TestGetNodeData_RequiresID verifies GetNodeData errors (without a request) when
// no IDs are supplied.
func TestGetNodeData_RequiresID(t *testing.T) {
	requestMade := false
	cic, srv := newTestCI(t, func(w http.ResponseWriter, r *http.Request) { requestMade = true })
	defer srv.Close()

	if _, _, err := cic.GetNodeData(CloudInitMetaData, "tok"); err == nil {
		t.Fatal("expected an error when no IDs are supplied, got nil")
	}
	if requestMade {
		t.Error("a request was made despite no IDs being supplied")
	}
}

// TestGetDefaults_UnsuccessfulHTTP verifies a non-2XX response surfaces as an
// UnsuccessfulHTTPError.
func TestGetDefaults_UnsuccessfulHTTP(t *testing.T) {
	cic, srv := newTestCI(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusForbidden)
	})
	defer srv.Close()

	_, err := cic.GetDefaults("tok")
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !errors.Is(err, client.UnsuccessfulHTTPError) {
		t.Errorf("error = %v, want it to wrap client.UnsuccessfulHTTPError", err)
	}
}

// TestDecodeCloudConfig_UnknownEncoding verifies that an unrecognized
// CloudConfigFile.Encoding value is rejected.
func TestDecodeCloudConfig_UnknownEncoding(t *testing.T) {
	if _, err := DecodeCloudConfig(cistore.CloudConfigFile{Content: []byte("x"), Encoding: "rot13"}); err == nil {
		t.Error("expected an error for unknown encoding, got nil")
	}
}

func TestPutGroups_EdgeCases(t *testing.T) {
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

func TestPutInstanceInfo_EdgeCases(t *testing.T) {
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
