// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package metadata_service

import (
	"io"
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name        string
		baseURI     string
		insecure    bool
		timeout     time.Duration
		apiVersion  string
		logger      zerolog.Logger
		wantErr     bool
		errContains string // substring to check in error message
	}{
		{
			name:       "valid URI with all parameters",
			baseURI:    "https://example.com",
			insecure:   false,
			timeout:    30 * time.Second,
			apiVersion: "v1beta2",
			logger:     zerolog.New(os.Stderr),
			wantErr:    false,
		},
		{
			name:       "valid URI with empty API version",
			baseURI:    "https://example.com",
			insecure:   false,
			timeout:    30 * time.Second,
			apiVersion: "",
			logger:     zerolog.New(os.Stderr),
			wantErr:    false,
		},
		{
			name:       "insecure mode enabled",
			baseURI:    "https://example.com",
			insecure:   true,
			timeout:    30 * time.Second,
			apiVersion: "",
			logger:     zerolog.New(os.Stderr),
			wantErr:    false,
		},
		{
			name:       "zero timeout",
			baseURI:    "https://example.com",
			insecure:   false,
			timeout:    0,
			apiVersion: "",
			logger:     zerolog.New(os.Stderr),
			wantErr:    false,
		},
		{
			name:       "large timeout",
			baseURI:    "https://example.com",
			insecure:   false,
			timeout:    24 * time.Hour,
			apiVersion: "",
			logger:     zerolog.New(os.Stderr),
			wantErr:    false,
		},
		{
			name:       "maximum duration timeout",
			baseURI:    "https://example.com",
			insecure:   false,
			timeout:    time.Duration(math.MaxInt64),
			apiVersion: "",
			logger:     zerolog.New(os.Stderr),
			wantErr:    false,
		},
		{
			name:       "negative timeout",
			baseURI:    "https://example.com",
			insecure:   false,
			timeout:    -1 * time.Second,
			apiVersion: "",
			logger:     zerolog.New(os.Stderr),
			wantErr:    false,
		},
		{
			name:       "URI with trailing slash",
			baseURI:    "https://example.com/",
			insecure:   false,
			timeout:    30 * time.Second,
			apiVersion: "",
			logger:     zerolog.New(os.Stderr),
			wantErr:    false,
		},
		{
			name:       "URI with path component",
			baseURI:    "https://example.com/metadata/v1",
			insecure:   false,
			timeout:    30 * time.Second,
			apiVersion: "",
			logger:     zerolog.New(os.Stderr),
			wantErr:    false,
		},
		{
			name:       "URI with query parameters",
			baseURI:    "https://example.com/path?query=value&foo=bar",
			insecure:   false,
			timeout:    30 * time.Second,
			apiVersion: "",
			logger:     zerolog.New(os.Stderr),
			wantErr:    false,
		},
		{
			name:       "localhost URI",
			baseURI:    "http://localhost:8080",
			insecure:   false,
			timeout:    30 * time.Second,
			apiVersion: "",
			logger:     zerolog.New(os.Stderr),
			wantErr:    false,
		},
		{
			name:       "IPv4 address URI",
			baseURI:    "http://192.168.1.1:8080",
			insecure:   false,
			timeout:    30 * time.Second,
			apiVersion: "",
			logger:     zerolog.New(os.Stderr),
			wantErr:    false,
		},
		{
			name:       "IPv6 address URI",
			baseURI:    "http://[::1]:8080",
			insecure:   false,
			timeout:    30 * time.Second,
			apiVersion: "",
			logger:     zerolog.New(os.Stderr),
			wantErr:    false,
		},
		{
			name:       "empty base URI",
			baseURI:    "",
			insecure:   false,
			timeout:    30 * time.Second,
			apiVersion: "",
			logger:     zerolog.New(os.Stderr),
			wantErr:    false, // url.Parse("") succeeds, returns empty URL
		},
		{
			name:       "whitespace only base URI",
			baseURI:    "   ",
			insecure:   false,
			timeout:    30 * time.Second,
			apiVersion: "",
			logger:     zerolog.New(os.Stderr),
			wantErr:    false, // url.Parse accepts whitespace
		},
		{
			name:        "invalid URI format - missing scheme",
			baseURI:     "://invalid",
			insecure:    false,
			timeout:     30 * time.Second,
			apiVersion:  "",
			logger:      zerolog.New(os.Stderr),
			wantErr:     true,
			errContains: "failed to create OchamiClient",
		},
		{
			name:       "malformed URI with spaces",
			baseURI:    "http://example .com",
			insecure:   false,
			timeout:    30 * time.Second,
			apiVersion: "",
			logger:     zerolog.New(os.Stderr),
			wantErr:    true,
		},
		{
			name:       "invalid URI - only path",
			baseURI:    "/just/a/path",
			insecure:   false,
			timeout:    30 * time.Second,
			apiVersion: "",
			logger:     zerolog.New(os.Stderr),
			wantErr:    false, // url.Parse accepts relative paths
		},
		{
			name:       "invalid scheme - ftp",
			baseURI:    "ftp://example.com",
			insecure:   false,
			timeout:    30 * time.Second,
			apiVersion: "",
			logger:     zerolog.New(os.Stderr),
			wantErr:    false, // url.Parse accepts any scheme
		},
		{
			name:       "extremely long URI",
			baseURI:    "https://example.com/" + strings.Repeat("a", 10000),
			insecure:   false,
			timeout:    30 * time.Second,
			apiVersion: "",
			logger:     zerolog.New(os.Stderr),
			wantErr:    false,
		},
		{
			name:       "API version with whitespace",
			baseURI:    "https://example.com",
			insecure:   false,
			timeout:    30 * time.Second,
			apiVersion: "  ",
			logger:     zerolog.New(os.Stderr),
			wantErr:    false,
		},
		{
			name:       "API version - invalid format",
			baseURI:    "https://example.com",
			insecure:   false,
			timeout:    30 * time.Second,
			apiVersion: "not-a-version",
			logger:     zerolog.New(os.Stderr),
			wantErr:    false, // API version validation is not enforced by client creation
		},
		{
			name:       "API version - very long string",
			baseURI:    "https://example.com",
			insecure:   false,
			timeout:    30 * time.Second,
			apiVersion: strings.Repeat("v", 1000),
			logger:     zerolog.New(os.Stderr),
			wantErr:    false,
		},
		{
			name:       "zero-value logger",
			baseURI:    "https://example.com",
			insecure:   false,
			timeout:    30 * time.Second,
			apiVersion: "",
			logger:     zerolog.Logger{}, // Zero value
			wantErr:    false,
		},
		{
			name:       "logger with disabled output",
			baseURI:    "https://example.com",
			insecure:   false,
			timeout:    30 * time.Second,
			apiVersion: "",
			logger:     zerolog.New(io.Discard),
			wantErr:    false,
		},
		{
			name:       "logger with debug level",
			baseURI:    "https://example.com",
			insecure:   false,
			timeout:    30 * time.Second,
			apiVersion: "",
			logger:     zerolog.New(os.Stderr).Level(zerolog.DebugLevel),
			wantErr:    false,
		},
		{
			name:       "logger with trace level",
			baseURI:    "https://example.com",
			insecure:   false,
			timeout:    30 * time.Second,
			apiVersion: "",
			logger:     zerolog.New(os.Stderr).Level(zerolog.TraceLevel),
			wantErr:    false,
		},
		{
			name:       "logger with error level",
			baseURI:    "https://example.com",
			insecure:   false,
			timeout:    30 * time.Second,
			apiVersion: "",
			logger:     zerolog.New(os.Stderr).Level(zerolog.ErrorLevel),
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		// Create per-iteration copy of test tt so that running
		// tests in parallel does not reuse the same test for
		// each run.
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			// Call NewClient
			client, err := NewClient(tc.baseURI, tc.insecure, tc.timeout, tc.apiVersion, tc.logger)

			// Check error expectation
			if (err != nil) != tc.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tc.wantErr)
				return
			}

			// If error expected, validate error message
			if tc.wantErr {
				if tc.errContains != "" && !strings.Contains(err.Error(), tc.errContains) {
					t.Errorf("error = %q, want substring %q", err.Error(), tc.errContains)
				}
				return
			}

			// Success case validations
			if client == nil {
				t.Fatal("NewClient() returned nil client without error")
			}

			// Verify OchamiClient is created and non-nil
			if client.OchamiClient == nil {
				t.Error("OchamiClient is nil")
			} else {
				// Verify OchamiClient has expected fields
				if client.OchamiClient.BaseURI == nil {
					t.Error("OchamiClient.BaseURI is nil")
				}
				if client.OchamiClient.ServiceName != serviceNameMetadataService {
					t.Errorf("OchamiClient.ServiceName = %q, want %q", client.OchamiClient.ServiceName, serviceNameMetadataService)
				}
				if client.OchamiClient.Client == nil {
					t.Error("OchamiClient.Client (http.Client) is nil")
				}
			}

			// Verify metadata-service Client is created and non-nil
			if client.Client == nil {
				t.Error("Client (metadata_service_client.Client) is nil")
			}

			// Verify Timeout matches input
			if client.Timeout != tc.timeout {
				t.Errorf("Timeout = %v, want %v", client.Timeout, tc.timeout)
			}

			// Additional validation for API version (if set)
			// Note: We can't directly inspect if WithVersion was called,
			// but we verify the client was created successfully with non-nil Client
			if tc.apiVersion != "" && client.Client == nil {
				t.Error("API version was set but Client is nil")
			}
		})
	}
}
