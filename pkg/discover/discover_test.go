package discover

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/google/uuid"

	"github.com/openchami/schemas/schemas"

	"github.com/OpenCHAMI/ochami/pkg/client/smd"
)

func TestDiscoveryInfoV2_Table(t *testing.T) {
	type wantRFE struct {
		count          int
		schemaVersions []int
		managerPerRFE  int
	}
	type wantComps struct {
		count int
	}
	type wantSys struct {
		perRFE []int // Systems per RFE, in order
	}
	type wantIfaces struct {
		total int // total smd.EthernetInterface entries (aggregated across nodes)
	}

	tests := []struct {
		name    string
		baseURI string
		di      DiscoveryItems
		wantErr bool

		wantComps  wantComps
		wantRFEs   wantRFE
		wantSys    wantSys
		wantIfaces wantIfaces
	}{
		{
			name:    "invalid base URI returns error",
			baseURI: "://bad_uri",
			di:      DiscoveryItems{},
			wantErr: true,
		},
		{
			name:    "single BMC, single node (explicit BMC name)",
			baseURI: "http://example.com",
			di: DiscoveryItems{
				BMCs: []BMC{
					{Name: "bmc-1", Xname: "x1000c0s0b0", MACAddr: "de:ca:fc:0f:fe:e1", IPAddr: "172.16.101.1", FQDN: "bmc-1.example.com"},
				},
				Nodes: []Node{
					{
						Name:   "n0",
						NID:    1,
						Xname:  "x1000c0s0b0n0",
						Groups: []string{"gA"},
						BMC:    "bmc-1", // match by name
						Ifaces: []Iface{
							{
								MACAddr: "de:ad:be:ee:ef:01",
								IPAddrs: []IfaceIP{
									{Name: "netA", IPAddr: "10.0.0.1"},
									{Name: "netB", IPAddr: "10.0.0.2"},
								},
							},
						},
					},
				},
			},
			wantErr:   false,
			wantComps: wantComps{count: 1},
			wantRFEs:  wantRFE{count: 1, schemaVersions: []int{1}, managerPerRFE: 1},
			wantSys:   wantSys{perRFE: []int{1}},
			wantIfaces: wantIfaces{
				total: 1, // one SMD EthernetInterface (per node iface)
			},
		},
		{
			name:    "multiple nodes on one BMC (BMC derived from xname)",
			baseURI: "http://example.com",
			di: DiscoveryItems{
				BMCs: []BMC{
					{Name: "rackA-bmc", Xname: "x1000c0s0b1", MACAddr: "de:ca:fc:0f:fe:e2", IPAddr: "172.16.101.2"},
				},
				Nodes: []Node{
					{
						Name:  "n1",
						NID:   2,
						Xname: "x1000c0s0b1n0",
						BMC:   "", // derive "x1000c0s0b1" from node xname
						Ifaces: []Iface{
							{
								MACAddr: "de:ad:be:ee:ef:02",
								IPAddrs: []IfaceIP{
									{Name: "data", IPAddr: "10.0.0.7"},
									{Name: "oob", IPAddr: "10.0.0.8"},
								},
							},
						},
					},
					{
						Name:  "n2",
						NID:   3,
						Xname: "x1000c0s0b1n1",
						BMC:   "",
						Ifaces: []Iface{
							{
								MACAddr: "de:ad:be:ee:ef:03",
								IPAddrs: []IfaceIP{
									{Name: "data", IPAddr: "10.0.0.9"},
								},
							},
						},
					},
				},
			},
			wantErr:   false,
			wantComps: wantComps{count: 2},
			wantRFEs:  wantRFE{count: 1, schemaVersions: []int{1}, managerPerRFE: 1},
			wantSys:   wantSys{perRFE: []int{2}},
			wantIfaces: wantIfaces{
				total: 2, // one per node iface
			},
		},
		{
			name:    "duplicate node entries dedup components and systems",
			baseURI: "http://example.com",
			di: DiscoveryItems{
				BMCs: []BMC{
					{Name: "bmc-dup", Xname: "x2000c0s0b0", MACAddr: "aa:bb:cc:dd:ee:ff", IPAddr: "172.16.50.10"},
				},
				Nodes: []Node{
					{
						Name:  "n-dupe",
						NID:   42,
						Xname: "x2000c0s0b0n0",
						BMC:   "bmc-dup",
						Ifaces: []Iface{
							{
								MACAddr: "00:00:00:00:de:ad",
								IPAddrs: []IfaceIP{{Name: "data", IPAddr: "192.0.2.10"}},
							},
						},
					},
					// Intentional duplicate of the same node/xname
					{
						Name:  "n-dupe",
						NID:   42,
						Xname: "x2000c0s0b0n0",
						BMC:   "bmc-dup",
						Ifaces: []Iface{
							{
								MACAddr: "00:00:00:00:de:ad",
								IPAddrs: []IfaceIP{{Name: "data", IPAddr: "192.0.2.10"}},
							},
						},
					},
				},
			},
			wantErr:   false,
			wantComps: wantComps{count: 1}, // deduped
			wantRFEs:  wantRFE{count: 1, schemaVersions: []int{1}, managerPerRFE: 1},
			wantSys:   wantSys{perRFE: []int{1}}, // deduped
			wantIfaces: wantIfaces{
				// The function appends an smd.EthernetInterface for each processed node iface,
				// but because systems are deduped by node xname, we still only produce one set
				// of interface entries for that unique system.
				total: 1,
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			comps, rfes, ifaces, err := DiscoveryInfoV2(tt.baseURI, tt.di)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("DiscoveryInfoV2 returned error: %v", err)
			}

			// Components
			if len(comps.Components) != tt.wantComps.count {
				t.Fatalf("got %d components, want %d", len(comps.Components), tt.wantComps.count)
			}
			for _, n := range tt.di.Nodes {
				// For nodes that are unique, ensure component fields look right.
				// (We won't fail if a duplicate wasn't added; we just verify one that *is* present.)
				for _, c := range comps.Components {
					if c.ID == n.Xname {
						if c.NID != n.NID || c.Type != "Node" || c.State != "On" || !c.Enabled {
							t.Errorf("Component = %+v (from node %+v)", c, n)
						}
					}
				}
			}

			// RedfishEndpoints
			if len(rfes.RedfishEndpoints) != tt.wantRFEs.count {
				t.Fatalf("got %d RFEs, want %d", len(rfes.RedfishEndpoints), tt.wantRFEs.count)
			}

			if len(tt.wantRFEs.schemaVersions) == tt.wantRFEs.count {
				for i, r := range rfes.RedfishEndpoints {
					if r.SchemaVersion != tt.wantRFEs.schemaVersions[i] {
						t.Errorf("RFE[%d].SchemaVersion = %d, want %d", i, r.SchemaVersion, tt.wantRFEs.schemaVersions[i])
					}
				}
			}

			// Each RFE should have exactly one Manager with a non-nil UUID and a BMC iface
			for i, r := range rfes.RedfishEndpoints {
				if len(r.Managers) != tt.wantRFEs.managerPerRFE {
					t.Fatalf("RFE[%d]: got %d managers, want %d", i, len(r.Managers), tt.wantRFEs.managerPerRFE)
				}
				m := r.Managers[0]
				// Manager System URI/Name
				expMgrURI := fmt.Sprintf("%s/redfish/v1/Managers/%s", tt.baseURI, r.ID)
				if m.System.URI != expMgrURI || m.System.Name != r.ID || m.Type != "NodeBMC" {
					t.Errorf("RFE[%d] Manager = %+v", i, m)
				}
				// Manager UUID should be non-nil
				if m.UUID == uuid.Nil.String() {
					t.Error("Manager.UUID is nil, want a real UUID")
				}
				// Manager should advertise one EthernetInterface matching the BMC's MAC/IP
				if len(m.EthernetInterfaces) != 1 {
					t.Fatalf("RFE[%d] Manager: got %d EthernetInterfaces, want 1", i, len(m.EthernetInterfaces))
				}
				me := m.EthernetInterfaces[0]
				if me.Name != r.ID || me.MAC != r.MACAddr || me.IP != r.IPAddress {
					t.Errorf("RFE[%d] Manager.EthernetInterface = %+v (vs RFE MAC/IP %+v/%+v)", i, me, r.MACAddr, r.IPAddress)
				}
			}

			// Systems per RFE (order preserved by implementation)
			if len(tt.wantSys.perRFE) == len(rfes.RedfishEndpoints) {
				for i, r := range rfes.RedfishEndpoints {
					if len(r.Systems) != tt.wantSys.perRFE[i] {
						t.Fatalf("RFE[%d]: got %d systems, want %d", i, len(r.Systems), tt.wantSys.perRFE[i])
					}
					if len(r.Systems) > 0 {
						s := r.Systems[0]
						// Check URI prefix
						wantPrefix := fmt.Sprintf("%s/redfish/v1/Systems/", tt.baseURI)
						if got := s.URI; len(got) < len(wantPrefix) || got[:len(wantPrefix)] != wantPrefix {
							t.Errorf("System.URI = %q, want prefix %q", s.URI, wantPrefix)
						}
						// Check name is present
						if s.Name == "" {
							t.Error("System.Name empty, want non-empty")
						}
						// Should have at least one EthernetInterface with the node xname as Name
						if len(s.EthernetInterfaces) < 1 {
							t.Errorf("System has no EthernetInterfaces")
						} else {
							se := s.EthernetInterfaces[0]
							if se.Name == "" || se.Description == "" || se.MAC == "" || se.IP == "" {
								t.Errorf("System.EthernetInterface has empty fields: %+v", se)
							}
						}
						// Should advertise the PCS action set (non-empty)
						if len(s.Actions) == 0 {
							t.Error("System.Actions empty, want non-empty default action set")
						}
					}
				}
			}

			// Check total of EthernetInterfaces
			if len(ifaces) != tt.wantIfaces.total {
				t.Fatalf("got %d smd.EthernetInterface(s), want %d", len(ifaces), tt.wantIfaces.total)
			}
			// Use first EthernetInterface (when present) as a litmus; check format
			if len(ifaces) > 0 {
				se := ifaces[0]
				if se.Type != "Node" || se.ComponentID == "" || se.MACAddress == "" {
					t.Errorf("smd.EthernetInterface = %+v", se)
				}
				if len(se.IPAddresses) == 0 {
					t.Errorf("smd.EthernetInterface.IPAddresses empty, want at least one")
				}
			}

			// Ensure BMC fields line up with RedfishEndpoint fields
			for i, r := range rfes.RedfishEndpoints {
				if i >= len(tt.di.BMCs) {
					continue
				}
				b := tt.di.BMCs[i]
				if r.Name != b.Name || r.ID != b.Xname || r.MACAddr != b.MACAddr || r.IPAddress != b.IPAddr || r.FQDN != b.FQDN {
					t.Errorf("RFE[%d] = %+v, want fields from BMC %+v", i, r, b)
				}
			}
		})
	}
}

