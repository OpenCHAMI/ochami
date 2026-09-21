// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package transition

import (
	"errors"
	"io"
	"testing"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
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
	// Speed up any polling that does occur.
	origInterval := pollInterval
	pollInterval = 0
	t.Cleanup(func() { pollInterval = origInterval })

	cmd := newCmdTransitionMonitorWithClient(provider)
	// The command relies on token handling, which inspects these flags.
	cmd.Flags().Bool("no-token", true, "")
	cmd.Flags().String("cluster", "", "")
	_ = cmd.Flags().Set("no-token", "true")
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs(args)
	return cmd.Execute()
}

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

func TestTransitionMonitor_ClientConstructionError(t *testing.T) {
	wantErr := errors.New("no client")
	err := runMonitor(t, transitionProvider(nil, wantErr), "abc-123")
	if !errors.Is(err, wantErr) {
		t.Fatalf("Execute() error = %v, want wrapped %v", err, wantErr)
	}
}

func TestTransitionMonitor_GetTransitionError(t *testing.T) {
	fake := &scriptedTransitionClient{
		responses: []client.HTTPEnvelope{{}},
		errs:      []error{errors.New("network down")},
	}
	err := runMonitor(t, transitionProvider(fake, nil), "abc-123")
	if err == nil {
		t.Fatal("Execute(): expected error, got nil")
	}
	if got := cli.ExitCode(err); got != cli.CodeNetwork {
		t.Errorf("exit code = %d, want CodeNetwork (%d)", got, cli.CodeNetwork)
	}
}

func TestTransitionMonitor_MalformedResponse(t *testing.T) {
	fake := &scriptedTransitionClient{
		responses: []client.HTTPEnvelope{{Body: []byte(`not json`)}},
	}
	err := runMonitor(t, transitionProvider(fake, nil), "abc-123")
	if err == nil {
		t.Fatal("Execute(): expected error, got nil")
	}
	if got := cli.ExitCode(err); got != cli.CodePayload {
		t.Errorf("exit code = %d, want CodePayload (%d)", got, cli.CodePayload)
	}
}

// TestTransitionMonitor_RealProviderIsWired ensures the production wrapper is
// wired with the real provider.
func TestTransitionMonitor_RealProviderIsWired(t *testing.T) {
	if cmd := newCmdTransitionMonitor(); cmd == nil || cmd.RunE == nil {
		t.Fatal("newCmdTransitionMonitor() did not produce a runnable command")
	}
	var _ pcsTransitionClientProvider = realPCSTransitionClient
}
