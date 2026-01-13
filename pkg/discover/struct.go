// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package discover

import (
	"fmt"
	"strings"

	"github.com/OpenCHAMI/ochami/pkg/xname"
)

const (
	BMCEmptyString = "(none)"
)

// DiscoveryItems serves as a structure to unmarshal discovery data into. It
// contains a list of BMCs and a list of Nodes.
type DiscoveryItems struct {
	BMCs  []BMC  `json:"bmcs" yaml:"bmcs"`
	Nodes []Node `json:"nodes" yaml:"nodes"`
}

func (di DiscoveryItems) String() string {
	// BMCs
	blStr := "bmcs=["
	for idx, bmc := range di.BMCs {
		if idx == 0 {
			blStr += fmt.Sprintf("bmc%d={%s}", idx, bmc)
		} else {
			blStr += fmt.Sprintf(" bmc%d={%s}", idx, bmc)
		}
	}
	blStr += "]"

	// Nodes
	nlStr := "nodes=["
	for idx, node := range di.Nodes {
		if idx == 0 {
			nlStr += fmt.Sprintf("node%d={%s}", idx, node)
		} else {
			nlStr += fmt.Sprintf(" node%d={%s}", idx, node)
		}
	}
	nlStr += "]"

	// Form entire discovery items string
	diStr := fmt.Sprintf("{%s,%s}", blStr, nlStr)

	return diStr
}

// BMC represents a Baseboard Management Controller for one or more nodes. It
// contains identification and network features that are used to communicate
// with it. A Node can reference the Name or Xname fields of a BMC to link the
// Node to the BMC.
type BMC struct {
	Name    string `json:"name" yaml:"name"`
	Xname   string `json:"xname" yaml:"xname"`
	MACAddr string `json:"mac" yaml:"mac"`
	IPAddr  string `json:"ip" yaml:"ip"`
	FQDN    string `json:"fqdn" yaml:"fqdn"`
}

func (b BMC) String() string {
	return fmt.Sprintf("name=%q xname=%s mac=%s ip=%s fqdn=%s",
		b.Name, b.Xname, b.MACAddr, b.IPAddr, b.FQDN)
}

// Node represents a computer object that posesses identification information,
// one or more network interfaces, optional membership to one or more groups,
// and a linked BMC that is attached to. A Node must be linked to a BMC to be
// known to SMD. A link can either be established by setting the BMC field to
// the name of an existing BMC in the BMCList or leaving it blank/unset and
// extracting the BMC component of the Node's Xname field.
type Node struct {
	Name   string   `json:"name" yaml:"name"`
	NID    int64    `json:"nid" yaml:"nid"`
	Xname  string   `json:"xname" yaml:"xname"`
	Groups []string `json:"groups" yaml:"groups"`
	BMC    string   `json:"bmc" yaml:"bmc"`
	Ifaces []Iface  `json:"interfaces" yaml:"interfaces"`
}

// ResolveBMC resolves the BMC reference of the node. If BMC is set for the
// node, that value will be returned. Otherwise, ResolveBMC attempts to
// determine the BMC xname using the Xname field. If this fails, "(none)" is
// returned along with an error.
func (n Node) ResolveBMC() (bmc_spec string, err error) {
	bmc_spec = BMCEmptyString
	if strings.Trim(n.BMC, " \t") != "" {
		bmc_spec = n.BMC
	} else {
		// Try and generate BMC xname from node xname
		var bmc_xname string
		if bmc_xname, err = xname.NodeXnameToBMCXname(n.Xname); err == nil {
			bmc_spec = bmc_xname
		} else {
			err = fmt.Errorf("failed to resolve BMC xname from node xname: %w", err)
		}
	}
	return bmc_spec, err
}

func (n Node) String() string {
	bmc_spec, _ := n.ResolveBMC()
	nStr := fmt.Sprintf("name=%q nid=%d xname=%s bmc=%s groups=%v interfaces=[",
		n.Name, n.NID, n.Xname, bmc_spec, n.Groups)
	for idx, iface := range n.Ifaces {
		if idx == 0 {
			nStr += fmt.Sprintf("iface%d={%s}", idx, iface)
		} else {
			nStr += fmt.Sprintf(" iface%d={%s}", idx, iface)
		}
	}
	nStr += "]"

	return nStr
}

// Iface represents a single interface with multiple IP addresses. Nodes can
// have multiple of these.
type Iface struct {
	MACAddr string    `json:"mac_addr" yaml:"mac_addr"`
	IPAddrs []IfaceIP `json:"ip_addrs" yaml:"ip_addrs"`
}