func TestAddMemberToGroup(t *testing.T) {
	newGroup := func(members []string) smd.Group {
		var g smd.Group
		g.Members.IDs = members
		return g
	}
	tests := []struct {
		name     string
		group    smd.Group
		xname    string
		expected smd.Group
	}{
		{
			name:     "add new member to empty group",
			group:    newGroup([]string{}),
			xname:    "x1000c0s0b0n0",
			expected: newGroup([]string{"x1000c0s0b0n0"}),
		},
		{
			name:     "add new member to non-empty group",
			group:    newGroup([]string{"x1000c0s0b0n0", "x1000c0s0b1n0"}),
			xname:    "x1000c0s0b2n0",
			expected: newGroup([]string{"x1000c0s0b0n0", "x1000c0s0b1n0", "x1000c0s0b2n0"}),
		},
		{
			name:     "member already exists in group",
			group:    newGroup([]string{"x1000c0s0b0n0", "x1000c0s0b1n0"}),
			xname:    "x1000c0s0b1n0",
			expected: newGroup([]string{"x1000c0s0b0n0", "x1000c0s0b1n0"}),
		},
		{
			name:     "add member when group has one element",
			group:    newGroup([]string{"x1000c0s0b0n0"}),
			xname:    "x1000c0s0b1n0",
			expected: newGroup([]string{"x1000c0s0b0n0", "x1000c0s0b1n0"}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AddMemberToGroup(tt.group, tt.xname)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("AddMemberToGroup() = %+v, want %+v", got, tt.expected)
			}
		})
	}
}

