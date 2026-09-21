// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cloud_init

// cloud_init_errors_test.go unit-tests the CloudInitClient wrapper methods'
// error arms: input rejected before a request is made, a non-2XX response
// surfacing as an UnsuccessfulHTTPError, and the per-item edge cases of the
// mutating helpers (blank names/IDs, empty lists, mixed batch results, and
// per-item HTTP failures). It also covers DecodeCloudConfig's own rejection
// case (see cloud_init_test.go for its success cases).

import (
	"context"
	"errors"
	"net/http"
	"strings"
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

	if _, err := cic.GetNodeData(context.Background(), CloudInitMetaData, "tok"); err == nil {
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

	_, err := cic.GetDefaults(context.Background(), "tok")
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

			results := cic.PutGroups(context.Background(), tc.groups, "tok")
			if len(results) != 1 || (results[0].Err != nil) != tc.wantErr {
				t.Errorf("results = %v, wantErr %v", results, tc.wantErr)
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

// TestCloudInitIterative_MixedResults verifies that each iterative writer
// returns a per-item aligned BatchResult when the first item succeeds and the
// second fails, covering both the success and HTTP-error arms for
// PostGroups, PutGroups, PutInstanceInfo, and DeleteGroups.
func TestCloudInitIterative_MixedResults(t *testing.T) {
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
				t.Fatalf("%s: results length = %d, want 2", tc.name, len(results))
			}
			if results[0].Err != nil || results[0].Value.StatusCode != http.StatusOK {
				t.Errorf("%s: first result = %+v, want success", tc.name, results[0])
			}
			if results[1].Err == nil || results[1].Value.StatusCode != http.StatusInternalServerError {
				t.Errorf("%s: second result = %+v, want HTTP failure", tc.name, results[1])
			}
		})
	}
}

// TestGetNodeGroupData_Guards verifies the blank-id and empty-groups guards.
func TestGetNodeGroupData_Guards(t *testing.T) {
	cic, srv := newTestCI(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	if _, err := cic.GetNodeGroupData(context.Background(), "tok", "  "); err == nil {
		t.Error("GetNodeGroupData with blank id = nil, want error")
	}
	if _, err := cic.GetNodeGroupData(context.Background(), "tok", "x0c0s0b0n0"); err == nil {
		t.Error("GetNodeGroupData with no groups = nil, want error")
	}
}

// TestPostDefaults_HTTPError verifies the PostDefaults HTTP error arm.
func TestPostDefaults_HTTPError(t *testing.T) {
	cic, srv := newTestCI(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	})
	defer srv.Close()

	if _, err := cic.PostDefaults(context.Background(), cistore.ClusterDefaults{}, "tok"); err == nil {
		t.Error("PostDefaults with HTTP error = nil, want error")
	}
}

// TestCloudConfigGetters_PreserveMalformedBodies verifies that successful GETs
// return server data unchanged. Parsing remains the caller's responsibility,
// so malformed cloud-config can be diagnosed without losing the response.
func TestCloudConfigGetters_PreserveMalformedBodies(t *testing.T) {
	const malformed = "not-base64!"
	tests := []struct {
		name string
		call func(*CloudInitClient) (client.BatchResult[client.HTTPEnvelope], error)
	}{
		{name: "node data", call: func(cic *CloudInitClient) (client.BatchResult[client.HTTPEnvelope], error) {
			return cic.GetNodeData(context.Background(), CloudInitUserData, "tok", "x0c0s0b0n0")
		}},
		{name: "node group data", call: func(cic *CloudInitClient) (client.BatchResult[client.HTTPEnvelope], error) {
			return cic.GetNodeGroupData(context.Background(), "tok", "x0c0s0b0n0", "compute")
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cic, srv := newTestCI(t, func(w http.ResponseWriter, r *http.Request) {
				if got := r.Header.Get("Authorization"); got != "Bearer tok" {
					t.Errorf("Authorization = %q, want %q", got, "Bearer tok")
				}
				_, _ = w.Write([]byte(malformed)) //nolint:errcheck // test response writes are observed by the client
			})
			defer srv.Close()

			results, err := tc.call(cic)
			if err != nil {
				t.Fatalf("control-flow error = %v", err)
			}
			if len(results) != 1 || results[0].Err != nil {
				t.Fatalf("results = %v, want one successful response", results)
			}
			if got := string(results[0].Value.Body); got != malformed {
				t.Errorf("body = %q, want %q", got, malformed)
			}
			if _, err := DecodeCloudConfig(cistore.CloudConfigFile{Content: []byte(malformed), Encoding: "base64"}); err == nil || !strings.Contains(err.Error(), "base64 decode") {
				t.Errorf("DecodeCloudConfig() error = %v, want contextual base64 error", err)
			}
		})
	}
}

func TestCloudInitMalformedPathGuards(t *testing.T) {
	cic, srv := newTestCI(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected request: %s", r.URL.Path)
	})
	defer srv.Close()

	tests := []struct {
		name string
		call func() error
	}{
		{name: "get group", call: func() error { return cic.GetGroups(context.Background(), "", "%zz")[0].Err }},
		{name: "get node data", call: func() error {
			results, err := cic.GetNodeData(context.Background(), CloudInitMetaData, "", "%zz")
			if err != nil {
				return err
			}
			return results[0].Err
		}},
		{name: "get node group data", call: func() error {
			results, err := cic.GetNodeGroupData(context.Background(), "", "%zz", "compute")
			if err != nil {
				return err
			}
			return results[0].Err
		}},
		{name: "put group", call: func() error {
			return cic.PutGroups(context.Background(), []cistore.GroupData{{Name: "%zz"}}, "")[0].Err
		}},
		{name: "put instance info", call: func() error {
			results, err := cic.PutInstanceInfo(context.Background(), []cistore.OpenCHAMIInstanceInfo{{ID: "%zz"}}, "")
			if err != nil {
				return err
			}
			return results[0].Err
		}},
		{name: "delete group", call: func() error { return cic.DeleteGroups(context.Background(), "", "%zz")[0].Err }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.call(); err == nil {
				t.Fatal("call returned nil error for malformed path")
			}
		})
	}
}

func TestCloudInitOperationGuardsAndCancellation(t *testing.T) {
	cic, srv := newTestCI(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	if results, err := cic.PutInstanceInfo(context.Background(), nil, ""); err == nil || results != nil {
		t.Fatalf("PutInstanceInfo(empty) = (%v, %v), want control-flow error", results, err)
	}
	results, err := cic.PutInstanceInfo(context.Background(), []cistore.OpenCHAMIInstanceInfo{{ID: " "}}, "")
	if err != nil || len(results) != 1 || results[0].Err == nil {
		t.Fatalf("PutInstanceInfo(blank) = (%v, %v), want one item error", results, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	results = cic.GetGroups(ctx, "", "one", "two")
	if len(results) != 2 || !errors.Is(results[0].Err, context.Canceled) || !errors.Is(results[1].Err, context.Canceled) {
		t.Fatalf("GetGroups(canceled) = %#v", results)
	}
}
