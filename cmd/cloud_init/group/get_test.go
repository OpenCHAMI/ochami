// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package group

import (
	"sync"
	"testing"

	"github.com/spf13/cobra"
)

func TestGroupGet_ConfigHeaderOptionsAreCommandLocal(t *testing.T) {
	t.Parallel()

	const runs = 20
	for i := 0; i < runs; i++ {
		commands := []*cobra.Command{newCmdGroupGetConfig(), newCmdGroupGetConfig()}
		modes := []string{"always", "never"}
		got := make([]string, len(commands))
		start := make(chan struct{})
		var wg sync.WaitGroup

		for idx := range commands {
			idx := idx
			commands[idx].RunE = func(cmd *cobra.Command, _ []string) error {
				<-start
				got[idx] = cmd.Flags().Lookup("headers").Value.String()
				return nil
			}
			commands[idx].SetArgs([]string{"--headers", modes[idx]})
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := commands[idx].Execute(); err != nil {
					t.Errorf("command %d Execute() error = %v", idx, err)
				}
			}()
		}

		close(start)
		wg.Wait()
		for idx := range modes {
			if got[idx] != modes[idx] {
				t.Errorf("run %d command %d headers = %q, want %q", i, idx, got[idx], modes[idx])
			}
		}
	}
}

func TestGroupGet_ConfigHeaderDefaultIsIndependent(t *testing.T) {
	t.Parallel()

	first := newCmdGroupGetConfig()
	second := newCmdGroupGetConfig()
	if err := first.Flags().Set("headers", "always"); err != nil {
		t.Fatalf("setting first command headers: %v", err)
	}
	if got := second.Flags().Lookup("headers").Value.String(); got != "multiple" {
		t.Errorf("second command headers = %q, want %q", got, "multiple")
	}
	if got := first.Flags().Lookup("headers").Value.String(); got != "always" {
		t.Errorf("first command headers = %q, want %q", got, "always")
	}
}
