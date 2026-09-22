// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package client

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestFabricaWrapHTTPError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		// wantWrapped is true if err matches the Fabrica HTTP-error message
		// shape and FabricaWrapHTTPError should add a new
		// UnsuccessfulHTTPError wrapping around it.
		wantWrapped bool
	}{
		{
			name:        "nil error",
			err:         nil,
			wantWrapped: false,
		},
		{
			name:        "HTTP error shape",
			err:         fmt.Errorf("HTTP error %d: %s", 404, "not found"),
			wantWrapped: true,
		},
		{
			name:        "API error shape",
			err:         fmt.Errorf("API error (%d): %s", 500, "boom"),
			wantWrapped: true,
		},
		{
			name:        "PATCH HTTP error shape",
			err:         fmt.Errorf("PATCH HTTP error %d: %s", 400, "bad"),
			wantWrapped: true,
		},
		{
			name:        "PATCH API error shape",
			err:         fmt.Errorf("PATCH API error (%d): %s", 409, "conflict"),
			wantWrapped: true,
		},
		{
			name:        "multi-digit status code",
			err:         fmt.Errorf("HTTP error %d: %s", 12345, "weird status"),
			wantWrapped: true,
		},
		{
			name:        "empty body after status code",
			err:         fmt.Errorf("HTTP error %d: %s", 404, ""),
			wantWrapped: true,
		},
		{
			name:        "transport/network error",
			err:         fmt.Errorf("request failed: %w", errors.New("dial tcp: connection refused")),
			wantWrapped: false,
		},
		{
			name:        "unrelated error",
			err:         errors.New("failed to convert data to JSON"),
			wantWrapped: false,
		},
		{
			name:        "shape appears mid-message, not at start",
			err:         fmt.Errorf("wrapped: HTTP error %d: %s", 404, "not found"),
			wantWrapped: false,
		},
		{
			// The message shape doesn't match (no status-code prefix), so
			// this shouldn't be double-wrapped, but errors.Is must still
			// report true because the sentinel is already in its chain.
			name:        "already wraps the sentinel via a different path",
			err:         fmt.Errorf("outer: %w", UnsuccessfulHTTPError),
			wantWrapped: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FabricaWrapHTTPError(tt.err)

			if tt.err == nil {
				if got != nil {
					t.Fatalf("FabricaWrapHTTPError(nil) = %v, want nil", got)
				}
				return
			}

			// Whether or not a new wrapping was added, errors.Is must be
			// true if the sentinel is anywhere in the resulting chain.
			wantIs := tt.wantWrapped || errors.Is(tt.err, UnsuccessfulHTTPError)
			if gotIs := errors.Is(got, UnsuccessfulHTTPError); gotIs != wantIs {
				t.Errorf("errors.Is(FabricaWrapHTTPError(%q), UnsuccessfulHTTPError) = %v, want %v", tt.err, gotIs, wantIs)
			}

			if tt.wantWrapped {
				// The original error's message must still be present so no
				// diagnostic information is lost by wrapping.
				if !strings.Contains(got.Error(), tt.err.Error()) {
					t.Errorf("FabricaWrapHTTPError(%q) = %q, want it to contain the original message", tt.err, got.Error())
				}
			} else {
				// A non-matching error must be returned unchanged (same
				// value), not wrapped or altered, even if it already
				// satisfies errors.Is via some other path.
				if got != tt.err {
					t.Errorf("FabricaWrapHTTPError(%q) = %v, want the original error returned unchanged", tt.err, got)
				}
			}
		})
	}
}
