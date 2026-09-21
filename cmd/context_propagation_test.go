// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// context_propagation_test.go verifies that a context set on the root Cobra
// command is wired through to the HTTP request layer. Context values and
// deadlines are local to the client and never cross the wire, so they can't
// be observed from the server side of an httptest round trip; cancellation
// is the one property that's both client-local and externally observable,
// since a cancelled context must cause the command to fail before the
// request completes against a live server. Deterministic coverage of
// cancellation/deadline handling within the client's request layer itself
// lives in pkg/client/request_layer_test.go; this test only proves the
// wiring between the command layer and the client is intact.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestContextPropagation_CancellationWiring verifies a context cancelled
// before command execution prevents the request from completing
// successfully, proving cancellation is wired through to the client.
func TestContextPropagation_CancellationWiring(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"smd", "component", "get", "--ignore-config", "--uri", srv.URL})
	if err := rootCmd.ExecuteContext(ctx); err == nil {
		t.Error("expected an error executing with an already-cancelled context, got nil")
	}
}
