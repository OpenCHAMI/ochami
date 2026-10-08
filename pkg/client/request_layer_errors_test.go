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

// TestPayloadInput_Failures verifies that the payload helpers return an error
// for empty bytes, a missing or directory payload file, a malformed payload,
// and a failing reader.
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
