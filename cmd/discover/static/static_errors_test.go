// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package static

// static_errors_test.go covers the rejection of a payload whose IDs could not
// be addressed in SMD. See static_test.go for the success paths.

import (
	"strings"
	"testing"

	"github.com/openchami/schemas/schemas/csm"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/client/smd"
)

// TestCheckIDs_RejectsInvalidIDs verifies that an invalid ID of each
// kind resolves to CodeUsage, and that an interface is checked by the ID its
// request would use, which may be adapted from its MAC address.
func TestCheckIDs_RejectsInvalidIDs(t *testing.T) {
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

			err := checkIDs(tc.comps, tc.rfes, tc.ifaces, tc.groups, true)
			if got := cli.ExitCode(err); got != cli.CodeUsage {
				t.Errorf("ExitCode() = %d, want %d (%s)", got, cli.CodeUsage, cli.CodeName(cli.CodeUsage))
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantMsg) {
				t.Errorf("error = %v, want it to contain %q", err, tc.wantMsg)
			}
		})
	}

	if err := checkIDs(validComps, validRFEs, validIfaces, validGroups, true); err != nil {
		t.Errorf("checkIDs() with valid IDs = %v, want nil", err)
	}
}

// TestCheckIDs_BlankRedfishEndpointID verifies that a redfish endpoint with a
// blank ID that SMD can adapt from its FQDN is only rejected with --overwrite,
// whose PUT needs an actual ID in its path.
func TestCheckIDs_BlankRedfishEndpointID(t *testing.T) {
	t.Parallel()

	rfes := smd.RedfishEndpointSliceV2{RedfishEndpoints: []smd.RedfishEndpointV2{{RedfishEndpoint: csm.RedfishEndpoint{FQDN: "x0c0s0b0.example.com"}}}}

	if err := checkIDs(smd.ComponentSlice{}, rfes, nil, nil, false); err != nil {
		t.Errorf("checkIDs() without overwrite = %v, want nil", err)
	}
	if got := cli.ExitCode(checkIDs(smd.ComponentSlice{}, rfes, nil, nil, true)); got != cli.CodeUsage {
		t.Errorf("ExitCode() with overwrite = %d, want %d (%s)", got, cli.CodeUsage, cli.CodeName(cli.CodeUsage))
	}
}

// TestCheckIDs_GroupMember verifies that a group's members are checked as well
// as its label.
func TestCheckIDs_GroupMember(t *testing.T) {
	t.Parallel()

	group := smd.Group{Label: "compute"}
	group.Members.IDs = []string{"x0c0s0b0n0", ".."}

	err := checkIDs(smd.ComponentSlice{}, smd.RedfishEndpointSliceV2{}, nil, []smd.Group{group}, false)
	if got := cli.ExitCode(err); got != cli.CodeUsage {
		t.Errorf("ExitCode() = %d, want %d (%s)", got, cli.CodeUsage, cli.CodeName(cli.CodeUsage))
	}
	if err == nil || !strings.Contains(err.Error(), `group member ".."`) {
		t.Errorf("error = %v, want it to name the group member", err)
	}
}
