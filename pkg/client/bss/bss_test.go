// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package bss

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/openchami/bss/pkg/bssTypes"

	"github.com/openchami/ochami/pkg/client"
)

// recordedRequest holds the parts of an incoming request that tests assert on.
type recordedRequest struct {
	Method   string
	Path     string
	RawQuery string
	Auth     string
	Body     []byte
}

// newTestClient starts an httptest server that records the last request it
// received and responds with status and respBody. It returns a BSSClient
// pointed at the server and a pointer to the recorded request.
func newTestClient(t *testing.T, status int, respBody string) (*BSSClient, *recordedRequest) {
	t.Helper()
	rec := &recordedRequest{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.Method = r.Method
		rec.Path = r.URL.Path
		rec.RawQuery = r.URL.RawQuery
		rec.Auth = r.Header.Get("Authorization")
		rec.Body, _ = io.ReadAll(r.Body)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(respBody))
	}))
	t.Cleanup(srv.Close)

	bc, err := NewClient(srv.URL)
	if err != nil {
		t.Fatalf("NewClient(%q) returned unexpected error: %v", srv.URL, err)
	}
	return bc, rec
}

// newUnreachableClient returns a BSSClient whose base URI points at a server
// that has already been shut down, so every request fails at the transport
// level.
func newUnreachableClient(t *testing.T) *BSSClient {
	t.Helper()
	srv := httptest.NewServer(http.NotFoundHandler())
	uri := srv.URL
	srv.Close()

	bc, err := NewClient(uri)
	if err != nil {
		t.Fatalf("NewClient(%q) returned unexpected error: %v", uri, err)
	}
	return bc
}

func TestNewClient(t *testing.T) {
	bc, err := NewClient("https://example.com/boot/v1", client.WithInsecure(true))
	if err != nil {
		t.Fatalf("NewClient() returned unexpected error: %v", err)
	}
	if bc.OchamiClient == nil {
		t.Fatal("NewClient() returned BSSClient with nil OchamiClient")
	}
	if bc.ServiceName != serviceNameBSS {
		t.Errorf("ServiceName = %q, want %q", bc.ServiceName, serviceNameBSS)
	}
	if got, want := bc.BaseURI.String(), "https://example.com/boot/v1"; got != want {
		t.Errorf("BaseURI = %q, want %q", got, want)
	}
}

func TestNewClient_InvalidURI(t *testing.T) {
	bc, err := NewClient("://bad-uri")
	if err == nil {
		t.Fatal("NewClient() with invalid URI returned nil error")
	}
	if bc != nil {
		t.Errorf("NewClient() with invalid URI returned non-nil client: %+v", bc)
	}
	if !strings.Contains(err.Error(), "failed to create OchamiClient for BSS") {
		t.Errorf("error %q does not mention OchamiClient creation failure", err)
	}
}

// bootParamsMethod describes one of the BSSClient methods that sends a
// bssTypes.BootParams body to /bootparameters.
type bootParamsMethod struct {
	name       string
	httpMethod string
	call       func(*BSSClient, bssTypes.BootParams, string) (client.HTTPEnvelope, error)
	errPrefix  string
}

var bootParamsMethods = []bootParamsMethod{
	{"PostBootParams", http.MethodPost, (*BSSClient).PostBootParams, "PostBootParams(): failed to POST"},
	{"PutBootParams", http.MethodPut, (*BSSClient).PutBootParams, "PutBootParams(): failed to PUT"},
	{"PatchBootParams", http.MethodPatch, (*BSSClient).PatchBootParams, "PatchBootParams(): failed to PATCH"},
	{"DeleteBootParams", http.MethodDelete, (*BSSClient).DeleteBootParams, "DeleteBootParams(): failed to DELETE"},
}

