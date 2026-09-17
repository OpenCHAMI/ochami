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
	"time"

	"github.com/openchami/ochami/pkg/format"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type failingReader struct {
	err error
}

func (r failingReader) Read([]byte) (int, error) {
	return 0, r.err
}

type controlledReadCloser struct {
	reader   io.Reader
	closeErr error
	closed   bool
}

func (r *controlledReadCloser) Read(p []byte) (int, error) {
	return r.reader.Read(p)
}

func (r *controlledReadCloser) Close() error {
	r.closed = true
	return r.closeErr
}

func TestGetURIRejectsInvalidInputs(t *testing.T) {
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

func TestNewOchamiClientRejectsMalformedBaseURI(t *testing.T) {
	if _, err := NewOchamiClient("test", "https://example.com/%zz"); err == nil || !strings.Contains(err.Error(), "failed to parse URI") {
		t.Fatalf("NewOchamiClient() error = %v, want parse error", err)
	}
}

func TestMakeRequestFailurePaths(t *testing.T) {
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

func TestOchamiClientsOwnIndependentTransports(t *testing.T) {
	secure, err := NewOchamiClient("secure", "https://example.com")
	if err != nil {
		t.Fatal(err)
	}
	insecure, err := NewOchamiClient("insecure", "https://example.com", WithInsecure(true))
	if err != nil {
		t.Fatal(err)
	}
	if secure.Client == http.DefaultClient || insecure.Client == http.DefaultClient {
		t.Fatal("OchamiClient reused process-global http.DefaultClient")
	}
	if secure.Transport == insecure.Transport {
		t.Fatal("secure and insecure clients share a transport")
	}
	secureTransport := secure.Transport.(*http.Transport)
	insecureTransport := insecure.Transport.(*http.Transport)
	if secureTransport.TLSClientConfig != nil && secureTransport.TLSClientConfig.InsecureSkipVerify {
		t.Error("secure client unexpectedly skips TLS verification")
	}
	if insecureTransport.TLSClientConfig == nil || !insecureTransport.TLSClientConfig.InsecureSkipVerify {
		t.Error("insecure client does not skip TLS verification")
	}
}

func TestMakeRequestPropagatesContextCancellation(t *testing.T) {
	oc, err := NewOchamiClient("test", "https://example.com")
	if err != nil {
		t.Fatal(err)
	}
	oc.Client = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		<-req.Context().Done()
		return nil, req.Context().Err()
	})}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := oc.MakeRequest(ctx, http.MethodGet, "https://example.com", nil, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("MakeRequest() error = %v, want context.Canceled", err)
	}
}

func TestMakeRequestPreservesCallerDeadline(t *testing.T) {
	oc, err := NewOchamiClient("test", "https://example.com")
	if err != nil {
		t.Fatal(err)
	}
	oc.Client = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		<-req.Context().Done()
		return nil, req.Context().Err()
	})}
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	if _, err := oc.MakeRequest(ctx, http.MethodGet, "https://example.com", nil, nil); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("MakeRequest() error = %v, want context.DeadlineExceeded", err)
	}
}

func TestPayloadInputFailures(t *testing.T) {
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

func TestUseCACertFileFailures(t *testing.T) {
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
