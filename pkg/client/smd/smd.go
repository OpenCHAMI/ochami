// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package smd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/openchami/schemas/schemas"
	"github.com/openchami/schemas/schemas/csm"

	"github.com/openchami/ochami/internal/log"
	"github.com/openchami/ochami/pkg/client"
)

// SMDClient is an OchamiClient that has its BasePath set configured to the one
// that BSS uses.
type SMDClient struct {
	*client.OchamiClient
}

const (
	serviceNameSMD = "SMD"

	SMDRelpathService            = "/service"
	SMDRelpathComponents         = "/State/Components"
	SMDRelpathEthernetInterfaces = "/Inventory/EthernetInterfaces"
	SMDRelpathRedfishEndpoints   = "/Inventory/RedfishEndpoints"
	SMDRelpathComponentEndpoints = "/Inventory/ComponentEndpoints"
	SMDRelpathGroups             = "/groups"
	SMDRelpathMemberships        = "/memberships"

	SMDSubpathBulkNID = "BulkNID"
)

// Component is a minimal subset of SMD's Component struct that contains only
// what is necessary for sending a valid Component request to SMD.
type Component struct {
	ID      string `json:"ID" yaml:"ID"`
	Type    string `json:"Type" yaml:"Type"`
	State   string `json:"State,omitempty" yaml:"State,omitempty"`
	Enabled bool   `json:"Enabled,omitempty" yaml:"Enabled,omitempty"`
	Role    string `json:"Role,omitempty" yaml:"Role,omitempty"`
	Arch    string `json:"Arch,omitempty" yaml:"Arch,omitempty"`
	NID     int64  `json:"NID,omitempty" yaml:"NID,omitempty"`
}

// ComponentSlice is a convenience data structure to make marshalling Component
// requests easier.
type ComponentSlice struct {
	Components []Component `json:"Components" yaml:"Components"`
}

// EthernetInterface is a minimal subset of SMD's EthernetInterface struct that
// contains only what is necessary for sending a valid EthernetInterface request
// to SMD.
type EthernetInterface struct {
	ID          string       `json:"ID" yaml:"ID"`
	ComponentID string       `json:"ComponentID" yaml:"ComponentID"`
	Type        string       `json:"Type" yaml:"Type"`
	Description string       `json:"Description" yaml:"Description"`
	MACAddress  string       `json:"MACAddress" yaml:"MACAddress"`
	IPAddresses []EthernetIP `json:"IPAddresses" yaml:"IPAddresses"`
}

type EthernetIP struct {
	IPAddress string `json:"IPAddress" yaml:"IPAddress"`
	Network   string `json:"Network" yaml:"Network"`
}

// RedfishEndpointSlice is a convenience data structure to make marshalling
// RedfishEndpoint requests easier.
type RedfishEndpointSlice struct {
	RedfishEndpoints []csm.RedfishEndpoint `json:"RedfishEndpoints" yaml:"RedfishEndpoints"`
}

// RedfishEndpointSliceV2 is a convenience data structure to make marshalling
// RedfishEndpointV2 requests easier.
type RedfishEndpointSliceV2 struct {
	RedfishEndpoints []RedfishEndpointV2 `json:"RedfishEndpoints" yaml:"RedfishEndpoints"`
}

// RedfishEndpointV2 holds the redfish endpoint data read from/into SMD using
// schema v2. This schema supports dynamic creation of Components,
// ComponentEndpoints, and EthernetInterfaces from the Systems and Managers
// contained in this struct.
type RedfishEndpointV2 struct {
	csm.RedfishEndpoint
	SchemaVersion int       `json:"SchemaVersion" yaml:"SchemaVersion"`
	Systems       []System  `json:"Systems" yaml:"Systems"`
	Managers      []Manager `json:"Managers" yaml:"Managers"`
}

// System represents data that would be retrieved from BMC System data, except
// reduced to a minimum needed for discovery.
type System struct {
	URI                string                      `json:"uri" yaml:"uri"`
	UUID               string                      `json:"uuid" yaml:"uuid"`
	Name               string                      `json:"name" yaml:"name"`
	EthernetInterfaces []schemas.EthernetInterface `json:"ethernet_interfaces" yaml:"ethernet_interfaces"`
	Actions            []string                    `json:"actions" yaml:"actions"`
}

// Manager represents data that would be retrieved from BMC Manager data, except
// reduced to a minimum needed for discovery.
type Manager struct {
	System
	Description string `json:"description" yaml:"description"`
	Type        string `json:"type" yaml:"type"`
}

// Group represents the payload structure for SMD groups.
type Group struct {
	Label          string   `json:"label" yaml:"label"`
	Description    string   `json:"description" yaml:"description"`
	Tags           []string `json:"tags,omitempty" yaml:"tags,omitempty"`
	ExclusiveGroup string   `json:"exclusiveGroup,omitempty" yaml:"exclusiveGroup,omitempty"`
	Members        struct {
		IDs []string `json:"ids,omitempty" yaml:"ids,omitempty"`
	} `json:"members,omitempty" yaml:"members,omitempty"`
}

// GroupMembers represents the payload structure for SMD group membership for
// PUT requests. It consists of only the group label and list of group IDs.
type GroupMembers struct {
	Group string   `json:"group" yaml:"group"`
	IDs   []string `json:"ids" yaml:"ids"`
}

// NewClient takes a baseURI and returns a pointer to a new SMDClient. If an
// error occurred creating the embedded OchamiClient, it is returned. Behavior
// such as TLS verification and token redaction is configured via functional
// options (e.g. client.WithInsecure, client.WithShowToken).
func NewClient(baseURI string, opts ...client.Option) (*SMDClient, error) {
	oc, err := client.NewOchamiClient(serviceNameSMD, baseURI, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OchamiClient for %s: %w", serviceNameSMD, err)
	}
	sc := &SMDClient{
		OchamiClient: oc,
	}

	return sc, err
}

