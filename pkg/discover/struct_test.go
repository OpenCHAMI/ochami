package discover

import (
	"strings"
	"testing"
)

func TestDiscoveryItems_String_Table(t *testing.T) {
	tests := []struct {
		name    string
		di      DiscoveryItems
		needles []string
	}{
		{
			name: "empty lists",
			di:   DiscoveryItems{BMCs: nil, Nodes: nil},
			needles: []string{
				"bmcs=[", "bmcs=[]",
				"nodes=[", "nodes=[]",
			},
		},
		{
			name: "two bmcs and two nodes",
			di: DiscoveryItems{
				BMCs: []BMC{
					{Name: "b1", Xname: "x1000c0s0b0", MACAddr: "de:ad:be:ef:aa:01", IPAddr: "172.16.101.1", FQDN: "b1.example.com"},
					{Name: "b2", Xname: "x1000c0s1b0", MACAddr: "de:ad:be:ef:aa:02", IPAddr: "172.16.101.2", FQDN: "b2.example.com"},
				},
				Nodes: []Node{
					{
						Name:   "nid1",
						NID:    1,
						Xname:  "x1000c0s0b0n0",
						Groups: []string{"compute"},
						BMC:    "x1000c0s0b0",
						Ifaces: []Iface{
							{MACAddr: "00:00:00:00:00:01", IPAddrs: []IfaceIP{{Name: "n1", IPAddr: "172.16.0.1"}}},
						},
					},
					{
						Name:   "nid2",
						NID:    2,
						Xname:  "x1000c0s1b0n0",
						Groups: []string{"compute"},
						// BMC left empty, it will be derived from node xname
						Ifaces: []Iface{
							{MACAddr: "00:00:00:00:00:02", IPAddrs: []IfaceIP{{Name: "n1", IPAddr: "172.16.0.2"}}},
						},
					},
				},
			},
			needles: []string{
				"bmcs=[", "bmc0={", "bmc1={", "nodes=[", "node0={", "node1={",
				`name="b1"`, `xname=x1000c0s0b0`, `ip=172.16.101.1`, `fqdn=b1.example.com`,
				`name="b2"`, `xname=x1000c0s1b0`, `ip=172.16.101.2`, `fqdn=b2.example.com`,
				`name="nid1"`, `bmc=x1000c0s0b0`,
				`name="nid2"`, `bmc=x1000c0s1b0`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.di.String()
			for _, n := range tt.needles {
				if !strings.Contains(got, n) {
					t.Fatalf("DiscoveryItems.String() missing %q in %q", n, got)
				}
			}
		})
	}
}

func TestBMC_String_Table(t *testing.T) {
	tests := []struct {
		name    string
		bmc     BMC
		needles []string
	}{
		{
			name: "all fields set",
			bmc: BMC{
				Name:    "bmc-1",
				Xname:   "x1000c0s0b0",
				MACAddr: "de:ad:be:ef:00:01",
				IPAddr:  "172.16.101.1",
				FQDN:    "bmc-1.example.com",
			},
			needles: []string{
				`name="bmc-1"`,
				`xname=x1000c0s0b0`,
				`mac=de:ad:be:ef:00:01`,
				`ip=172.16.101.1`,
				`fqdn=bmc-1.example.com`,
			},
		},
		{
			name: "empty FQDN",
			bmc: BMC{
				Name:    "bmc-2",
				Xname:   "x1000c0s1b0",
				MACAddr: "de:ad:be:ef:00:02",
				IPAddr:  "172.16.101.2",
				FQDN:    "",
			},
			needles: []string{
				`name="bmc-2"`,
				`xname=x1000c0s1b0`,
				`mac=de:ad:be:ef:00:02`,
				`ip=172.16.101.2`,
				`fqdn=`, // explicitly rendered
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.bmc.String()
			for _, n := range tt.needles {
				if !strings.Contains(got, n) {
					t.Fatalf("BMC.String() missing %q in %q", n, got)
				}
			}
		})
	}
}

