// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package static

// static_errors_test.go covers the rejection of a payload whose IDs cannot be
// used in the request paths that --overwrite sends. See static_test.go for the
// success paths.

import (
	"strings"
	"testing"

	"github.com/openchami/schemas/schemas/csm"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/client/smd"
)

// TestCheckOverwriteIDs_RejectsInvalidIDs verifies that an invalid ID of each
// kind resolves to CodeUsage, and that an interface is checked by the ID its
// request would use, which may be adapted from its MAC address.
func TestCheckOverwriteIDs_RejectsInvalidIDs(t *testing.T) {
	t.Parallel()

	validComps := smd.ComponentSlice{Components: []smd.Component{{ID: "x0c0s0b0n0"}}}
	validRFEs := smd.RedfishEndpointSliceV2{RedfishEndpoints: []smd.RedfishEndpointV2{{RedfishEndpoint: csm.RedfishEndpoint{ID: "x0c0s0b0"}}}}
	validIfaces := []smd.EthernetInterface{{MACAddress: "de:ad:be:ef:00:00"}}
	validGroups := []smd.Group{{Label: "compute"}}

	tests := []struct {
		name    string
		comps   smd.ComponentSlice
		rfes    smd.RedfishEndpointSliceV2
		ifaces  []smd.EthernetInterface
		groups  []smd.Group
		wantMsg string
	}{
		{
			name:    "component",
			comps:   smd.ComponentSlice{Components: []smd.Component{{ID: ".."}}},
			rfes:    validRFEs,
			ifaces:  validIfaces,
			groups:  validGroups,
			wantMsg: "component ID",
		},
		{
			name:    "redfish endpoint",
			comps:   validComps,
			rfes:    smd.RedfishEndpointSliceV2{RedfishEndpoints: []smd.RedfishEndpointV2{{RedfishEndpoint: csm.RedfishEndpoint{ID: "a/b"}}}},
			ifaces:  validIfaces,
			groups:  validGroups,
			wantMsg: "redfish endpoint ID",
		},
		{
			name:    "interface with blank adapted ID",
			comps:   validComps,
			rfes:    validRFEs,
			ifaces:  []smd.EthernetInterface{{MACAddress: "::"}},
			groups:  validGroups,
			wantMsg: "ethernet interface ID cannot be blank",
		},
		{
			name:    "group",
			comps:   validComps,
			rfes:    validRFEs,
			ifaces:  validIfaces,
			groups:  []smd.Group{{Label: "."}},
			wantMsg: "group label",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := checkOverwriteIDs(tc.comps, tc.rfes, tc.ifaces, tc.groups)
			if got := cli.ExitCode(err); got != cli.CodeUsage {
				t.Errorf("ExitCode() = %d, want %d (%s)", got, cli.CodeUsage, cli.CodeName(cli.CodeUsage))
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantMsg) {
				t.Errorf("error = %v, want it to contain %q", err, tc.wantMsg)
			}
		})
	}

	if err := checkOverwriteIDs(validComps, validRFEs, validIfaces, validGroups); err != nil {
		t.Errorf("checkOverwriteIDs() with valid IDs = %v, want nil", err)
	}
}
