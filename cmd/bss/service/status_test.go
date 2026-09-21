// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package service

import (
	"testing"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/pkg/client"
)

// fakeBSSStatusClient is a test double for bssStatusClient.
type fakeBSSStatusClient struct {
	gotComponent string
	env          client.HTTPEnvelope
	err          error
}

func (f *fakeBSSStatusClient) GetStatus(component string) (client.HTTPEnvelope, error) {
	f.gotComponent = component
	return f.env, f.err
}

// providerFor returns a bssStatusClientProvider that yields the given client and
// error.
func providerFor(c bssStatusClient, err error) bssStatusClientProvider {
	return func(*cobra.Command) (bssStatusClient, error) { return c, err }
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
			fake := &fakeBSSStatusClient{env: client.HTTPEnvelope{Body: []byte(`{"ok":true}`)}}
			cmd := newCmdServiceStatusWithClient(providerFor(fake, nil))
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

// TestServiceStatus_RealProviderIsWired ensures newCmdServiceStatus is
// constructed with the production provider (the wrapper compiles and returns a
// usable command). Full end-to-end behavior of the real client is covered by
// the command-level tests that run against an httptest.Server.
func TestServiceStatus_RealProviderIsWired(t *testing.T) {
	if cmd := newCmdServiceStatus(); cmd == nil || cmd.RunE == nil {
		t.Fatal("newCmdServiceStatus() did not produce a runnable command")
	}
	// The production provider must satisfy the consumer interface.
	var _ bssStatusClientProvider = realBSSStatusClient
}
