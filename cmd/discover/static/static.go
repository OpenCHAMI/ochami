// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package static

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/internal/log"
	"github.com/openchami/ochami/pkg/client"
	"github.com/openchami/ochami/pkg/client/smd"
	"github.com/openchami/ochami/pkg/discover"
)

// nodeCommon keeps basic node information that is common between the deprecated
// discovery format and the new format. It exists so that group-building logic
// does not have to be duplicated per format. Once the deprecated format is
// removed, the Group field can go away.
type nodeCommon struct {
	Name   string
	Xname  string
	Group  string
	Groups []string
}

// buildGroupList constructs the deduplicated list of SMD groups (with their
// members) from a slice of nodeCommon. It merges the deprecated per-node
// "group" field with the "groups" slice, deduplicating members. This is a pure
// function extracted from the static discovery command to make the
// group-assembly logic independently testable.
func buildGroupList(nodesCommon []nodeCommon) []smd.Group {
	groupsToAdd := make(map[string]smd.Group)
	addToGroup := func(label, xname string) {
		if g, ok := groupsToAdd[label]; !ok {
			newGroup := smd.Group{
				Label:       label,
				Description: fmt.Sprintf("The %s group", label),
			}
			groupsToAdd[label] = discover.AddMemberToGroup(newGroup, xname)
		} else {
			groupsToAdd[label] = discover.AddMemberToGroup(g, xname)
		}
	}
	for _, node := range nodesCommon {
		// node.Group IS DEPRECATED IN FAVOR OF node.Groups. This block
		// should be deleted when node.Group is removed. For now, we merge
		// node.Group with node.Groups; since a map is used for
		// deduplication, this is trivial.
		if node.Group != "" {
			if len(strings.Trim(node.Name, " \t")) == 0 {
				log.Logger.Warn().Msgf("node %s contains 'group', which is deprecated; use 'groups' instead", node.Xname)
			} else {
				log.Logger.Warn().Msgf("node %s (%s) contains 'group', which is deprecated; use 'groups' instead", node.Xname, node.Name)
			}
			addToGroup(node.Group, node.Xname)
		}
		for _, group := range node.Groups {
			addToGroup(group, node.Xname)
		}
	}
	groupList := make([]smd.Group, len(groupsToAdd))
	var idx = 0
	for _, g := range groupsToAdd {
		groupList[idx] = g
		idx++
	}
	return groupList
}

func singleBatchResult(results client.BatchResult[client.HTTPEnvelope]) (client.HTTPEnvelope, error) {
	if len(results) != 1 {
		return client.HTTPEnvelope{}, fmt.Errorf("batch returned %d results, want one", len(results))
	}
	return results[0].Value, results[0].Err
}

