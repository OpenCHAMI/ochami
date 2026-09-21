// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// bss_boot_image_test.go covers the branch families of "bss boot image set"
// that the happy path (TestBSSBootImageSet_Success in bss_test.go) does not reach:
// selector fan-out with per-selector "not found" warnings, malformed/empty
// GET responses, selection by --xname/--nid, and GET/PUT HTTP error mapping.

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

func TestBSSBootImageSet_SelectorsAndWarnings(t *testing.T) {
	t.Parallel()

	var gotQuery string
	var putBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			gotQuery = r.URL.RawQuery
			_, _ = io.WriteString(w, `[{"hosts":["x0c0s0b0n0"],"nids":[1],"macs":["de:ad:be:ef:00:00"],"params":"console=tty0 root=old"}]`) //nolint:errcheck // test response writes are observed by the client
		case http.MethodPut:
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("read PUT body: %v", err)
			}
			putBody = string(body)
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "--log-level", "warning", "bss", "boot", "image", "set",
		"--uri", srv.URL, "--token", "t",
		"--xname", "x0c0s0b0n0,x0c0s0b0n1", "--nid", "1,2", "--mac", "de:ad:be:ef:00:00,de:ad:be:ef:00:01", "/dev/newroot")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	for _, query := range []string{"name=x0c0s0b0n0", "name=x0c0s0b0n1", "nid=1", "nid=2", "mac=de%3Aad%3Abe%3Aef%3A00%3A00", "mac=de%3Aad%3Abe%3Aef%3A00%3A01"} {
		if !strings.Contains(gotQuery, query) {
			t.Errorf("query = %q, want %q", gotQuery, query)
		}
	}
	for _, warning := range []string{"host x0c0s0b0n1 not found", "node ID 2 not found", "mac de:ad:be:ef:00:01 not found"} {
		if !strings.Contains(res.stdout, warning) {
			t.Errorf("output = %q, want warning %q", res.stdout, warning)
		}
	}
	if !strings.Contains(putBody, `root=/dev/newroot`) {
		t.Errorf("PUT body = %q, want updated root argument", putBody)
	}
}

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
				_, _ = io.WriteString(w, tc.body) //nolint:errcheck // test response writes are observed by the client
			}))
			defer srv.Close()

			res := runOchamiWithRuntime(t, "--ignore-config", "bss", "boot", "image", "set", "--uri", srv.URL,
				"--token", "t", "--mac", "de:ad:be:ef:00:00", "/dev/newroot")
			if res.err == nil || res.exitCode != tc.wantCode {
				t.Fatalf("result = (err %v, exit %d), want exit %d", res.err, res.exitCode, tc.wantCode)
			}
			if !strings.Contains(res.err.Error(), tc.wantErr) {
				t.Errorf("error = %q, want %q", res.err, tc.wantErr)
			}
		})
	}
}

// TestBSSBootImageSet_ByXnameAndNid verifies "boot image set" selects nodes by
// --xname and --nid, fetching then PUTting the modified boot parameters.
func TestBSSBootImageSet_ByXnameAndNid(t *testing.T) {
	for _, sel := range [][]string{{"--xname", "x0c0s0b0n0"}, {"--nid", "1"}} {
		t.Run(sel[0], func(t *testing.T) {
			var puts int
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.Method {
				case http.MethodGet:
					_, _ = w.Write([]byte(`[{"macs":["de:ad:be:ef:00:00"],"kernel":"http://s3/vmlinuz","params":"root=live:old"}]`)) //nolint:errcheck // test response writes are observed by the client
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

// TestBSSBootImageSet_GetHTTPError verifies a failing GET of boot params resolves
// to CodeHTTP.
func TestBSSBootImageSet_GetHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	t.Parallel()
	res := runOchamiWithRuntime(t, "--ignore-config", "bss", "boot", "image", "set", "--uri", srv.URL, "--token", "t",
		"--mac", "de:ad:be:ef:00:00", "https://example.com/new-image")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestBSSBootImageSet_PutHTTPError verifies a failing PUT resolves to CodeHTTP
// via the per-item aggregate.
func TestBSSBootImageSet_PutHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`[{"macs":["de:ad:be:ef:00:00"],"kernel":"http://s3/vmlinuz","params":"root=live:old"}]`)) //nolint:errcheck // test response writes are observed by the client
			return
		}
		http.Error(w, "bad", http.StatusBadRequest)
	}))
	defer srv.Close()

	t.Parallel()
	res := runOchamiWithRuntime(t, "--ignore-config", "bss", "boot", "image", "set", "--uri", srv.URL, "--token", "t",
		"--mac", "de:ad:be:ef:00:00", "https://example.com/new-image")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}
