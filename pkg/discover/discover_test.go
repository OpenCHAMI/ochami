package discover

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/google/uuid"

	"github.com/openchami/schemas/schemas"

	"github.com/OpenCHAMI/ochami/pkg/client/smd"
)

func TestNodeList_String(t *testing.T) {
	nl := NodeList{
		Nodes: []Node{
			{
				Name:   "nid1",
				NID:    1,
				Xname:  "x1000c0s0b0n0",
				Group:  "compute",
				BMCMac: "de:ad:be:ee:ef:00",
				BMCIP:  "172.16.101.1",
				Ifaces: []Iface{
					{
						MACAddr: "de:ca:fc:0f:fe:e1",
						IPAddrs: []IfaceIP{
							{
								Network: "mgmt",
								IPAddr:  "172.16.100.1",
							},
						},
					},
				},
			},
			{
				Name:    "nid2",
				NID:     2,
				Xname:   "x1000c0s1b0n0",
				Group:   "compute",
				BMCMac:  "de:ad:be:ee:ef:01",
				BMCIP:   "172.16.101.2",
				BMCFQDN: "nid2.bmc.example.com",
				Ifaces: []Iface{
					{
						MACAddr: "de:ca:fc:0f:fe:e2",
						IPAddrs: []IfaceIP{
							{
								Network: "mgmt",
								IPAddr:  "172.16.100.2",
							},
						},
					},
				},
			},
		},
	}
	want := `[` +
		`node0={name="nid1" nid=1 xname=x1000c0s0b0n0 bmc_mac=de:ad:be:ee:ef:00 bmc_ip=172.16.101.1 bmc_fqdn= interfaces=[iface0={mac_addr=de:ca:fc:0f:fe:e1 ip_addrs=[ip0={network="mgmt" ip_addr=172.16.100.1}]}]} ` +
		`node1={name="nid2" nid=2 xname=x1000c0s1b0n0 bmc_mac=de:ad:be:ee:ef:01 bmc_ip=172.16.101.2 bmc_fqdn=nid2.bmc.example.com interfaces=[iface0={mac_addr=de:ca:fc:0f:fe:e2 ip_addrs=[ip0={network="mgmt" ip_addr=172.16.100.2}]}]}` +
		`]`
	if got := nl.String(); got != want {
		t.Errorf("NodeList.String() = %q, want %q", got, want)
	}
}

func TestNode_String(t *testing.T) {
	node := Node{
		Name:    "node1",
		NID:     1,
		Xname:   "x1000c0s0b0n0",
		Group:   "grp",
		BMCMac:  "de:ca:fc:0f:fe:e1",
		BMCIP:   "172.16.101.1",
		BMCFQDN: "node1.bmc.example.com",
		Ifaces: []Iface{
			{
				MACAddr: "de:ad:be:ee:ef:01",
				IPAddrs: []IfaceIP{
					{Network: "net", IPAddr: "10.0.0.1"},
				},
			},
		},
	}
	want := `name="node1" nid=1 xname=x1000c0s0b0n0 bmc_mac=de:ca:fc:0f:fe:e1 bmc_ip=172.16.101.1 bmc_fqdn=node1.bmc.example.com ` +
		`interfaces=[iface0={mac_addr=de:ad:be:ee:ef:01 ip_addrs=[ip0={network="net" ip_addr=10.0.0.1}]}]`
	if got := node.String(); got != want {
		t.Errorf("Node.String() = %q, want %q", got, want)
	}
}

func TestIface_String(t *testing.T) {
	iface := Iface{
		MACAddr: "00:00:00:00:00:00",
		IPAddrs: []IfaceIP{
			{Network: "n1", IPAddr: "172.16.0.1"},
			{Network: "n2", IPAddr: "172.16.0.2"},
		},
	}
	want := `mac_addr=00:00:00:00:00:00 ip_addrs=[ip0={network="n1" ip_addr=172.16.0.1} ip1={network="n2" ip_addr=172.16.0.2}]`
	if got := iface.String(); got != want {
		t.Errorf("Iface.String() = %q, want %q", got, want)
	}
}

func TestIfaceIP_String(t *testing.T) {
	ip := IfaceIP{Network: "nw", IPAddr: "1.2.3.4"}
	want := `network="nw" ip_addr=1.2.3.4`
	if got := ip.String(); got != want {
		t.Errorf("IfaceIP.String() = %q, want %q", got, want)
	}
}

func TestDiscoveryInfoV2_InvalidURI(t *testing.T) {
	_, _, _, err := DiscoveryInfoV2("://bad_uri", NodeList{})
	if err == nil {
		t.Fatal("expected error for invalid URI, got nil")
	}
}

func TestDiscoveryInfoV2_Success(t *testing.T) {
	base := "http://example.com"
	nl := NodeList{
		Nodes: []Node{
			{
				Name:    "n42",
				NID:     42,
				Xname:   "invalid", // force xname->BMCXname to error & fallback
				Group:   "g",
				BMCMac:  "de:ca:fc:0f:fe:e1",
				BMCIP:   "172.16.101.1",
				BMCFQDN: "n42.bmc.example.com",
				Ifaces: []Iface{
					{
						MACAddr: "de:ad:be:ee:ef:01",
						IPAddrs: []IfaceIP{
							{Network: "netA", IPAddr: "10.0.0.1"},
							{Network: "netB", IPAddr: "10.0.0.2"},
						},
					},
				},
			},
		},
	}

	comps, rfes, ifaces, err := DiscoveryInfoV2(base, nl)
	if err != nil {
		t.Fatalf("DiscoveryInfoV2 returned error: %v", err)
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

func TestDiscoveryInfoV2_MultipleNodesPerBMC(t *testing.T) {
	base := "http://example.com"
	bmc0Xname := "x1000c0s0b0"
	// bmc1Xname := "x1000c0s0b0"
	nodes := NodeList{
		Nodes: []Node{
			{
				Name:   "x1000c0s0b0n0",
				NID:    42,
				Xname:  "x1000c0s0b0n0",
				Group:  "g",
				BMCMac: "de:ca:fc:0f:fe:e1",
				BMCIP:  "172.16.101.1",
				Ifaces: []Iface{
					{
						MACAddr: "de:ad:be:ee:ef:01",
						IPAddrs: []IfaceIP{
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
				Ifaces: []Iface{
					{
						MACAddr: "de:ad:be:ee:ef:02",
						IPAddrs: []IfaceIP{
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
				Ifaces: []Iface{
					{
						MACAddr: "de:ad:be:ee:ef:01",
						IPAddrs: []IfaceIP{
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
				Ifaces: []Iface{
					{
						MACAddr: "de:ad:be:ee:ef:02",
						IPAddrs: []IfaceIP{
							{Network: "netA", IPAddr: "10.0.0.7"},
							{Network: "netB", IPAddr: "10.0.0.8"},
						},
					},
				},
			},
		},
	}

	comps, rfes, ifaces, err := DiscoveryInfoV2(base, nodes)
	if err != nil {
		t.Fatalf("DiscoveryInfoV2 returned error: %v", err)
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
