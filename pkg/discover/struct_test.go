package discover

import (
	"testing"
)

func TestNodeListDeprecated_String(t *testing.T) {
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
		`node0={name="nid1" nid=1 xname=x1000c0s0b0n0 bmc_mac=de:ad:be:ee:ef:00 bmc_ip=172.16.101.1 bmc_fqdn= interfaces=[iface0={mac_addr=de:ca:fc:0f:fe:e1 ip_addrs=[ip0={network="mgmt" ip_addr=172.16.100.1}]}]} ` +
		`node1={name="nid2" nid=2 xname=x1000c0s1b0n0 bmc_mac=de:ad:be:ee:ef:01 bmc_ip=172.16.101.2 bmc_fqdn=nid2.bmc.example.com interfaces=[iface0={mac_addr=de:ca:fc:0f:fe:e2 ip_addrs=[ip0={network="mgmt" ip_addr=172.16.100.2}]}]}` +
		`]`
	if got := nl.String(); got != want {
		t.Errorf("NodeList.String() = %q, want %q", got, want)
	}
}

func TestNodeDeprecated_String(t *testing.T) {
	node := NodeDeprecated{
		Name:    "node1",
		NID:     1,
		Xname:   "x1000c0s0b0n0",
		Group:   "grp",
		BMCMac:  "de:ca:fc:0f:fe:e1",
		BMCIP:   "172.16.101.1",
		BMCFQDN: "node1.bmc.example.com",
		Ifaces: []IfaceDeprecated{
			{
				MACAddr: "de:ad:be:ee:ef:01",
				IPAddrs: []IfaceIPDeprecated{
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

func TestIfaceDeprecated_String(t *testing.T) {
	iface := IfaceDeprecated{
		MACAddr: "00:00:00:00:00:00",
		IPAddrs: []IfaceIPDeprecated{
			{Network: "n1", IPAddr: "172.16.0.1"},
			{Network: "n2", IPAddr: "172.16.0.2"},
		},
	}
	want := `mac_addr=00:00:00:00:00:00 ip_addrs=[ip0={network="n1" ip_addr=172.16.0.1} ip1={network="n2" ip_addr=172.16.0.2}]`
	if got := iface.String(); got != want {
		t.Errorf("Iface.String() = %q, want %q", got, want)
	}
}

func TestIfaceIPDeprecated_String(t *testing.T) {
	ip := IfaceIPDeprecated{Network: "nw", IPAddr: "1.2.3.4"}
	want := `network="nw" ip_addr=1.2.3.4`
	if got := ip.String(); got != want {
		t.Errorf("IfaceIP.String() = %q, want %q", got, want)
	}
}
