// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package boot_service

// boot_service_boundary_test.go covers UIDs at the edge of what client.CheckPathSegment
// accepts. The Fabrica-generated client builds each path with fmt.Sprintf and
// does not escape the UID, and how it then treats the path varies by Fabrica
// version, so these tests fail if a library upgrade lets such a UID out of its
// path segment.

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/openchami/ochami/pkg/client"
)

// TestBootHelpers_UIDStaysInSegment verifies that a UID that passes
// client.CheckPathSegment arrives as a single path segment below the collection
// endpoint with no query, for both request paths that the generated client has.
func TestBootHelpers_UIDStaysInSegment(t *testing.T) {
	for _, uid := range []string{"...", "a b", "a%20b", "a+b", "a:b", "a%25b"} {
		t.Run(uid, func(t *testing.T) {
			uris := make(chan string, 2)
			c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				uris <- r.RequestURI
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte("{}"))
			})
			defer srv.Close()

			if results := c.DeleteNodes(context.Background(), "", []string{uid}); results.HasErrors() {
				t.Errorf("DeleteNodes() errors = %v, want none", results.Errors())
			}
			if _, err := c.PatchNode(context.Background(), "", client.PatchMethodRFC7386, uid, map[string]any{}); err != nil {
				t.Errorf("PatchNode() error = %v, want nil", err)
			}

			for range 2 {
				select {
				case got := <-uris:
					segment, ok := strings.CutPrefix(got, "/nodes/")
					if !ok || segment == "" || strings.ContainsAny(segment, "/?#") {
						t.Errorf("request URI = %q, want one segment below /nodes/", got)
					}
				default:
					t.Fatal("request not received")
				}
			}
		})
	}
}