// Test that EthernetInterfaces created from Redfish System uses first IP address.
//
// SMD does not support multiple IP addresses in its inventory detail.
func TestDiscoveryInfoV2_SystemInterface_UsesFirstIP(t *testing.T) {
	base := "http://example.com"
	di := DiscoveryItems{
		BMCs: []BMC{
			{Name: "bmc-1", Xname: "x3000c0s0b0", MACAddr: "10:10:10:10:10:10", IPAddr: "172.16.200.1"},
		},
		Nodes: []Node{
			{
				Name:  "n1",
				NID:   7,
				Xname: "x3000c0s0b0n0",
				BMC:   "bmc-1",
				Ifaces: []Iface{
					{
						MACAddr: "aa:aa:aa:aa:aa:aa",
						IPAddrs: []IfaceIP{
							{Name: "A", IPAddr: "10.1.1.1"},
							{Name: "B", IPAddr: "10.1.1.2"},
						},
					},
				},
			},
		},
	}

	_, rfes, _, err := DiscoveryInfoV2(base, di)
	if err != nil {
		t.Fatalf("DiscoveryInfoV2 returned error: %v", err)
	}
	if len(rfes.RedfishEndpoints) != 1 || len(rfes.RedfishEndpoints[0].Systems) != 1 {
		t.Fatalf("unexpected RFE/System counts: %+v", rfes.RedfishEndpoints)
	}
	sys := rfes.RedfishEndpoints[0].Systems[0]
	if len(sys.EthernetInterfaces) != 1 {
		t.Fatalf("got %d System.EthernetInterfaces, want 1", len(sys.EthernetInterfaces))
	}
	got := sys.EthernetInterfaces[0]
	want := schemas.EthernetInterface{
		Name:        di.Nodes[0].Xname,
		Description: fmt.Sprintf("Interface 0 for %s", di.Nodes[0].Name),
		MAC:         di.Nodes[0].Ifaces[0].MACAddr,
		IP:          di.Nodes[0].Ifaces[0].IPAddrs[0].IPAddr, // FIRST IP ONLY
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("System.EthernetInterface = %+v, want %+v", got, want)
	}
}

