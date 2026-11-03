// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cli

import (
	"context"
	"errors"
	"os"
	"syscall"
	"testing"
	"time"
)

// TestNotifyContext_SignalCancelsWithCause verifies that the first signal
// cancels the context with an InterruptError naming it, and then releases the
// signals so a second one gets its default behavior.
func TestNotifyContext_SignalCancelsWithCause(t *testing.T) {
	t.Parallel()

	for _, sig := range []os.Signal{os.Interrupt, syscall.SIGTERM} {
		t.Run(sig.String(), func(t *testing.T) {
			t.Parallel()

			signals := make(chan os.Signal, 1)
			released := make(chan struct{})
			ctx, stop := notifyContext(context.Background(), signals, func() { close(released) })
			defer stop()

			signals <- sig
			select {
			case <-released:
			case <-time.After(5 * time.Second):
				t.Fatal("signals were not released after the first signal")
			}
			var ie *InterruptError
			if !errors.As(context.Cause(ctx), &ie) || ie.Signal != sig {
				t.Fatalf("context.Cause() = %v, want InterruptError for %v", context.Cause(ctx), sig)
			}
			if !errors.Is(ctx.Err(), context.Canceled) {
				t.Errorf("ctx.Err() = %v, want context.Canceled", ctx.Err())
			}
		})
	}
}

// TestNotifyContext_StopReleasesSignals verifies that stop releases the
// signals once and cancels the context without an interruption cause.
func TestNotifyContext_StopReleasesSignals(t *testing.T) {
	t.Parallel()

	releases := make(chan struct{}, 2)
	ctx, stop := notifyContext(context.Background(), make(chan os.Signal), func() { releases <- struct{}{} })
	stop()
	stop()

	<-ctx.Done()
	var ie *InterruptError
	if cause := context.Cause(ctx); errors.As(cause, &ie) || !errors.Is(cause, context.Canceled) {
		t.Errorf("context.Cause() = %v, want plain context.Canceled", cause)
	}
	// The watcher goroutine also releases once the context is done; give it
	// time to run so a second release would be seen.
	time.Sleep(10 * time.Millisecond)
	if n := len(releases); n != 1 {
		t.Errorf("signals released %d times, want 1", n)
	}
}
