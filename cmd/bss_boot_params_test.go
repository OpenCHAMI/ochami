// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// bss_boot_params_test.go exercises the "bss boot params" verbs (get, add,
// set, update, delete) end-to-end against an httptest.Server, asserting
// outbound request method/path/query/body. HTTP-failure and rejection-path
// cases are covered in bss_boot_params_errors_test.go.

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"
)

// TestBSSBootParamsGet_All verifies "bss boot params get" issues GET
// /bootparameters and prints the response body.
func TestBSSBootParamsGet_All(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.Write([]byte(`[{"hosts":["x0c0s0b0n0"]}]`))
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "bss", "boot", "params", "get",
		"--ignore-config", "--uri", srv.URL, "--token", "faketoken")

	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("method = %q, want GET", gotMethod)
	}
	if gotPath != "/bootparameters" {
		t.Errorf("path = %q, want /bootparameters", gotPath)
	}
	if !strings.Contains(res.stdout, "x0c0s0b0n0") {
		t.Errorf("stdout = %q, want it to contain the host", res.stdout)
	}
}

// TestBSSBootParamsGet_WithMAC verifies that --mac is encoded into the query
// string sent to /bootparameters.
func TestBSSBootParamsGet_WithMAC(t *testing.T) {
	t.Parallel()

	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		gotPath = r.URL.Path
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "bss", "boot", "params", "get",
		"--ignore-config", "--uri", srv.URL, "--token", "faketoken",
		"--mac", "de:ad:be:ef:00:00")

	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.Contains(gotQuery, "mac=de%3Aad%3Abe%3Aef%3A00%3A00") {
		t.Errorf("query = %q, want it to contain the URL-encoded mac", gotQuery)
	}
	if gotPath != "/bootparameters" {
		t.Errorf("path = %q, want /bootparameters", gotPath)
	}
}

// TestBSSBootParamsAdd_ViaFlags verifies "bss boot params add" issues POST
// /bootparameters with the kernel and macs encoded in the body.
func TestBSSBootParamsAdd_ViaFlags(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		var err error
		gotBody, err = io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "bss", "boot", "params", "add",
		"--ignore-config", "--uri", srv.URL, "--token", "faketoken",
		"--mac", "de:ad:be:ef:00:00",
		"--kernel", "https://example.com/vmlinuz")

	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/bootparameters" {
		t.Errorf("path = %q, want /bootparameters", gotPath)
	}
	var bp map[string]any
	if err := json.Unmarshal(gotBody, &bp); err != nil {
		t.Fatalf("failed to unmarshal body %q: %v", string(gotBody), err)
	}
	if k, _ := bp["kernel"].(string); k != "https://example.com/vmlinuz" {
		t.Errorf("kernel = %v, want https://example.com/vmlinuz", bp["kernel"])
	}
}

// TestBSSBootParamsDelete_NoConfirm verifies "bss boot params delete --no-confirm"
// issues DELETE /bootparameters.
func TestBSSBootParamsDelete_NoConfirm(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "bss", "boot", "params", "delete",
		"--ignore-config", "--uri", srv.URL, "--token", "faketoken",
		"--no-confirm", "--mac", "de:ad:be:ef:00:00",
		"--kernel", "https://example.com/vmlinuz")

	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want DELETE", gotMethod)
	}
	if gotPath != "/bootparameters" {
		t.Errorf("path = %q, want /bootparameters", gotPath)
	}
}

// TestBSSBootParamsGet_Query verifies the query builder emits name/mac/nid query
// parameters for the corresponding flags.
func TestBSSBootParamsGet_Query(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    []string
		wantKey string
		wantVal string
	}{
		{"xname", []string{"--xname", "x0c0s0b0n0"}, "name", "x0c0s0b0n0"},
		{"mac", []string{"--mac", "de:ad:be:ef:00:00"}, "mac", "de:ad:be:ef:00:00"},
		{"nid", []string{"--nid", "42"}, "nid", "42"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotQuery url.Values
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotQuery = r.URL.Query()
				w.Write([]byte(`[]`))
			}))
			defer srv.Close()

			args := append([]string{"bss", "boot", "params", "get",
				"--ignore-config", "--uri", srv.URL, "--token", "t"}, tc.args...)
			res := runOchamiWithRuntime(t, args...)
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
			if got := gotQuery.Get(tc.wantKey); got != tc.wantVal {
				t.Errorf("query %s = %q, want %q", tc.wantKey, got, tc.wantVal)
			}
		})
	}
}

