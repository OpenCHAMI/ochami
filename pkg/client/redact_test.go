// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package client

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/rs/zerolog"

	"github.com/openchami/ochami/internal/log"
)

func TestRedactToken(t *testing.T) {
	tests := []struct {
		name  string
		show  bool
		token string
		want  string
	}{
		{
			name:  "empty token",
			show:  false,
			token: "",
			want:  "",
		},
		{
			name:  "short token fully masked",
			show:  false,
			token: "abc",
			want:  "...",
		},
		{
			name:  "token exactly prefix length fully masked",
			show:  false,
			token: "abcdef",
			want:  "...",
		},
		{
			name:  "long token truncated with prefix and ellipsis",
			show:  false,
			token: "eyJhbGciOiJIUzI1NiJ9.payload.sig",
			want:  "eyJhbG...",
		},
		{
			name:  "show token returns full value",
			show:  true,
			token: "eyJhbGciOiJIUzI1NiJ9.payload.sig",
			want:  "eyJhbGciOiJIUzI1NiJ9.payload.sig",
		},
		{
			name:  "show token with empty stays empty",
			show:  true,
			token: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RedactToken(tt.token, tt.show)
			if got != tt.want {
				t.Errorf("RedactToken(%q, %v) = %q, want %q", tt.token, tt.show, got, tt.want)
			}
		})
	}
}

func TestRedactAuthHeaderValues(t *testing.T) {
	tests := []struct {
		name string
		show bool
		vals []string
		want []string
	}{
		{
			name: "bearer token truncated by default",
			show: false,
			vals: []string{"Bearer eyJhbGciOiJIUzI1NiJ9.payload.sig"},
			want: []string{"Bearer eyJhbG..."},
		},
		{
			name: "non-bearer value truncated wholesale",
			show: false,
			vals: []string{"eyJhbGciOiJIUzI1NiJ9.payload.sig"},
			want: []string{"eyJhbG..."},
		},
		{
			name: "multiple values",
			show: false,
			vals: []string{"Bearer eyJhbGciOiJIUzI1NiJ9.payload.sig", "Bearer another-long-token-value"},
			want: []string{"Bearer eyJhbG...", "Bearer anothe..."},
		},
		{
			name: "show token returns values unchanged",
			show: true,
			vals: []string{"Bearer eyJhbGciOiJIUzI1NiJ9.payload.sig"},
			want: []string{"Bearer eyJhbGciOiJIUzI1NiJ9.payload.sig"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := redactAuthHeaderValues(tt.vals, tt.show)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("redactAuthHeaderValues(%v, %v) = %v, want %v", tt.vals, tt.show, got, tt.want)
			}
		})
	}
}

func TestIsAuthorizationHeader(t *testing.T) {
	tests := []struct {
		key  string
		want bool
	}{
		{"Authorization", true},
		{"authorization", true},
		{"AUTHORIZATION", true},
		{"Content-Type", false},
		{"X-Auth", false},
	}
	for _, tt := range tests {
		if got := isAuthorizationHeader(tt.key); got != tt.want {
			t.Errorf("isAuthorizationHeader(%q) = %v, want %v", tt.key, got, tt.want)
		}
	}
}

// TestMakeRequestRedactsAuthorizationInLogs verifies that the Authorization
// header is truncated in debug logs by default (WithShowToken(false)) and shown
// in full when the client is created with WithShowToken(true).
func TestMakeRequestRedactsAuthorizationInLogs(t *testing.T) {
	origLogger := log.Logger
	defer func() { log.Logger = origLogger }()

	fullToken := "eyJhbGciOiJIUzI1NiJ9.payload.sig"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	run := func(showToken bool) string {
		var buf bytes.Buffer
		log.Logger = zerolog.New(&buf).Level(zerolog.DebugLevel)

		oc, err := NewOchamiClient("test", ts.URL, WithShowToken(showToken))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}
		headers := NewHTTPHeaders()
		if err := headers.SetAuthorization(fullToken); err != nil {
			t.Fatalf("failed to set authorization: %v", err)
		}
		res, err := oc.MakeRequest(http.MethodGet, ts.URL, headers, nil)
		if err != nil {
			t.Fatalf("MakeRequest failed: %v", err)
		}
		if res != nil && res.Body != nil {
			res.Body.Close()
		}
		return buf.String()
	}

	// Default (redacted).
	logs := run(false)
	if strings.Contains(logs, fullToken) {
		t.Errorf("logs contained full token by default, expected it to be redacted; logs: %s", logs)
	}
	if !strings.Contains(logs, "eyJhbG...") {
		t.Errorf("logs did not contain truncated token; logs: %s", logs)
	}

	// ShowToken enabled (full).
	logs = run(true)
	if !strings.Contains(logs, fullToken) {
		t.Errorf("logs did not contain full token with WithShowToken(true); logs: %s", logs)
	}
}
