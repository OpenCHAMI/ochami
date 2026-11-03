// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

//go:build unix

package cli

import (
	"os"
	"os/signal"
	"syscall"
	"time"
)

// dieFromSignal restores sig's default action and sends it to the process,
// returning only if the process survives it. See DieFromSignal.
func dieFromSignal(sig os.Signal) {
	s, ok := sig.(syscall.Signal)
	if !ok {
		return
	}
	signal.Reset(s)
	if err := syscall.Kill(os.Getpid(), s); err != nil {
		return
	}
	// The signal is delivered asynchronously, to any of the process's
	// threads. Wait for it to end the process rather than racing it to the
	// caller's os.Exit, which would make the death an ordinary exit again.
	time.Sleep(time.Second)
}
