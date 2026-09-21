// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package service

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/client"
)

// fakeBSSStatusClient is a test double for bssStatusClient.
type fakeBSSStatusClient struct {
	gotComponent string
	env          client.HTTPEnvelope
	err          error
}

func (f *fakeBSSStatusClient) GetStatus(_ context.Context, component string) (client.HTTPEnvelope, error) {
	f.gotComponent = component
	return f.env, f.err
}

// providerFor returns a bssStatusClientProvider that yields the given client and
// error.
func providerFor(c bssStatusClient, err error) bssStatusClientProvider {
	return func(*cobra.Command) (bssStatusClient, error) { return c, err }
}

// createTestRuntime creates a test runtime with isolated I/O streams.
func createTestRuntime() (*cli.Runtime, *bytes.Buffer) {
	var stdoutBuf bytes.Buffer
	rt := cli.NewTestRuntime(strings.NewReader(""), &stdoutBuf, &stdoutBuf)
	return rt, &stdoutBuf
}

// TestServiceStatus_ComponentSelection verifies each component flag selects the
// expected status endpoint.
func TestServiceStatus_ComponentSelection(t *testing.T) {
	tests := []struct {
		name string
		flag string
		want string
	}{
		{"default", "", ""},
		{"all", "all", "all"},
		{"storage", "storage", "storage"},
		{"smd", "smd", "smd"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test runtime and inject into command context
			rt, stdoutBuf := createTestRuntime()
			fake := &fakeBSSStatusClient{env: client.HTTPEnvelope{Body: []byte(`{"ok":true}`)}}
			cmd := newCmdServiceStatusWithClient(providerFor(fake, nil))
			cmd.SetContext(cli.ContextWithRuntime(context.Background(), rt))
			cmd.SetOut(stdoutBuf)
			cmd.SetErr(stdoutBuf)
			cmd.SetArgs(nil)
			if tt.flag != "" {
				if err := cmd.Flags().Set(tt.flag, "true"); err != nil {
					t.Fatalf("set flag %q: %v", tt.flag, err)
				}
			}

			if err := cmd.Execute(); err != nil {
				t.Fatalf("Execute(): unexpected error: %v", err)
			}
			if fake.gotComponent != tt.want {
				t.Errorf("GetStatus component = %q, want %q", fake.gotComponent, tt.want)
			}
		})
	}
}

// TestServiceStatus_RealProviderIsWired verifies that newCmdServiceStatus
// builds a runnable command and that the production client provider has the
// type the command consumes.
func TestServiceStatus_RealProviderIsWired(t *testing.T) {
	if cmd := newCmdServiceStatus(); cmd == nil || cmd.RunE == nil {
		t.Fatal("newCmdServiceStatus() did not produce a runnable command")
	}
	// The production provider must satisfy the consumer interface.
	var _ bssStatusClientProvider = realBSSStatusClient
}