func TestBootParamsMethods_Success(t *testing.T) {
	bp := bssTypes.BootParams{
		Hosts:  []string{"x1000c0s0b0n0"},
		Macs:   []string{"00:11:22:33:44:55"},
		Nids:   []int32{1},
		Params: "console=ttyS0",
		Kernel: "http://example.com/kernel",
		Initrd: "http://example.com/initrd",
	}

	for _, m := range bootParamsMethods {
		t.Run(m.name, func(t *testing.T) {
			bc, rec := newTestClient(t, http.StatusOK, `{"ok":true}`)

			henv, err := m.call(bc, bp, "my-token")
			if err != nil {
				t.Fatalf("%s() returned unexpected error: %v", m.name, err)
			}
			if rec.Method != m.httpMethod {
				t.Errorf("method = %q, want %q", rec.Method, m.httpMethod)
			}
			if rec.Path != BSSRelpathBootParams {
				t.Errorf("path = %q, want %q", rec.Path, BSSRelpathBootParams)
			}
			if rec.Auth != "Bearer my-token" {
				t.Errorf("Authorization = %q, want %q", rec.Auth, "Bearer my-token")
			}

			var gotBP bssTypes.BootParams
			if err := json.Unmarshal(rec.Body, &gotBP); err != nil {
				t.Fatalf("request body is not valid BootParams JSON: %v (body: %s)", err, rec.Body)
			}
			if !reflect.DeepEqual(gotBP.Hosts, bp.Hosts) ||
				!reflect.DeepEqual(gotBP.Macs, bp.Macs) ||
				!reflect.DeepEqual(gotBP.Nids, bp.Nids) ||
				gotBP.Params != bp.Params ||
				gotBP.Kernel != bp.Kernel ||
				gotBP.Initrd != bp.Initrd {
				t.Errorf("request body = %+v, want %+v", gotBP, bp)
			}

			if henv.StatusCode != http.StatusOK {
				t.Errorf("StatusCode = %d, want %d", henv.StatusCode, http.StatusOK)
			}
			if string(henv.Body) != `{"ok":true}` {
				t.Errorf("Body = %q, want %q", henv.Body, `{"ok":true}`)
			}
		})
	}
}

func TestBootParamsMethods_NoToken(t *testing.T) {
	for _, m := range bootParamsMethods {
		t.Run(m.name, func(t *testing.T) {
			bc, rec := newTestClient(t, http.StatusOK, "")

			if _, err := m.call(bc, bssTypes.BootParams{}, ""); err != nil {
				t.Fatalf("%s() returned unexpected error: %v", m.name, err)
			}
			if rec.Auth != "" {
				t.Errorf("Authorization header = %q, want it to be unset", rec.Auth)
			}
		})
	}
}

func TestBootParamsMethods_MarshalError(t *testing.T) {
	// A func value inside the free-form cloud-init data cannot be marshaled
	// to JSON, so the request must fail before anything is sent.
	bp := bssTypes.BootParams{
		CloudInit: bssTypes.CloudInit{
			MetaData: bssTypes.CloudDataType{"bad": func() {}},
		},
	}

	for _, m := range bootParamsMethods {
		t.Run(m.name, func(t *testing.T) {
			bc, rec := newTestClient(t, http.StatusOK, "")

			_, err := m.call(bc, bp, "")
			if err == nil {
				t.Fatalf("%s() with unmarshalable BootParams returned nil error", m.name)
			}
			if want := m.name + "(): failed to marshal BootParams"; !strings.Contains(err.Error(), want) {
				t.Errorf("error %q does not contain %q", err, want)
			}
			if rec.Method != "" {
				t.Errorf("request was sent (%s %s) despite marshal error", rec.Method, rec.Path)
			}
		})
	}
}

func TestBootParamsMethods_HTTPError(t *testing.T) {
	for _, m := range bootParamsMethods {
		t.Run(m.name, func(t *testing.T) {
			bc, _ := newTestClient(t, http.StatusBadRequest, "bad request")

			henv, err := m.call(bc, bssTypes.BootParams{}, "")
			if err == nil {
				t.Fatalf("%s() with 400 response returned nil error", m.name)
			}
			if !errors.Is(err, client.UnsuccessfulHTTPError) {
				t.Errorf("error %q does not wrap client.UnsuccessfulHTTPError", err)
			}
			if !strings.Contains(err.Error(), m.errPrefix) {
				t.Errorf("error %q does not contain %q", err, m.errPrefix)
			}
			if henv.StatusCode != http.StatusBadRequest {
				t.Errorf("StatusCode = %d, want %d", henv.StatusCode, http.StatusBadRequest)
			}
		})
	}
}