// GetStatus is a wrapper function around OchamiClient.GetData that takes an
// optional component and uses it to determine which subpath of the SMD /service
// endpoint to query. If empty, the /service/ready endpoint is queried.
// Otherwise:
//
// "all" -> "/service/values"
func (sc *SMDClient) GetStatus(ctx context.Context, component string) (client.HTTPEnvelope, error) {
	var (
		henv              client.HTTPEnvelope
		err               error
		smdStatusEndpoint string
	)
	switch component {
	case "":
		smdStatusEndpoint, err = url.JoinPath(SMDRelpathService, "ready")
	case "all":
		smdStatusEndpoint, err = url.JoinPath(SMDRelpathService, "values")
	default:
		return henv, fmt.Errorf("GetStatus(): unknown status component: %s", component)
	}

	// Check that the JoinPath call was successful
	if err != nil {
		return henv, fmt.Errorf("GetStatus(): error creating SMD status endpoint: %w", err)
	}

	henv, err = sc.GetData(ctx, smdStatusEndpoint, "", nil)
	if err != nil {
		err = fmt.Errorf("GetStatus(): error getting SMD all status: %w", err)
	}

	return henv, err
}

// GetComponentsAll is a wrapper function around OchamiClient.GetData that queries
// /State/Components.
func (sc *SMDClient) GetComponentsAll(ctx context.Context) (client.HTTPEnvelope, error) {
	henv, err := sc.GetData(ctx, SMDRelpathComponents, "", nil)
	if err != nil {
		err = fmt.Errorf("GetComponentsAll(): error getting components: %w", err)
	}

	return henv, err
}

// GetComponentsXname is like GetComponentsAll except that it takes a token and
// queries /State/Components/{xname}.
func (sc *SMDClient) GetComponentsXname(ctx context.Context, xname, token string) (client.HTTPEnvelope, error) {
	var henv client.HTTPEnvelope
	finalEP := SMDRelpathComponents + "/" + xname
	headers := client.NewHTTPHeaders()
	if token != "" {
		_ = headers.SetAuthorization(token) //nolint:errcheck // headers was allocated above and cannot be nil
	}
	henv, err := sc.GetData(ctx, finalEP, "", headers)
	if err != nil {
		err = fmt.Errorf("GetComponentsXname(): error getting component for xname %q: %w", xname, err)
	}

	return henv, err
}

// GetComponentsNid is like GetComponentsAll except that it takes a token and
// queries /State/Components/ByNID/{nid}.
func (sc *SMDClient) GetComponentsNid(ctx context.Context, nid int32, token string) (client.HTTPEnvelope, error) {
	var henv client.HTTPEnvelope
	finalEP := SMDRelpathComponents + "/ByNID/" + fmt.Sprint(nid)
	headers := client.NewHTTPHeaders()
	if token != "" {
		_ = headers.SetAuthorization(token) //nolint:errcheck // headers was allocated above and cannot be nil
	}
	henv, err := sc.GetData(ctx, finalEP, "", headers)
	if err != nil {
		err = fmt.Errorf("GetComponentsNid(): error getting component for NID %d: %w", nid, err)
	}

	return henv, err
}

// GetRedfishEndpoints is a wrapper around OchamiClient.GetData that takes an
// optional query string (without the "?") and a token. It sets token as the
// authorization bearer in the headers and passes the query string and headers
// to OchamiClient.GetData, using the SMD RedfishEndpoints API endpoint.
func (sc *SMDClient) GetRedfishEndpoints(ctx context.Context, query, token string) (client.HTTPEnvelope, error) {
	var (
		henv    client.HTTPEnvelope
		headers *client.HTTPHeaders
		err     error
	)
	headers = client.NewHTTPHeaders()
	if token != "" {
		_ = headers.SetAuthorization(token) //nolint:errcheck // headers was allocated above and cannot be nil
	}
	henv, err = sc.GetData(ctx, SMDRelpathRedfishEndpoints, query, headers)
	if err != nil {
		err = fmt.Errorf("GetRedfishEndpoints(): error getting redfish endpoints: %w", err)
	}

	return henv, err
}

// GetEthernetInterfaces is a wrapper around OchamiClient.GetData that takes a
// query string and passes it to OchamiClient.GetData using SMD's ethernet
// interfaces endpoint.
func (sc *SMDClient) GetEthernetInterfaces(ctx context.Context, query string) (client.HTTPEnvelope, error) {
	henv, err := sc.GetData(ctx, SMDRelpathEthernetInterfaces, query, nil)
	if err != nil {
		err = fmt.Errorf("GetEthernetInterfaces(): error getting ethernet interfaces: %w", err)
	}

	return henv, err
}

// GetEthernetInterfacesByID is a wrapper around OchamiClient.GetData that takes
// an ethernet interface ID, token, and a flag indicating if the ethernet
// interface itself should be retrieved or a list of its IPs. It passes these to
// OchamiClient.GetData, setting the token as the authorization bearer in the
// request headers.
func (sc *SMDClient) GetEthernetInterfaceByID(ctx context.Context, id, token string, getIPs bool) (client.HTTPEnvelope, error) {
	var (
		ep      string
		err     error
		henv    client.HTTPEnvelope
		headers *client.HTTPHeaders
	)
	headers = client.NewHTTPHeaders()
	if token != "" {
		_ = headers.SetAuthorization(token) //nolint:errcheck // headers was allocated above and cannot be nil
	}
	if getIPs {
		if ep, err = url.JoinPath(SMDRelpathEthernetInterfaces, id); err != nil {
			return henv, fmt.Errorf("GetEthernetInterfacesByID(): failed to join ethernet path (%s) with id (%s): %w", SMDRelpathEthernetInterfaces, id, err)
		}
		if ep, err = url.JoinPath(ep, "IPAddresses"); err != nil {
			return henv, fmt.Errorf("GetEthernetInterfacesByID(): failed to join endpoint %s with \"IPAddresses\": %w", ep, err)
		}
	} else {
		if ep, err = url.JoinPath(SMDRelpathEthernetInterfaces, id); err != nil {
			return henv, fmt.Errorf("GetEthernetInterfacesByID(): failed to join endpoint %s with id %q: %w", ep, id, err)
		}
	}
	henv, err = sc.GetData(ctx, ep, "", headers)
	if err != nil {
		err = fmt.Errorf("GetEthernetInterfacesByID(): failed to GET ethernet interfaces in SMD: %w", err)
	}

	return henv, err
}

