// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

//go:build unix

package cli

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"
)

// These tests change the test process's own signal handling, so they don't
// run in parallel.

// TestNotifyContext_ProcessSignal verifies that a signal sent to the process
// cancels the context with an InterruptError naming it instead of
// terminating the process.
func TestNotifyContext_ProcessSignal(t *testing.T) {
	ctx, stop := NotifyContext(context.Background())
	defer stop()

	if err := syscall.Kill(os.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatalf("kill: %v", err)
	}
	select {
	case <-ctx.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("context was not cancelled by SIGTERM")
	}
	var ie *InterruptError
	if !errors.As(context.Cause(ctx), &ie) || ie.Signal != syscall.SIGTERM {
		t.Errorf("context.Cause() = %v, want InterruptError for SIGTERM", context.Cause(ctx))
	}
}

// TestNotifyContext_IgnoredSignalStaysIgnored verifies that NotifyContext
// leaves a signal the process ignores ignored, rather than re-enabling it by
// registering for it.
func TestNotifyContext_IgnoredSignalStaysIgnored(t *testing.T) {
	signal.Ignore(os.Interrupt)
	defer signal.Reset(os.Interrupt)

	_, stop := NotifyContext(context.Background())
	defer stop()

	if !signal.Ignored(os.Interrupt) {
		t.Error("NotifyContext re-enabled an ignored SIGINT")
	}
}
