// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client"
	"github.com/OpenCHAMI/ochami/pkg/client/smd"
	"github.com/OpenCHAMI/ochami/pkg/discover"
)

func discoverStaticDeprecatedFormat(cmd *cobra.Command) bool {
	discoveryData := make(map[string]([]map[string]any))
	if cmd.Flag("data").Changed {
		handlePayload(cmd, &discoveryData)
	} else {
		handlePayloadStdin(cmd, &discoveryData)
	}
	deprecatedFormat := false
	for _, node := range discoveryData["nodes"] {
		if _, bmcIpFound := node["bmc_ip"]; bmcIpFound {
			log.Logger.Warn().Msg("deprecated nodes key bmc_ip found, using old discovery format")
			deprecatedFormat = true
			break
		} else if _, bmcMacFound := node["bmc_mac"]; bmcMacFound {
			log.Logger.Warn().Msg("deprecated nodes key bmc_mac found, using old discovery format")
			deprecatedFormat = true
			break
		}
	}
	return deprecatedFormat
}

// discoverStaticCmd represents the discover-static command
var discoverStaticCmd = &cobra.Command{
	Use:   "static [--overwrite] [-d (<data> | @<path>)] [-f <format>]",
	Short: "Populate SMD with data statically",
	Long: `Populate SMD using static data. This data can be from a file (if an
argument is passed) or from standard input. This "fake" discovery
data is read by ochami, which then interprets the data and figures
out which SMD data structures to create. This is meant to be a
reproduceable alternative to dynamic discovery as is done by
Magellan.

The format of the payload file is an array of node specifications.
In YAML, each node entry would look something like:

nodes:
- name: node01
  nid: 1
  xname: x1000c1s7b0n0
  bmc_mac: de:ca:fc:0f:ee:ee
  bmc_ip: 172.16.0.101
  groups:
  - compute
  - slurm
  interfaces:
  - mac_addr: de:ad:be:ee:ee:f1
    ip_addrs:
    - name: internal
      ip_addr: 172.16.0.1
  - mac_addr: de:ad:be:ee:ee:f2
    ip_addrs:
    - name: external
      ip_addr: 10.15.3.100
  - mac_addr: 02:00:00:91:31:b3
    ip_addrs:
    - name: HSN
      ip_addr: 192.168.0.1

See ochami-discover(1) for more details.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Without a base URI, we cannot do anything
		smdBaseURI, err := getBaseURISMD(cmd)
		if err != nil {
			log.Logger.Error().Err(err).Msg("failed to get base URI for SMD")
			logHelpError(cmd)
			os.Exit(1)
		}

		// This endpoint requires authentication, so a token is needed
		setToken(cmd)
		checkToken(cmd)

		// Create client to make request to SMD
		smdClient, err := smd.NewClient(smdBaseURI, insecure)
		if err != nil {
			log.Logger.Error().Err(err).Msg("error creating new SMD client")
			logHelpError(cmd)
			os.Exit(1)
		}

		// Check if a CA certificate was passed and load it into client if valid
		useCACert(smdClient.OchamiClient)

		if cmd.Flag("overwrite").Changed {
			log.Logger.Warn().Msg("--overwrite passed; overwriting any existing data")
		}

		// Declare structures to send to SMD here so either discovery
		// format can be used to generate them.
		var (
			comps  smd.ComponentSlice
			rfes   smd.RedfishEndpointSliceV2
			ifaces []smd.EthernetInterface
		)

		// This is a temporary structure that keeps basic node
		// information that is common between the deprecated format and
		// the new format for discovery. It's here so that we don't have
		// to duplicate loops due to the differing formats. Once the
		// deprecated format is removed, this can go away.
		type NodeCommon struct {
			Name   string
			Xname  string
			Group  string
			Groups []string
		}
		var nodesCommon []NodeCommon

		// Read data from file or stdin into map to determine which
		// discovery method to use.
		useDeprecatedFormat := discoverStaticDeprecatedFormat(cmd)
		if useDeprecatedFormat {
			log.Logger.Warn().Msg("using deprecated discovery format which will be removed in a future version")

			// Read data from file or stdin
			nodes := discover.NodeListDeprecated{}
			if cmd.Flag("data").Changed {
				handlePayload(cmd, &nodes)
			} else {
				handlePayloadStdin(cmd, &nodes)
			}
			log.Logger.Debug().Msgf("read %d nodes", len(nodes.Nodes))
			log.Logger.Debug().Msgf("nodes: %s", nodes)

			// Add nodes to node list in common format
			for _, n := range nodes.Nodes {
				commonNode := NodeCommon{
					Name:   n.Name,
					Xname:  n.Xname,
					Group:  n.Group,
					Groups: n.Groups,
				}
				nodesCommon = append(nodesCommon, commonNode)
			}

			// Put together payload for different endpoints
			log.Logger.Debug().Msg("generating redfish structures to send to SMD")
			var err error
			comps, rfes, ifaces, err = discover.DiscoveryInfoV2Deprecated(smdBaseURI, nodes)
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to construct structures to send to SMD")
				logHelpError(cmd)
				os.Exit(1)
			}
			log.Logger.Debug().Msgf("generated redfish structures: %v", rfes.RedfishEndpoints)
		} else {
			// Read data from file or stdin
			items := discover.DiscoveryItems{}
			if cmd.Flag("data").Changed {
				handlePayload(cmd, &items)
			} else {
				handlePayloadStdin(cmd, &items)
			}
			log.Logger.Debug().Msgf("read %d bmcs", len(items.BMCs))
			log.Logger.Debug().Msgf("bmcs: %s", items.BMCs)
			log.Logger.Debug().Msgf("read %d nodes", len(items.Nodes))
			log.Logger.Debug().Msgf("nodes: %s", items.Nodes)

			// Add nodes to node list in common format
			for _, n := range items.Nodes {
				commonNode := NodeCommon{
					Name:   n.Name,
					Xname:  n.Xname,
					Groups: n.Groups,
				}
				nodesCommon = append(nodesCommon, commonNode)
			}

			// Put together payload for different endpoints
			log.Logger.Debug().Msg("generating redfish structures to send to SMD")
			var err error
			comps, rfes, ifaces, err = discover.DiscoveryInfoV2(smdBaseURI, items)
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to construct structures to send to SMD")
				logHelpError(cmd)
				os.Exit(1)
			}
			log.Logger.Debug().Msgf("generated redfish structures: %v", rfes.RedfishEndpoints)
		}

		// Send Component requests
		// NOTE: These are sent *before* the RedfishEndpoints so the
		// user-specified NIDs get used instead of the SMD-generated
		// ones. The NIDs generated by SMD assume starting at 1 and
		// increment up in the order added.
		compErrorsOccurred := false
		if cmd.Flag("overwrite").Changed {
			// Send a PUT if --overwrite specified to overwrite any existing components
			_, errs, err := smdClient.PutComponents(comps, token)
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to add/overwrite components in SMD")
				compErrorsOccurred = true
			}
			for _, err := range errs {
				if err != nil {
					var errMsg string
					if errors.Is(err, client.UnsuccessfulHTTPError) {
						errMsg = "SMD component request yielded unsuccessful HTTP response"
					} else {
						errMsg = "failed to add/overwrite component in SMD"
					}
					log.Logger.Error().Err(err).Msg(errMsg)
					compErrorsOccurred = true
				}
			}

			// The SMD Components API does not modify the NID for
			// PUTs. Thus, we explicitly do it with a PATCH to a
			// specific endpoint that does it.
			if _, err := smdClient.PatchComponentsNID(comps, token); err != nil {
				log.Logger.Error().Err(err).Msg("failed to update NIDs for components in SMD")
				compErrorsOccurred = true
			}
		} else {
			// Otherwise send a normal POST
			_, err = smdClient.PostComponents(comps, token)
			if err != nil {
				var errMsg string
				if errors.Is(err, client.UnsuccessfulHTTPError) {
					errMsg = "SMD component request yielded unsuccessful HTTP response"
				} else {
					errMsg = "failed to add components to SMD"
				}
				log.Logger.Error().Err(err).Msg(errMsg)
				compErrorsOccurred = true
			}
		}

		// Send RedfishEndpoint requests
		var (
			rfeErrorsOccurred bool = false
			rfeHenvs          []client.HTTPEnvelope
			rfeErrs           []error
			rfeErr            error
		)
		if cmd.Flag("overwrite").Changed {
			// SMD's RedfishEndpoint API for PUT behaves more like
			// PATCH. In other words, the RedfishEndpoint must exist
			// _first_ before PUTting. This means that, to get
			// normal PUT behavior, we have to first try to POST,
			// then, if 409 is returned, try to PUT.
			for _, rfe := range rfes.RedfishEndpoints {
				// Attempt to POST the redfish endpoint
				rfeListWrapper := smd.RedfishEndpointSliceV2{
					RedfishEndpoints: []smd.RedfishEndpointV2{rfe},
				}
				rfeHenvs, rfeErrs, rfeErr = smdClient.PostRedfishEndpointsV2(rfeListWrapper, token)

				if rfeErr != nil {
					// An error in the function occurred,
					// err for this redfish endpoint and
					// move on.
					log.Logger.Error().Err(rfeErr).Msg("failed to add redfish endpoint to SMD")
					rfeErrorsOccurred = true
					continue
				}

				if rfeErrs[0] != nil {
					// An HTTP error occurred
					if errors.Is(rfeErrs[0], client.UnsuccessfulHTTPError) {
						if rfeHenvs[0].StatusCode == 409 {
							// RFE exists, PUT it
							log.Logger.Info().Msgf("redfish endpoint %s exists, attempting to update it", rfe.ID)
							_, putErrs, putErr := smdClient.PutRedfishEndpointsV2(rfeListWrapper, token)
							if putErr != nil {
								log.Logger.Error().Err(putErr).Msg("failed to update existing redfish endpoint in SMD")
								rfeErrorsOccurred = true
								continue
							}
							if putErrs[0] != nil {
								var errMsg string
								if errors.Is(putErrs[0], client.UnsuccessfulHTTPError) {
									errMsg = "SMD redfish endpoint PUT yielded unsuccessful HTTP response"
								} else {
									errMsg = "failed to update existing redfish endpoint in SMD"
								}
								log.Logger.Error().Err(putErrs[0]).Msg(errMsg)
								rfeErrorsOccurred = true
								continue
							}
						} else {
							// Some other HTTP error occurred, err
							log.Logger.Error().Err(rfeErrs[0]).Msg("SMD redfish endpoint POST yielded non-409 (duplicate) failure")
							rfeErrorsOccurred = true
							continue
						}
					} else {
						log.Logger.Error().Err(rfeErrs[0]).Msg("failed to add redfish endpoint to SMD")
						rfeErrorsOccurred = true
						continue
					}
				}
			}
		} else {
			// --overwrite was not passed, perform regular POST.
			_, rfeErrs, rfeErr = smdClient.PostRedfishEndpointsV2(rfes, token)
			if rfeErr != nil {
				log.Logger.Error().Err(rfeErr).Msg("failed to add redfish endpoints to SMD")
				rfeErrorsOccurred = true
			}
			for _, err := range rfeErrs {
				if err != nil {
					var errMsg string
					if errors.Is(err, client.UnsuccessfulHTTPError) {
						errMsg = "SMD redfish endpoint request yielded unsuccessful HTTP response"
					} else {
						if cmd.Flag("overwrite").Changed {
							errMsg = "failed to add/overwrite redfish endpoint in SMD"
						} else {
							errMsg = "failed to add redfish endpoint to SMD"
						}
					}
					log.Logger.Error().Err(err).Msg(errMsg)
					rfeErrorsOccurred = true
				}
			}
		}

		// Send EthernetInterface requests
		var (
			ifaceErrorsOccurred bool = false
			ifaceHenvs          []client.HTTPEnvelope
			ifaceErrs           []error
			ifaceErr            error
		)
		// Get discovery version value (err handled in cmd.Args).
		// Send EthernetInterfaces to SMD if discoverVersion is 1.
		if discoveryVersion == discover.DiscoveryMethodV1 {
			if cmd.Flag("overwrite").Changed {
				// SMD's EthernetInterface API does not allow the PUT
				// method. Instead, we loop over each ethernet interface
				// to add and attempt a POST. If a 409 is returned for
				// that interface, a PATCH is attempted. Otherwise, an
				// error has occurred.
				for _, iface := range ifaces {
					// Attempt to POST the ethernet interface
					ifaceListWrapper := []smd.EthernetInterface{iface}
					ifaceHenvs, ifaceErrs, ifaceErr = smdClient.PostEthernetInterfaces(ifaceListWrapper, token)

					if ifaceErr != nil {
						// An error in the function occurred, err for
						// this interface and move on.
						log.Logger.Error().Err(ifaceErr).Msg("failed to add ethernet interface to SMD")
						ifaceErrorsOccurred = true
						continue
					}

					if ifaceErrs[0] != nil {
						// An HTTP error occurred
						if errors.Is(ifaceErrs[0], client.UnsuccessfulHTTPError) {
							if ifaceHenvs[0].StatusCode == 409 {
								// Ethernet interface exists, patch it
								log.Logger.Info().Msgf("ethernet interface with MAC address %s exists, attempting to update it", iface.MACAddress)
								_, patchErrs, patchErr := smdClient.PatchEthernetInterfaces(ifaceListWrapper, token)
								if patchErr != nil {
									log.Logger.Error().Err(patchErr).Msg("failed to update existing ethernet interface in SMD")
									ifaceErrorsOccurred = true
									continue
								}
								if patchErrs[0] != nil {
									var errMsg string
									if errors.Is(patchErrs[0], client.UnsuccessfulHTTPError) {
										errMsg = "SMD ethernet interface PATCH yielded unsuccessful HTTP response"
									} else {
										errMsg = "failed to update existing ethernet interface in SMD"
									}
									log.Logger.Error().Err(patchErrs[0]).Msg(errMsg)
									ifaceErrorsOccurred = true
									continue
								}
							} else {
								// Some other HTTP error occurred, err
								log.Logger.Error().Err(ifaceErrs[0]).Msg("SMD ethernet interface POST yield non-409 (duplicate) failure")
								ifaceErrorsOccurred = true
								continue
							}
						} else {
							log.Logger.Error().Err(ifaceErrs[0]).Msg("failed to add ethernet interface to SMD")
							ifaceErrorsOccurred = true
							continue
						}
					}
				}
			} else {
				// --overwrite was not passed, perform regular POST.
				_, ifaceErrs, ifaceErr = smdClient.PostEthernetInterfaces(ifaces, token)
				if ifaceErr != nil {
					log.Logger.Error().Err(ifaceErr).Msg("failed to add ethernet interfaces to SMD")
					ifaceErrorsOccurred = true
				}
				for _, err := range ifaceErrs {
					if err != nil {
						var errMsg string
						if errors.Is(err, client.UnsuccessfulHTTPError) {
							errMsg = "SMD ethernet interface request yielded unsuccessful HTTP response"
						} else {
							errMsg = "failed to add ethernet interface to SMD"
						}
						log.Logger.Error().Err(err).Msg(errMsg)
						ifaceErrorsOccurred = true
					}
				}
			}
		}

		// Put together list of groups to add and which components to add to those groups
		groupsToAdd := make(map[string]smd.Group)
		for _, node := range nodesCommon {
			// nodesCommon is temporary. Once the deprecated
			// discovery format is removed, this can be switched to
			// items.Nodes.

			// node.Group IS DEPRECATED IN FAVOR OF node.groups. This block should be
			// deleted when node.Group is removed.
			//
			// For now, we merge node.Group with node.Groups. Since we use a dictionary for
			// deduplication, this is trivial.
			if node.Group != "" {
				if len(strings.Trim(node.Name, " \t")) == 0 {
					log.Logger.Warn().Msgf("node %s contains 'group', which is deprecated; use 'groups' instead", node.Xname)
				} else {
					log.Logger.Warn().Msgf("node %s (%s) contains 'group', which is deprecated; use 'groups' instead", node.Xname, node.Name)
				}
				if g, ok := groupsToAdd[node.Group]; !ok {
					// Group doesn't exist yet, populate groupsToAdd with it
					newGroup := smd.Group{
						Label:       node.Group,
						Description: fmt.Sprintf("The %s group", node.Group),
					}
					groupsToAdd[node.Group] = discover.AddMemberToGroup(newGroup, node.Xname)
				} else {
					// Update group membership with new node in groupsToAdd map
					groupsToAdd[node.Group] = discover.AddMemberToGroup(g, node.Xname)
				}
			}
			for _, group := range node.Groups {
				if g, ok := groupsToAdd[group]; !ok {
					// Group doesn't exist yet, populate groupsToAdd with it
					newGroup := smd.Group{
						Label:       group,
						Description: fmt.Sprintf("The %s group", group),
					}
					groupsToAdd[group] = discover.AddMemberToGroup(newGroup, node.Xname)
				} else {
					// Update group membership with new node in groupsToAdd map
					groupsToAdd[group] = discover.AddMemberToGroup(g, node.Xname)
				}
			}
		}
		groupList := make([]smd.Group, len(groupsToAdd))
		var idx = 0
		for _, g := range groupsToAdd {
			groupList[idx] = g
			idx++
		}

		// Add groups and components to those groups
		var (
			groupErrorsOccurred bool = false
			groupHenvs          []client.HTTPEnvelope
			groupErrs           []error
			groupErr            error
		)
		if cmd.Flag("overwrite").Changed {
			// SMD's groups API does not allow the PUT method.
			// Instead, we loop over each group to add and attempt a
			// POST. Iff a 409 is returned for that interface, a
			// PATCH is attempted. Otherwise, an error has occurred.
			for _, group := range groupList {
				// Attempt to POST the group
				groupListWrapper := []smd.Group{group}
				groupHenvs, groupErrs, groupErr = smdClient.PostGroups(groupListWrapper, token)

				if groupErr != nil {
					// An error in the function occurred,
					// err for this group and move on.
					log.Logger.Error().Err(groupErr).Msg("failed to add group to SMD")
					groupErrorsOccurred = true
					continue
				}

				if groupErrs[0] != nil {
					// An HTTP error occurred
					if errors.Is(groupErrs[0], client.UnsuccessfulHTTPError) {
						if groupHenvs[0].StatusCode == 409 {
							// Group exists, patch it
							log.Logger.Info().Msgf("group %s exists, attempting to update it", group.Label)
							_, patchErrs, patchErr := smdClient.PatchGroups(groupListWrapper, token)
							if patchErr != nil {
								log.Logger.Error().Err(patchErr).Msg("failed to update existing group in SMD")
								groupErrorsOccurred = true
								continue
							}
							if patchErrs[0] != nil {
								var errMsg string
								if errors.Is(patchErrs[0], client.UnsuccessfulHTTPError) {
									errMsg = "SMD group PATCH yielded unsuccessful HTTP response"
								} else {
									errMsg = "failed to update existing group in SMD"
								}
								log.Logger.Error().Err(patchErrs[0]).Msg(errMsg)
								groupErrorsOccurred = true
								continue
							}
						} else {
							// Some other HTTP error occurred, err
							log.Logger.Error().Err(groupErrs[0]).Msg("SMD group POST yielded non-409 (duplicate) failure")
							groupErrorsOccurred = true
							continue
						}
					} else {
						log.Logger.Error().Err(groupErrs[0]).Msg("failed to add group to SMD")
						groupErrorsOccurred = true
						continue
					}
				}
			}
		} else {
			_, groupErrs, groupErr = smdClient.PostGroups(groupList, token)
			if groupErr != nil {
				log.Logger.Error().Err(groupErr).Msg("failed to add groups to SMD")
				groupErrorsOccurred = true
			}
			for _, err := range groupErrs {
				if err != nil {
					var errMsg string
					if errors.Is(err, client.UnsuccessfulHTTPError) {
						errMsg = "SMD groups request yielded unsuccessful HTTP response"
					} else {
						errMsg = "failed to add groups to SMD"
					}
					log.Logger.Error().Err(err).Msg(errMsg)
					groupErrorsOccurred = true
				}
			}
		}

		// Notify user if any request errors occurred
		exitStatus := 0
		if compErrorsOccurred || rfeErrorsOccurred || ifaceErrorsOccurred || groupErrorsOccurred {
			logHelpError(cmd)
		}
		if compErrorsOccurred {
			log.Logger.Warn().Msg("component requests completed with errors")
			exitStatus = 1
		}
		if rfeErrorsOccurred {
			log.Logger.Warn().Msg("redfish endpoint requests completed with errors")
			exitStatus = 1
		}
		if ifaceErrorsOccurred {
			log.Logger.Warn().Msg("ethernet interface requests completed with errors")
			exitStatus = 1
		}
		if groupErrorsOccurred {
			log.Logger.Warn().Msg("group requests completed with errors")
			exitStatus = 1
		}
		os.Exit(exitStatus)
	},
}

func init() {
	discoverStaticCmd.Flags().Var(&discoveryVersion, "discovery-version", "set version for discovery method to use")
	discoverStaticCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")
	discoverStaticCmd.Flags().VarP(&formatInput, "format-input", "f", "format of input payload data (json,json-pretty,yaml)")
	discoverStaticCmd.Flags().Bool("overwrite", false, "overwrite any existing information instead of failing")

	discoverStaticCmd.RegisterFlagCompletionFunc("format-input", completionFormatData)
	discoverStaticCmd.RegisterFlagCompletionFunc("discovery-version", completionDiscoveryVersion)

	discoverCmd.AddCommand(discoverStaticCmd)
}