///////////////////////////
//                       //
// DEPRECATED STRUCTURES //
//                       //
///////////////////////////

func TestDiscoveryInfoV2Deprecated_InvalidURI(t *testing.T) {
	_, _, _, err := DiscoveryInfoV2Deprecated("://bad_uri", NodeListDeprecated{})
	if err == nil {
		t.Fatal("expected error for invalid URI, got nil")
	}
}

func TestDiscoveryInfoV2Deprecated_Success(t *testing.T) {
	base := "http://example.com"
	nl := NodeListDeprecated{
		Nodes: []NodeDeprecated{
			{
				Name:    "n42",
				NID:     42,
				Xname:   "invalid", // force node-to-BMC xname conversion to fail and fallback to node xname
				Group:   "g",
				BMCMac:  "de:ca:fc:0f:fe:e1",
				BMCIP:   "172.16.101.1",
				BMCFQDN: "n42.bmc.example.com",
				Ifaces: []IfaceDeprecated{
					{
						MACAddr: "de:ad:be:ee:ef:01",
						IPAddrs: []IfaceIPDeprecated{
							{Network: "netA", IPAddr: "10.0.0.1"},
							{Network: "netB", IPAddr: "10.0.0.2"},
						},
					},
				},
			},
		},
	}

	comps, rfes, ifaces, err := DiscoveryInfoV2Deprecated(base, nl)
	if err != nil {
		t.Fatalf("DiscoveryInfoV2Deprecated returned error: %v", err)
	}

	// Components
	if len(comps.Components) != 1 {
		t.Fatalf("got %d components, want 1", len(comps.Components))
	}
	c := comps.Components[0]
	if want := nl.Nodes[0].Xname; c.ID != want {
		t.Errorf("component ID = %q, want %q", c.ID, want)
	}
	if c.NID != nl.Nodes[0].NID || c.Type != "Node" || c.State != "On" || !c.Enabled {
		t.Errorf("component = %+v", c)
	}

	// RedfishEndpoints
	if len(rfes.RedfishEndpoints) != 1 {
		t.Fatalf("got %d redfish endpoints, want 1", len(rfes.RedfishEndpoints))
	}
	r := rfes.RedfishEndpoints[0]
	node := nl.Nodes[0]
	if r.Name != node.Name || r.Type != "NodeBMC" || r.MACAddr != node.BMCMac || r.IPAddress != node.BMCIP || r.FQDN != node.BMCFQDN {
		t.Errorf("RedfishEndpoint fields = %+v", r)
	}
	if r.SchemaVersion != 1 {
		t.Errorf("SchemaVersion = %d, want 1", r.SchemaVersion)
	}

	// Systems
	if len(r.Systems) != 1 {
		t.Fatalf("got %d systems, want 1", len(r.Systems))
	}
	sys := r.Systems[0]
	expectedSysURI := fmt.Sprintf("%s/redfish/v1/Systems/%s", base, node.Xname)
	if sys.URI != expectedSysURI || sys.Name != node.Name {
		t.Errorf("System = %+v", sys)
	}
	if len(sys.EthernetInterfaces) != 1 {
		t.Fatalf("got %d system EthernetInterfaces, want 1", len(sys.EthernetInterfaces))
	}
	e := sys.EthernetInterfaces[0]
	if want := (schemas.EthernetInterface{
		Name:        node.Xname,
		Description: fmt.Sprintf("Interface 0 for %s", node.Name),
		MAC:         node.Ifaces[0].MACAddr,
		IP:          node.Ifaces[0].IPAddrs[0].IPAddr,
	}); !reflect.DeepEqual(e, want) {
		t.Errorf("System.EthernetInterface = %+v, want %+v", e, want)
	}
	if !reflect.DeepEqual(
		sys.Actions,
		[]string{"On", "ForceOff", "GracefulShutdown", "GracefulRestart", "ForceRestart", "Nmi", "ForceOn",
			"PushPowerButton", "PowerCycle", "Suspend", "Pause", "Resume"},
	) {
		t.Error("System.Actions does not match the expected value")
	}

	// Managers
	if len(r.Managers) != 1 {
		t.Fatalf("got %d managers, want 1", len(r.Managers))
	}
	m := r.Managers[0]
	expectedMgrURI := fmt.Sprintf("%s/redfish/v1/Managers/%s", base, node.Xname)
	if m.System.URI != expectedMgrURI || m.System.Name != node.Xname || m.Type != "NodeBMC" {
		t.Errorf("Manager = %+v", m)
	}
	if m.UUID == uuid.Nil.String() {
		t.Error("Manager.UUID is nil, want a real UUID")
	}

	// EthernetInterface slice
	if len(ifaces) != len(node.Ifaces) {
		t.Fatalf("got %d smd.EthernetInterface, want %d", len(ifaces), len(node.Ifaces))
	}
	se := ifaces[0]
	if se.ComponentID != node.Xname || se.Type != "Node" {
		t.Errorf("EthernetInterface = %+v", se)
	}
	if len(se.IPAddresses) != len(node.Ifaces[0].IPAddrs) {
		t.Fatalf("got %d IPAddresses, want %d", len(se.IPAddresses), len(node.Ifaces[0].IPAddrs))
	}
	for i, ip := range se.IPAddresses {
		orig := node.Ifaces[0].IPAddrs[i]
		if ip.IPAddress != orig.IPAddr || ip.Network != orig.Network {
			t.Errorf("IPAddresses[%d] = %+v, want %+v", i, ip, orig)
		}
	}
}

