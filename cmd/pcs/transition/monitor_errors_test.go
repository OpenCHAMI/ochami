// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package transition

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

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

// inProgressTransition is a transition response that never completes, so the
// monitor keeps polling.
var inProgressTransition = client.HTTPEnvelope{Body: []byte(`{"transitionStatus":"in-progress","taskCounts":{"total":1,"in-progress":1}}`)}

// cancellingTransitionClient cancels the monitor's context from inside a
// GetTransition call, then returns err, standing in for a Ctrl+C that arrives
// while a request is in flight.
type cancellingTransitionClient struct {
	cancel context.CancelFunc
	err    error
}

func (c cancellingTransitionClient) GetTransition(context.Context, string, string) (client.HTTPEnvelope, error) {
	c.cancel()
	return inProgressTransition, c.err
}

// TestTransitionMonitor_Cancelled verifies that cancelling the context stops
// the monitor promptly, whether it is waiting out the poll interval or in a
// request, and returns the cancellation rather than a request failure.
func TestTransitionMonitor_Cancelled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		client func(cancel context.CancelFunc) pcsTransitionClient
	}{
		{name: "during poll interval", client: func(cancel context.CancelFunc) pcsTransitionClient {
			time.AfterFunc(10*time.Millisecond, cancel)
			return &scriptedTransitionClient{responses: []client.HTTPEnvelope{inProgressTransition}}
		}},
		{name: "during request", client: func(cancel context.CancelFunc) pcsTransitionClient {
			return cancellingTransitionClient{cancel: cancel, err: fmt.Errorf("Get: %w", context.Canceled)}
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			provider := transitionProvider(tc.client(cancel), nil)
			done := make(chan error, 1)
			go func() { done <- runMonitorContext(t, ctx, provider, "--poll-interval", "30", "abc-123") }()

			select {
			case err := <-done:
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("Execute() error = %v, want context.Canceled", err)
				}
				if strings.Contains(err.Error(), "failed to get transition") {
					t.Errorf("Execute() error = %q, want the cancellation, not a request failure", err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("monitor did not stop within 5s of cancellation (poll interval is 30s)")
			}
		})
	}
}
