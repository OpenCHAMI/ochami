// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package client

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/openchami/ochami/pkg/format"
)

// TestGetURI_RejectsInvalidInputs verifies that GetURI rejects a nil base URI
// and a malformed endpoint, and that it builds the URI without changing the
// client's base URI.
func TestGetURI_RejectsInvalidInputs(t *testing.T) {
	t.Run("nil base URI", func(t *testing.T) {
		oc := &OchamiClient{}
		if _, err := oc.GetURI("items", ""); err == nil || !strings.Contains(err.Error(), "base URI is nil") {
			t.Fatalf("GetURI() error = %v, want nil-base-URI error", err)
		}
	})

	t.Run("malformed endpoint", func(t *testing.T) {
		oc := &OchamiClient{BaseURI: &url.URL{Scheme: "https", Host: "example.com"}}
		if _, err := oc.GetURI("bad%zz", ""); err == nil || !strings.Contains(err.Error(), "failed to join path") {
			t.Fatalf("GetURI() error = %v, want path error", err)
		}
	})

	t.Run("base URI remains unchanged", func(t *testing.T) {
		base, err := url.Parse("https://example.com/api?original=yes")
		if err != nil {
			t.Fatal(err)
		}
		oc := &OchamiClient{BaseURI: base}
		got, err := oc.GetURI("items", "replacement=yes")
		if err != nil {
			t.Fatal(err)
		}
		if got != "https://example.com/api/items?replacement=yes" {
			t.Errorf("GetURI() = %q", got)
		}
		if base.String() != "https://example.com/api?original=yes" {
			t.Errorf("base URI mutated to %q", base)
		}
	})
}

// TestNewOchamiClient_RejectsMalformedBaseURI verifies that NewOchamiClient
// rejects a base URI that can't be parsed.
func TestNewOchamiClient_RejectsMalformedBaseURI(t *testing.T) {
	if _, err := NewOchamiClient("test", "https://example.com/%zz"); err == nil || !strings.Contains(err.Error(), "failed to parse URI") {
		t.Fatalf("NewOchamiClient() error = %v, want parse error", err)
	}
}

// TestMakeRequest_FailurePaths verifies that MakeRequest returns
// request-creation and transport failures, and that NewHTTPEnvelopeFromResponse
// returns response-body read and close failures and closes the body after a
// failed read.
func TestMakeRequest_FailurePaths(t *testing.T) {
	oc, err := NewOchamiClient("test", "https://example.com")
	if err != nil {
		t.Fatal(err)
	}

	t.Run("request creation", func(t *testing.T) {
		if _, err := oc.MakeRequest(context.Background(), "BAD\nMETHOD", "https://example.com", nil, nil); err == nil || !strings.Contains(err.Error(), "create new HTTP request") {
			t.Fatalf("MakeRequest() error = %v, want request creation error", err)
		}
	})

	t.Run("transport", func(t *testing.T) {
		transportErr := errors.New("transport unavailable")
		oc.Client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, transportErr
		})}
		if _, err := oc.MakeRequest(context.Background(), http.MethodGet, "https://example.com", nil, nil); !errors.Is(err, transportErr) {
			t.Fatalf("MakeRequest() error = %v, want wrapped transport error", err)
		}
	})

	t.Run("response body read", func(t *testing.T) {
		readErr := errors.New("read failed")
		body := &controlledReadCloser{reader: failingReader{err: readErr}}
		oc.Client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{Status: "200 OK", StatusCode: http.StatusOK, Header: make(http.Header), Body: body, ContentLength: 1}, nil
		})}
		res, err := oc.MakeRequest(context.Background(), http.MethodGet, "https://example.com", nil, nil)
		if err != nil {
			t.Fatalf("MakeRequest() error = %v", err)
		}
		if _, err := NewHTTPEnvelopeFromResponse(res); !errors.Is(err, readErr) {
			t.Fatalf("NewHTTPEnvelopeFromResponse() error = %v, want wrapped read error", err)
		}
		if !body.closed {
			t.Error("response body was not closed after read failure")
		}
	})

	t.Run("response body close", func(t *testing.T) {
		closeErr := errors.New("close failed")
		body := &controlledReadCloser{reader: strings.NewReader("ok"), closeErr: closeErr}
		oc.Client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{Status: "200 OK", StatusCode: http.StatusOK, Header: make(http.Header), Body: body, ContentLength: 2}, nil
		})}
		res, err := oc.MakeRequest(context.Background(), http.MethodGet, "https://example.com", nil, nil)
		if err != nil {
			t.Fatalf("MakeRequest() error = %v", err)
		}
		if _, err := NewHTTPEnvelopeFromResponse(res); !errors.Is(err, closeErr) {
			t.Fatalf("NewHTTPEnvelopeFromResponse() error = %v, want wrapped close error", err)
		}
	})
}

