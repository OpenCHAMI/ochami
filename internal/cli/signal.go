// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cli

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// NotifyContext returns a copy of parent that is cancelled when the process
// receives SIGINT or SIGTERM, with an *InterruptError naming the signal as
// its cause (see Interrupted). Only the first signal is caught: after it, the
// signals return to their default behavior, so a second Ctrl+C kills a
// process whose cleanup is stuck.
//
// A signal the process inherited as ignored stays ignored, since registering
// for it would re-enable it. Shells start background jobs of non-interactive
// scripts with SIGINT ignored, and such a job should stay uninterruptible.
//
// Calling stop releases the signals and cancels the context.
//
// Catching a signal changes how the process ends: a process that exits after
// catching SIGINT looks to its shell as if it handled the signal itself. Once
// the command has stopped, call DieFromSignal so the process still ends by
// the signal.
func NotifyContext(parent context.Context) (ctx context.Context, stop context.CancelFunc) {
	var sigs []os.Signal
	for _, sig := range []os.Signal{os.Interrupt, syscall.SIGTERM} {
		if !signal.Ignored(sig) {
			sigs = append(sigs, sig)
		}
	}
	ch := make(chan os.Signal, 1)
	if len(sigs) > 0 {
		signal.Notify(ch, sigs...)
	}
	return notifyContext(parent, ch, func() { signal.Stop(ch) })
}

// notifyContext implements NotifyContext with the signals delivered on
// signals and released by releaseSignals, so tests can deliver signals
// without sending them to the process.
func notifyContext(parent context.Context, signals <-chan os.Signal, releaseSignals func()) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancelCause(parent)
	var once sync.Once
	release := func() { once.Do(releaseSignals) }
	go func() {
		select {
		case sig := <-signals:
			cancel(&InterruptError{Signal: sig})
		case <-ctx.Done():
		}
		release()
	}()
	return ctx, func() {
		release()
		cancel(nil)
	}
}

// DieFromSignal ends the process by the signal that cancelled ctx, as if the
// signal had never been caught, and returns only if ctx wasn't cancelled by a
// signal or the process survived (in which case the caller should exit with
// the signal's code, which is what a shell reports for a death by it anyway).
// Call it once the command has stopped and cleaned up, in place of exiting
// with CodeInterrupted or CodeTerminated.
//
// A shell can tell a child that was killed by SIGINT from one that exited,
// even with status 130, and acts differently on each. When the user presses
// Ctrl+C, the terminal sends SIGINT to the shell as well as to the child.
// bash, waiting for the child, defers acting on it until the child ends
// ("wait and cooperative exit"): if the child was killed by SIGINT, bash
// treats the Ctrl+C as meant for it and aborts the script or loop, but if the
// child exited, bash assumes the child used Ctrl+C for its own purposes, as
// an editor does, and carries on with the next command. A process that
// catches SIGINT to clean up must therefore, after cleaning up, restore the
// signal's default action and kill itself with it, or Ctrl+C stops working
// for loops that run it. This is the convention described in Martin
// Cracauer's "Proper handling of SIGINT/SIGQUIT"
// (https://www.cons.org/cracauer/sigint.html), and the one git, GNU sort, and
// Python (since 3.8) follow. A command that consumes the signal as input
// instead, as "rcs console connect" does by ending its session cleanly,
// returns success, and the caller doesn't call DieFromSignal.
//
// SIGTERM is re-raised the same way, so a parent or supervisor sees that
// the process was terminated. Outside Unix, DieFromSignal does nothing.
func DieFromSignal(ctx context.Context) {
	var ie *InterruptError
	if errors.As(context.Cause(ctx), &ie) {
		dieFromSignal(ie.Signal)
	}
}
