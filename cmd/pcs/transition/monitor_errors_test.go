// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package transition

import (
	"errors"
	"testing"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/client"
)

// TestTransitionMonitor_ClientConstructionError verifies client setup failures
// are propagated.
func TestTransitionMonitor_ClientConstructionError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("no client")
	err := runMonitor(t, transitionProvider(nil, wantErr), "abc-123")
	if !errors.Is(err, wantErr) {
		t.Fatalf("Execute() error = %v, want wrapped %v", err, wantErr)
	}
}

// TestTransitionMonitor_GetTransitionError verifies polling failures are
// classified and returned.
func TestTransitionMonitor_GetTransitionError(t *testing.T) {
	t.Parallel()

	fake := &scriptedTransitionClient{
		responses: []client.HTTPEnvelope{{}},
		errs:      []error{errors.New("network down")},
	}
	err := runMonitor(t, transitionProvider(fake, nil), "abc-123")
	if err == nil {
		t.Fatal("Execute(): expected error, got nil")
	}
	if got := cli.ExitCode(err); got != cli.CodeNetwork {
		t.Errorf("exit code = %d, want %d (%s)", got, cli.CodeNetwork, cli.CodeName(cli.CodeNetwork))
	}
}

// TestTransitionMonitor_MalformedResponse verifies invalid transition payloads
// produce a payload error.
func TestTransitionMonitor_MalformedResponse(t *testing.T) {
	t.Parallel()

	fake := &scriptedTransitionClient{
		responses: []client.HTTPEnvelope{{Body: []byte(`not json`)}},
	}
	err := runMonitor(t, transitionProvider(fake, nil), "abc-123")
	if err == nil {
		t.Fatal("Execute(): expected error, got nil")
	}
	if got := cli.ExitCode(err); got != cli.CodePayload {
		t.Errorf("exit code = %d, want %d (%s)", got, cli.CodePayload, cli.CodeName(cli.CodePayload))
	}
}