func upsertOnConflict[T any](items []T, create, update func(T) (client.HTTPEnvelope, error)) []error {
	var errs []error
	for _, item := range items {
		henv, err := create(item)
		if err == nil {
			continue
		}
		if errors.Is(err, client.UnsuccessfulHTTPError) && henv.StatusCode == 409 {
			_, err = update(item)
		}
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

func NewCmd() *cobra.Command {
	discoveryVersion := discover.DiscoveryMethodV2

	// staticCmd represents the "discover static" command
	var staticCmd = &cobra.Command{
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

bmcs:
- xname: x1000c1s7b0
  mac: de:ca:fc:0f:ee:ee
  ip: 172.16.0.101
nodes:
- name: node01
  nid: 1
  xname: x1000c1s7b0n0
  bmc: x1000c1s7b0
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
		RunE: func(cmd *cobra.Command, args []string) error {
			// Without a base URI, we cannot do anything
			smdBaseURI, err := cli.GetBaseURISMD(cmd)
			if err != nil {
				return cli.Errorf(cli.CodeConfig, "failed to get base URI for SMD: %w", err)
			}

			// This endpoint requires authentication, so a token is needed
			if err := cli.HandleToken(cmd); err != nil {
				return err
			}

			// Create client to make request to SMD
			smdClient, err := smd.NewClient(smdBaseURI, client.WithInsecure(cli.Insecure), client.WithShowToken(cli.ShowToken(cmd)))
			if err != nil {
				return cli.Errorf(cli.CodeConfig, "error creating new SMD client: %w", err)
			}

			// Check if a CA certificate was passed and load it into client if valid
			if err := cli.UseCACert(smdClient.OchamiClient); err != nil {
				return err
			}

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

			// nodeCommon (package scope) keeps basic node information that
			// is common between the deprecated format and the new format for
			// discovery. It's here so that we don't have to duplicate loops
			// due to the differing formats. Once the deprecated format is
			// removed, this can go away.
			var nodesCommon []nodeCommon

			// Read data from file or stdin into map to determine which
			// discovery method to use.
			discoveryData := make(map[string]([]map[string]any))
			if cmd.Flag("data").Changed {
				if err := cli.HandlePayload(cmd, &discoveryData); err != nil {
					return err
				}
			} else {
				if err := cli.HandlePayloadStdin(cmd, &discoveryData); err != nil {
					return err
				}
			}
			useDeprecatedFormat := discoverStaticDeprecatedFormat(cmd, discoveryData)
			var rawData []byte
			if useDeprecatedFormat {
				log.Logger.Warn().Msg("using deprecated discovery format which will be removed in a future version")

				// Convert discovery data to struct
				rawData, err := json.Marshal(discoveryData)
				if err != nil {
					return cli.Errorf(cli.CodePayload, "unable to marshal discovery data to json: %w", err)
				}
				nodes := discover.NodeListDeprecated{}
				err = json.Unmarshal(rawData, &nodes)
				if err != nil {
					return cli.Errorf(cli.CodePayload, "unable to unmarshal discovery data from json: %w", err)
				}

				log.Logger.Debug().Msgf("read %d nodes", len(nodes.Nodes))
				log.Logger.Debug().Msgf("nodes: %s", nodes)

				// Add nodes to node list in common format
				for _, n := range nodes.Nodes {
					commonNode := nodeCommon{
						Name:   n.Name,
						Xname:  n.Xname,
						Group:  n.Group,
						Groups: n.Groups,
					}
					nodesCommon = append(nodesCommon, commonNode)
				}

				// Put together payload for different endpoints
				log.Logger.Debug().Msg("generating redfish structures to send to SMD")
				comps, rfes, ifaces, err = discover.DiscoveryInfoV2Deprecated(smdBaseURI, nodes)
				if err != nil {
					return cli.Errorf(cli.CodePayload, "failed to construct structures to send to SMD: %w", err)
				}
				log.Logger.Debug().Msgf("generated redfish structures: %v", rfes.RedfishEndpoints)
			} else {
				// Convert discovery data to struct
				rawData, err = json.Marshal(discoveryData)
				if err != nil {
					return cli.Errorf(cli.CodePayload, "unable to marshal discovery items to json: %w", err)
				}
				items := discover.DiscoveryItems{}
				err = json.Unmarshal(rawData, &items)
				if err != nil {
					return cli.Errorf(cli.CodePayload, "unable to unmarshal discovery items from json: %w", err)
				}

				log.Logger.Debug().Msgf("read %d bmcs", len(items.BMCs))
				log.Logger.Debug().Msgf("bmcs: %s", items.BMCs)
				log.Logger.Debug().Msgf("read %d nodes", len(items.Nodes))
				log.Logger.Debug().Msgf("nodes: %s", items.Nodes)

				// Add nodes to node list in common format
				for _, n := range items.Nodes {
					commonNode := nodeCommon{
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
					return cli.Errorf(cli.CodePayload, "failed to construct structures to send to SMD: %w", err)
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
				results := smdClient.PutComponents(cmd.Context(), comps, cli.Token)
				for _, err := range results.Errors() {
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
				if _, err := smdClient.PatchComponentsNID(cmd.Context(), comps, cli.Token); err != nil {
					log.Logger.Error().Err(err).Msg("failed to update NIDs for components in SMD")
					compErrorsOccurred = true
				}
			} else {
				// Otherwise send a normal POST
				_, err = smdClient.PostComponents(cmd.Context(), comps, cli.Token)
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
				rfeErrs           []error
			)
			if cmd.Flag("overwrite").Changed {
				// SMD's RedfishEndpoint API for PUT behaves more like
				// PATCH. In other words, the RedfishEndpoint must exist
				// _first_ before PUTting. This means that, to get
				// normal PUT behavior, we have to first try to POST,
				// then, if 409 is returned, try to PUT.
				rfeErrs = upsertOnConflict(rfes.RedfishEndpoints,
					func(rfe smd.RedfishEndpointV2) (client.HTTPEnvelope, error) {
						return singleBatchResult(smdClient.PostRedfishEndpointsV2(cmd.Context(), smd.RedfishEndpointSliceV2{RedfishEndpoints: []smd.RedfishEndpointV2{rfe}}, cli.Token))
					},
					func(rfe smd.RedfishEndpointV2) (client.HTTPEnvelope, error) {
						return singleBatchResult(smdClient.PutRedfishEndpointsV2(cmd.Context(), smd.RedfishEndpointSliceV2{RedfishEndpoints: []smd.RedfishEndpointV2{rfe}}, cli.Token))
					})
				for _, err := range rfeErrs {
					log.Logger.Error().Err(err).Msg("failed to add or update redfish endpoint in SMD")
					rfeErrorsOccurred = true
				}
			} else {
				// --overwrite was not passed, perform regular POST.
				rfeErrs = smdClient.PostRedfishEndpointsV2(cmd.Context(), rfes, cli.Token).Errors()
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
				ifaceErrs           []error
			)
			// Get discovery version value (err handled in cmd.Args).
			// Send EthernetInterfaces to SMD if discoverVersion is 1.
			if discoveryVersion == discover.DiscoveryMethodV1 {
				if cmd.Flag("overwrite").Changed {
					ifaceErrs = upsertOnConflict(ifaces,
						func(iface smd.EthernetInterface) (client.HTTPEnvelope, error) {
							return singleBatchResult(smdClient.PostEthernetInterfaces(cmd.Context(), []smd.EthernetInterface{iface}, cli.Token))
						},
						func(iface smd.EthernetInterface) (client.HTTPEnvelope, error) {
							return singleBatchResult(smdClient.PatchEthernetInterfaces(cmd.Context(), []smd.EthernetInterface{iface}, cli.Token))
						})
					for _, err := range ifaceErrs {
						log.Logger.Error().Err(err).Msg("failed to add or update ethernet interface in SMD")
						ifaceErrorsOccurred = true
					}
				} else {
					// --overwrite was not passed, perform regular POST.
					ifaceErrs = smdClient.PostEthernetInterfaces(cmd.Context(), ifaces, cli.Token).Errors()
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

			// Put together list of groups to add and which components to
			// add to those groups.
			groupList := buildGroupList(nodesCommon)

			// Add groups and components to those groups
			var (
				groupErrorsOccurred bool = false
				groupErrs           []error
			)
			if cmd.Flag("overwrite").Changed {
				groupErrs = upsertOnConflict(groupList,
					func(group smd.Group) (client.HTTPEnvelope, error) {
						return singleBatchResult(smdClient.PostGroups(cmd.Context(), []smd.Group{group}, cli.Token))
					},
					func(group smd.Group) (client.HTTPEnvelope, error) {
						return singleBatchResult(smdClient.PatchGroups(cmd.Context(), []smd.Group{group}, cli.Token))
					})
				for _, err := range groupErrs {
					log.Logger.Error().Err(err).Msg("failed to add or update group in SMD")
					groupErrorsOccurred = true
				}
			} else {
				groupErrs = smdClient.PostGroups(cmd.Context(), groupList, cli.Token).Errors()
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
			if compErrorsOccurred {
				log.Logger.Warn().Msg("component requests completed with errors")
			}
			if rfeErrorsOccurred {
				log.Logger.Warn().Msg("redfish endpoint requests completed with errors")
			}
			if ifaceErrorsOccurred {
				log.Logger.Warn().Msg("ethernet interface requests completed with errors")
			}
			if groupErrorsOccurred {
				log.Logger.Warn().Msg("group requests completed with errors")
			}
			if compErrorsOccurred || rfeErrorsOccurred || ifaceErrorsOccurred || groupErrorsOccurred {
				return cli.Errorf(cli.CodeHTTP, "static discovery completed with errors")
			}

			return nil
		},
	}

	// Create flags
	staticCmd.Flags().Var(&discoveryVersion, "discovery-version", "set version for discovery method to use")
	staticCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")
	staticCmd.Flags().VarP(&cli.FormatInput, "format-input", "f", "format of input payload data (json,json-pretty,yaml)")
	staticCmd.Flags().Bool("overwrite", false, "overwrite any existing information instead of failing")
	staticCmd.Flags().String("uri", "", "absolute base URI or relative base path of SMD")

	staticCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)
	staticCmd.RegisterFlagCompletionFunc("discovery-version", cli.CompletionDiscoveryVersion)

	return staticCmd
}

func discoverStaticDeprecatedFormat(cmd *cobra.Command, discoveryData map[string]([]map[string]any)) bool {
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