func TestNode_String_Table(t *testing.T) {
	tests := []struct {
		name    string
		node    Node
		needles []string
	}{
		{
			name: "with explicit BMC and two ifaces",
			node: Node{
				Name:   "nid1",
				NID:    1,
				Xname:  "x1000c0s0b0n0",
				Groups: []string{"compute"},
				BMC:    "x1000c0s0b0",
				Ifaces: []Iface{
					{
						MACAddr: "00:00:00:00:00:01",
						IPAddrs: []IfaceIP{
							{Name: "n1", IPAddr: "172.16.0.1"},
							{Name: "n2", IPAddr: "172.16.0.2"},
						},
					},
					{
						MACAddr: "de:ad:be:ef:00:02",
						IPAddrs: []IfaceIP{
							{Name: "n3", IPAddr: "172.16.0.3"},
						},
					},
				},
			},
			needles: []string{
				`name="nid1"`,
				`nid=1`,
				`xname=x1000c0s0b0n0`,
				`bmc=x1000c0s0b0`,
				`groups=[compute]`,
				`mac_addr=00:00:00:00:00:01`,
				`network="n1" ip_addr=172.16.0.1`,
				`network="n2" ip_addr=172.16.0.2`,
				`mac_addr=de:ad:be:ef:00:02`,
				`network="n3" ip_addr=172.16.0.3`,
			},
		},
		{
			name: "derive BMC from xname, empty groups and ifaces",
			node: Node{
				Name:   "nid2",
				NID:    2,
				Xname:  "x1000c0s1b0n0",
				Groups: []string{},
				// BMC empty since it will be derived from node xname
				Ifaces: nil,
			},
			needles: []string{
				`name="nid2"`,
				`nid=2`,
				`xname=x1000c0s1b0n0`,
				`bmc=x1000c0s1b0`,
				`groups=[]`,
				`interfaces=[`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.node.String()
			for _, n := range tt.needles {
				if !strings.Contains(got, n) {
					t.Fatalf("Node.String() missing %q in %q", n, got)
				}
			}
		})
	}
}

func TestNode_ResolveBMC_Table(t *testing.T) {
	tests := []struct {
		name    string
		node    Node
		wantBMC string
		wantErr bool
	}{
		{
			name:    "BMC explicitly set wins",
			node:    Node{Name: "n1", Xname: "x1000c0s0b0n0", BMC: "x1000c0s0b0"},
			wantBMC: "x1000c0s0b0",
			wantErr: false,
		},
		{
			name:    "derive from Xname when BMC unset",
			node:    Node{Name: "n2", Xname: "x1000c0s1b0n0", BMC: ""},
			wantBMC: "x1000c0s1b0",
			wantErr: false,
		},
		{
			name:    "invalid Xname falls back to (none)",
			node:    Node{Name: "bad", Xname: "not-an-xname", BMC: ""},
			wantBMC: BMCEmptyString,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotBMC, err := tt.node.ResolveBMC()
			if gotBMC != tt.wantBMC {
				t.Fatalf("ResolveBMC() bmc = %q, want %q", gotBMC, tt.wantBMC)
			}
			if (err != nil) != tt.wantErr {
				t.Fatalf("ResolveBMC() error presence=%v, wantErr=%v (err=%v)", err != nil, tt.wantErr, err)
			}
		})
	}
}

func TestIface_String_Table(t *testing.T) {
	tests := []struct {
		name string
		ifc  Iface
		want string
	}{
		{
			name: "no IPs",
			ifc:  Iface{MACAddr: "aa:bb:cc:dd:ee:ff", IPAddrs: nil},
			want: `mac_addr=aa:bb:cc:dd:ee:ff ip_addrs=[]`,
		},
		{
			name: "one IP",
			ifc: Iface{
				MACAddr: "00:00:00:00:00:01",
				IPAddrs: []IfaceIP{{Name: "n1", IPAddr: "172.16.0.1"}},
			},
			want: `mac_addr=00:00:00:00:00:01 ip_addrs=[ip0={network="n1" ip_addr=172.16.0.1}]`,
		},
		{
			name: "two IPs",
			ifc: Iface{
				MACAddr: "00:00:00:00:00:02",
				IPAddrs: []IfaceIP{
					{Name: "n1", IPAddr: "172.16.0.1"},
					{Name: "n2", IPAddr: "172.16.0.2"},
				},
			},
			want: `mac_addr=00:00:00:00:00:02 ip_addrs=[ip0={network="n1" ip_addr=172.16.0.1} ip1={network="n2" ip_addr=172.16.0.2}]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.ifc.String()
			if got != tt.want {
				t.Fatalf("Iface.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIfaceIP_String(t *testing.T) {
	tests := []struct {
		name string
		ip   IfaceIP
		want string
	}{
		{
			name: "simple",
			ip:   IfaceIP{Name: "nw", IPAddr: "1.2.3.4"},
			want: `network="nw" ip_addr=1.2.3.4`,
		},
		{
			name: "empty network ok",
			ip:   IfaceIP{Name: "", IPAddr: "10.0.0.1"},
			want: `network="" ip_addr=10.0.0.1`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.ip.String()
			if got != tt.want {
				t.Fatalf("IfaceIP.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

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