func TestDiscoveryInfoV2Deprecated_MultipleNodesPerBMC(t *testing.T) {
	base := "http://example.com"
	bmc0Xname := "x1000c0s0b0"
	// bmc1Xname := "x1000c0s0b0"
	nodes := NodeListDeprecated{
		Nodes: []NodeDeprecated{
			{
				Name:   "x1000c0s0b0n0",
				NID:    42,
				Xname:  "x1000c0s0b0n0",
				Group:  "g",
				BMCMac: "de:ca:fc:0f:fe:e1",
				BMCIP:  "172.16.101.1",
				Ifaces: []IfaceDeprecated{
					{
						MACAddr: "de:ad:be:ee:ef:01",
						IPAddrs: []IfaceIPDeprecated{
							{Network: "netA", IPAddr: "10.0.0.1"},
							{Network: "netB", IPAddr: "10.0.0.2"},
						},
					},
				},
			},
			{
				Name:   "x1000c0s0b0n1",
				NID:    43,
				Xname:  "x1000c0s0b0n1",
				Group:  "g",
				BMCMac: "de:ca:fc:0f:fe:e1",
				BMCIP:  "172.16.101.1",
				Ifaces: []IfaceDeprecated{
					{
						MACAddr: "de:ad:be:ee:ef:02",
						IPAddrs: []IfaceIPDeprecated{
							{Network: "netA", IPAddr: "10.0.0.3"},
							{Network: "netB", IPAddr: "10.0.0.4"},
						},
					},
				},
			},
			{
				Name:   "x1000c0s0b1n0",
				NID:    44,
				Xname:  "x1000c0s0b1n0",
				Group:  "g",
				BMCMac: "de:ca:fc:0f:fe:e2",
				BMCIP:  "172.16.101.2",
				Ifaces: []IfaceDeprecated{
					{
						MACAddr: "de:ad:be:ee:ef:01",
						IPAddrs: []IfaceIPDeprecated{
							{Network: "netA", IPAddr: "10.0.0.5"},
							{Network: "netB", IPAddr: "10.0.0.6"},
						},
					},
				},
			},
			{
				Name:   "x1000c0s0b1n1",
				NID:    45,
				Xname:  "x1000c0s0b1n1",
				Group:  "g",
				BMCMac: "de:ca:fc:0f:fe:e2",
				BMCIP:  "172.16.101.2",
				Ifaces: []IfaceDeprecated{
					{
						MACAddr: "de:ad:be:ee:ef:02",
						IPAddrs: []IfaceIPDeprecated{
							{Network: "netA", IPAddr: "10.0.0.7"},
							{Network: "netB", IPAddr: "10.0.0.8"},
						},
					},
				},
			},
		},
	}

	comps, rfes, ifaces, err := DiscoveryInfoV2Deprecated(base, nodes)
	if err != nil {
		t.Fatalf("DiscoveryInfoV2Deprecated returned error: %v", err)
	}

	// Components
	if len(comps.Components) != 4 {
		t.Fatalf("got %d components, want 4", len(comps.Components))
	}
	c := comps.Components[0]
	if want := nodes.Nodes[0].Xname; c.ID != want {
		t.Errorf("component ID = %q, want %q", c.ID, want)
	}
	if c.NID != nodes.Nodes[0].NID || c.Type != "Node" || c.State != "On" || !c.Enabled {
		t.Errorf("component = %+v", c)
	}

	// RedfishEndpoints
	if len(rfes.RedfishEndpoints) != 2 {
		t.Fatalf("got %d redfish endpoints, want 2", len(rfes.RedfishEndpoints))
	}
	r := rfes.RedfishEndpoints[0]
	node := nodes.Nodes[0]
	if r.Name != node.Name || r.Type != "NodeBMC" || r.MACAddr != node.BMCMac || r.IPAddress != node.BMCIP || r.FQDN != node.BMCFQDN {
		t.Errorf("RedfishEndpoint fields = %+v", r)
	}
	if r.SchemaVersion != 1 {
		t.Errorf("SchemaVersion = %d, want 1", r.SchemaVersion)
	}

	// Systems
	if len(r.Systems) != 2 {
		t.Fatalf("got %d systems, want 2", len(r.Systems))
	}
	sys := r.Systems[0]
	expectedSysURI := fmt.Sprintf("%s/redfish/v1/Systems/%s", base, node.Xname)
	if sys.URI != expectedSysURI || sys.Name != node.Name {
		t.Errorf("System = %+v", sys)
	}
	if len(sys.EthernetInterfaces) != 1 {
		t.Fatalf("got %d system EthernetInterfaces, want 1", len(sys.EthernetInterfaces))
	}
	e := sys.EthernetInterfaces[0]
	if want := (schemas.EthernetInterface{
		Name:        node.Xname,
		Description: fmt.Sprintf("Interface 0 for %s", node.Name),
		MAC:         node.Ifaces[0].MACAddr,
		IP:          node.Ifaces[0].IPAddrs[0].IPAddr,
	}); !reflect.DeepEqual(e, want) {
		t.Errorf("System.EthernetInterface = %+v, want %+v", e, want)
	}
	if !reflect.DeepEqual(
		sys.Actions,
		[]string{"On", "ForceOff", "GracefulShutdown", "GracefulRestart", "ForceRestart", "Nmi", "ForceOn",
			"PushPowerButton", "PowerCycle", "Suspend", "Pause", "Resume"},
	) {
		t.Error("System.Actions does not match the expected value")
	}

	// Managers
	if len(r.Managers) != 1 {
		t.Fatalf("got %d managers, want 1", len(r.Managers))
	}
	m := r.Managers[0]
	expectedMgrURI := fmt.Sprintf("%s/redfish/v1/Managers/%s", base, bmc0Xname)
	if m.System.URI != expectedMgrURI || m.System.Name != bmc0Xname || m.Type != "NodeBMC" {
		t.Errorf("URI: %s, expected: %s", m.System.URI, expectedMgrURI)
		t.Errorf("Name: %s, expected: %s", m.System.Name, node.Xname)
		t.Errorf("Type: %s, expected: NodeBMC", m.Type)
		t.Errorf("Manager = %+v", m)
	}
	if m.UUID == uuid.Nil.String() {
		t.Error("Manager.UUID is nil, want a real UUID")
	}

	// EthernetInterface slice
	if len(ifaces) != 4 {
		t.Fatalf("got %d smd.EthernetInterface, want 4", len(ifaces))
	}
	se := ifaces[0]
	if se.ComponentID != node.Xname || se.Type != "Node" {
		t.Errorf("EthernetInterface = %+v", se)
	}
	if len(se.IPAddresses) != len(node.Ifaces[0].IPAddrs) {
		t.Fatalf("got %d IPAddresses, want %d", len(se.IPAddresses), len(node.Ifaces[0].IPAddrs))
	}
	for i, ip := range se.IPAddresses {
		orig := node.Ifaces[0].IPAddrs[i]
		if ip.IPAddress != orig.IPAddr || ip.Network != orig.Network {
			t.Errorf("IPAddresses[%d] = %+v, want %+v", i, ip, orig)
		}
	}
}