func TestBootParamsMethods_TransportError(t *testing.T) {
	for _, m := range bootParamsMethods {
		t.Run(m.name, func(t *testing.T) {
			bc := newUnreachableClient(t)

			_, err := m.call(bc, bssTypes.BootParams{}, "")
			if err == nil {
				t.Fatalf("%s() against unreachable server returned nil error", m.name)
			}
			if errors.Is(err, client.UnsuccessfulHTTPError) {
				t.Errorf("transport error %q unexpectedly wraps client.UnsuccessfulHTTPError", err)
			}
			if !strings.Contains(err.Error(), m.errPrefix) {
				t.Errorf("error %q does not contain %q", err, m.errPrefix)
			}
		})
	}
}

func TestGetBootParams(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		token     string
		wantQuery string
		wantAuth  string
	}{
		{name: "no query, no token"},
		{name: "query and token", query: "name=x1000c0s0b0n0&mac=00:11:22:33:44:55", token: "tok", wantQuery: "name=x1000c0s0b0n0&mac=00:11:22:33:44:55", wantAuth: "Bearer tok"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bc, rec := newTestClient(t, http.StatusOK, `[]`)

			henv, err := bc.GetBootParams(tt.query, tt.token)
			if err != nil {
				t.Fatalf("GetBootParams() returned unexpected error: %v", err)
			}
			if rec.Method != http.MethodGet {
				t.Errorf("method = %q, want GET", rec.Method)
			}
			if rec.Path != BSSRelpathBootParams {
				t.Errorf("path = %q, want %q", rec.Path, BSSRelpathBootParams)
			}
			if rec.RawQuery != tt.wantQuery {
				t.Errorf("query = %q, want %q", rec.RawQuery, tt.wantQuery)
			}
			if rec.Auth != tt.wantAuth {
				t.Errorf("Authorization = %q, want %q", rec.Auth, tt.wantAuth)
			}
			if string(henv.Body) != `[]` {
				t.Errorf("Body = %q, want %q", henv.Body, `[]`)
			}
		})
	}
}

func TestGetBootParams_Error(t *testing.T) {
	bc, _ := newTestClient(t, http.StatusNotFound, "not found")

	_, err := bc.GetBootParams("", "")
	if err == nil {
		t.Fatal("GetBootParams() with 404 response returned nil error")
	}
	if !errors.Is(err, client.UnsuccessfulHTTPError) {
		t.Errorf("error %q does not wrap client.UnsuccessfulHTTPError", err)
	}
	if !strings.Contains(err.Error(), "GetBootParams(): error getting boot parameters") {
		t.Errorf("error %q is missing GetBootParams context", err)
	}
}

