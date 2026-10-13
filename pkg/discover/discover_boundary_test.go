// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package discover

import "testing"

// TestDiscoveryInfoV2_DeduplicatesBMCsAndSkipsUnresolvedNodes verifies that
// DiscoveryInfoV2 creates one Redfish endpoint for BMCs that share a MAC
// address, and still creates a component, but no system, for a node whose BMC
// can't be resolved.
func TestDiscoveryInfoV2_DeduplicatesBMCsAndSkipsUnresolvedNodes(t *testing.T) {
	di := DiscoveryItems{
		BMCs: []BMC{
			{Name: "first", Xname: "x0c0s0b0", MACAddr: "aa:bb:cc:dd:ee:ff"},
			{Name: "duplicate", Xname: "x0c0s0b1", MACAddr: "aa:bb:cc:dd:ee:ff"},
		},
		Nodes: []Node{{Name: "bad", Xname: "not-an-xname"}},
	}
	components, rfes, _, err := DiscoveryInfoV2("https://example.com", di)
	if err != nil {
		t.Fatalf("DiscoveryInfoV2() error = %v", err)
	}
	if len(rfes.RedfishEndpoints) != 1 {
		t.Fatalf("redfish endpoints = %d, want one deduplicated endpoint", len(rfes.RedfishEndpoints))
	}
	if len(components.Components) != 1 || len(rfes.RedfishEndpoints[0].Systems) != 0 {
		t.Fatalf("components/RFEs = (%#v, %#v), want unresolved node component without system", components, rfes)
	}
}

// TestDiscoveryInfoV2_SkipsNodeWithUnknownBMC verifies that DiscoveryInfoV2
// creates a component but no Redfish endpoint for a node whose BMC isn't
// listed.
func TestDiscoveryInfoV2_SkipsNodeWithUnknownBMC(t *testing.T) {
	di := DiscoveryItems{Nodes: []Node{{Name: "node", Xname: "x0c0s0b0n0", BMC: "missing"}}}
	components, rfes, _, err := DiscoveryInfoV2("https://example.com", di)
	if err != nil {
		t.Fatalf("DiscoveryInfoV2() error = %v", err)
	}
	if len(components.Components) != 1 || len(rfes.RedfishEndpoints) != 0 {
		t.Fatalf("components/RFEs = (%#v, %#v), want component and no RFE", components, rfes)
	}
}

// TestDeprecatedDiscovery_DeduplicatesRepeatedNodes verifies that
// DiscoveryInfoV2Deprecated produces one component, Redfish endpoint, system,
// and manager for a node listed twice.
func TestDeprecatedDiscovery_DeduplicatesRepeatedNodes(t *testing.T) {
	node := NodeDeprecated{Name: "node", Xname: "x0c0s0b0n0", BMCMac: "aa:bb:cc:dd:ee:ff"}
	components, rfes, _, err := DiscoveryInfoV2Deprecated("https://example.com", NodeListDeprecated{Nodes: []NodeDeprecated{node, node}})
	if err != nil {
		t.Fatalf("DiscoveryInfoV2Deprecated() error = %v", err)
	}
	if len(components.Components) != 1 || len(rfes.RedfishEndpoints) != 1 || len(rfes.RedfishEndpoints[0].Systems) != 1 || len(rfes.RedfishEndpoints[0].Managers) != 1 {
		t.Fatalf("deduplicated values = (%#v, %#v)", components, rfes)
	}
}