// GetComponentEndpoints is similar to GetComponentEndpointsAll except that it
// iteratively calls OchamiClient.GetData on each xname passed. Each request
// has a corresponding aligned result containing its HTTP envelope and error.
func (sc *SMDClient) GetComponentEndpoints(ctx context.Context, token string, xnames ...string) client.BatchResult[client.HTTPEnvelope] {
	headers := client.NewHTTPHeaders()
	if token != "" {
		_ = headers.SetAuthorization(token) //nolint:errcheck // headers was allocated above and cannot be nil
	}
	return executeBatch(ctx, xnames, func(ctx context.Context, xname string) (client.HTTPEnvelope, error) {
		henv, err := sc.GetData(ctx, SMDRelpathComponentEndpoints+"/"+xname, "", headers)
		if err != nil {
			log.Logger.Debug().Err(err).Msg("failed to get component endpoint")
			return henv, fmt.Errorf("GetComponentEndpoints(): failed to GET component endpoint from SMD: %w", err)
		}
		return henv, nil
	})
}

// GetComponentEndpointsAll is a wrapper function around OchamiClient.GetData
// that takes a token and puts it in the request headers as an authorization
// bearer, then sends a get to the SMD component endpoint API endpoint.
func (sc *SMDClient) GetComponentEndpointsAll(ctx context.Context, token string) (client.HTTPEnvelope, error) {
	var (
		err     error
		henv    client.HTTPEnvelope
		headers *client.HTTPHeaders
	)
	headers = client.NewHTTPHeaders()
	if token != "" {
		_ = headers.SetAuthorization(token) //nolint:errcheck // headers was allocated above and cannot be nil
	}
	henv, err = sc.GetData(ctx, SMDRelpathComponentEndpoints, "", headers)
	if err != nil {
		err = fmt.Errorf("GetComponentEndpointsAll(): error getting component endpoints: %w", err)
	}

	return henv, err
}

// GetGroups is a wrapper function around OchamiClient.GetData that takes a
// query string and token. It puts the token in the request headers as an
// authorization bearer, then sends a get to the SMD groups API endpoint with
// the query string, returning the response as an client.HTTPEnvelope and an
// error if one occurred.
func (sc *SMDClient) GetGroups(ctx context.Context, query, token string) (client.HTTPEnvelope, error) {
	var (
		henv    client.HTTPEnvelope
		headers *client.HTTPHeaders
		err     error
	)
	headers = client.NewHTTPHeaders()
	if token != "" {
		_ = headers.SetAuthorization(token) //nolint:errcheck // headers was allocated above and cannot be nil
	}
	henv, err = sc.GetData(ctx, SMDRelpathGroups, query, headers)
	if err != nil {
		err = fmt.Errorf("GetGroups(): error getting groups: %w", err)
	}

	return henv, err
}

// GetGroupMembers is a wrapper function around OchamiClient.GetData that takes
// a group name, which it passes to the GetData function using the SMD group
// membership endpoint. It also takes a token, which it puts into the headers as
// the authorization bearer.
func (sc *SMDClient) GetGroupMembers(ctx context.Context, group, token string) (client.HTTPEnvelope, error) {
	if group == "" {
		return client.HTTPEnvelope{}, fmt.Errorf("GetGroupMembers(): group label cannot be empty")
	}
	finalEP, err := url.JoinPath(SMDRelpathGroups, group, "members")
	if err != nil {
		return client.HTTPEnvelope{}, fmt.Errorf("GetGroupMembers(): failed to join group path (%s) with membership path for gorup %s: %w", SMDRelpathGroups, group, err)
	}
	headers := client.NewHTTPHeaders()
	if token != "" {
		_ = headers.SetAuthorization(token) //nolint:errcheck // headers was allocated above and cannot be nil
	}
	henv, err := sc.GetData(ctx, finalEP, "", headers)
	if err != nil {
		err = fmt.Errorf("GetGroupMembers(): error getting group members for group %s: %w", group, err)
	}

	return henv, err
}

// GetGroupMembership is a wrapper function around OchamiClient.GetData that takes
// a node name, which it passes to the GetData function using the SMD node
// membership endpoint. It also takes a token, which it puts into the headers as
// the authorization bearer.
func (sc *SMDClient) GetGroupMembership(ctx context.Context, qstr, token string) (client.HTTPEnvelope, error) {
	headers := client.NewHTTPHeaders()
	if token != "" {
		_ = headers.SetAuthorization(token) //nolint:errcheck // headers was allocated above and cannot be nil
	}
	henv, err := sc.GetData(ctx, SMDRelpathMemberships, qstr, headers)
	if err != nil {
		err = fmt.Errorf("GetGroupMembership(): error getting group memberships for query %s: %w", qstr, err)
	}

	return henv, err
}

// PostComponents is a wrapper function around OchamiClient.PostData that takes
// a ComponentSlice and a token, puts the token in the request headers as an
// authorization bearer, marshalls compSlice as JSON and sets it as the request
// body, then passes it to Ochami.PostData.
func (sc *SMDClient) PostComponents(ctx context.Context, compSlice ComponentSlice, token string) (client.HTTPEnvelope, error) {
	var (
		henv    client.HTTPEnvelope
		headers *client.HTTPHeaders
		body    client.HTTPBody
		err     error
	)
	if body, err = json.Marshal(compSlice); err != nil {
		return henv, fmt.Errorf("PostComponents(): failed to marshal ComponentArray: %w", err)
	}
	headers = client.NewHTTPHeaders()
	if token != "" {
		_ = headers.SetAuthorization(token) //nolint:errcheck // headers was allocated above and cannot be nil
	}
	henv, err = sc.PostData(ctx, SMDRelpathComponents, "", headers, body)
	if err != nil {
		err = fmt.Errorf("PostComponents(): failed to POST component(s) to SMD: %w", err)
	}

	return henv, err
}

