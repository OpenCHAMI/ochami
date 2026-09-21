// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cli

import (
	"bytes"
	"strings"
	"sync"
	"testing"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/pkg/config"
)

func loggingCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("log-format", "", "")
	cmd.Flags().String("log-level", "", "")
	cmd.Flags().String("log-color", "", "")
	return cmd
}

func TestConcurrentRuntimeLoggingIsIsolated(t *testing.T) {
	t.Parallel()

	var debugOutput, errorOutput bytes.Buffer
	debugRuntime := NewTestRuntime(nil, &bytes.Buffer{}, &debugOutput).WithConfig(config.Config{
		Log: config.ConfigLog{Level: "debug", Format: "basic", Color: "off"},
	})
	errorRuntime := NewTestRuntime(nil, &bytes.Buffer{}, &errorOutput).WithConfig(config.Config{
		Log: config.ConfigLog{Level: "error", Format: "basic", Color: "off"},
	})

	var wg sync.WaitGroup
	for _, tc := range []struct {
		runtime *Runtime
		cmd     *cobra.Command
	}{
		{runtime: debugRuntime, cmd: loggingCommand()},
		{runtime: errorRuntime, cmd: loggingCommand()},
	} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := tc.runtime.InitLogging(tc.cmd); err != nil {
				t.Errorf("InitLogging() error = %v", err)
			}
		}()
	}
	wg.Wait()

	debugRuntime.Logger.Debug().Msg("debug-runtime-only")
	errorRuntime.Logger.Debug().Msg("filtered-debug-message")
	errorRuntime.Logger.Error().Msg("error-runtime-only")

	debugLogs := debugOutput.String()
	errorLogs := errorOutput.String()
	if !strings.Contains(debugLogs, "debug-runtime-only") || strings.Contains(debugLogs, "error-runtime-only") {
		t.Fatalf("debug runtime logs crossed writers: %q", debugLogs)
	}
	if !strings.Contains(errorLogs, "error-runtime-only") || strings.Contains(errorLogs, "debug-runtime-only") || strings.Contains(errorLogs, "filtered-debug-message") {
		t.Fatalf("error runtime logs crossed writers or ignored level: %q", errorLogs)
	}
}

// TestInitLogging_FlagOverrides covers InitLogging's log-format/level/color
// flag-override arms.
func TestInitLogging_FlagOverrides(t *testing.T) {
	cmd := loggingCommand()
	if err := cmd.Flags().Set("log-format", "json"); err != nil {
		t.Fatalf("set log-format: %v", err)
	}
	if err := cmd.Flags().Set("log-level", "warning"); err != nil {
		t.Fatalf("set log-level: %v", err)
	}
	if err := cmd.Flags().Set("log-color", "off"); err != nil {
		t.Fatalf("set log-color: %v", err)
	}

	rt := NewTestRuntime(nil, &bytes.Buffer{}, &bytes.Buffer{})
	if err := rt.InitLogging(cmd); err != nil {
		t.Fatalf("InitLogging = %v, want nil", err)
	}
	if rt.Config.Log.Format != "json" {
		t.Errorf("Log.Format = %q, want json", rt.Config.Log.Format)
	}
	if rt.Config.Log.Level != "warning" {
		t.Errorf("Log.Level = %q, want warning", rt.Config.Log.Level)
	}
}

func TestInitLogging_WritesToRuntimeErrorStream(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer
	rt := NewTestRuntime(nil, &bytes.Buffer{}, &stderr).WithConfig(config.Config{
		Log: config.ConfigLog{Level: "debug", Format: "basic", Color: "off"},
	})
	if err := rt.InitLogging(loggingCommand()); err != nil {
		t.Fatalf("InitLogging() error = %v", err)
	}
	if got := stderr.String(); !strings.Contains(got, "logging has been initialized") {
		t.Fatalf("runtime stderr = %q, want initialization message", got)
	}
}
