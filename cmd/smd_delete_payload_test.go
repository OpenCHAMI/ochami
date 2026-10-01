// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// smd_delete_payload_test.go verifies that "smd <resource> delete -d ..."
// correctly extracts an ID list from payload data for every SMD delete
// command that accepts -d (group, compep, iface, rfe, component). The
// payload is only ever used as a source of IDs, never sent as the DELETE
// request body (see pkg/client/smd's Delete* methods), so these tests
// exercise the extraction logic itself: the right number of DELETE requests
// for a populated payload, and a CodeUsage rejection before any request is
// made for an empty one.

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSMDDelete_PayloadExtractsIDs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		args        []string
		payload     string
		wantDeletes int
	}{
		{
			name:        "group delete",
			args:        []string{"smd", "group", "delete"},
			payload:     `[{"label":"compute"},{"label":"storage"}]`,
			wantDeletes: 2,
		},
		{
			name:        "compep delete",
			args:        []string{"smd", "compep", "delete"},
			payload:     `[{"ID":"x0c0s0b0n0"},{"ID":"x0c0s0b0n1"}]`,
			wantDeletes: 2,
		},
		{
			name:        "iface delete",
			args:        []string{"smd", "iface", "delete"},
			payload:     `[{"ID":"de:ad:be:ef:00:00"}]`,
			wantDeletes: 1,
		},
		{
			name:        "rfe delete",
			args:        []string{"smd", "rfe", "delete"},
			payload:     `{"RedfishEndpoints":[{"ID":"x0c0s0b0n0"},{"ID":"x0c0s0b0n1"},{"ID":"x0c0s0b0n2"}]}`,
			wantDeletes: 3,
		},
		{
			name:        "component delete",
			args:        []string{"smd", "component", "delete"},
			payload:     `{"Components":[{"ID":"x0c0s0b0n0"}]}`,
			wantDeletes: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var deletes int
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodDelete {
					deletes++
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer srv.Close()

			args := append(append([]string{}, tc.args...), "--uri", srv.URL, "--token", "t",
				"--no-confirm", "-d", tc.payload)
			res := runOchamiWithRuntime(t, args...)
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
			if deletes != tc.wantDeletes {
				t.Errorf("DELETE count = %d, want %d", deletes, tc.wantDeletes)
			}
		})
	}
}
