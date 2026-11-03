// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

//go:build unix

package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

// processTimeout bounds how long a test waits for a signalled process to end.
const processTimeout = 10 * time.Second

// hangingServer starts a server that never answers and returns its URL, a
// channel that receives a value as each request arrives, and a count of
// requests received.
func hangingServer(t *testing.T) (string, <-chan struct{}, *atomic.Int32) {
	t.Helper()
	arrived := make(chan struct{}, 16)
	var count atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		count.Add(1)
		arrived <- struct{}{}
		<-r.Context().Done()
	}))
	t.Cleanup(srv.Close)
	return srv.URL, arrived, &count
}

// helperEnv returns the environment that makes the test binary run main with
// args (see TestProcessHelper).
func helperEnv(t *testing.T, args ...string) []string {
	t.Helper()
	argsJSON, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("encode process arguments: %v", err)
	}
	return replaceProcessEnv(os.Environ(),
		processHelperEnv+"=1",
		processArgsEnv+"="+string(argsJSON),
		"HOME="+t.TempDir(),
	)
}

// waitSignaled waits for cmd to end and reports the signal that killed it,
// failing the test if it exited instead or outlived processTimeout.
func waitSignaled(t *testing.T, cmd *exec.Cmd, output *bytes.Buffer) syscall.Signal {
	t.Helper()
	done := make(chan struct{})
	go func() {
		_ = cmd.Wait() //nolint:errcheck // a signal death is an error; ProcessState is inspected below
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(processTimeout):
		// Kill the whole process group, in case cmd is a shell still
		// running ochami.
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) //nolint:errcheck // best-effort cleanup before failing
		<-done
		t.Fatalf("process did not end within %v of the signal; output:\n%s", processTimeout, output)
	}
	ws, ok := cmd.ProcessState.Sys().(syscall.WaitStatus)
	if !ok || !ws.Signaled() {
		t.Fatalf("process exited with status %d, want it killed by a signal; output:\n%s", cmd.ProcessState.ExitCode(), output)
	}
	return ws.Signal()
}

// TestProcess_KilledBySignal verifies that ochami, signalled while a command
// is blocked in a request, stops the command and then ends by the same
// signal rather than exiting, so its parent sees a death by the signal.
func TestProcess_KilledBySignal(t *testing.T) {
	tests := []struct {
		sig     syscall.Signal
		wantLog string
	}{
		{sig: syscall.SIGINT, wantLog: "interrupted by SIGINT"},
		{sig: syscall.SIGTERM, wantLog: "terminated by SIGTERM"},
	}
	for _, tc := range tests {
		sig := tc.sig
		t.Run(sig.String(), func(t *testing.T) {
			url, arrived, _ := hangingServer(t)
			var output bytes.Buffer
			cmd := exec.Command(os.Args[0], "-test.run=^TestProcessHelper$")
			cmd.Env = helperEnv(t, "--ignore-config", "--no-token", "--cluster-uri", url, "smd", "component", "get")
			cmd.Stdout = &output
			cmd.Stderr = &output
			cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
			if err := cmd.Start(); err != nil {
				t.Fatalf("start helper process: %v", err)
			}
			select {
			case <-arrived:
			case <-time.After(processTimeout):
				_ = cmd.Process.Kill() //nolint:errcheck // best-effort cleanup before failing
				t.Fatalf("helper process never sent its request; output:\n%s", &output)
			}

			if err := cmd.Process.Signal(sig); err != nil {
				t.Fatalf("signal helper process: %v", err)
			}
			if got := waitSignaled(t, cmd, &output); got != sig {
				t.Errorf("process killed by %v, want %v", got, sig)
			}
			// The command was stopped and reported before the process died.
			if !strings.Contains(output.String(), tc.wantLog) {
				t.Errorf("output does not report the interruption; got:\n%s", &output)
			}
		})
	}
}

// TestProcess_CtrlCStopsShellLoop verifies that Ctrl+C stops a bash loop
// running ochami. A terminal's Ctrl+C sends SIGINT to the whole foreground
// process group, bash included; bash goes on to the next iteration unless
// ochami is killed by the signal ("wait and cooperative exit", see
// cli.DieFromSignal).
func TestProcess_CtrlCStopsShellLoop(t *testing.T) {
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash not found")
	}
	if signal.Ignored(os.Interrupt) {
		// bash and ochami would inherit the ignored SIGINT and never see
		// the simulated Ctrl+C.
		t.Skip("SIGINT is ignored in the test process")
	}

	url, arrived, requests := hangingServer(t)
	var output bytes.Buffer
	script := `for i in 1 2; do "$1" -test.run='^TestProcessHelper$'; echo "after ochami: $?"; done; echo "loop finished"`
	cmd := exec.Command(bash, "-c", script, "bash", os.Args[0])
	cmd.Env = helperEnv(t, "--ignore-config", "--no-token", "--cluster-uri", url, "smd", "component", "get")
	cmd.Stdout = &output
	cmd.Stderr = &output
	// Its own process group stands in for the terminal's foreground group.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start bash: %v", err)
	}
	select {
	case <-arrived:
	case <-time.After(processTimeout):
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) //nolint:errcheck // best-effort cleanup before failing
		t.Fatalf("ochami never sent its request; output:\n%s", &output)
	}

	if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGINT); err != nil {
		t.Fatalf("signal process group: %v", err)
	}
	if got := waitSignaled(t, cmd, &output); got != syscall.SIGINT {
		t.Errorf("bash killed by %v, want SIGINT", got)
	}
	if strings.Contains(output.String(), "after ochami") {
		t.Errorf("bash went on with the loop after Ctrl+C; output:\n%s", &output)
	}
	if n := requests.Load(); n != 1 {
		t.Errorf("requests = %d, want 1: the loop ran ochami again after Ctrl+C", n)
	}
}