// TestMakeOchamiRequest_ReportsURIFailures verifies that when the request URI
// can't be built, MakeOchamiRequest's error names the endpoint and, if one was
// given, the query.
func TestMakeOchamiRequest_ReportsURIFailures(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  string
	}{
		{name: "without query", want: "endpoint items"},
		{name: "with query", query: "limit=1", want: "endpoint items and query limit=1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oc := &OchamiClient{}
			_, err := oc.MakeOchamiRequest(context.Background(), http.MethodGet, "items", tt.query, nil, nil)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("MakeOchamiRequest() error = %v, want text %q", err, tt.want)
			}
		})
	}
}

// TestDataMethods_ReportResponseReadFailures verifies that GetData, PostData,
// PutData, PatchData, and DeleteData return a failure to read the response
// body, wrapped with the request's method.
func TestDataMethods_ReportResponseReadFailures(t *testing.T) {
	readErr := errors.New("response read failed")
	tests := []struct {
		name string
		call func(*OchamiClient) error
		want string
	}{
		{name: "get", call: func(c *OchamiClient) error {
			_, err := c.GetData(context.Background(), "items", "", nil)
			return err
		}, want: "GET response"},
		{name: "post", call: func(c *OchamiClient) error {
			_, err := c.PostData(context.Background(), "items", "", nil, nil)
			return err
		}, want: "POST response"},
		{name: "put", call: func(c *OchamiClient) error {
			_, err := c.PutData(context.Background(), "items", "", nil, nil)
			return err
		}, want: "PUT response"},
		{name: "patch", call: func(c *OchamiClient) error {
			_, err := c.PatchData(context.Background(), "items", "", nil, nil)
			return err
		}, want: "PATCH response"},
		{name: "delete", call: func(c *OchamiClient) error {
			_, err := c.DeleteData(context.Background(), "items", "", nil, nil)
			return err
		}, want: "DELETE response"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oc, err := NewOchamiClient("test", "https://example.com")
			if err != nil {
				t.Fatal(err)
			}
			oc.Client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{
					Status: "200 OK", StatusCode: http.StatusOK, Header: make(http.Header),
					Body: &controlledReadCloser{reader: failingReader{err: readErr}},
				}, nil
			})}
			err = tt.call(oc)
			if !errors.Is(err, readErr) || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("call error = %v, want wrapped read error with %q", err, tt.want)
			}
		})
	}
}

// TestPayloadInput_Failures verifies that the payload helpers return an error
// for empty bytes, a missing or directory payload file, a missing stdin reader,
// a failing reader, and malformed file, data, or reader payloads.
func TestPayloadInput_Failures(t *testing.T) {
	dir := t.TempDir()
	malformed := filepath.Join(dir, "malformed.yaml")
	if err := os.WriteFile(malformed, []byte("key: [\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		call func() error
	}{
		{name: "empty bytes", call: func() error { _, err := BytesToHTTPBody(nil, format.DataFormatJson); return err }},
		{name: "missing file", call: func() error {
			_, err := FileToHTTPBody(filepath.Join(dir, "missing"), format.DataFormatJson)
			return err
		}},
		{name: "directory payload", call: func() error { _, err := FileToHTTPBody(dir, format.DataFormatJson); return err }},
		{name: "malformed payload", call: func() error { _, err := FileToHTTPBody(malformed, format.DataFormatYaml); return err }},
		{name: "reader failure", call: func() error {
			var v any
			return ReadPayloadReader(failingReader{err: io.ErrUnexpectedEOF}, format.DataFormatJson, &v)
		}},
		{name: "slice reader failure", call: func() error {
			var v []any
			return ReadPayloadReaderSlice(failingReader{err: io.ErrUnexpectedEOF}, format.DataFormatJson, &v)
		}},
		{name: "nil slice stdin", call: func() error {
			var v []any
			return ReadPayloadFileSliceWithReader("-", nil, format.DataFormatJson, &v)
		}},
		{name: "malformed file slice", call: func() error {
			var v []any
			return ReadPayloadFileSliceWithReader[any](malformed, nil, format.DataFormatYaml, &v)
		}},
		{name: "malformed scalar data", call: func() error {
			var v any
			return ReadPayloadData("{", format.DataFormatJson, &v)
		}},
		{name: "malformed slice data", call: func() error {
			var v []any
			return ReadPayloadDataSlice("{", format.DataFormatJson, &v)
		}},
		{name: "malformed scalar reader", call: func() error {
			var v any
			return ReadPayloadReader(strings.NewReader("{"), format.DataFormatJson, &v)
		}},
		{name: "malformed slice reader", call: func() error {
			var v []any
			return ReadPayloadReaderSlice(strings.NewReader("{"), format.DataFormatJson, &v)
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.call(); err == nil {
				t.Fatal("call returned nil error")
			}
		})
	}
}

// TestUseCACert_FileFailures verifies that UseCACert rejects a missing file, a
// directory, and an empty file.
func TestUseCACert_FileFailures(t *testing.T) {
	oc, err := NewOchamiClient("test", "https://example.com")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	empty := filepath.Join(dir, "empty.pem")
	if err := os.WriteFile(empty, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{filepath.Join(dir, "missing.pem"), dir, empty} {
		if err := oc.UseCACert(path); err == nil {
			t.Errorf("UseCACert(%q) error = nil", path)
		}
	}
}
