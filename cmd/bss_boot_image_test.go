// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestBSSBootImageSet_ByXnameAndNid verifies "boot image set" selects nodes by
// --xname and --nid, fetching then PUTting the modified boot parameters.
func TestBSSBootImageSet_ByXnameAndNid(t *testing.T) {
	t.Parallel()

	for _, sel := range [][]string{{"--xname", "x0c0s0b0n0"}, {"--nid", "1"}} {
		t.Run(sel[0], func(t *testing.T) {
			var puts int
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.Method {
				case http.MethodGet:
					w.Write([]byte(`[{"macs":["de:ad:be:ef:00:00"],"kernel":"http://s3/vmlinuz","params":"root=live:old"}]`))
				case http.MethodPut:
					puts++
					w.WriteHeader(http.StatusOK)
				}
			}))
			defer srv.Close()

			args := append([]string{"--ignore-config", "bss", "boot", "image", "set", "--uri", srv.URL, "--token", "t"},
				append(sel, "https://example.com/new-image")...)
			t.Parallel()
			res := runOchamiWithRuntime(t, args...)
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
			if puts == 0 {
				t.Error("expected at least one PUT to update boot params, got none")
			}
		})
	}
}
