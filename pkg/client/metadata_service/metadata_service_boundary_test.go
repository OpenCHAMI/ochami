// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package metadata_service

// metadata_service_boundary_test.go covers UIDs at the edge of what client.CheckPathSegment
// accepts. The Fabrica-generated client builds each path with fmt.Sprintf and
// does no escaping of its own, so these tests fail if a library upgrade stops
// keeping such a UID inside its path segment.

import (
	"context"
	"net/http"
	"testing"

	"github.com/openchami/ochami/pkg/client"
)

// TestMetadataHelpers_EscapesUID verifies that a UID containing URL
// metacharacters stays in a single escaped path segment, rather than starting a
// query or fragment that would leave the request on the collection endpoint.
func TestMetadataHelpers_EscapesUID(t *testing.T) {
	tests := []struct {
		uid     string
		wantURI string
	}{
		{"?", "/wireguardpeers/%3F"},
		{"#", "/wireguardpeers/%23"},
		{"a?force=true", "/wireguardpeers/a%3Fforce=true"},
		{"a%2Fb", "/wireguardpeers/a%252Fb"},
		{"%2e%2e", "/wireguardpeers/%252e%252e"},
	}
	for _, tc := range tests {
		t.Run(tc.uid, func(t *testing.T) {
			uris := make(chan string, 2)
			c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				uris <- r.Method + " " + r.RequestURI
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte("{}"))
			})
			defer srv.Close()

			if results := c.DeleteWireGuardPeers(context.Background(), "", []string{tc.uid}); results.HasErrors() {
				t.Errorf("DeleteWireGuardPeers() errors = %v, want none", results.Errors())
			}
			if _, err := c.PatchWireGuardPeer(context.Background(), "", client.PatchMethodRFC7386, tc.uid, map[string]any{}); err != nil {
				t.Errorf("PatchWireGuardPeer() error = %v, want nil", err)
			}

			for _, method := range []string{http.MethodDelete, http.MethodPatch} {
				want := method + " " + tc.wantURI
				select {
				case got := <-uris:
					if got != want {
						t.Errorf("request = %q, want %q", got, want)
					}
				default:
					t.Fatalf("no request received, want %q", want)
				}
			}
		})
	}
}
