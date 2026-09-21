// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package transition

import (
	"io"
	"testing"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/pkg/client"
)

// scriptedTransitionClient returns a preset sequence of transition responses,
// advancing by one on each call and repeating the last entry thereafter. This
// lets a test drive the monitor's polling loop to completion deterministically.
type scriptedTransitionClient struct {
	responses []client.HTTPEnvelope
	errs      []error
	calls     int
}

func (s *scriptedTransitionClient) GetTransition(transitionID, token string) (client.HTTPEnvelope, error) {
	i := s.calls
	if i >= len(s.responses) {
		i = len(s.responses) - 1
	}
	s.calls++
	var err error
	if i < len(s.errs) {
		err = s.errs[i]
	}
	return s.responses[i], err
}

func transitionProvider(c pcsTransitionClient, err error) pcsTransitionClientProvider {
	return func(*cobra.Command) (pcsTransitionClient, error) { return c, err }
}

// runMonitor executes the monitor command with the given provider and args,
// registering the flags the command's RunE depends on (via cli.HandleToken) and
// discarding progress-bar output.
func runMonitor(t *testing.T, provider pcsTransitionClientProvider, args ...string) error {
	t.Helper()
	cmd := newCmdTransitionMonitorWithClient(provider)

	// Speed up any polling that does occur. This must happen after the
	// command is built: newCmdTransitionMonitorWithClient registers
	// --poll-interval via IntVarP, which resets pollInterval to its default
	// (1) as a side effect of flag registration, so setting it beforehand
	// would be silently overwritten.
	origInterval := pollInterval
	pollInterval = 0
	t.Cleanup(func() { pollInterval = origInterval })

	// The command relies on token handling, which inspects these flags.
	cmd.Flags().Bool("no-token", true, "")
	cmd.Flags().String("cluster", "", "")
	if err := cmd.Flags().Set("no-token", "true"); err != nil {
		t.Fatal(err)
	}
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs(args)
	return cmd.Execute()
}

// TestTransitionMonitor_CompletesOnCompletedStatus verifies monitoring stops
// after a completed transition.
func TestTransitionMonitor_CompletesOnCompletedStatus(t *testing.T) {
	fake := &scriptedTransitionClient{
		responses: []client.HTTPEnvelope{
			{Body: []byte(`{"transitionStatus":"in-progress","taskCounts":{"total":2,"in-progress":2}}`)},
			{Body: []byte(`{"transitionStatus":"completed","taskCounts":{"total":2,"succeeded":2}}`)},
		},
	}

	if err := runMonitor(t, transitionProvider(fake, nil), "abc-123"); err != nil {
		t.Fatalf("Execute(): unexpected error: %v", err)
	}
	if fake.calls < 2 {
		t.Errorf("GetTransition called %d times, want at least 2 (poll then complete)", fake.calls)
	}
}

// TestTransitionMonitor_CompletesOnAbortedStatus verifies monitoring stops
// after an aborted transition.
func TestTransitionMonitor_CompletesOnAbortedStatus(t *testing.T) {
	fake := &scriptedTransitionClient{
		responses: []client.HTTPEnvelope{
			{Body: []byte(`{"transitionStatus":"aborted","taskCounts":{"total":1,"failed":1}}`)},
		},
	}
	if err := runMonitor(t, transitionProvider(fake, nil), "abc-123"); err != nil {
		t.Fatalf("Execute(): unexpected error: %v", err)
	}
	if fake.calls != 1 {
		t.Errorf("GetTransition called %d times, want 1 (immediate abort)", fake.calls)
	}
}

// TestTransitionMonitor_RealProviderIsWired verifies that
// newCmdTransitionMonitor builds a runnable command and that the production
// client provider has the type the command consumes.
func TestTransitionMonitor_RealProviderIsWired(t *testing.T) {
	if cmd := newCmdTransitionMonitor(); cmd == nil || cmd.RunE == nil {
		t.Fatal("newCmdTransitionMonitor() did not produce a runnable command")
	}
	var _ pcsTransitionClientProvider = realPCSTransitionClient
}
