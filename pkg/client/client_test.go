// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package client

import (
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/OpenCHAMI/ochami/pkg/format"
)

func TestGetData(t *testing.T) {
	// Test server for GET
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok":
			w.Header().Set("X-Test", "yes")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"msg":"success"}`))
		case "/fail":
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("oops"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	oc, err := NewOchamiClient("svc", ts.URL, false)
	if err != nil {
		t.Fatalf("NewOchamiClient: %v", err)
	}

	tests := []struct {
		name      string
		endpoint  string
		wantErr   bool
		wantIsErr error
	}{
		{
			name:     "GET success",
			endpoint: "ok",
			wantErr:  false,
		},
		{
			name:      "GET fail",
			endpoint:  "fail",
			wantErr:   true,
			wantIsErr: UnsuccessfulHTTPError,
		},
	}

	for _, tt := range tests {
		// Create per-iteration copy of test tt so that running
		// tests in parallel does not reuse the same test for
		// each run.
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			hdrs := NewHTTPHeaders()
			env, err := oc.GetData(tc.endpoint, "", hdrs)
			if (err != nil) != tc.wantErr {
				t.Fatalf("GetData error = %v, wantErr %v", err, tc.wantErr)
			}
			if tc.wantIsErr != nil && !errors.Is(err, tc.wantIsErr) {
				t.Errorf("GetData error = %v, want Is(%v)", err, tc.wantIsErr)
			}
			if !tc.wantErr {
				if env.StatusCode != 200 {
					t.Errorf("StatusCode = %d, want 200", env.StatusCode)
				}
				if got := string(env.Body); got != `{"msg":"success"}` {
					t.Errorf("Body = %q, want %q", got, `{"msg":"success"}`)
				}
			}
		})
	}
}

func TestPostData(t *testing.T) {
	// Test server for POST
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"msg":"created"}`))
		case "/fail":
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("boom"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	oc, err := NewOchamiClient("svc", ts.URL, false)
	if err != nil {
		t.Fatalf("NewOchamiClient: %v", err)
	}

	tests := []struct {
		name      string
		endpoint  string
		body      HTTPBody
		wantErr   bool
		wantIsErr error
	}{
		{
			name:     "POST success",
			endpoint: "ok",
			body:     HTTPBody(`{"foo":"bar"}`),
			wantErr:  false,
		},
		{
			name:      "POST fail",
			endpoint:  "fail",
			body:      HTTPBody(`{"x":1}`),
			wantErr:   true,
			wantIsErr: UnsuccessfulHTTPError,
		},
	}

	for _, tt := range tests {
		// Create per-iteration copy of test tt so that running
		// tests in parallel does not reuse the same test for
		// each run.
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			hdrs := NewHTTPHeaders()
			env, err := oc.PostData(tc.endpoint, "", hdrs, tc.body)
			if (err != nil) != tc.wantErr {
				t.Fatalf("PostData error = %v, wantErr %v", err, tc.wantErr)
			}
			if tc.wantIsErr != nil && !errors.Is(err, tc.wantIsErr) {
				t.Errorf("PostData error = %v, want Is(%v)", err, tc.wantIsErr)
			}
			if !tc.wantErr {
				if env.StatusCode != 200 {
					t.Errorf("StatusCode = %d, want 200", env.StatusCode)
				}
				if got := string(env.Body); got != `{"msg":"created"}` {
					t.Errorf("Body = %q, want %q", got, `{"msg":"created"}`)
				}
			}
		})
	}
}

func TestFileToHTTPBody(t *testing.T) {
	// Prepare a temp JSON file
	dir := t.TempDir()
	path := filepath.Join(dir, "payload.json")
	if err := ioutil.WriteFile(path, []byte(`{"n":42}`), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	tests := []struct {
		name    string
		path    string
		format  format.DataFormat
		want    string
		wantErr bool
	}{
		{
			name:    "valid JSON file",
			path:    path,
			format:  format.DataFormatJson,
			want:    `{"n":42}`,
			wantErr: false,
		},
		{
			name:    "empty path",
			path:    "",
			format:  format.DataFormatJson,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		// Create per-iteration copy of test tt so that running
		// tests in parallel does not reuse the same test for
		// each run.
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			b, err := FileToHTTPBody(tc.path, tc.format)
			if (err != nil) != tc.wantErr {
				t.Fatalf("FileToHTTPBody error = %v, wantErr %v", err, tc.wantErr)
			}
			if !tc.wantErr {
				if got := string(b); got != tc.want {
					t.Errorf("FileToHTTPBody = %q, want %q", got, tc.want)
				}
			}
		})
	}
}

