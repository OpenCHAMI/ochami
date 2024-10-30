package discover

import (
	"fmt"
	"net/url"

	"github.com/google/uuid"
	"github.com/openchami/schemas/schemas"
	"github.com/synackd/ochami/internal/client"
	"github.com/synackd/ochami/internal/log"
	"github.com/synackd/ochami/internal/xname"
)

// NodeList is simply a list of Nodes. Data from a payload file is unmarshalled
// into this.
type NodeList []Node

func (nl NodeList) String() string {
	nlStr := "["
	for idx, node := range nl {
		if idx == 0 {
			nlStr += fmt.Sprintf("node%d={%s}", idx, node)
		} else {
			nlStr += fmt.Sprintf(" node%d={%s}", idx, node)
		}
	}
	nlStr += "]"

	return nlStr
}

// Node represents a node entry in a payload file. Multiple of these are send to
// SMD to "discover" them.
type Node struct {
	Name   string  `json:"name"`
	NID    int64   `json:"nid"`
	Xname  string  `json:"xname"`
	BMCMac string  `json:"bmc_mac"`
	BMCIP  string  `json:"bmc_ip"`
	Ifaces []Iface `json:"interfaces"`
}

func (n Node) String() string {
	nStr := fmt.Sprintf("name=%q nid=%d xname=%s bmc_mac=%s bmc_ip=%s interfaces=[",
		n.Name, n.NID, n.Xname, n.BMCMac, n.BMCIP)
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
	MACAddr string    `json:"mac_addr"`
	IPAddrs []IfaceIP `json:"ip_addrs"`
}

func (i Iface) String() string {
	ipStr := "ip_addrs=["
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
	Network string `json:"network"`
	IPAddr  string `json:"ip_addr"`
}

func (i IfaceIP) String() string {
	return fmt.Sprintf("network=%q ip_addr=%s", i.Network, i.IPAddr)
}

// DiscoveryInfoV2 is given the baseURI for the cluster and a NodeList
// (presumably read from a file) and generates the SMD structures that can be
// passed to Ochami send functions directly. This function represents
// "discovering" nodes and returning the information that would be sent to SMD.
// Fake discovery is similar to real discovery (like
// [Magellan](https://github.com/OpenCHAMI/magellan) would do), except the
// information is sourced from a file instead of dynamically reaching out to
// BMCs.
func DiscoveryInfoV2(baseURI string, nl NodeList) (client.RedfishEndpointSliceV2, []client.EthernetInterface, error) {
	var rfes client.RedfishEndpointSliceV2
	var ifaces []client.EthernetInterface
	base, err := url.Parse(baseURI)
	if err != nil {
		return rfes, ifaces, fmt.Errorf("invalid URI: %s", baseURI)
	}
	for _, node := range nl {
		log.Logger.Debug().Msgf("generating redfish structure for node with xname %s", node.Xname)
		var rfe client.RedfishEndpointV2

		// Differentiate node Xname from BMC Xname
		bmcXname, err := xname.NodeXnameToBMCXname(node.Xname)
		if err != nil {
			log.Logger.Warn().Err(err).Msgf("node %s: falling back to node xname as BMC xname", node.Xname)
			bmcXname = node.Xname
		}

		// Populate rfe base data
		rfe.Name = node.Name
		rfe.Type = "NodeBMC"
		rfe.ID = bmcXname
		rfe.MACAddr = node.BMCMac
		rfe.IPAddress = node.BMCIP
		rfe.SchemaVersion = 1  // Tells SMD to use new (v2) parsing code

		// Deduplication maps for fake BMC Managers and Systems
		systemMap := make(map[string]string)
		managerMap := make(map[string]string)

		// Create fake BMC "System" for node if it doesn't already exist
		if _, ok := systemMap[node.Xname]; !ok {
			log.Logger.Debug().Msgf("node %s: generating fake BMC System", node.Xname)
			base.Path = "/redfish/v1/Systems/" + node.Xname

			s := client.System{
				URI: base.String(),
				Name: node.Name,
			}

			// Create unique identifier for system
			if sysUUID, err := uuid.NewRandom(); err != nil {
				log.Logger.Warn().Err(err).Msgf("node %s: could not generate UUID for fake BMC System, it will be zero", node.Xname)
			} else {
				s.UUID = sysUUID.String()
			}

			// Node interfaces
			for idx, iface := range node.Ifaces {
				newIface := schemas.EthernetInterface{
					Name:        node.Xname,
					Description: fmt.Sprintf("Interface %d for %s", idx, node.Name),
					MAC:         iface.MACAddr,
					IP:          iface.IPAddrs[0].IPAddr,
				}
				s.EthernetInterfaces = append(s.EthernetInterfaces, newIface)
				SMDIface := client.EthernetInterface{
					ComponentID: newIface.Name,
					Type:        "Node",
					Description: newIface.Description,
					MACAddress:  newIface.MAC,
				}
				for _, ip := range iface.IPAddrs {
					SMDIface.IPAddresses = append(SMDIface.IPAddresses, client.EthernetIP{
						IPAddress: ip.IPAddr,
						Network:   ip.Network,
					})
				}
				ifaces = append(ifaces, SMDIface)
			}

			systemMap[node.Xname] = "present"
			log.Logger.Debug().Msgf("node %s: generated system: %v", node.Xname, s)
			rfe.Systems = append(rfe.Systems, s)
		} else {
			log.Logger.Debug().Msgf("node %s: fake BMC System already exists, skipping creation", node.Xname)
		}

		// Create fake BMC "Manager" for node if it doesn't already exist
		// BMC interface
		if _, ok := managerMap[bmcXname]; !ok {
			log.Logger.Debug().Msgf("BMC %s: generating fake BMC Manager", bmcXname)
			base.Path = "/redfish/v1/Managers/" + bmcXname

			m := client.Manager{
				System: client.System{
					URI: base.String(),
					Name: bmcXname,
				},
				Type: "NodeBMC",
			}

			// Create unique identifier for manager
			if mngerUUID, err := uuid.NewRandom(); err != nil {
				log.Logger.Warn().Err(err).Msgf("BMC %s: could not generate UUID for fake BMC Manager, it will be zero", bmcXname)
			} else {
				m.UUID = mngerUUID.String()
				rfe.UID = mngerUUID  // Redfish UUID will be fake Manager's UUID
			}

			// BMC interface
			ifaceBMC := schemas.EthernetInterface{
				Name:        bmcXname,
				Description: fmt.Sprintf("Interface for BMC %s", bmcXname),
				MAC:         node.BMCMac,
				IP:          node.BMCIP,
			}
			m.EthernetInterfaces = append(m.EthernetInterfaces, ifaceBMC)
			managerMap[bmcXname] = "present"
			log.Logger.Debug().Msgf("BMC %s: generated manager: %v", bmcXname, m)
			rfe.Managers = append(rfe.Managers, m)
		} else {
			log.Logger.Debug().Msgf("BMC %s: fake BMC Manager already exists, skipping creation", bmcXname)
		}
		rfes.RedfishEndpoints = append(rfes.RedfishEndpoints, rfe)
	}
	return rfes, ifaces, nil
}