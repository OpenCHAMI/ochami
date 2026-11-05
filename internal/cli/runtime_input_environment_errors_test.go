// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cli

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

// TestRuntimeHandlePayload_StdinCancelled verifies that cancelling the
// command's context returns a payload read blocked on standard input.
func TestRuntimeHandlePayload_StdinCancelled(t *testing.T) {
	t.Parallel()

	rt := NewTestRuntime(gatedReader{make(chan string)}, io.Discard, io.Discard)
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(10*time.Millisecond, cancel)

	tests := []struct {
		name string
		read func(cmd *cobra.Command) error
	}{
		{name: "data @-", read: func(cmd *cobra.Command) error {
			var v map[string]any
			return rt.HandlePayload(cmd, &v)
		}},
		{name: "stdin", read: func(cmd *cobra.Command) error {
			var v map[string]any
			return rt.HandlePayloadStdin(cmd, &v)
		}},
		{name: "slice data @-", read: func(cmd *cobra.Command) error {
			var v []map[string]any
			return HandlePayloadSlice(rt, cmd, &v)
		}},
		{name: "slice stdin", read: func(cmd *cobra.Command) error {
			var v []map[string]any
			return HandlePayloadStdinSlice(rt, cmd, &v)
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.Flags().String("data", "", "")
			if err := cmd.Flags().Set("data", "@-"); err != nil {
				t.Fatalf("set data flag: %v", err)
			}
			cmd.SetContext(ctx)

			done := make(chan error, 1)
			go func() { done <- tc.read(cmd) }()
			select {
			case err := <-done:
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("error = %v, want context.Canceled", err)
				}
			case <-time.After(cancelDeadline):
				t.Fatal("payload read did not return after its context was cancelled")
			}
		})
	}
}