// PostRedfishEndpoints is a wrapper function around OchamiClient.PostData that
// takes a RedfishEndpointSlice and a token, puts the token in the request
// headers as an authorization bearer, and iteratively calls
// OchamiClient.PostData using each RedfishEndpoint in the slice.
func (sc *SMDClient) PostRedfishEndpoints(ctx context.Context, rfes RedfishEndpointSlice, token string) client.BatchResult[client.HTTPEnvelope] {
	headers := client.NewHTTPHeaders()
	if token != "" {
		_ = headers.SetAuthorization(token) //nolint:errcheck // headers was allocated above and cannot be nil
	}
	return executeBatch(ctx, rfes.RedfishEndpoints, func(ctx context.Context, rfe csm.RedfishEndpoint) (client.HTTPEnvelope, error) {
		body, err := json.Marshal(rfe)
		if err != nil {
			return client.HTTPEnvelope{}, fmt.Errorf("PostRedfishEndpoints(): failed to marshal RedfishEndpoint: %w", err)
		}
		henv, err := sc.PostData(ctx, SMDRelpathRedfishEndpoints, "", headers, body)
		if err != nil {
			return henv, fmt.Errorf("PostRedfishEndpoints(): failed to POST redfish endpoint to SMD: %w", err)
		}
		return henv, nil
	})
}

// PostRedfishEndpointsV2 behaves like PostRedfishEndpoints except that it works
// with a RedfishEndpointSliceV2.
func (sc *SMDClient) PostRedfishEndpointsV2(ctx context.Context, rfes RedfishEndpointSliceV2, token string) client.BatchResult[client.HTTPEnvelope] {
	headers := client.NewHTTPHeaders()
	if token != "" {
		_ = headers.SetAuthorization(token) //nolint:errcheck // headers was allocated above and cannot be nil
	}
	return executeBatch(ctx, rfes.RedfishEndpoints, func(ctx context.Context, rfe RedfishEndpointV2) (client.HTTPEnvelope, error) {
		body, err := json.Marshal(rfe)
		if err != nil {
			return client.HTTPEnvelope{}, fmt.Errorf("PostRedfishEndpointsV2(): failed to marshal RedfishEndpoint: %w", err)
		}
		henv, err := sc.PostData(ctx, SMDRelpathRedfishEndpoints, "", headers, body)
		if err != nil {
			return henv, fmt.Errorf("PostRedfishEndpointsV2(): failed to POST redfish endpoint to SMD: %w", err)
		}
		return henv, nil
	})
}

// PostEthernetInterfaces is a wrapper function around OchamiClient.PostData
// that takes a slice of EthernetInterfaces and a token, puts the token in the
// request headers as an authorization bearer, and iteratively calls
// OchamiClient.PostData using each EthernetInterface in the slice.
func (sc *SMDClient) PostEthernetInterfaces(ctx context.Context, eis []EthernetInterface, token string) client.BatchResult[client.HTTPEnvelope] {
	headers := client.NewHTTPHeaders()
	if token != "" {
		_ = headers.SetAuthorization(token) //nolint:errcheck // headers was allocated above and cannot be nil
	}
	return executeBatch(ctx, eis, func(ctx context.Context, ei EthernetInterface) (client.HTTPEnvelope, error) {
		body, err := json.Marshal(ei)
		if err != nil {
			return client.HTTPEnvelope{}, fmt.Errorf("PostEthernetInterfaces(): failed to marshal EthernetInterface: %w", err)
		}
		henv, err := sc.PostData(ctx, SMDRelpathEthernetInterfaces, "", headers, body)
		if err != nil {
			return henv, fmt.Errorf("PostEthernetInterfaces(): failed to POST ethernet interface(s) to SMD: %w", err)
		}
		return henv, nil
	})
}

// PostGroups is a wrapper function around OchamiClient.PostData that takes a
// Group slice and a token, puts the token in the request headers as an
// authorization bearer, and iteratively calls OchamiClient.PostData using each
// Group in the slice.
func (sc *SMDClient) PostGroups(ctx context.Context, groups []Group, token string) client.BatchResult[client.HTTPEnvelope] {
	headers := client.NewHTTPHeaders()
	if token != "" {
		_ = headers.SetAuthorization(token) //nolint:errcheck // headers was allocated above and cannot be nil
	}
	return executeBatch(ctx, groups, func(ctx context.Context, group Group) (client.HTTPEnvelope, error) {
		body, err := json.Marshal(group)
		if err != nil {
			return client.HTTPEnvelope{}, fmt.Errorf("PostGroups(): failed to marshal Group: %w", err)
		}
		henv, err := sc.PostData(ctx, SMDRelpathGroups, "", headers, body)
		if err != nil {
			return henv, fmt.Errorf("PostGroups(): failed to POST group to SMD: %w", err)
		}
		return henv, nil
	})
}

// PostGroupMembers is a wrapper function around OchamiClient.PostData that
// takes a token, group name, and a list of one or more component IDs. It puts
// the token in the request headers as an authorization bearer, and iteratively
// calls OchamiClient.PostData for each member on the group.
func (sc *SMDClient) PostGroupMembers(ctx context.Context, token, group string, members ...string) (client.BatchResult[client.HTTPEnvelope], error) {
	if group == "" {
		return nil, fmt.Errorf("PostGroupMembers(): no group label specified to add members to")
	}
	if len(members) == 0 {
		return nil, fmt.Errorf("PostGroupMembers(): no new members specified to add to group")
	}
	headers := client.NewHTTPHeaders()
	if token != "" {
		_ = headers.SetAuthorization(token) //nolint:errcheck // headers was allocated above and cannot be nil
	}
	results := executeBatch(ctx, members, func(ctx context.Context, member string) (client.HTTPEnvelope, error) {
		groupPath, err := url.JoinPath(SMDRelpathGroups, group, "members")
		if err != nil {
			return client.HTTPEnvelope{}, fmt.Errorf("PostGroupMembers(): failed to join group path (%s) with group label (%s): %w", SMDRelpathGroups, group, err)
		}
		body, err := json.Marshal(map[string]string{"id": member})
		if err != nil {
			return client.HTTPEnvelope{}, fmt.Errorf("PostGroupMembers(): failed to marshal member id %s: %w", member, err)
		}
		henv, err := sc.PostData(ctx, groupPath, "", headers, body)
		if err != nil {
			return henv, fmt.Errorf("PostGroupMembers(): failed to POST member %s to group %s: %w", member, group, err)
		}
		return henv, nil
	})
	return results, nil
}

