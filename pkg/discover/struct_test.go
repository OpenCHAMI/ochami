package discover

import (
	"strings"
	"testing"
)

///////////////////////////
//                       //
// DEPRECATED STRUCTURES //
//                       //
///////////////////////////

func TestNodeListDeprecated_String_Full(t *testing.T) {
	nl := NodeListDeprecated{
		Nodes: []NodeDeprecated{
			{
				Name:   "nid1",
				NID:    1,
				Xname:  "x1000c0s0b0n0",
				Group:  "compute",
				BMCMac: "de:ad:be:ee:ef:00",
				BMCIP:  "172.16.101.1",
				Ifaces: []IfaceDeprecated{
					{
						MACAddr: "de:ca:fc:0f:fe:e1",
						IPAddrs: []IfaceIPDeprecated{
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
				Ifaces: []IfaceDeprecated{
					{
						MACAddr: "de:ca:fc:0f:fe:e2",
						IPAddrs: []IfaceIPDeprecated{
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
		`node0={name="nid1" nid=1 xname=x1000c0s0b0n0 group="compute" groups=[] bmc_mac=de:ad:be:ee:ef:00 bmc_ip=172.16.101.1 bmc_fqdn= interfaces=[iface0={mac_addr=de:ca:fc:0f:fe:e1 ip_addrs=[ip0={network="mgmt" ip_addr=172.16.100.1}]}]} ` +
		`node1={name="nid2" nid=2 xname=x1000c0s1b0n0 group="compute" groups=[] bmc_mac=de:ad:be:ee:ef:01 bmc_ip=172.16.101.2 bmc_fqdn=nid2.bmc.example.com interfaces=[iface0={mac_addr=de:ca:fc:0f:fe:e2 ip_addrs=[ip0={network="mgmt" ip_addr=172.16.100.2}]}]}` +
		`]`
	if got := nl.String(); got != want {
		t.Errorf("NodeListDeprecated.String() = %q, want %q", got, want)
	}
}

func TestNodeListDeprecated_String_Empty(t *testing.T) {
	nl := NodeListDeprecated{Nodes: nil}

	if got := nl.String(); got != "[]" {
		t.Fatalf("NodeListDeprecated.String() should render empty list, got: %q", got)
	}
}

func TestIfaceDeprecated_String_Format(t *testing.T) {
	iface := IfaceDeprecated{
		MACAddr: "00:00:00:00:00:00",
		IPAddrs: []IfaceIPDeprecated{
			{Network: "n1", IPAddr: "172.16.0.1"},
			{Network: "n2", IPAddr: "172.16.0.2"},
		},
	}
	want := `mac_addr=00:00:00:00:00:00 ip_addrs=[ip0={network="n1" ip_addr=172.16.0.1} ip1={network="n2" ip_addr=172.16.0.2}]`
	if got := iface.String(); got != want {
		t.Errorf("IfaceDeprecated.String() = %q, want %q", got, want)
	}
}

func TestIfaceDeprecated_String_WithTwoIPs(t *testing.T) {
	iface := IfaceDeprecated{
		MACAddr: "00:00:00:00:00:00",
		IPAddrs: []IfaceIPDeprecated{
			{Network: "n1", IPAddr: "172.16.0.1"},
			{Network: "n2", IPAddr: "172.16.0.2"},
		},
	}
	got := iface.String()
	want := `mac_addr=00:00:00:00:00:00 ip_addrs=[ip0={network="n1" ip_addr=172.16.0.1} ip1={network="n2" ip_addr=172.16.0.2}]`
	if got != want {
		t.Fatalf("IfaceDeprecated.String() = %q, want %q", got, want)
	}
}

func TestIfaceDeprecated_String_NoIPs(t *testing.T) {
	iface := IfaceDeprecated{
		MACAddr: "de:ad:be:ef:00:01",
		IPAddrs: nil,
	}
	got := iface.String()

	// Expect MAC present and an explicitly empty list for ip_addrs.
	if want := "mac_addr=de:ad:be:ef:00:01"; !strings.Contains(got, want) {
		t.Fatalf("IfaceDeprecated.String() missing %q in %q", want, got)
	}
	if !strings.Contains(got, "ip_addrs=[]") && !strings.Contains(got, "ip_addrs=[ ]") {
		t.Fatalf("IfaceDeprecated.String() should render an empty ip_addrs list, got: %q", got)
	}
}

func TestIfaceIPDeprecated_String_Format(t *testing.T) {
	ip := IfaceIPDeprecated{Network: "nw", IPAddr: "1.2.3.4"}
	got := ip.String()
	want := `network="nw" ip_addr=1.2.3.4`
	if got != want {
		t.Fatalf("IfaceIPDeprecated.String() = %q, want %q", got, want)
	}
}

func TestNodeDeprecated_String_Full(t *testing.T) {
	n := NodeDeprecated{
		Name:   "nid1",
		NID:    1,
		Xname:  "x1000c0s0b0n0",
		Group:  "compute",
		BMCMac: "de:ad:be:ee:ef:00",
		BMCIP:  "172.16.101.1",
		Ifaces: []IfaceDeprecated{
			{
				MACAddr: "00:00:00:00:00:00",
				IPAddrs: []IfaceIPDeprecated{
					{Network: "n1", IPAddr: "172.16.0.1"},
					{Network: "n2", IPAddr: "172.16.0.2"},
				},
			},
			{
				MACAddr: "de:ad:be:ef:00:02",
				IPAddrs: []IfaceIPDeprecated{
					{Network: "n3", IPAddr: "172.16.0.3"},
				},
			},
		},
	}

	got := n.String()

	// Flexible assertions to avoid overfitting to internal exact formatting.
	needles := []string{
		`name="nid1"`,
		`nid=1`,
		`xname=x1000c0s0b0n0`,
		`group="compute"`,
		`groups=[]`,
		`bmc_mac=de:ad:be:ee:ef:00`,
		`bmc_ip=172.16.101.1`,
		`mac_addr=00:00:00:00:00:00`,
		`network="n1" ip_addr=172.16.0.1`,
		`network="n2" ip_addr=172.16.0.2`,
		`mac_addr=de:ad:be:ef:00:02`,
		`network="n3" ip_addr=172.16.0.3`,
	}
	for _, s := range needles {
		if !strings.Contains(got, s) {
			t.Fatalf("NodeDeprecated.String() missing %q in %q", s, got)
		}
	}
}

func TestNodeListDeprecated_String_MultipleNodes(t *testing.T) {
	nl := NodeListDeprecated{
		Nodes: []NodeDeprecated{
			{
				Name:   "nid1",
				NID:    1,
				Xname:  "x1000c0s0b0n0",
				Group:  "compute",
				BMCMac: "de:ad:be:ee:ef:00",
				BMCIP:  "172.16.101.1",
			},
			{
				Name:   "nid2",
				NID:    2,
				Xname:  "x1000c0s0b0n1",
				Group:  "compute",
				BMCMac: "de:ad:be:ee:ef:01",
				BMCIP:  "172.16.101.2",
			},
		},
	}
	got := nl.String()

	if !(strings.Contains(got, `nid1`) && strings.Contains(got, `nid2`)) {
		t.Fatalf("NodeListDeprecated.String() should contain both nodes, got: %q", got)
	}
	if !strings.Contains(got, "node0={") || !strings.Contains(got, "node1={") {
		t.Fatalf("NodeListDeprecated.String() should index nodes node0/node1, got: %q", got)
	}
}