// TestBSSBootParamsGet_Formats verifies the output-format variants.
func TestBSSBootParamsGet_Formats(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[{"macs":["de:ad:be:ef:00:00"],"params":"console=tty0"}]`))
	}))
	defer srv.Close()

	for _, f := range []string{"json", "json-pretty", "yaml"} {
		res := runOchamiWithRuntime(t, "bss", "boot", "params", "get",
			"--ignore-config", "--uri", srv.URL, "--token", "t", "-F", f)
		if res.err != nil {
			t.Fatalf("format %s: unexpected error: %v (exit %d)", f, res.err, res.exitCode)
		}
		assertFormattedOutput(t, f, res.stdout, "params", "console=tty0")
	}
}

// TestBSSBootParamsAdd_AllFlags verifies that "bss boot params add" accepts
// every component selector and config-field flag at once and sends a POST.
func TestBSSBootParamsAdd_AllFlags(t *testing.T) {
	t.Parallel()

	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "bss", "boot", "params", "add",
		"--ignore-config", "--uri", srv.URL, "--token", "t",
		"--xname", "x0c0s0b0n0", "--mac", "de:ad:be:ef:00:00", "--nid", "1",
		"--kernel", "https://example.com/vmlinuz", "--initrd", "https://example.com/initrd",
		"--params", "console=tty0")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
}

// TestBSSBootParamsDelete_AllFlags verifies that "bss boot params delete
// --no-confirm" accepts every component selector and config-field flag at once
// and sends a DELETE.
func TestBSSBootParamsDelete_AllFlags(t *testing.T) {
	t.Parallel()

	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "bss", "boot", "params", "delete",
		"--ignore-config", "--uri", srv.URL, "--token", "t", "--no-confirm",
		"--xname", "x0c0s0b0n0", "--mac", "de:ad:be:ef:00:00", "--nid", "1",
		"--kernel", "https://example.com/vmlinuz", "--initrd", "https://example.com/initrd",
		"--params", "console=tty0")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want DELETE", gotMethod)
	}
}

// TestBSSBootParamsSet_AllSelectorFlags verifies that "bss boot params set"
// accepts every component selector (--xname, --mac, --nid) and config field
// (--kernel, --initrd, --params) at once and sends a PUT.
func TestBSSBootParamsSet_AllSelectorFlags(t *testing.T) {
	t.Parallel()

	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "bss", "boot", "params", "set",
		"--ignore-config", "--uri", srv.URL, "--token", "t",
		"--xname", "x0c0s0b0n0", "--mac", "de:ad:be:ef:00:00", "--nid", "1",
		"--kernel", "https://example.com/vmlinuz", "--initrd", "https://example.com/initrd",
		"--params", "console=tty0")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodPut {
		t.Errorf("method = %q, want PUT", gotMethod)
	}
}

// TestBSSBootParamsUpdate_AllSelectorFlags verifies that "bss boot params
// update" accepts the --xname and --nid selectors together with every
// config-field flag and sends a PATCH.
func TestBSSBootParamsUpdate_AllSelectorFlags(t *testing.T) {
	t.Parallel()

	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "bss", "boot", "params", "update",
		"--ignore-config", "--uri", srv.URL, "--token", "t",
		"--xname", "x0c0s0b0n0", "--nid", "1",
		"--kernel", "https://example.com/vmlinuz", "--initrd", "https://example.com/initrd",
		"--params", "console=tty0")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodPatch {
		t.Errorf("method = %q, want PATCH", gotMethod)
	}
}

// TestBSSBootParamsDelete_ByFlags verifies "delete --no-confirm" with component
// and config flags issues a DELETE.
func TestBSSBootParamsDelete_ByFlags(t *testing.T) {
	t.Parallel()

	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "bss", "boot", "params", "delete",
		"--ignore-config", "--uri", srv.URL, "--token", "t", "--no-confirm",
		"--xname", "x0c0s0b0n0", "--nid", "1", "--kernel", "https://example.com/vmlinuz",
		"--initrd", "https://example.com/initrd", "--params", "quiet")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want DELETE", gotMethod)
	}
}

// TestBSSBootParamsDelete_OneFlag verifies "delete --no-confirm" accepts a
// lone component selector or a lone boot config flag, and sends a DELETE
// whose body carries that field and leaves the others empty.
func TestBSSBootParamsDelete_OneFlag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		flag       string
		value      string
		wantMacs   []string
		wantKernel string
	}{
		{"mac", "--mac", "de:ad:be:ef:00:00", []string{"de:ad:be:ef:00:00"}, ""},
		{"kernel", "--kernel", "https://example.com/vmlinuz", nil, "https://example.com/vmlinuz"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var gotMethod string
			var gotBody []byte
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotMethod = r.Method
				var err error
				gotBody, err = io.ReadAll(r.Body)
				if err != nil {
					t.Errorf("read request body: %v", err)
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer srv.Close()

			res := runOchamiWithRuntime(t, "bss", "boot", "params", "delete",
				"--ignore-config", "--uri", srv.URL, "--token", "t", "--no-confirm",
				tt.flag, tt.value)
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
			if gotMethod != http.MethodDelete {
				t.Errorf("method = %q, want DELETE", gotMethod)
			}
			var bp struct {
				Macs   []string `json:"macs"`
				Kernel string   `json:"kernel"`
			}
			if err := json.Unmarshal(gotBody, &bp); err != nil {
				t.Fatalf("failed to unmarshal body %q: %v", string(gotBody), err)
			}
			if !slices.Equal(bp.Macs, tt.wantMacs) || bp.Kernel != tt.wantKernel {
				t.Errorf("body macs = %v, kernel = %q; want %v, %q", bp.Macs, bp.Kernel, tt.wantMacs, tt.wantKernel)
			}
		})
	}
}

// TestBSSBootParamsDelete_ByData verifies "delete -d <payload>" issues a DELETE.
func TestBSSBootParamsDelete_ByData(t *testing.T) {
	t.Parallel()

	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "bss", "boot", "params", "delete",
		"--ignore-config", "--uri", srv.URL, "--token", "t", "--no-confirm",
		"-d", `{"macs":["de:ad:be:ef:00:00"]}`)
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want DELETE", gotMethod)
	}
}

// TestBSSBootParamsDelete_Confirm verifies the interactive confirm path issues
// the DELETE when the user answers "y".
func TestBSSBootParamsDelete_Confirm(t *testing.T) {
	t.Parallel()

	var deletes int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deletes++
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithInputAndRuntime(t, "y\n",
		"bss", "boot", "params", "delete", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"--mac", "de:ad:be:ef:00:00", "--kernel", "https://example.com/vmlinuz")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if deletes != 1 {
		t.Errorf("DELETE count = %d, want 1", deletes)
	}
}

// TestBSSBootParams_PayloadArray verifies that add, set, update, and delete
// accept a -d payload holding an array of boot parameters, in JSON or YAML,
// and send one request per element, in order.
func TestBSSBootParams_PayloadArray(t *testing.T) {
	t.Parallel()

	const (
		jsonPayload = `[{"macs":["de:ad:be:ef:00:00"],"kernel":"k1"},{"macs":["de:ad:be:ef:00:01"],"kernel":"k2"}]`
		yamlPayload = "- macs: [de:ad:be:ef:00:00]\n  kernel: k1\n- macs: [de:ad:be:ef:00:01]\n  kernel: k2\n"
	)
	tests := []struct {
		name       string
		verb       string
		args       []string
		status     int
		wantMethod string
	}{
		{"add", "add", []string{"-d", jsonPayload}, http.StatusCreated, http.MethodPost},
		{"set", "set", []string{"-d", jsonPayload}, http.StatusOK, http.MethodPut},
		{"set yaml", "set", []string{"-d", yamlPayload, "-f", "yaml"}, http.StatusOK, http.MethodPut},
		{"update", "update", []string{"-d", jsonPayload}, http.StatusOK, http.MethodPatch},
		{"delete", "delete", []string{"-d", jsonPayload, "--no-confirm"}, http.StatusOK, http.MethodDelete},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var gotKernels []string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != tt.wantMethod {
					t.Errorf("method = %q, want %s", r.Method, tt.wantMethod)
				}
				var bp struct {
					Kernel string `json:"kernel"`
				}
				if err := json.NewDecoder(r.Body).Decode(&bp); err != nil {
					t.Errorf("decode request body: %v", err)
				}
				gotKernels = append(gotKernels, bp.Kernel)
				w.WriteHeader(tt.status)
			}))
			defer srv.Close()

			args := append([]string{"bss", "boot", "params", tt.verb,
				"--ignore-config", "--uri", srv.URL, "--token", "t"}, tt.args...)
			res := runOchamiWithRuntime(t, args...)
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
			if !slices.Equal(gotKernels, []string{"k1", "k2"}) {
				t.Errorf("request kernels = %v, want [k1 k2]", gotKernels)
			}
		})
	}
}

// TestBSSBootParamsSet_Success verifies "bss boot params set" issues PUT /bootparameters.
func TestBSSBootParamsSet_Success(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "bss", "boot", "params", "set",
		"--ignore-config", "--uri", srv.URL, "--token", "faketoken",
		"--mac", "de:ad:be:ef:00:00", "--kernel", "https://example.com/vmlinuz")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodPut {
		t.Errorf("method = %q, want PUT", gotMethod)
	}
	if gotPath != "/bootparameters" {
		t.Errorf("path = %q, want /bootparameters", gotPath)
	}
}

// TestBSSBootParamsUpdate_Success verifies "bss boot params update" issues PATCH
// /bootparameters.
func TestBSSBootParamsUpdate_Success(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "bss", "boot", "params", "update",
		"--ignore-config", "--uri", srv.URL, "--token", "faketoken",
		"--mac", "de:ad:be:ef:00:00", "--kernel", "https://example.com/vmlinuz")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodPatch {
		t.Errorf("method = %q, want PATCH", gotMethod)
	}
	if gotPath != "/bootparameters" {
		t.Errorf("path = %q, want /bootparameters", gotPath)
	}
}