// PutComponents takes a ComponentSlice and a token and iteratively calls
// OchamiClient.PutData for each Component in the contained list. This is
// necessary because SMD only allows sending a PUT for a single Component using
// its ID in the endpoint path, forcing the client to send only a single
// Component per request. Each input has one aligned result.
func (sc *SMDClient) PutComponents(ctx context.Context, compSlice ComponentSlice, token string) client.BatchResult[client.HTTPEnvelope] {
	headers := client.NewHTTPHeaders()
	if token != "" {
		_ = headers.SetAuthorization(token) //nolint:errcheck // headers was allocated above and cannot be nil
	}
	return executeBatch(ctx, compSlice.Components, func(ctx context.Context, comp Component) (client.HTTPEnvelope, error) {
		if comp.ID == "" {
			return client.HTTPEnvelope{}, fmt.Errorf("PutComponents(): unable to update component with blank ID")
		}
		xnamePath, err := url.JoinPath(SMDRelpathComponents, comp.ID)
		if err != nil {
			return client.HTTPEnvelope{}, fmt.Errorf("PutComponents(): failed join component path (%s) with xname (%s): %w", SMDRelpathComponents, comp.ID, err)
		}
		// SMD is weird and requires the PUT body to be a structure that
		// _contains_ the component, so we do that here.
		putComp := map[string]any{"Component": comp, "Force": true}
		body, marshalErr := json.Marshal(putComp)
		if marshalErr != nil {
			return client.HTTPEnvelope{}, fmt.Errorf("PutComponents(): failed to marshal component into JSON: %w", marshalErr)
		}
		henv, err := sc.PutData(ctx, xnamePath, "", headers, body)
		if err != nil {
			return henv, fmt.Errorf("PutComponents(): failed to PUT component %s in SMD: %w", comp.ID, err)
		}
		return henv, nil
	})
}

// PutRedfishEndpoints is a wrapper function around OchamiClient.PutData that
// takes a RedfishEndpointSlice and a token, puts the token in the request
// headers as an authorization bearer, and iteratively calls
// OchamiClient.PutData using each RedfishEndpoint in the slice.
func (sc *SMDClient) PutRedfishEndpoints(ctx context.Context, rfes RedfishEndpointSlice, token string) client.BatchResult[client.HTTPEnvelope] {
	headers := client.NewHTTPHeaders()
	if token != "" {
		_ = headers.SetAuthorization(token) //nolint:errcheck // headers was allocated above and cannot be nil
	}
	return executeBatch(ctx, rfes.RedfishEndpoints, func(ctx context.Context, rfe csm.RedfishEndpoint) (client.HTTPEnvelope, error) {
		if rfe.ID == "" {
			return client.HTTPEnvelope{}, fmt.Errorf("PutRedfishEndpoints(): unable to update redfish endpoint with blank ID")
		}
		xnamePath, err := url.JoinPath(SMDRelpathRedfishEndpoints, rfe.ID)
		if err != nil {
			return client.HTTPEnvelope{}, fmt.Errorf("PutRedfishEndpoints(): failed to join redfish endpoint path (%s) with xname (%s): %w", SMDRelpathRedfishEndpoints, rfe.ID, err)
		}
		body, err := json.Marshal(rfe)
		if err != nil {
			return client.HTTPEnvelope{}, fmt.Errorf("PutRedfishEndpoints(): failed to marshal RedfishEndpoint: %w", err)
		}
		henv, err := sc.PutData(ctx, xnamePath, "", headers, body)
		if err != nil {
			return henv, fmt.Errorf("PutRedfishEndpoints(): failed to PUT redfish endpoint to SMD: %w", err)
		}
		return henv, nil
	})
}

// PutRedfishEndpointsV2 behaves like PutRedfishEndpoints except that it works
// with a RedfishEndpointSliceV2.
func (sc *SMDClient) PutRedfishEndpointsV2(ctx context.Context, rfes RedfishEndpointSliceV2, token string) client.BatchResult[client.HTTPEnvelope] {
	headers := client.NewHTTPHeaders()
	if token != "" {
		_ = headers.SetAuthorization(token) //nolint:errcheck // headers was allocated above and cannot be nil
	}
	return executeBatch(ctx, rfes.RedfishEndpoints, func(ctx context.Context, rfe RedfishEndpointV2) (client.HTTPEnvelope, error) {
		if rfe.ID == "" {
			return client.HTTPEnvelope{}, fmt.Errorf("PutRedfishEndpointsV2(): unable to update redfish endpoint with blank ID")
		}
		xnamePath, err := url.JoinPath(SMDRelpathRedfishEndpoints, rfe.ID)
		if err != nil {
			return client.HTTPEnvelope{}, fmt.Errorf("PutRedfishEndpointsV2(): failed to join redfish endpoint path (%s) with xname (%s): %w", SMDRelpathRedfishEndpoints, rfe.ID, err)
		}
		body, err := json.Marshal(rfe)
		if err != nil {
			return client.HTTPEnvelope{}, fmt.Errorf("PutRedfishEndpointsV2(): failed to marshal RedfishEndpoint: %w", err)
		}
		henv, err := sc.PutData(ctx, xnamePath, "", headers, body)
		if err != nil {
			return henv, fmt.Errorf("PutRedfishEndpointsV2(): failed to PUT redfish endpoint to SMD: %w", err)
		}
		return henv, nil
	})
}

// PutGroupMembers is a wrapper function around OchamiClient.PutData that takes
// a token, group name, and a list of one or more component IDs. It puts the
// token in the request headers as an authorization bearer and calls
// OchamiClient.PostData on the SMD group members API endpoint with the group
// and member list.
func (sc *SMDClient) PutGroupMembers(ctx context.Context, token, group string, members ...string) (client.HTTPEnvelope, error) {
	var (
		henv    client.HTTPEnvelope
		headers *client.HTTPHeaders
		body    client.HTTPBody
	)

	// Check that group and member list are non-empty
	if group == "" {
		return henv, fmt.Errorf("PutGroupMembers(): no group label specified to set members of")
	}
	if len(members) == 0 {
		return henv, fmt.Errorf("PutGroupMembers(): no members specified")
	}

	// Add token to headers
	headers = client.NewHTTPHeaders()
	if token != "" {
		_ = headers.SetAuthorization(token) //nolint:errcheck // headers was allocated above and cannot be nil
	}

	// Calculate endpoint path for group
	groupPath, err := url.JoinPath(SMDRelpathGroups, group, "members")
	if err != nil {
		return henv, fmt.Errorf("PutGroupMembers(): failed to join group path (%s) with group label (%s): %w", SMDRelpathGroups, group, err)
	}

	// Send request and return response
	g := GroupMembers{
		Group: group,
		IDs:   members,
	}
	if body, err = json.Marshal(g); err != nil {
		return henv, fmt.Errorf("PutGroupMembers(): failed to marshal group data: %w", err)
	}
	henv, err = sc.PutData(ctx, groupPath, "", headers, body)
	if err != nil {
		err = fmt.Errorf("PutGroupMembers(): failed to PUT members to group %s: %w", group, err)
	}

	return henv, err
}