// TestSimpleGetters covers the BSSClient GET wrappers that take at most a query
// string and send no authorization header.
func TestSimpleGetters(t *testing.T) {
	tests := []struct {
		name      string
		call      func(*BSSClient) (client.HTTPEnvelope, error)
		wantPath  string
		wantQuery string
		errPrefix string
	}{
		{
			name:      "GetBootScript",
			call:      func(bc *BSSClient) (client.HTTPEnvelope, error) { return bc.GetBootScript("mac=00:11:22:33:44:55") },
			wantPath:  BSSRelpathBootScript,
			wantQuery: "mac=00:11:22:33:44:55",
			errPrefix: "GetBootScript(): error getting boot script",
		},
		{
			name:      "GetDumpstate",
			call:      (*BSSClient).GetDumpstate,
			wantPath:  BSSRelpathDumpstate,
			errPrefix: "GetDumpstate(): error getting dump state",
		},
		{
			name:      "GetEndpointHistory",
			call:      func(bc *BSSClient) (client.HTTPEnvelope, error) { return bc.GetEndpointHistory("name=x1000c0s0b0n0") },
			wantPath:  BSSRelpathEndpointHistory,
			wantQuery: "name=x1000c0s0b0n0",
			errPrefix: "GetEndpointHistory(): error getting endpoint history",
		},
		{
			name:      "GetHosts",
			call:      func(bc *BSSClient) (client.HTTPEnvelope, error) { return bc.GetHosts("nid=1") },
			wantPath:  BSSRelpathHosts,
			wantQuery: "nid=1",
			errPrefix: "GetHosts(): error getting hosts",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name+"/success", func(t *testing.T) {
			bc, rec := newTestClient(t, http.StatusOK, "ok")

			henv, err := tt.call(bc)
			if err != nil {
				t.Fatalf("%s() returned unexpected error: %v", tt.name, err)
			}
			if rec.Method != http.MethodGet {
				t.Errorf("method = %q, want GET", rec.Method)
			}
			if rec.Path != tt.wantPath {
				t.Errorf("path = %q, want %q", rec.Path, tt.wantPath)
			}
			if rec.RawQuery != tt.wantQuery {
				t.Errorf("query = %q, want %q", rec.RawQuery, tt.wantQuery)
			}
			if rec.Auth != "" {
				t.Errorf("Authorization = %q, want it to be unset", rec.Auth)
			}
			if string(henv.Body) != "ok" {
				t.Errorf("Body = %q, want %q", henv.Body, "ok")
			}
		})
		t.Run(tt.name+"/error", func(t *testing.T) {
			bc, _ := newTestClient(t, http.StatusInternalServerError, "boom")

			_, err := tt.call(bc)
			if err == nil {
				t.Fatalf("%s() with 500 response returned nil error", tt.name)
			}
			if !errors.Is(err, client.UnsuccessfulHTTPError) {
				t.Errorf("error %q does not wrap client.UnsuccessfulHTTPError", err)
			}
			if !strings.Contains(err.Error(), tt.errPrefix) {
				t.Errorf("error %q does not contain %q", err, tt.errPrefix)
			}
		})
	}
}

func TestGetStatus(t *testing.T) {
	tests := []struct {
		component string
		wantPath  string
	}{
		{component: "", wantPath: "/service/status"},
		{component: "all", wantPath: "/service/status/all"},
		{component: "storage", wantPath: "/service/storage/status"},
		{component: "smd", wantPath: "/service/hsm"},
		{component: "version", wantPath: "/service/version"},
	}
	for _, tt := range tests {
		t.Run("component="+tt.component, func(t *testing.T) {
			bc, rec := newTestClient(t, http.StatusOK, `{"bss-status":"running"}`)

			henv, err := bc.GetStatus(tt.component)
			if err != nil {
				t.Fatalf("GetStatus(%q) returned unexpected error: %v", tt.component, err)
			}
			if rec.Method != http.MethodGet {
				t.Errorf("method = %q, want GET", rec.Method)
			}
			if rec.Path != tt.wantPath {
				t.Errorf("path = %q, want %q", rec.Path, tt.wantPath)
			}
			if henv.StatusCode != http.StatusOK {
				t.Errorf("StatusCode = %d, want %d", henv.StatusCode, http.StatusOK)
			}
		})
	}
}

func TestGetStatus_UnknownComponent(t *testing.T) {
	bc, rec := newTestClient(t, http.StatusOK, "")

	_, err := bc.GetStatus("bogus")
	if err == nil {
		t.Fatal("GetStatus() with unknown component returned nil error")
	}
	if want := "unknown status component: bogus"; !strings.Contains(err.Error(), want) {
		t.Errorf("error %q does not contain %q", err, want)
	}
	if rec.Method != "" {
		t.Errorf("request was sent (%s %s) for unknown component", rec.Method, rec.Path)
	}
}

func TestGetStatus_HTTPError(t *testing.T) {
	bc, _ := newTestClient(t, http.StatusServiceUnavailable, "down")

	_, err := bc.GetStatus("all")
	if err == nil {
		t.Fatal("GetStatus() with 503 response returned nil error")
	}
	if !errors.Is(err, client.UnsuccessfulHTTPError) {
		t.Errorf("error %q does not wrap client.UnsuccessfulHTTPError", err)
	}
	if !strings.Contains(err.Error(), "GetStatus():") {
		t.Errorf("error %q is missing GetStatus context", err)
	}
}
