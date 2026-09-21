// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package discover

import "testing"

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

func TestDeprecatedDiscoveryDeduplicatesRepeatedNodes(t *testing.T) {
	node := NodeDeprecated{Name: "node", Xname: "x0c0s0b0n0", BMCMac: "aa:bb:cc:dd:ee:ff"}
	components, rfes, _, err := DiscoveryInfoV2Deprecated("https://example.com", NodeListDeprecated{Nodes: []NodeDeprecated{node, node}})
	if err != nil {
		t.Fatalf("DiscoveryInfoV2Deprecated() error = %v", err)
	}
	if len(components.Components) != 1 || len(rfes.RedfishEndpoints) != 1 || len(rfes.RedfishEndpoints[0].Systems) != 1 || len(rfes.RedfishEndpoints[0].Managers) != 1 {
		t.Fatalf("deduplicated values = (%#v, %#v)", components, rfes)
	}
}