// PatchComponentsNID is a wrapper function around OchamiClient.PatchData that
// takes a slice of Components and a token. It doesn't read any data fields
// within each Component except ID (xname) and NID, and for each Component, all
// fields except these are blanked. These modified components are then passed
// with the token to OchamiClient.PatchData to SMD's BulkNID endpoint to update
// the NIDs of the Components.
func (sc *SMDClient) PatchComponentsNID(ctx context.Context, comps ComponentSlice, token string) (client.HTTPEnvelope, error) {
	// Set token in request headers
	headers := client.NewHTTPHeaders()
	if token != "" {
		_ = headers.SetAuthorization(token) //nolint:errcheck // headers was allocated above and cannot be nil
	}

	// Create base path
	nidPath, err := url.JoinPath(SMDRelpathComponents, SMDSubpathBulkNID)
	if err != nil {
		return client.HTTPEnvelope{}, fmt.Errorf("PatchComponentsNID(): failed to join component path (%s) with BulkNID path (%s): %w", SMDRelpathComponents, SMDSubpathBulkNID, err)
	}

	// Blank out all except ID and NID fields from Components
	compsStripped := ComponentSlice{}
	for _, comp := range comps.Components {
		compsStripped.Components = append(compsStripped.Components, Component{
			ID:  comp.ID,
			NID: comp.NID,
		})
	}

	// Create request body
	body, err := json.Marshal(compsStripped)
	if err != nil {
		return client.HTTPEnvelope{}, fmt.Errorf("PatchComponentsNID(): failed to marshal stripped components: %w", err)
	}

	// Send request
	henv, err := sc.PatchData(ctx, nidPath, "", headers, body)
	if err != nil {
		err = fmt.Errorf("PatchComponentsNID(): failed to PATCH stripped components in SMD: %w", err)
	}

	return henv, err
}

// PatchEthernetInterfaces is a wrapper function around OchamiClient.PatchData
// that takes a slice of EthernetInterfaces and a token, puts the token in the
// request headers as an authorization bearer, and iteratively calls
// OchamiClient.PatchData using each EthernetInterface in the slice.
func (sc *SMDClient) PatchEthernetInterfaces(ctx context.Context, eis []EthernetInterface, token string) client.BatchResult[client.HTTPEnvelope] {
	headers := client.NewHTTPHeaders()
	if token != "" {
		_ = headers.SetAuthorization(token) //nolint:errcheck // headers was allocated above and cannot be nil
	}
	return executeBatch(ctx, eis, func(ctx context.Context, ei EthernetInterface) (client.HTTPEnvelope, error) {
		if ei.ID == "" {
			if ei.MACAddress != "" {
				log.Logger.Warn().Msgf("PatchEthernetInterfaces(): ID for ethernet interface is blank, attempting to adapt from MAC address (%s)", ei.MACAddress)
				newID := strings.ToLower(ei.MACAddress)
				newID = strings.ReplaceAll(newID, ":", "")
				newID = strings.ReplaceAll(newID, "-", "")
				newID = strings.ReplaceAll(newID, "_", "")
				ei.ID = newID
			} else {
				return client.HTTPEnvelope{}, fmt.Errorf("PatchEthernetInterfaces(): unable to patch ethernet interface with both blank ID and blank MAC address")
			}
		}
		eiPath, err := url.JoinPath(SMDRelpathEthernetInterfaces, ei.ID)
		if err != nil {
			return client.HTTPEnvelope{}, fmt.Errorf("PatchEthernetInterfaces(): failed to join ethernet interface path (%s) with ethernet interface ID (%s): %w", SMDRelpathEthernetInterfaces, ei.ID, err)
		}
		body, err := json.Marshal(ei)
		if err != nil {
			return client.HTTPEnvelope{}, fmt.Errorf("PatchEthernetInterfaces(): failed to marshal EthernetInterface: %w", err)
		}
		henv, err := sc.PatchData(ctx, eiPath, "", headers, body)
		if err != nil {
			return henv, fmt.Errorf("PatchEthernetInterfaces(): failed to PATCH ethernet interface(s) to SMD: %w", err)
		}
		return henv, nil
	})
}

// PatchGroups is a wrapper function around OchamiClient.PatchData that takes a
// Group slice and a token, puts token in the request headers as an
// authorization bearer, marshals each group as JSON and sets it as the request
// body, then passes it to OchamiClient.PatchData using the group label in the
// path.
func (sc *SMDClient) PatchGroups(ctx context.Context, groups []Group, token string) client.BatchResult[client.HTTPEnvelope] {
	headers := client.NewHTTPHeaders()
	if token != "" {
		_ = headers.SetAuthorization(token) //nolint:errcheck // headers was allocated above and cannot be nil
	}
	return executeBatch(ctx, groups, func(ctx context.Context, group Group) (client.HTTPEnvelope, error) {
		if group.Label == "" {
			return client.HTTPEnvelope{}, fmt.Errorf("PatchGroups(): no group label specified to update")
		}
		groupPath, err := url.JoinPath(SMDRelpathGroups, group.Label)
		if err != nil {
			return client.HTTPEnvelope{}, fmt.Errorf("PatchGroups(): failed to join group path (%s) with group label (%s): %w", SMDRelpathGroups, group.Label, err)
		}
		body, err := json.Marshal(group)
		if err != nil {
			return client.HTTPEnvelope{}, fmt.Errorf("PatchGroups(): failed to marshal Group: %w", err)
		}
		henv, err := sc.PatchData(ctx, groupPath, "", headers, body)
		if err != nil {
			return henv, fmt.Errorf("PatchGroups(): failed to PATCH group %s in SMD: %w", group.Label, err)
		}
		return henv, nil
	})
}

