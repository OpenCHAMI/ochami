// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// TestBSSBootImageSet_MalformedAndEmptyResponses verifies that "bss boot image
// set" fails with CodePayload for a malformed boot parameters response and with
// CodeGeneric when no boot parameters match.
func TestBSSBootImageSet_MalformedAndEmptyResponses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		body     string
		wantCode int
		wantErr  string
	}{
		{name: "malformed", body: `{`, wantCode: cli.CodePayload, wantErr: "failed to unmarshal boot params"},
		{name: "empty", body: `[]`, wantCode: cli.CodeGeneric, wantErr: "no boot params to edit"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				io.WriteString(w, tc.body)
			}))
			defer srv.Close()

			res := runOchamiWithRuntime(t, "--ignore-config", "bss", "boot", "image", "set", "--uri", srv.URL,
				"--token", "t", "--mac", "de:ad:be:ef:00:00", "/dev/newroot")
			if res.err == nil || res.exitCode != tc.wantCode {
				t.Fatalf("result = (err %v, exit %d), want exit %d (%s)", res.err, res.exitCode, tc.wantCode, cli.CodeName(tc.wantCode))
			}
			if !strings.Contains(res.err.Error(), tc.wantErr) {
				t.Errorf("error = %q, want %q", res.err, tc.wantErr)
			}
		})
	}
}

// TestBSSBootImageSet_GetHTTPError verifies a failing GET of boot params resolves
// to CodeHTTP.
func TestBSSBootImageSet_GetHTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "bss", "boot", "image", "set", "--uri", srv.URL, "--token", "t",
		"--mac", "de:ad:be:ef:00:00", "https://example.com/new-image")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestBSSBootImageSet_PutHTTPError verifies a failing PUT resolves to CodeHTTP
// via the per-item aggregate.
func TestBSSBootImageSet_PutHTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Write([]byte(`[{"macs":["de:ad:be:ef:00:00"],"kernel":"http://s3/vmlinuz","params":"root=live:old"}]`))
			return
		}
		http.Error(w, "bad", http.StatusBadRequest)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "bss", "boot", "image", "set", "--uri", srv.URL, "--token", "t",
		"--mac", "de:ad:be:ef:00:00", "https://example.com/new-image")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}
