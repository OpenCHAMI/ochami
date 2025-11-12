package discover

import (
	"fmt"
	"net/url"

	"github.com/google/uuid"
	"github.com/openchami/schemas/schemas"

	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client/smd"
	"github.com/OpenCHAMI/ochami/pkg/xname"
)

// DiscoveryInfoV2 is given the baseURI for the cluster and a NodeList
// (presumably read from a file) and generates the SMD structures that can be
// passed to Ochami send functions directly. This function represents
// "discovering" nodes and returning the information that would be sent to SMD.
// Fake discovery is similar to real discovery (like
// [Magellan](https://github.com/OpenCHAMI/magellan) would do), except the
// information is sourced from a file instead of dynamically reaching out to
// BMCs.
func DiscoveryInfoV2(baseURI string, di DiscoveryItems) (smd.ComponentSlice, smd.RedfishEndpointSliceV2, []smd.EthernetInterface, error) {
	var (
		comps  smd.ComponentSlice
		rfes   smd.RedfishEndpointSliceV2
		ifaces []smd.EthernetInterface
	)
	base, err := url.Parse(baseURI)
	if err != nil {
		return comps, rfes, ifaces, fmt.Errorf("invalid URI: %s", baseURI)
	}

	var (
		compMap   = make(map[string]string) // Deduplication map for SMD Components
		systemMap = make(map[string]string) // Deduplication map for BMC Systems

		// RedfishEndpoints for each BMC.
		//
		// The primary key is the BMC MAC address, but BMC name (if present) and xname will also
		// be used as keys (alongside MAC), which are used when mapping a node to its BMC by either
		// the BMC name or xname.
		bmcs        = make(map[string]*smd.RedfishEndpointV2)
		bmcsInOrder []*smd.RedfishEndpointV2 // Ordered list of BMCs in map above
	)

	for _, bmc := range di.BMCs {
		log.Logger.Debug().Msgf("generating redfish endpoint structure for bmc %s", bmc)

		// Create SMD RedfishEndpoint for BMC
		var rfe *smd.RedfishEndpointV2
		if r, found := bmcs[bmc.MACAddr]; found {
			rfe = r
		} else {
			// Populate rfe base data
			rfe = &smd.RedfishEndpointV2{}
			rfe.Name = bmc.Name
			rfe.Type = "NodeBMC"
			rfe.ID = bmc.Xname
			rfe.MACAddr = bmc.MACAddr
			rfe.IPAddress = bmc.IPAddr
			rfe.FQDN = bmc.FQDN
			rfe.SchemaVersion = 1 // Tells SMD to use new (v2) parsing code

			// Add RFE to map so it can be referenced by node(s)
			bmcs[rfe.MACAddr] = rfe
			bmcs[rfe.Name] = rfe
			bmcs[rfe.ID] = rfe

			// Add RFE to ordered list that is used to send to SMD
			bmcsInOrder = append(bmcsInOrder, rfe)

			// Create fake Redfish "Manager" for BMC
			log.Logger.Debug().Msgf("BMC %s: generating fake BMC Manager", rfe.ID)
			base.Path = "/redfish/v1/Managers/" + rfe.ID
			m := smd.Manager{
				System: smd.System{
					URI:  base.String(),
					Name: rfe.ID,
				},
				Type: "NodeBMC",
			}

			// Create unique identifier for manager
			if mngerUUID, err := uuid.NewRandom(); err != nil {
				log.Logger.Warn().Err(err).Msgf("BMC %s: could not generate UUID for fake BMC Manager, it will be zero", rfe.ID)
			} else {
				m.UUID = mngerUUID.String()
				rfe.UID = mngerUUID // Redfish UUID will be fake Manager's UUID
			}

			// Network interface for BMC manager
			ifaceBMC := schemas.EthernetInterface{
				Name:        rfe.ID,
				Description: fmt.Sprintf("Interface for BMC %s", rfe.ID),
				MAC:         rfe.MACAddr,
				IP:          rfe.IPAddress,
			}
			m.EthernetInterfaces = append(m.EthernetInterfaces, ifaceBMC)
			log.Logger.Debug().Msgf("BMC %s: generated manager: %v", rfe.ID, m)
			rfe.Managers = append(rfe.Managers, m)
		}
	}

	for _, node := range di.Nodes {
		// Create SMD component for node
		log.Logger.Debug().Msgf("generating component structure for node with xname %s", node.Xname)
		if _, ok := compMap[node.Xname]; !ok {
			comp := smd.Component{
				ID:      node.Xname,
				NID:     node.NID,
				Type:    "Node",
				State:   "On",
				Enabled: true,
			}
			log.Logger.Debug().Msgf("adding component %v", comp)
			compMap[node.Xname] = "present"
			comps.Components = append(comps.Components, comp)
		} else {
			log.Logger.Warn().Msgf("component with xname %s already exists (duplicate?), not adding", node.Xname)
		}

		log.Logger.Debug().Msgf("matching node %s to BMC", node.Xname)

		// Attempt to match node with BMC
		bmcSpec, err := node.ResolveBMC()
		if err != nil {
			// Either:
			//
			// 1. node did not define a BMC name/xname to match to, or
			// 2. an error occurred deriving the BMC xname from the node's xname
			//
			log.Logger.Error().Err(err).Msgf("failed to resolve BMC for node %s", node.Xname)
			continue
		}
		rfe, bmcFound := bmcs[bmcSpec]
		if !bmcFound {
			// BMC spec not defined
			log.Logger.Error().Msgf("no such bmc %q defined for node %s", bmcSpec, node.Xname)
			continue
		}

		// Create fake BMC "System" for node if it doesn't already exist and add to
		// found BMC's Systems list.
		if _, ok := systemMap[node.Xname]; !ok {
			log.Logger.Debug().Msgf("node %s: generating fake BMC System", node.Xname)
			base.Path = "/redfish/v1/Systems/" + node.Xname

			s := smd.System{
				URI:  base.String(),
				Name: node.Name,
			}

			// Create unique identifier for system
			if sysUUID, err := uuid.NewRandom(); err != nil {
				log.Logger.Warn().Err(err).Msgf("node %s: could not generate UUID for fake BMC System, it will be zero", node.Xname)
			} else {
				s.UUID = sysUUID.String()
			}

			// Fake discovery as of v0.5.1 does not have a field to
			// indicate supported power actions, and PCS requires
			// them. We don't have direct configuration for the
			// System struct that contains this either, so in lieu
			// of that, simply add every possible action from from
			// the Redfish Reference 6.5.5.1 ResetType:
			// https://www.dmtf.org/sites/default/files/standards/documents/DSP2046_2023.3.html#aggregate-102
			s.Actions = []string{"On", "ForceOff", "GracefulShutdown", "GracefulRestart", "ForceRestart", "Nmi",
				"ForceOn", "PushPowerButton", "PowerCycle", "Suspend", "Pause", "Resume"}

			// Node interfaces
			for idx, iface := range node.Ifaces {
				newIface := schemas.EthernetInterface{
					Name:        node.Xname,
					Description: fmt.Sprintf("Interface %d for %s", idx, node.Name),
					MAC:         iface.MACAddr,
					IP:          iface.IPAddrs[0].IPAddr,
				}
				s.EthernetInterfaces = append(s.EthernetInterfaces, newIface)
				SMDIface := smd.EthernetInterface{
					ComponentID: newIface.Name,
					Type:        "Node",
					Description: newIface.Description,
					MACAddress:  newIface.MAC,
				}
				for _, ip := range iface.IPAddrs {
					SMDIface.IPAddresses = append(SMDIface.IPAddresses, smd.EthernetIP{
						IPAddress: ip.IPAddr,
						Network:   ip.Name,
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
	}
	for _, rfe := range bmcsInOrder {
		rfes.RedfishEndpoints = append(rfes.RedfishEndpoints, *rfe)
	}
	return comps, rfes, ifaces, nil
}

// DiscoveryInfoV2Deprecated is given the baseURI for the cluster and a NodeList
// (presumably read from a file) and generates the SMD structures that can be
// passed to Ochami send functions directly. This function represents
// "discovering" nodes and returning the information that would be sent to SMD.
// Fake discovery is similar to real discovery (like
// [Magellan](https://github.com/OpenCHAMI/magellan) would do), except the
// information is sourced from a file instead of dynamically reaching out to
// BMCs.
//
// This function is DEPRECATED and will be removed in a future version. It is
// here for compatibility.
func DiscoveryInfoV2Deprecated(baseURI string, nl NodeListDeprecated) (smd.ComponentSlice, smd.RedfishEndpointSliceV2, []smd.EthernetInterface, error) {
	var (
		comps  smd.ComponentSlice
		rfes   smd.RedfishEndpointSliceV2
		ifaces []smd.EthernetInterface
	)
	base, err := url.Parse(baseURI)
	if err != nil {
		return comps, rfes, ifaces, fmt.Errorf("invalid URI: %s", baseURI)
	}

	var (
		compMap     = make(map[string]string)                 // Deduplication map for SMD Components
		systemMap   = make(map[string]string)                 // Deduplication map for BMC Systems
		managerMap  = make(map[string]string)                 // Deduplication map for BMC Managers
		bmcs        = make(map[string]*smd.RedfishEndpointV2) // RedfishEndpoints for each BMC, the key is the mac to the BMC
		bmcsInOrder []*smd.RedfishEndpointV2                  // Contains the same objects as the bmcs map. This maintains the order that the objects were created
	)
	for _, node := range nl.Nodes {
		log.Logger.Debug().Msgf("generating component structure for node with xname %s", node.Xname)
		if _, ok := compMap[node.Xname]; !ok {
			comp := smd.Component{
				ID:      node.Xname,
				NID:     node.NID,
				Type:    "Node",
				State:   "On",
				Enabled: true,
			}
			log.Logger.Debug().Msgf("adding component %v", comp)
			compMap[node.Xname] = "present"
			comps.Components = append(comps.Components, comp)
		} else {
			log.Logger.Warn().Msgf("component with xname %s already exists (duplicate?), not adding", node.Xname)
		}

		log.Logger.Debug().Msgf("generating redfish structure for node with xname %s", node.Xname)

		// Differentiate node Xname from BMC Xname
		bmcXname, err := xname.NodeXnameToBMCXname(node.Xname)
		if err != nil {
			log.Logger.Warn().Err(err).Msgf("node %s: falling back to node xname as BMC xname", node.Xname)
			bmcXname = node.Xname
		}

		var rfe *smd.RedfishEndpointV2
		if r, found := bmcs[node.BMCMac]; found {
			rfe = r
		} else {
			// Populate rfe base data
			rfe = &smd.RedfishEndpointV2{}
			rfe.Name = node.Name
			rfe.Type = "NodeBMC"
			rfe.ID = bmcXname
			rfe.MACAddr = node.BMCMac
			rfe.IPAddress = node.BMCIP
			rfe.FQDN = node.BMCFQDN
			rfe.SchemaVersion = 1 // Tells SMD to use new (v2) parsing code
			bmcs[rfe.MACAddr] = rfe
			bmcsInOrder = append(bmcsInOrder, rfe)
		}

		// Create fake BMC "System" for node if it doesn't already exist
		if _, ok := systemMap[node.Xname]; !ok {
			log.Logger.Debug().Msgf("node %s: generating fake BMC System", node.Xname)
			base.Path = "/redfish/v1/Systems/" + node.Xname

			s := smd.System{
				URI:  base.String(),
				Name: node.Name,
			}

			// Create unique identifier for system
			if sysUUID, err := uuid.NewRandom(); err != nil {
				log.Logger.Warn().Err(err).Msgf("node %s: could not generate UUID for fake BMC System, it will be zero", node.Xname)
			} else {
				s.UUID = sysUUID.String()
			}

			// Fake discovery as of v0.5.1 does not have a field to
			// indicate supported power actions, and PCS requires
			// them. We don't have direct configuration for the
			// System struct that contains this either, so in lieu
			// of that, simply add every possible action from from
			// the Redfish Reference 6.5.5.1 ResetType:
			// https://www.dmtf.org/sites/default/files/standards/documents/DSP2046_2023.3.html#aggregate-102
			s.Actions = []string{"On", "ForceOff", "GracefulShutdown", "GracefulRestart", "ForceRestart", "Nmi",
				"ForceOn", "PushPowerButton", "PowerCycle", "Suspend", "Pause", "Resume"}

			// Node interfaces
			for idx, iface := range node.Ifaces {
				newIface := schemas.EthernetInterface{
					Name:        node.Xname,
					Description: fmt.Sprintf("Interface %d for %s", idx, node.Name),
					MAC:         iface.MACAddr,
					IP:          iface.IPAddrs[0].IPAddr,
				}
				s.EthernetInterfaces = append(s.EthernetInterfaces, newIface)
				SMDIface := smd.EthernetInterface{
					ComponentID: newIface.Name,
					Type:        "Node",
					Description: newIface.Description,
					MACAddress:  newIface.MAC,
				}
				for _, ip := range iface.IPAddrs {
					SMDIface.IPAddresses = append(SMDIface.IPAddresses, smd.EthernetIP{
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

			m := smd.Manager{
				System: smd.System{
					URI:  base.String(),
					Name: bmcXname,
				},
				Type: "NodeBMC",
			}

			// Create unique identifier for manager
			if mngerUUID, err := uuid.NewRandom(); err != nil {
				log.Logger.Warn().Err(err).Msgf("BMC %s: could not generate UUID for fake BMC Manager, it will be zero", bmcXname)
			} else {
				m.UUID = mngerUUID.String()
				rfe.UID = mngerUUID // Redfish UUID will be fake Manager's UUID
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
	}
	for _, rfe := range bmcsInOrder {
		rfes.RedfishEndpoints = append(rfes.RedfishEndpoints, *rfe)
	}
	return comps, rfes, ifaces, nil
}

// AddMemberToGroup adds xname to group, ensuring deduplication.
func AddMemberToGroup(group smd.Group, xname string) smd.Group {
	for _, x := range group.Members.IDs {
		if x == xname {
			return group
		}
	}
	g := group
	g.Members.IDs = append(g.Members.IDs, xname)
	return g
}
