// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package client

import (
	"context"
	"errors"
	"io"
	"net/http"
	"testing"
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

// TestOchamiClient_IndependentTransports verifies that each OchamiClient gets
// its own HTTP client and transport, so an insecure client doesn't turn off TLS
// verification for another.
func TestOchamiClient_IndependentTransports(t *testing.T) {
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

// TestMakeRequest_PropagatesContextCancellation verifies that MakeRequest sends
// the request with the caller's context, so a canceled context fails the
// request with context.Canceled.
func TestMakeRequest_PropagatesContextCancellation(t *testing.T) {
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
