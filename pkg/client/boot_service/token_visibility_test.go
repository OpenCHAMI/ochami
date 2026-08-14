// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package boot_service

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/openchami/ochami/pkg/client"
	"github.com/openchami/ochami/pkg/format"
)

func TestNewClientPropagatesShowToken(t *testing.T) {
	const token = "eyJhbGciOiJIUzI1NiJ9.payload.sig"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("[]"))
	}))
	defer srv.Close()

	tests := []struct {
		name        string
		showToken   bool
		wantToken   string
		rejectToken string
	}{
		{
			name:        "token truncated by default",
			showToken:   false,
			wantToken:   "eyJhbG...",
			rejectToken: token,
		},
		{
			name:      "full token shown when enabled",
			showToken: true,
			wantToken: token,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logs bytes.Buffer
			logger := zerolog.New(&logs).Level(zerolog.DebugLevel)

			c, err := NewClient(
				srv.URL,
				5*time.Second,
				"",
				logger,
				client.WithShowToken(tt.showToken),
			)
			if err != nil {
				t.Fatalf("failed to create client: %v", err)
			}

			if _, err := c.ListNodes(token, format.DataFormatJson); err != nil {
				t.Fatalf("failed to list nodes: %v", err)
			}

			got := logs.String()
			if !strings.Contains(got, tt.wantToken) {
				t.Errorf("debug logs did not contain expected token value %q: %s", tt.wantToken, got)
			}
			if tt.rejectToken != "" && strings.Contains(got, tt.rejectToken) {
				t.Errorf("debug logs contained full token when it should be truncated: %s", got)
			}
		})
	}
}