func (i Iface) String() string {
	ipStr := fmt.Sprintf("mac_addr=%s ip_addrs=[", i.MACAddr)
	for idx, ip := range i.IPAddrs {
		if idx == 0 {
			ipStr += fmt.Sprintf("ip%d={%s}", idx, ip)
		} else {
			ipStr += fmt.Sprintf(" ip%d={%s}", idx, ip)
		}
	}
	ipStr += "]"

	return ipStr
}

// IfaceIP represents a single IP address of an Iface. An IP address can have an
// associated Network which represents the human-readable name of the network
// the IP address is on. Note that Network is NOT the subnet mask or CIDR of the
// IPAddr.
type IfaceIP struct {
	Name   string `json:"name" yaml:"name"` // Network name (human-readable)
	IPAddr string `json:"ip_addr" yaml:"ip_addr"`
}

func (i IfaceIP) String() string {
	return fmt.Sprintf("network=%q ip_addr=%s", i.Name, i.IPAddr)
}

///////////////////////////
//                       //
// DEPRECATED STRUCTURES //
//                       //
///////////////////////////

// NodeListDeprecated is simply a list of Nodes. The purpose for a list of nodes
// for discovery is so that they can be iterated on to create the necessary
// structures to be sent to SMD.
type NodeListDeprecated struct {
	Nodes []NodeDeprecated `json:"nodes" yaml:"nodes"`
}

func (nl NodeListDeprecated) String() string {
	nlStr := "["
	for idx, node := range nl.Nodes {
		if idx == 0 {
			nlStr += fmt.Sprintf("node%d={%s}", idx, node)
		} else {
			nlStr += fmt.Sprintf(" node%d={%s}", idx, node)
		}
	}
	nlStr += "]"

	return nlStr
}

// NodeDeprecated represents a node entry in a payload file. Multiple of these
// are send to SMD to "discover" them.
//
// This struct is DEPRECATED in favor of the new Node struct above and will be
// removed in a future version. It is present for compatibility.
type NodeDeprecated struct {
	Name    string            `json:"name" yaml:"name"`
	NID     int64             `json:"nid" yaml:"nid"`
	Xname   string            `json:"xname" yaml:"xname"`
	Group   string            `json:"group" yaml:"group"` // DEPRECATED
	Groups  []string          `json:"groups" yaml:"groups"`
	BMCMac  string            `json:"bmc_mac" yaml:"bmc_mac"`
	BMCIP   string            `json:"bmc_ip" yaml:"bmc_ip"`
	BMCFQDN string            `json:"bmc_fqdn" yaml:"bmc_fqdn"`
	Ifaces  []IfaceDeprecated `json:"interfaces" yaml:"interfaces"`
}

func (n NodeDeprecated) String() string {
	nStr := fmt.Sprintf("name=%q nid=%d xname=%s group=%q groups=%v bmc_mac=%s bmc_ip=%s bmc_fqdn=%s interfaces=[",
		n.Name, n.NID, n.Xname, n.Group, n.Groups, n.BMCMac, n.BMCIP, n.BMCFQDN)
	for idx, iface := range n.Ifaces {
		if idx == 0 {
			nStr += fmt.Sprintf("iface%d={%s}", idx, iface)
		} else {
			nStr += fmt.Sprintf(" iface%d={%s}", idx, iface)
		}
	}
	nStr += "]"

	return nStr
}

// IfaceDeprecated represents a single interface with multiple IP addresses.
// Nodes can have multiple of these.
type IfaceDeprecated struct {
	MACAddr string              `json:"mac_addr" yaml:"mac_addr"`
	IPAddrs []IfaceIPDeprecated `json:"ip_addrs" yaml:"ip_addrs"`
}

func (i IfaceDeprecated) String() string {
	ipStr := fmt.Sprintf("mac_addr=%s ip_addrs=[", i.MACAddr)
	for idx, ip := range i.IPAddrs {
		if idx == 0 {
			ipStr += fmt.Sprintf("ip%d={%s}", idx, ip)
		} else {
			ipStr += fmt.Sprintf(" ip%d={%s}", idx, ip)
		}
	}
	ipStr += "]"

	return ipStr
}

// IfaceIPDeprecated represents a single IP address of an Iface. An IP address
// can have an associated Network which represents the human-readable name of
// the network the IP address is on. Note that Network is NOT the subnet mask or
// CIDR of the IPAddr.
type IfaceIPDeprecated struct {
	Network string `json:"network" yaml:"network"`
	IPAddr  string `json:"ip_addr" yaml:"ip_addr"`
}

func (i IfaceIPDeprecated) String() string {
	return fmt.Sprintf("network=%q ip_addr=%s", i.Network, i.IPAddr)
}