// DeleteComponents takes a token and xnames and iteratively calls
// OchamiClient.DeleteData for each xname. This is necessary because SMD only
// allows deleting one xname at a time. Each input has one aligned result.
func (sc *SMDClient) DeleteComponents(ctx context.Context, token string, xnames ...string) client.BatchResult[client.HTTPEnvelope] {
	headers := client.NewHTTPHeaders()
	if token != "" {
		_ = headers.SetAuthorization(token) //nolint:errcheck // headers was allocated above and cannot be nil
	}
	return executeBatch(ctx, xnames, func(ctx context.Context, xname string) (client.HTTPEnvelope, error) {
		xnamePath, err := url.JoinPath(SMDRelpathComponents, xname)
		if err != nil {
			return client.HTTPEnvelope{}, fmt.Errorf("DeleteComponents(): failed join component path (%s) with xname (%s): %w", SMDRelpathComponents, xname, err)
		}
		henv, err := sc.DeleteData(ctx, xnamePath, "", headers, nil)
		if err != nil {
			return henv, fmt.Errorf("DeleteComponents(): failed to DELETE component %s in SMD: %w", xname, err)
		}
		return henv, nil
	})
}

// DeleteComponentsAll is a wrapper function around OchamiClient.DeleteData that
// takes a token, puts it in the request headers as an authorization bearer, and
// sends it in a DELETE request to the SMD components endpoint. This should
// delete all components SMD knows about if the token is authorized.
func (sc *SMDClient) DeleteComponentsAll(ctx context.Context, token string) (client.HTTPEnvelope, error) {
	var (
		henv    client.HTTPEnvelope
		headers *client.HTTPHeaders
		err     error
	)

	headers = client.NewHTTPHeaders()
	if token != "" {
		_ = headers.SetAuthorization(token) //nolint:errcheck // headers was allocated above and cannot be nil
	}
	henv, err = sc.DeleteData(ctx, SMDRelpathComponents, "", headers, nil)
	if err != nil {
		err = fmt.Errorf("DeleteComponentsAll(): failed to DELETE component(s) to SMD: %w", err)
	}

	return henv, err
}

// DeleteRedfishEndpoints takes a token and xnames and iteratively calls
// OchamiClient.DeleteData for each xname. This is necessary because SMD only
// allows deleting one xname at a time. Each input has one aligned result.
func (sc *SMDClient) DeleteRedfishEndpoints(ctx context.Context, token string, xnames ...string) client.BatchResult[client.HTTPEnvelope] {
	headers := client.NewHTTPHeaders()
	if token != "" {
		_ = headers.SetAuthorization(token) //nolint:errcheck // headers was allocated above and cannot be nil
	}
	return executeBatch(ctx, xnames, func(ctx context.Context, xname string) (client.HTTPEnvelope, error) {
		xnamePath, err := url.JoinPath(SMDRelpathRedfishEndpoints, xname)
		if err != nil {
			return client.HTTPEnvelope{}, fmt.Errorf("DeleteRedfishEndpoints(): failed join redfish endpoint path (%s) with xname (%s): %w", SMDRelpathRedfishEndpoints, xname, err)
		}
		henv, err := sc.DeleteData(ctx, xnamePath, "", headers, nil)
		if err != nil {
			return henv, fmt.Errorf("DeleteRedfishEndpoints(): failed to DELETE redfish endpoint %s in SMD: %w", xname, err)
		}
		return henv, nil
	})
}

// DeleteRedfishEndpointsAll is a wrapper function around
// OchamiClient.DeleteData that takes a token, puts it in the request headers as
// an authorization bearer, and sends it in a DELETE request to the SMD redfish
// endpoints endpoint. This should delete all redfish endpoints SMD knows about
// if the token is authorized.
func (sc *SMDClient) DeleteRedfishEndpointsAll(ctx context.Context, token string) (client.HTTPEnvelope, error) {
	var (
		henv    client.HTTPEnvelope
		headers *client.HTTPHeaders
		err     error
	)

	headers = client.NewHTTPHeaders()
	if token != "" {
		_ = headers.SetAuthorization(token) //nolint:errcheck // headers was allocated above and cannot be nil
	}
	henv, err = sc.DeleteData(ctx, SMDRelpathRedfishEndpoints, "", headers, nil)
	if err != nil {
		err = fmt.Errorf("DeleteRedfishEndpointsAll(): failed to DELETE redfish endpoint(s) to SMD: %w", err)
	}

	return henv, err
}

// DeleteEthernetInterfaces takes a token and one or more ethernet interface
// IDs and iteratively calls OchamiClient.DeleteData for each ID. This is
// necessary because SMD only allows deleting one ethernet interface at a time.
// Each input has one aligned result.
func (sc *SMDClient) DeleteEthernetInterfaces(ctx context.Context, token string, eIds ...string) client.BatchResult[client.HTTPEnvelope] {
	headers := client.NewHTTPHeaders()
	if token != "" {
		_ = headers.SetAuthorization(token) //nolint:errcheck // headers was allocated above and cannot be nil
	}
	return executeBatch(ctx, eIds, func(ctx context.Context, eID string) (client.HTTPEnvelope, error) {
		eIdPath, err := url.JoinPath(SMDRelpathEthernetInterfaces, eID)
		if err != nil {
			return client.HTTPEnvelope{}, fmt.Errorf("DeleteEthernetInterfaces(): failed join ethernet interface path (%s) with ethernet interface %s: %w", SMDRelpathEthernetInterfaces, eID, err)
		}
		henv, err := sc.DeleteData(ctx, eIdPath, "", headers, nil)
		if err != nil {
			return henv, fmt.Errorf("DeleteEthernetInterfaces(): failed to DELETE ethernet interface %s in SMD: %w", eID, err)
		}
		return henv, nil
	})
}