func TestReadPayload(t *testing.T) {
	// Prepare a temp JSON file
	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")
	if err := ioutil.WriteFile(path, []byte(`{"k":7}`), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	tests := []struct {
		name    string
		fn      func(string, format.DataFormat, interface{}) error
		input   string
		fmt     format.DataFormat
		want    map[string]int
		wantErr bool
	}{
		{
			name:    "ReadPayloadFile",
			fn:      ReadPayloadFile,
			input:   path,
			fmt:     format.DataFormatJson,
			want:    map[string]int{"k": 7},
			wantErr: false,
		},
		{
			name:    "ReadPayload with @ prefix",
			fn:      ReadPayload,
			input:   "@" + path,
			fmt:     format.DataFormatJson,
			want:    map[string]int{"k": 7},
			wantErr: false,
		},
		{
			name:    "ReadPayloadData",
			fn:      ReadPayloadData,
			input:   `{"k":99}`,
			fmt:     format.DataFormatJson,
			want:    map[string]int{"k": 99},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		// Create per-iteration copy of test tt so that running
		// tests in parallel does not reuse the same test for
		// each run.
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			var m map[string]int
			err := tc.fn(tc.input, tc.fmt, &m)
			if (err != nil) != tc.wantErr {
				t.Fatalf("%s error = %v, wantErr %v", tc.name, err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			// compare maps
			if len(m) != len(tc.want) {
				t.Fatalf("%s map length = %d, want %d", tc.name, len(m), len(tc.want))
			}
			for k, v := range tc.want {
				if got := m[k]; got != v {
					t.Errorf("%s[%q] = %d, want %d", tc.name, k, got, v)
				}
			}
		})
	}
}

func TestReadPayloadSlice(t *testing.T) {
	type Item struct {
		K int `json:"k" yaml:"k"`
	}

	// Prepare temp files
	dir := t.TempDir()
	jsonSinglePath := filepath.Join(dir, "single.json")
	if err := ioutil.WriteFile(jsonSinglePath, []byte(`{"k":7}`), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	jsonArrayPath := filepath.Join(dir, "array.json")
	if err := ioutil.WriteFile(jsonArrayPath, []byte(`[{"k":1},{"k":2}]`), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	yamlSinglePath := filepath.Join(dir, "single.yaml")
	if err := ioutil.WriteFile(yamlSinglePath, []byte("k: 9\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	yamlArrayPath := filepath.Join(dir, "array.yaml")
	if err := ioutil.WriteFile(yamlArrayPath, []byte("- k: 3\n- k: 4\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Prepare stdin payload for "@-" case
	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdin = r
	t.Cleanup(func() {
		_ = r.Close()
		os.Stdin = oldStdin
	})
	if _, err := w.Write([]byte(`{"k":42}`)); err != nil {
		t.Fatalf("write stdin pipe: %v", err)
	}
	_ = w.Close()

	tests := []struct {
		name    string
		input   string
		fmt     format.DataFormat
		want    []Item
		wantErr bool
	}{
		{
			name:  "json single object (inline)",
			input: `{"k":7}`,
			fmt:   format.DataFormatJson,
			want:  []Item{{K: 7}},
		},
		{
			name:  "json array (inline)",
			input: `[{"k":1},{"k":2}]`,
			fmt:   format.DataFormatJson,
			want:  []Item{{K: 1}, {K: 2}},
		},
		{
			name:  "yaml single mapping (inline)",
			input: "k: 9\n",
			fmt:   format.DataFormatYaml,
			want:  []Item{{K: 9}},
		},
		{
			name:  "yaml sequence (inline)",
			input: "- k: 3\n- k: 4\n",
			fmt:   format.DataFormatYaml,
			want:  []Item{{K: 3}, {K: 4}},
		},
		{
			name:  "json single object via @file",
			input: "@" + jsonSinglePath,
			fmt:   format.DataFormatJson,
			want:  []Item{{K: 7}},
		},
		{
			name:  "json array via @file",
			input: "@" + jsonArrayPath,
			fmt:   format.DataFormatJson,
			want:  []Item{{K: 1}, {K: 2}},
		},
		{
			name:  "yaml single mapping via @file",
			input: "@" + yamlSinglePath,
			fmt:   format.DataFormatYaml,
			want:  []Item{{K: 9}},
		},
		{
			name:  "yaml sequence via @file",
			input: "@" + yamlArrayPath,
			fmt:   format.DataFormatYaml,
			want:  []Item{{K: 3}, {K: 4}},
		},
		{
			name:  "stdin via @-",
			input: "@-",
			fmt:   format.DataFormatJson,
			want:  []Item{{K: 42}},
		},
		{
			name:    "empty input",
			input:   "   \n\t",
			fmt:     format.DataFormatJson,
			wantErr: true,
		},
		{
			name:    "missing file via @file",
			input:   "@" + filepath.Join(dir, "does_not_exist.json"),
			fmt:     format.DataFormatJson,
			wantErr: true,
		},
		{
			name:    "malformed json",
			input:   `{"k":}`,
			fmt:     format.DataFormatJson,
			wantErr: true,
		},
		{
			name:    "wrong top-level (scalar)",
			input:   `123`,
			fmt:     format.DataFormatJson,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		// Create per-iteration copy of test tt so that running
		// tests in parallel does not reuse the same test for
		// each run.
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			// Start with a non-empty slice to ensure the function overwrites it.
			got := []Item{{K: -1}}
			err := ReadPayloadSlice[Item](tc.input, tc.fmt, &got)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ReadPayloadSlice error = %v, wantErr %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %#v, want %#v", got, tc.want)
			}
		})
	}
}