// DeleteEthernetInterfacesAll is a wrapper function around
// OchamiClient.DeleteData that takes a token, puts it in the request headers as
// an authorization bearer, and sends it in a DELETE request to the SMD ethernet
// interfaces endpoint. This should delete all ethernet interfaces SMD knows
// about if the token is authorized.
func (sc *SMDClient) DeleteEthernetInterfacesAll(ctx context.Context, token string) (client.HTTPEnvelope, error) {
	var (
		henv    client.HTTPEnvelope
		headers *client.HTTPHeaders
		err     error
	)

	headers = client.NewHTTPHeaders()
	if token != "" {
		_ = headers.SetAuthorization(token) //nolint:errcheck // headers was allocated above and cannot be nil
	}
	henv, err = sc.DeleteData(ctx, SMDRelpathEthernetInterfaces, "", headers, nil)
	if err != nil {
		err = fmt.Errorf("DeleteEthernetInterfacesAll(): failed to DELETE ethernet interface(s) to SMD: %w", err)
	}

	return henv, err
}

// DeleteComponentEndpoints takes a token and one or more xnames and
// iteratively calls OchamiClient.DeleteData for each xname. This is necessary
// because SMD only allows deleting one component endpoint at a time. Each input
// has one aligned result.
func (sc *SMDClient) DeleteComponentEndpoints(ctx context.Context, token string, xnames ...string) client.BatchResult[client.HTTPEnvelope] {
	headers := client.NewHTTPHeaders()
	if token != "" {
		_ = headers.SetAuthorization(token) //nolint:errcheck // headers was allocated above and cannot be nil
	}
	return executeBatch(ctx, xnames, func(ctx context.Context, xname string) (client.HTTPEnvelope, error) {
		finalEP, err := url.JoinPath(SMDRelpathComponentEndpoints, xname)
		if err != nil {
			return client.HTTPEnvelope{}, fmt.Errorf("DeleteComponentEndpoints(): failed join component endpoint path (%s) with xname %s: %w", SMDRelpathComponentEndpoints, xname, err)
		}
		henv, err := sc.DeleteData(ctx, finalEP, "", headers, nil)
		if err != nil {
			return henv, fmt.Errorf("DeleteComponentEndpoints(): failed to DELETE component endpoint %s in SMD: %w", xname, err)
		}
		return henv, nil
	})
}

// DeleteComponentEndpointsAll is a wrapper function around
// OchamiClient.DeleteData that takes a token, puts it in the request headers as
// an authorization bearer, and sends it in a DELETE request to the SMD
// component endpoints endpoint. This should delete all component endpoints SMD
// knows about if the token is authorized.
func (sc *SMDClient) DeleteComponentEndpointsAll(ctx context.Context, token string) (client.HTTPEnvelope, error) {
	var (
		henv    client.HTTPEnvelope
		headers *client.HTTPHeaders
		err     error
	)

	headers = client.NewHTTPHeaders()
	if token != "" {
		_ = headers.SetAuthorization(token) //nolint:errcheck // headers was allocated above and cannot be nil
	}
	henv, err = sc.DeleteData(ctx, SMDRelpathComponentEndpoints, "", headers, nil)
	if err != nil {
		err = fmt.Errorf("DeleteComponentEndpointsAll(): failed to DELETE component endpoint(s) to SMD: %w", err)
	}

	return henv, err
}

// DeleteGroups takes a token and one or more group labels and iteratively
// calls OchamiClient.DeleteData for each label. This is necessary because SMD
// only allows deleting one group at a time. Each input has one aligned result.
func (sc *SMDClient) DeleteGroups(ctx context.Context, token string, groupLabels ...string) client.BatchResult[client.HTTPEnvelope] {
	headers := client.NewHTTPHeaders()
	if token != "" {
		_ = headers.SetAuthorization(token) //nolint:errcheck // headers was allocated above and cannot be nil
	}
	return executeBatch(ctx, groupLabels, func(ctx context.Context, label string) (client.HTTPEnvelope, error) {
		labelPath, err := url.JoinPath(SMDRelpathGroups, label)
		if err != nil {
			return client.HTTPEnvelope{}, fmt.Errorf("DeleteGroups(): failed join group path (%s) with group label (%s): %w", SMDRelpathGroups, label, err)
		}
		henv, err := sc.DeleteData(ctx, labelPath, "", headers, nil)
		if err != nil {
			return henv, fmt.Errorf("DeleteGroups(): failed to DELETE group %s in SMD: %w", label, err)
		}
		return henv, nil
	})
}

// DeleteGroupMembers takes a token, group name, and one or more component IDs
// and iteratively calls OchamiClient.DeleteData for each member for the group.
// This is necessary because SMD only allows deleting one member at a time. Each
// input has one aligned result; operation-wide validation errors are returned
// separately.
func (sc *SMDClient) DeleteGroupMembers(ctx context.Context, token, group string, members ...string) (client.BatchResult[client.HTTPEnvelope], error) {
	if group == "" {
		return nil, fmt.Errorf("DeleteGroupMembers(): no group label specified to delete members from")
	}
	if len(members) == 0 {
		return nil, fmt.Errorf("DeleteGroupMembers(): no members specified to delete from group")
	}
	headers := client.NewHTTPHeaders()
	if token != "" {
		_ = headers.SetAuthorization(token) //nolint:errcheck // headers was allocated above and cannot be nil
	}
	results := executeBatch(ctx, members, func(ctx context.Context, member string) (client.HTTPEnvelope, error) {
		memberPath, err := url.JoinPath(SMDRelpathGroups, group, "members", member)
		if err != nil {
			return client.HTTPEnvelope{}, fmt.Errorf("DeleteGroupMembers(): failed join group path (%s) with group %s and member %s: %w", SMDRelpathGroups, group, member, err)
		}
		henv, err := sc.DeleteData(ctx, memberPath, "", headers, nil)
		if err != nil {
			return henv, fmt.Errorf("DeleteGroupMembers(): failed to DELETE member %s from group %s in SMD: %w", member, group, err)
		}
		return henv, nil
	})
	return results, nil
}

func executeBatch[T any](ctx context.Context, items []T, operation func(context.Context, T) (client.HTTPEnvelope, error)) client.BatchResult[client.HTTPEnvelope] {
	results := make(client.BatchResult[client.HTTPEnvelope], len(items))
	for i, item := range items {
		if err := ctx.Err(); err != nil {
			for ; i < len(results); i++ {
				results[i].Err = err
			}
			break
		}
		results[i].Value, results[i].Err = operation(ctx, item)
	}
	return results
}
