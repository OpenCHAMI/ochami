package smd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client"
	"github.com/openchami/schemas/schemas"
	"github.com/openchami/schemas/schemas/csm"
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

	SMDSubpathBulkNID = "BulkNID"
)

// Component is a minimal subset of SMD's Component struct that contains only
// what is necessary for sending a valid Component request to SMD.
type Component struct {
	ID      string `json:"ID"`
	Type    string `json:"Type"`
	State   string `json:"State,omitempty"`
	Enabled bool   `json:"Enabled,omitempty"`
	Role    string `json:"Role,omitempty"`
	Arch    string `json:"Arch,omitempty"`
	NID     int64  `json:"NID,omitempty"`
}

// ComponentSlice is a convenience data structure to make marshalling Component
// requests easier.
type ComponentSlice struct {
	Components []Component `json:"Components"`
}

// EthernetInterface is a minimal subset of SMD's EthernetInterface struct that
// contains only what is necessary for sending a valid EthernetInterface request
// to SMD.
type EthernetInterface struct {
	ID          string       `json:"ID"`
	ComponentID string       `json:"ComponentID"`
	Type        string       `json:"Type"`
	Description string       `json:"Description"`
	MACAddress  string       `json:"MACAddress"`
	IPAddresses []EthernetIP `json:"IPAddresses"`
}

type EthernetIP struct {
	IPAddress string `json:"IPAddress"`
	Network   string `json:"Network"`
}

// RedfishEndpointSlice is a convenience data structure to make marshalling
// RedfishEndpoint requests easier.
type RedfishEndpointSlice struct {
	RedfishEndpoints []csm.RedfishEndpoint `json:"RedfishEndpoints"`
}

// RedfishEndpointSliceV2 is a convenience data structure to make marshalling
// RedfishEndpointV2 requests easier.
type RedfishEndpointSliceV2 struct {
	RedfishEndpoints []RedfishEndpointV2 `json:"RedfishEndpoints"`
}

// RedfishEndpointV2 holds the redfish endpoint data read from/into SMD using
// schema v2. This schema supports dynamic creation of Components,
// ComponentEndpoints, and EthernetInterfaces from the Systems and Managers
// contained in this struct.
type RedfishEndpointV2 struct {
	csm.RedfishEndpoint
	SchemaVersion int       `json:"SchemaVersion"`
	Systems       []System  `json:"Systems"`
	Managers      []Manager `json:"Managers"`
}

// System represents data that would be retrieved from BMC System data, except
// reduced to a minimum needed for discovery.
type System struct {
	URI                string                      `json:"uri"`
	UUID               string                      `json:"uuid"`
	Name               string                      `json:"name"`
	EthernetInterfaces []schemas.EthernetInterface `json:"ethernet_interfaces"`
}

// Manager represents data that would be retrieved from BMC Manager data, except
// reduced to a minimum needed for discovery.
type Manager struct {
	System
	Description string `json:"description"`
	Type        string `json:"type"`
}

// Group represents the payload structure for SMD groups.
type Group struct {
	Label          string   `json:"label"`
	Description    string   `json:"description"`
	Tags           []string `json:"tags,omitempty"`
	ExclusiveGroup string   `json:"exclusiveGroup,omitempty"`
	Members        struct {
		IDs []string `json:"ids,omitempty"`
	} `json:"members,omitempty"`
}

// GroupMembers represents the payload structure for SMD group membership for
// PUT requests. It consists of only the group label and list of group IDs.
type GroupMembers struct {
	Label string   `json:"label"`
	IDs   []string `json:"ids"`
}

// NewClient takes a baseURI and returns a pointer to a new SMDClient. If an
// error occurred creating the embedded OchamiClient, it is returned. If
// insecure is true, TLS certificates will not be verified.
func NewClient(baseURI string, insecure bool) (*SMDClient, error) {
	oc, err := client.NewOchamiClient(serviceNameSMD, baseURI, insecure)
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
func (sc *SMDClient) GetStatus(component string) (client.HTTPEnvelope, error) {
	var (
		henv              client.HTTPEnvelope
		err               error
		smdStatusEndpoint string
	)
	switch component {
	case "":
		smdStatusEndpoint = path.Join(SMDRelpathService, "ready")
	case "all":
		smdStatusEndpoint = path.Join(SMDRelpathService, "values")
	default:
		return henv, fmt.Errorf("GetStatus(): unknown status component: %s", component)
	}

	henv, err = sc.GetData(smdStatusEndpoint, "", nil)
	if err != nil {
		err = fmt.Errorf("GetStatus(): error getting SMD all status: %w", err)
	}

	return henv, err
}

// GetComponentsAll is a wrapper function around OchamiClient.GetData that queries
// /State/Components.
func (sc *SMDClient) GetComponentsAll() (client.HTTPEnvelope, error) {
	henv, err := sc.GetData(SMDRelpathComponents, "", nil)
	if err != nil {
		err = fmt.Errorf("GetComponentsAll(): error getting components: %w", err)
	}

	return henv, err
}

// GetComponentsXname is like GetComponentsAll except that it takes a token and
// queries /State/Components/{xname}.
func (sc *SMDClient) GetComponentsXname(xname, token string) (client.HTTPEnvelope, error) {
	var henv client.HTTPEnvelope
	finalEP := SMDRelpathComponents + "/" + xname
	headers := client.NewHTTPHeaders()
	if token != "" {
		if err := headers.SetAuthorization(token); err != nil {
			return henv, fmt.Errorf("GetComponentsXname(): error setting token in HTTP headers: %w", err)
		}
	}
	henv, err := sc.GetData(finalEP, "", headers)
	if err != nil {
		err = fmt.Errorf("GetComponentsXname(): error getting component for xname %q: %w", xname, err)
	}

	return henv, err
}

// GetComponentsNid is like GetComponentsAll except that it takes a token and
// queries /State/Components/ByNID/{nid}.
func (sc *SMDClient) GetComponentsNid(nid int32, token string) (client.HTTPEnvelope, error) {
	var henv client.HTTPEnvelope
	finalEP := SMDRelpathComponents + "/ByNID/" + fmt.Sprint(nid)
	headers := client.NewHTTPHeaders()
	if token != "" {
		if err := headers.SetAuthorization(token); err != nil {
			return henv, fmt.Errorf("GetComponentsNid(): error setting token in HTTP headers: %w", err)
		}
	}
	henv, err := sc.GetData(finalEP, "", headers)
	if err != nil {
		err = fmt.Errorf("GetComponentsNid(): error getting component for NID %d: %w", nid, err)
	}

	return henv, err
}

// GetRedfishEndpoints is a wrapper around OchamiClient.GetData that takes an
// optional query string (without the "?") and a token. It sets token as the
// authorization bearer in the headers and passes the query string and headers
// to OchamiClient.GetData, using the SMD RedfishEndpoints API endpoint.
func (sc *SMDClient) GetRedfishEndpoints(query, token string) (client.HTTPEnvelope, error) {
	var (
		henv    client.HTTPEnvelope
		headers *client.HTTPHeaders
		err     error
	)
	headers = client.NewHTTPHeaders()
	if token != "" {
		if err = headers.SetAuthorization(token); err != nil {
			return henv, fmt.Errorf("GetRedfishEndpoints(): error setting token in HTTP headers: %w")
		}
	}
	henv, err = sc.GetData(SMDRelpathRedfishEndpoints, query, headers)
	if err != nil {
		err = fmt.Errorf("GetRedfishEndpoints(): error getting redfish endpoints: %w", err)
	}

	return henv, err
}

// GetEthernetInterfaces is a wrapper around OchamiClient.GetData that takes a
// query string and passes it to OchamiClient.GetData using SMD's ethernet
// interfaces endpoint.
func (sc *SMDClient) GetEthernetInterfaces(query string) (client.HTTPEnvelope, error) {
	henv, err := sc.GetData(SMDRelpathEthernetInterfaces, query, nil)
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
func (sc *SMDClient) GetEthernetInterfaceByID(id, token string, getIPs bool) (client.HTTPEnvelope, error) {
	var (
		ep      string
		err     error
		henv    client.HTTPEnvelope
		headers *client.HTTPHeaders
	)
	headers = client.NewHTTPHeaders()
	if token != "" {
		if err = headers.SetAuthorization(token); err != nil {
			return henv, fmt.Errorf("GetEthernetInterfaceByID(): error setting token in HTTP headers: %w", err)
		}
	}
	if getIPs {
		if ep, err = url.JoinPath(SMDRelpathEthernetInterfaces, id); err != nil {
			return henv, fmt.Errorf("GetEthernetInterfacesByID(): failed to join ethernet path (%s) with id (%s): %w", SMDRelpathEthernetInterfaces, id, err)
		}
		if ep, err = url.JoinPath(ep, "IPAddresses"); err != nil {
			return henv, fmt.Errorf("GetEthernetInterfacesByID(): failed to join endpoint %s with \"IPAddresses\": %w", ep, err)
		}
	} else {
		ep, err = url.JoinPath(SMDRelpathEthernetInterfaces, id)
	}
	henv, err = sc.GetData(ep, "", headers)
	if err != nil {
		err = fmt.Errorf("GetEthernetInterfacesByID(): failed to GET ethernet interfaces in SMD: %w", err)
	}

	return henv, err
}

// GetComponentEndpoints is similar to GetComponentEndpointsAll except that it
// iteratively calls OchamiClient.GetData on each xname passed. Each request
// has a corresponding client.HTTPEnvelope and error in returned slices. The
// function also returns a separate error if a control flow error occurs.
func (sc *SMDClient) GetComponentEndpoints(token string, xnames ...string) ([]client.HTTPEnvelope, []error, error) {
	var (
		errors  []error
		henvs   []client.HTTPEnvelope
		headers *client.HTTPHeaders
	)
	headers = client.NewHTTPHeaders()
	if token != "" {
		if err := headers.SetAuthorization(token); err != nil {
			return henvs, errors, fmt.Errorf("GetComponentEndpoints(): error setting token in HTTP headers: %w", err)
		}
	}
	for _, xname := range xnames {
		henv, err := sc.GetData(SMDRelpathComponentEndpoints+"/"+xname, "", headers)
		if err != nil {
			newErr := fmt.Errorf("GetComponentEndpoints(): failed to GET component endpoint from SMD: %w", err)
			log.Logger.Debug().Err(err).Msg("failed to get component endpoint")
			errors = append(errors, newErr)
			continue
		}
		errors = append(errors, nil)
		henvs = append(henvs, henv)
	}

	return henvs, errors, nil
}

// GetComponentEndpointsAll is a wrapper function around OchamiClient.GetData
// that takes a token and puts it in the request headers as an authorization
// bearer, then sends a get to the SMD component endpoint API endpoint.
func (sc *SMDClient) GetComponentEndpointsAll(token string) (client.HTTPEnvelope, error) {
	var (
		err     error
		henv    client.HTTPEnvelope
		headers *client.HTTPHeaders
	)
	headers = client.NewHTTPHeaders()
	if token != "" {
		if err = headers.SetAuthorization(token); err != nil {
			return henv, fmt.Errorf("GetComponentEndpointsAll(): error setting token in HTTP headers: %w", err)
		}
	}
	henv, err = sc.GetData(SMDRelpathComponentEndpoints, "", headers)
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
func (sc *SMDClient) GetGroups(query, token string) (client.HTTPEnvelope, error) {
	var (
		henv    client.HTTPEnvelope
		headers *client.HTTPHeaders
		err     error
	)
	headers = client.NewHTTPHeaders()
	if token != "" {
		if err = headers.SetAuthorization(token); err != nil {
			return henv, fmt.Errorf("GetGroups(): error setting token in HTTP headers: %w", err)
		}
	}
	henv, err = sc.GetData(SMDRelpathGroups, query, headers)
	if err != nil {
		err = fmt.Errorf("GetGroups(): error getting groups: %w", err)
	}

	return henv, err
}

// GetGroupMembers is a wrapper function around OchamiClient.GetData that takes
// a group name, which it passes to the GetData function using the SMD group
// membership endpoint. It also takes a token, which it puts into the headers as
// the authorization bearer.
func (sc *SMDClient) GetGroupMembers(group, token string) (client.HTTPEnvelope, error) {
	if group == "" {
		return client.HTTPEnvelope{}, fmt.Errorf("GetGroupMembers(): group label cannot be empty")
	}
	finalEP, err := url.JoinPath(SMDRelpathGroups, group, "members")
	if err != nil {
		return client.HTTPEnvelope{}, fmt.Errorf("GetGroupMembers(): failed to join group path (%s) with membership path for gorup %s: %w", SMDRelpathGroups, group)
	}
	headers := client.NewHTTPHeaders()
	if token != "" {
		if err := headers.SetAuthorization(token); err != nil {
			return client.HTTPEnvelope{}, fmt.Errorf("GetGroupMembers(): error setting token in HTTP headers: %w", err)
		}
	}
	henv, err := sc.GetData(finalEP, "", headers)
	if err != nil {
		err = fmt.Errorf("GetGroupMembers(): error getting group members for group %s: %w", group, err)
	}

	return henv, err
}

// PostComponents is a wrapper function around OchamiClient.PostData that takes
// a ComponentSlice and a token, puts the token in the request headers as an
// authorization bearer, marshalls compSlice as JSON and sets it as the request
// body, then passes it to Ochami.PostData.
func (sc *SMDClient) PostComponents(compSlice ComponentSlice, token string) (client.HTTPEnvelope, error) {
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
		if err := headers.SetAuthorization(token); err != nil {
			return henv, fmt.Errorf("PostComponents(): error setting token in HTTP headers: %w", err)
		}
	}
	henv, err = sc.PostData(SMDRelpathComponents, "", headers, body)
	if err != nil {
		err = fmt.Errorf("PostComponents(): failed to POST component(s) to SMD: %w", err)
	}

	return henv, err
}

// PostRedfishEndpoints is a wrapper function around OchamiClient.PostData that
// takes a RedfishEndpointSlice and a token, puts the token in the request
// headers as an authorization bearer, and iteratively calls
// OchamiClient.PostData using each RedfishEndpoint in the slice.
func (sc *SMDClient) PostRedfishEndpoints(rfes RedfishEndpointSlice, token string) ([]client.HTTPEnvelope, []error, error) {
	var (
		errors  []error
		henvs   []client.HTTPEnvelope
		headers *client.HTTPHeaders
	)
	headers = client.NewHTTPHeaders()
	if token != "" {
		if err := headers.SetAuthorization(token); err != nil {
			return henvs, errors, fmt.Errorf("PostRedfishEndpoints(): error setting token in HTTP headers: %w", err)
		}
	}
	for _, rfe := range rfes.RedfishEndpoints {
		var body client.HTTPBody
		var err error
		if body, err = json.Marshal(rfe); err != nil {
			newErr := fmt.Errorf("PostRedfishEndpoints(): failed to marshal RedfishEndpoint: %w", err)
			errors = append(errors, newErr)
			henvs = append(henvs, client.HTTPEnvelope{})
			continue
		}
		henv, err := sc.PostData(SMDRelpathRedfishEndpoints, "", headers, body)
		henvs = append(henvs, henv)
		if err != nil {
			newErr := fmt.Errorf("PostRedfishEndpoints(): failed to POST redfish endpoint to SMD: %w", err)
			errors = append(errors, newErr)
			continue
		}
		errors = append(errors, nil)
	}

	return henvs, errors, nil
}

// PostRedfishEndpointsV2 behaves like PostRedfishEndpoints except that it works
// with a RedfishEndpointSliceV2.
func (sc *SMDClient) PostRedfishEndpointsV2(rfes RedfishEndpointSliceV2, token string) ([]client.HTTPEnvelope, []error, error) {
	var (
		errors  []error
		henvs   []client.HTTPEnvelope
		headers *client.HTTPHeaders
	)
	headers = client.NewHTTPHeaders()
	if token != "" {
		if err := headers.SetAuthorization(token); err != nil {
			return henvs, errors, fmt.Errorf("PostRedfishEndpointsV2(): error setting token in HTTP headers: %w", err)
		}
	}
	for _, rfe := range rfes.RedfishEndpoints {
		var body client.HTTPBody
		var err error
		if body, err = json.Marshal(rfe); err != nil {
			newErr := fmt.Errorf("PostRedfishEndpointsV2(): failed to marshal RedfishEndpoint: %w", err)
			errors = append(errors, newErr)
			henvs = append(henvs, client.HTTPEnvelope{})
			continue
		}
		henv, err := sc.PostData(SMDRelpathRedfishEndpoints, "", headers, body)
		henvs = append(henvs, henv)
		if err != nil {
			newErr := fmt.Errorf("PostRedfishEndpointsV2(): failed to POST redfish endpoint to SMD: %w", err)
			errors = append(errors, newErr)
			continue
		}
		errors = append(errors, nil)
	}

	return henvs, errors, nil
}

// PostEthernetInterfaces is a wrapper function around OchamiClient.PostData
// that takes a slice of EthernetInterfaces and a token, puts the token in the
// request headers as an authorization bearer, and iteratively calls
// OchamiClient.PostData using each EthernetInterface in the slice.
func (sc *SMDClient) PostEthernetInterfaces(eis []EthernetInterface, token string) ([]client.HTTPEnvelope, []error, error) {
	var (
		errors  []error
		henvs   []client.HTTPEnvelope
		headers *client.HTTPHeaders
	)
	headers = client.NewHTTPHeaders()
	if token != "" {
		if err := headers.SetAuthorization(token); err != nil {
			return henvs, errors, fmt.Errorf("PostEthernetInterfaces(): error setting token in HTTP headers: %w", err)
		}
	}
	for _, ei := range eis {
		var body client.HTTPBody
		var err error
		if body, err = json.Marshal(ei); err != nil {
			newErr := fmt.Errorf("PostEthernetInterfaces(): failed to marshal EthernetInterface: %w", err)
			errors = append(errors, newErr)
			henvs = append(henvs, client.HTTPEnvelope{})
			continue
		}
		henv, err := sc.PostData(SMDRelpathEthernetInterfaces, "", headers, body)
		henvs = append(henvs, henv)
		if err != nil {
			newErr := fmt.Errorf("PostEthernetInterfaces(): failed to POST ethernet interface(s) to SMD: %w", err)
			errors = append(errors, newErr)
			continue
		}
		errors = append(errors, nil)
	}

	return henvs, errors, nil
}

// PostGroups is a wrapper function around OchamiClient.PostData that takes a
// Group slice and a token, puts the token in the request headers as an
// authorization bearer, and iteratively calls OchamiClient.PostData using each
// Group in the slice.
func (sc *SMDClient) PostGroups(groups []Group, token string) ([]client.HTTPEnvelope, []error, error) {
	var (
		errors  []error
		henvs   []client.HTTPEnvelope
		headers *client.HTTPHeaders
	)
	headers = client.NewHTTPHeaders()
	if token != "" {
		if err := headers.SetAuthorization(token); err != nil {
			return henvs, errors, fmt.Errorf("PostGroups(): error setting token in HTTP headers: %w", err)
		}
	}
	for _, group := range groups {
		var body client.HTTPBody
		var err error
		if body, err = json.Marshal(group); err != nil {
			newErr := fmt.Errorf("PostGroups(): failed to marshal Group: %w", err)
			errors = append(errors, newErr)
			henvs = append(henvs, client.HTTPEnvelope{})
			continue
		}
		henv, err := sc.PostData(SMDRelpathGroups, "", headers, body)
		henvs = append(henvs, henv)
		if err != nil {
			newErr := fmt.Errorf("PostGroups(): failed to POST group to SMD: %w", err)
			errors = append(errors, newErr)
			continue
		}
		errors = append(errors, nil)
	}

	return henvs, errors, nil
}

// PostGroupMembers is a wrapper function around OchamiClient.PostData that
// takes a token, group name, and a list of one or more component IDs. It puts
// the token in the request headers as an authorization bearer, and iteratively
// calls OchamiClient.PostData for each member on the group.
func (sc *SMDClient) PostGroupMembers(token, group string, members ...string) ([]client.HTTPEnvelope, []error, error) {
	var (
		henvs   []client.HTTPEnvelope
		headers *client.HTTPHeaders
		body    client.HTTPBody
		errors  []error
	)
	if group == "" {
		return henvs, errors, fmt.Errorf("PostGroupMembers(): no group label specified to add members to")
	}
	if len(members) == 0 {
		return henvs, errors, fmt.Errorf("PostGroupMembers(): no new members specified to add to group")
	}
	headers = client.NewHTTPHeaders()
	if token != "" {
		if err := headers.SetAuthorization(token); err != nil {
			return henvs, errors, fmt.Errorf("PostGroupMembers(): error setting token in HTTP headers: %w", err)
		}
	}
	for _, member := range members {
		groupPath, err := url.JoinPath(SMDRelpathGroups, group, "members")
		if err != nil {
			newErr := fmt.Errorf("PostGroupMembers(): failed to join group path (%s) with group label (%s): %w", SMDRelpathGroups, group)
			henvs = append(henvs, client.HTTPEnvelope{})
			errors = append(errors, newErr)
			continue
		}
		m := make(map[string]string)
		m["id"] = member
		if body, err = json.Marshal(m); err != nil {
			newErr := fmt.Errorf("PostGroupMembers(): failed to marshal member id %s: %w", member, err)
			henvs = append(henvs, client.HTTPEnvelope{})
			errors = append(errors, newErr)
			continue
		}
		henv, err := sc.PostData(groupPath, "", headers, body)
		henvs = append(henvs, henv)
		if err != nil {
			newErr := fmt.Errorf("PostGroupMembers(): failed to POST member %s to group %s: %w", member, group, err)
			errors = append(errors, newErr)
			continue
		}
		errors = append(errors, nil)
	}

	return henvs, errors, nil
}

// PutComponents takes a ComponentSlice and a token and iteratively calls
// OchamiClient.PutData for each Component in the contained list. This is
// necessary because SMD only allows sending a PUT for a single Component using
// its ID in the endpoint path, forcing the client to send only a single
// Component per request. A slice of client.HTTPEnvelopes is returned
// containing one client.HTTPEnvelope per HTTP request, as well as an error
// slice containing errors corresponding to each HTTP request. The indexes of
// these should correspond.  If an error in the function itself occurred, a
// separate error is returned.  This is to distinguish iterative HTTP request
// errors from control flow errors.
func (sc *SMDClient) PutComponents(compSlice ComponentSlice, token string) ([]client.HTTPEnvelope, []error, error) {
	var (
		henvs   []client.HTTPEnvelope
		errors  []error
		headers *client.HTTPHeaders
	)
	headers = client.NewHTTPHeaders()
	if token != "" {
		if err := headers.SetAuthorization(token); err != nil {
			return henvs, errors, fmt.Errorf("PutComponents(): error setting token in HTTP headers: %w", err)
		}
	}
	for _, comp := range compSlice.Components {
		if comp.ID == "" {
			newErr := fmt.Errorf("PutComponents(): unable to update component with blank ID")
			henvs = append(henvs, client.HTTPEnvelope{})
			errors = append(errors, newErr)
			continue
		}
		xnamePath, err := url.JoinPath(SMDRelpathComponents, comp.ID)
		if err != nil {
			newErr := fmt.Errorf("PutComponents(): failed join component path (%s) with xname (%s): %w", SMDRelpathComponents, comp.ID, err)
			henvs = append(henvs, client.HTTPEnvelope{})
			errors = append(errors, newErr)
			continue
		}
		// SMD is weird and requires the PUT body to be a structure that
		// _contains_ the component, so we do that here.
		putComp := map[string]any{"Component": comp, "Force": true}
		body, marshalErr := json.Marshal(putComp)
		if marshalErr != nil {
			newErr := fmt.Errorf("PutComponents(): failed to marshal component into JSON: %w", marshalErr)
			errors = append(errors, newErr)
		}
		henv, err := sc.PutData(xnamePath, "", headers, body)
		henvs = append(henvs, henv)
		if err != nil {
			newErr := fmt.Errorf("PutComponents(): failed to PUT component %s in SMD: %w", comp.ID, err)
			errors = append(errors, newErr)
			continue
		}
		errors = append(errors, nil)
	}

	return henvs, errors, nil
}

// PutRedfishEndpoints is a wrapper function around OchamiClient.PutData that
// takes a RedfishEndpointSlice and a token, puts the token in the request
// headers as an authorization bearer, and iteratively calls
// OchamiClient.PutData using each RedfishEndpoint in the slice.
func (sc *SMDClient) PutRedfishEndpoints(rfes RedfishEndpointSlice, token string) ([]client.HTTPEnvelope, []error, error) {
	var (
		errors  []error
		henvs   []client.HTTPEnvelope
		headers *client.HTTPHeaders
	)
	headers = client.NewHTTPHeaders()
	if token != "" {
		if err := headers.SetAuthorization(token); err != nil {
			return henvs, errors, fmt.Errorf("PutRedfishEndpoints(): error setting token in HTTP headers: %w", err)
		}
	}
	for _, rfe := range rfes.RedfishEndpoints {
		var body client.HTTPBody
		var err error
		if rfe.ID == "" {
			newErr := fmt.Errorf("PutRedfishEndpoints(): unable to update redfish endpoint with blank ID")
			henvs = append(henvs, client.HTTPEnvelope{})
			errors = append(errors, newErr)
			continue
		}
		xnamePath, err := url.JoinPath(SMDRelpathRedfishEndpoints, rfe.ID)
		if err != nil {
			newErr := fmt.Errorf("PutRedfishEndpoints(): failed to join redfish endpoint path (%s) with xname (%s): %w", SMDRelpathRedfishEndpoints, rfe.ID, err)
			henvs = append(henvs, client.HTTPEnvelope{})
			errors = append(errors, newErr)
			continue
		}
		if body, err = json.Marshal(rfe); err != nil {
			newErr := fmt.Errorf("PutRedfishEndpoints(): failed to marshal RedfishEndpoint: %w", err)
			errors = append(errors, newErr)
			henvs = append(henvs, client.HTTPEnvelope{})
			continue
		}
		henv, err := sc.PutData(xnamePath, "", headers, body)
		henvs = append(henvs, henv)
		if err != nil {
			newErr := fmt.Errorf("PutRedfishEndpoints(): failed to PUT redfish endpoint to SMD: %w", err)
			errors = append(errors, newErr)
			continue
		}
		errors = append(errors, nil)
	}

	return henvs, errors, nil
}

// PutRedfishEndpointsV2 behaves like PutRedfishEndpoints except that it works
// with a RedfishEndpointSliceV2.
func (sc *SMDClient) PutRedfishEndpointsV2(rfes RedfishEndpointSliceV2, token string) ([]client.HTTPEnvelope, []error, error) {
	var (
		errors  []error
		henvs   []client.HTTPEnvelope
		headers *client.HTTPHeaders
	)
	headers = client.NewHTTPHeaders()
	if token != "" {
		if err := headers.SetAuthorization(token); err != nil {
			return henvs, errors, fmt.Errorf("PutRedfishEndpointsV2(): error setting token in HTTP headers: %w", err)
		}
	}
	for _, rfe := range rfes.RedfishEndpoints {
		var body client.HTTPBody
		var err error
		if rfe.ID == "" {
			newErr := fmt.Errorf("PutRedfishEndpointsV2(): unable to update redfish endpoint with blank ID")
			henvs = append(henvs, client.HTTPEnvelope{})
			errors = append(errors, newErr)
			continue
		}
		xnamePath, err := url.JoinPath(SMDRelpathRedfishEndpoints, rfe.ID)
		if err != nil {
			newErr := fmt.Errorf("PutRedfishEndpointsV2(): failed to join redfish endpoint path (%s) with xname (%s): %w", SMDRelpathRedfishEndpoints, rfe.ID, err)
			henvs = append(henvs, client.HTTPEnvelope{})
			errors = append(errors, newErr)
			continue
		}
		if body, err = json.Marshal(rfe); err != nil {
			newErr := fmt.Errorf("PutRedfishEndpointsV2(): failed to marshal RedfishEndpoint: %w", err)
			errors = append(errors, newErr)
			henvs = append(henvs, client.HTTPEnvelope{})
			continue
		}
		henv, err := sc.PutData(xnamePath, "", headers, body)
		henvs = append(henvs, henv)
		if err != nil {
			newErr := fmt.Errorf("PutRedfishEndpointsV2(): failed to PUT redfish endpoint to SMD: %w", err)
			errors = append(errors, newErr)
			continue
		}
		errors = append(errors, nil)
	}

	return henvs, errors, nil
}

// PutGroupMembers is a wrapper function around OchamiClient.PutData that takes
// a token, group name, and a list of one or more component IDs. It puts the
// token in the request headers as an authorization bearer and calls
// OchamiClient.PostData on the SMD group members API endpoint with the group
// and member list.
func (sc *SMDClient) PutGroupMembers(token, group string, members ...string) (client.HTTPEnvelope, error) {
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
		if err := headers.SetAuthorization(token); err != nil {
			return henv, fmt.Errorf("PutGroupMembers(): error setting token in HTTP headers: %w", err)
		}
	}

	// Calculate endpoint path for group
	groupPath, err := url.JoinPath(SMDRelpathGroups, group, "members")
	if err != nil {
		return henv, fmt.Errorf("PutGroupMembers(): failed to join group path (%s) with group label (%s): %w", SMDRelpathGroups, group)
	}

	// Send request and return response
	g := GroupMembers{
		Label: group,
		IDs:   members,
	}
	if body, err = json.Marshal(g); err != nil {
		return henv, fmt.Errorf("PutGroupMembers(): failed to marshal group data: %w", err)
	}
	henv, err = sc.PutData(groupPath, "", headers, body)
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
func (sc *SMDClient) PatchComponentsNID(comps ComponentSlice, token string) (client.HTTPEnvelope, error) {
	// Set token in request headers
	headers := client.NewHTTPHeaders()
	if token != "" {
		if err := headers.SetAuthorization(token); err != nil {
			return client.HTTPEnvelope{}, fmt.Errorf("PatchComponentsNID(): error setting token in HTTP headers: %w", err)
		}
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
	henv, err := sc.PatchData(nidPath, "", headers, body)
	if err != nil {
		err = fmt.Errorf("PatchComponentsNID(): failed to PATCH stripped components in SMD: %w", err)
	}

	return henv, err
}

// PatchEthernetInterfaces is a wrapper function around OchamiClient.PatchData
// that takes a slice of EthernetInterfaces and a token, puts the token in the
// request headers as an authorization bearer, and iteratively calls
// OchamiClient.PatchData using each EthernetInterface in the slice.
func (sc *SMDClient) PatchEthernetInterfaces(eis []EthernetInterface, token string) ([]client.HTTPEnvelope, []error, error) {
	var (
		errors  []error
		henvs   []client.HTTPEnvelope
		headers *client.HTTPHeaders
	)
	headers = client.NewHTTPHeaders()
	if token != "" {
		if err := headers.SetAuthorization(token); err != nil {
			return henvs, errors, fmt.Errorf("PatchEthernetInterfaces(): error setting token in HTTP headers: %w", err)
		}
	}
	for _, ei := range eis {
		var body client.HTTPBody
		var err error
		if ei.ID == "" {
			if ei.MACAddress != "" {
				log.Logger.Warn().Msgf("PatchEthernetInterfaces(): ID for ethernet interface is blank, attempting to adapt from MAC address (%s)", ei.MACAddress)
				newID := strings.ToLower(ei.MACAddress)
				newID = strings.ReplaceAll(newID, ":", "")
				newID = strings.ReplaceAll(newID, "-", "")
				newID = strings.ReplaceAll(newID, "_", "")
				ei.ID = newID
			} else {
				newErr := fmt.Errorf("PatchEthernetInterfaces(): unable to patch ethernet interface with both blank ID and blank MAC address")
				henvs = append(henvs, client.HTTPEnvelope{})
				errors = append(errors, newErr)
				continue
			}
		}
		eiPath, err := url.JoinPath(SMDRelpathEthernetInterfaces, ei.ID)
		if err != nil {
			newErr := fmt.Errorf("PatchEthernetInterfaces(): failed to join ethernet interface path (%s) with ethernet interface ID (%s): %w", SMDRelpathEthernetInterfaces, ei.ID, err)
			henvs = append(henvs, client.HTTPEnvelope{})
			errors = append(errors, newErr)
			continue
		}
		if body, err = json.Marshal(ei); err != nil {
			newErr := fmt.Errorf("PatchEthernetInterfaces(): failed to marshal EthernetInterface: %w", err)
			errors = append(errors, newErr)
			henvs = append(henvs, client.HTTPEnvelope{})
			continue
		}
		henv, err := sc.PatchData(eiPath, "", headers, body)
		henvs = append(henvs, henv)
		if err != nil {
			newErr := fmt.Errorf("PatchEthernetInterfaces(): failed to PATCH ethernet interface(s) to SMD: %w", err)
			errors = append(errors, newErr)
			continue
		}
		errors = append(errors, nil)
	}

	return henvs, errors, nil
}

// PatchGroups is a wrapper function around OchamiClient.PatchData that takes a
// Group slice and a token, puts token in the request headers as an
// authorization bearer, marshals each group as JSON and sets it as the request
// body, then passes it to OchamiClient.PatchData using the group label in the
// path.
func (sc *SMDClient) PatchGroups(groups []Group, token string) ([]client.HTTPEnvelope, []error, error) {
	var (
		henvs   []client.HTTPEnvelope
		headers *client.HTTPHeaders
		body    client.HTTPBody
		errors  []error
	)
	headers = client.NewHTTPHeaders()
	if token != "" {
		if err := headers.SetAuthorization(token); err != nil {
			return henvs, errors, fmt.Errorf("PatchGroups(): error setting token in HTTP headers: %w", err)
		}
	}
	for _, group := range groups {
		if group.Label == "" {
			newErr := fmt.Errorf("PatchGroups(): no group label specified to update")
			henvs = append(henvs, client.HTTPEnvelope{})
			errors = append(errors, newErr)
			continue
		}
		groupPath, err := url.JoinPath(SMDRelpathGroups, group.Label)
		if err != nil {
			newErr := fmt.Errorf("PatchGroups(): failed to join group path (%s) with group label (%s): %w", SMDRelpathGroups, group.Label)
			henvs = append(henvs, client.HTTPEnvelope{})
			errors = append(errors, newErr)
			continue
		}
		if body, err = json.Marshal(group); err != nil {
			newErr := fmt.Errorf("PatchGroups(): failed to marshal Group: %w")
			henvs = append(henvs, client.HTTPEnvelope{})
			errors = append(errors, newErr)
			continue
		}
		henv, err := sc.PatchData(groupPath, "", headers, body)
		henvs = append(henvs, henv)
		if err != nil {
			newErr := fmt.Errorf("PatchGroups(): failed to PATCH group %s in SMD: %w", group.Label, err)
			errors = append(errors, newErr)
			continue
		}
		errors = append(errors, nil)
	}

	return henvs, errors, nil
}

// DeleteComponents takes a token and xnames and iteratively calls
// OchamiClient.DeleteData for each xname. This is necessary because SMD only
// allows deleting one xname at a time. A slice of client.HTTPEnvelopes is
// returned containing one client.HTTPEnvelope per deletion, as well as an
// error slice containing errors corresponding to each deletion. The indexes of
// these should correspond. If an error in the function itself occurred, a
// separate error is returned. This is to distinguish HTTP request errors from
// control flow errors.
func (sc *SMDClient) DeleteComponents(token string, xnames ...string) ([]client.HTTPEnvelope, []error, error) {
	var (
		errors  []error
		henvs   []client.HTTPEnvelope
		headers *client.HTTPHeaders
	)
	headers = client.NewHTTPHeaders()
	if token != "" {
		if err := headers.SetAuthorization(token); err != nil {
			return henvs, errors, fmt.Errorf("DeleteComponents(): error setting token in HTTP headers: %w", err)
		}
	}
	for _, xname := range xnames {
		xnamePath, err := url.JoinPath(SMDRelpathComponents, xname)
		if err != nil {
			newErr := fmt.Errorf("DeleteComponents(): failed join component path (%s) with xname (%s): %w", SMDRelpathComponents, xname, err)
			henvs = append(henvs, client.HTTPEnvelope{})
			errors = append(errors, newErr)
			continue
		}
		henv, err := sc.DeleteData(xnamePath, "", headers, nil)
		henvs = append(henvs, henv)
		if err != nil {
			newErr := fmt.Errorf("DeleteComponents(): failed to DELETE component %s in SMD: %w", xname, err)
			errors = append(errors, newErr)
			continue
		}
		errors = append(errors, nil)
	}

	return henvs, errors, nil
}

// DeleteComponentsAll is a wrapper function around OchamiClient.DeleteData that
// takes a token, puts it in the request headers as an authorization bearer, and
// sends it in a DELETE request to the SMD components endpoint. This should
// delete all components SMD knows about if the token is authorized.
func (sc *SMDClient) DeleteComponentsAll(token string) (client.HTTPEnvelope, error) {
	var (
		henv    client.HTTPEnvelope
		headers *client.HTTPHeaders
		err     error
	)

	headers = client.NewHTTPHeaders()
	if token != "" {
		if err := headers.SetAuthorization(token); err != nil {
			return henv, fmt.Errorf("DeleteComponentsAll(): error setting token in HTTP headers: %w", err)
		}
	}
	henv, err = sc.DeleteData(SMDRelpathComponents, "", headers, nil)
	if err != nil {
		err = fmt.Errorf("DeleteComponentsAll(): failed to DELETE component(s) to SMD: %w", err)
	}

	return henv, err
}

// DeleteRedfishEndpoints takes a token and xnames and iteratively calls
// OchamiClient.DeleteData for each xname. This is necessary because SMD only
// allows deleting one xname at a time. A slice of client.HTTPEnvelopes is
// returned containing one client.HTTPEnvelope per deletion, as well as an
// error slice containing errors corresponding to each deletion. The indexes of
// these should correspond. If an error in the function itself occurred, a
// separate error is returned. This is to distinguish HTTP request errors from
// control flow errors.
func (sc *SMDClient) DeleteRedfishEndpoints(token string, xnames ...string) ([]client.HTTPEnvelope, []error, error) {
	var (
		errors  []error
		henvs   []client.HTTPEnvelope
		headers *client.HTTPHeaders
	)
	headers = client.NewHTTPHeaders()
	if token != "" {
		if err := headers.SetAuthorization(token); err != nil {
			return henvs, errors, fmt.Errorf("DeleteRedfishEndpoints(): error setting token in HTTP headers: %w", err)
		}
	}
	for _, xname := range xnames {
		xnamePath, err := url.JoinPath(SMDRelpathRedfishEndpoints, xname)
		if err != nil {
			newErr := fmt.Errorf("DeleteRedfishEndpoints(): failed join redfish endpoint path (%s) with xname (%s): %w", SMDRelpathRedfishEndpoints, xname, err)
			henvs = append(henvs, client.HTTPEnvelope{})
			errors = append(errors, newErr)
			continue
		}
		henv, err := sc.DeleteData(xnamePath, "", headers, nil)
		henvs = append(henvs, henv)
		if err != nil {
			newErr := fmt.Errorf("DeleteRedfishEndpoints(): failed to DELETE redfish endpoint %s in SMD: %w", xname, err)
			errors = append(errors, newErr)
			continue
		}
		errors = append(errors, nil)
	}

	return henvs, errors, nil
}

// DeleteRedfishEndpointsAll is a wrapper function around
// OchamiClient.DeleteData that takes a token, puts it in the request headers as
// an authorization bearer, and sends it in a DELETE request to the SMD redfish
// endpoints endpoint. This should delete all redfish endpoints SMD knows about
// if the token is authorized.
func (sc *SMDClient) DeleteRedfishEndpointsAll(token string) (client.HTTPEnvelope, error) {
	var (
		henv    client.HTTPEnvelope
		headers *client.HTTPHeaders
		err     error
	)

	headers = client.NewHTTPHeaders()
	if token != "" {
		if err := headers.SetAuthorization(token); err != nil {
			return henv, fmt.Errorf("DeleteRedfishEndpointsAll(): error setting token in HTTP headers: %w", err)
		}
	}
	henv, err = sc.DeleteData(SMDRelpathRedfishEndpoints, "", headers, nil)
	if err != nil {
		err = fmt.Errorf("DeleteRedfishEndpointsAll(): failed to DELETE redfish endpoint(s) to SMD: %w", err)
	}

	return henv, err
}

// DeleteEthernetInterfaces takes a token and one or more ethernet interface
// IDs and iteratively calls OchamiClient.DeleteData for each ID. This is
// necessary because SMD only allows deleting one ethernet interface at a time.
// A slice of client.HTTPEnvelopes is returned containing one
// client.HTTPEnvelope per deletion, as well as an error slice containing
// errors corresponding to each deletion. The indexes of these should
// correspond. If an error in the function itself occurred, a separate error is
// returned. This is to distinguish HTTP request errors from control flow
// errors.
func (sc *SMDClient) DeleteEthernetInterfaces(token string, eIds ...string) ([]client.HTTPEnvelope, []error, error) {
	var (
		errors  []error
		henvs   []client.HTTPEnvelope
		headers *client.HTTPHeaders
	)
	headers = client.NewHTTPHeaders()
	if token != "" {
		if err := headers.SetAuthorization(token); err != nil {
			return henvs, errors, fmt.Errorf("DeleteEthernetInterfaces(): error setting token in HTTP headers: %w", err)
		}
	}
	for _, eId := range eIds {
		eIdPath, err := url.JoinPath(SMDRelpathEthernetInterfaces, eId)
		if err != nil {
			newErr := fmt.Errorf("DeleteEthernetInterfaces(): failed join ethernet interface path (%s) with ethernet interface %s: %w", SMDRelpathEthernetInterfaces, eId, err)
			henvs = append(henvs, client.HTTPEnvelope{})
			errors = append(errors, newErr)
			continue
		}
		henv, err := sc.DeleteData(eIdPath, "", headers, nil)
		henvs = append(henvs, henv)
		if err != nil {
			newErr := fmt.Errorf("DeleteEthernetInterfaces(): failed to DELETE ethernet interface %s in SMD: %w", eId, err)
			errors = append(errors, newErr)
			continue
		}
		errors = append(errors, nil)
	}

	return henvs, errors, nil
}

// DeleteEthernetInterfacesAll is a wrapper function around
// OchamiClient.DeleteData that takes a token, puts it in the request headers as
// an authorization bearer, and sends it in a DELETE request to the SMD ethernet
// interfaces endpoint. This should delete all ethernet interfaces SMD knows
// about if the token is authorized.
func (sc *SMDClient) DeleteEthernetInterfacesAll(token string) (client.HTTPEnvelope, error) {
	var (
		henv    client.HTTPEnvelope
		headers *client.HTTPHeaders
		err     error
	)

	headers = client.NewHTTPHeaders()
	if token != "" {
		if err := headers.SetAuthorization(token); err != nil {
			return henv, fmt.Errorf("DeleteEthernetInterfacesAll(): error setting token in HTTP headers: %w", err)
		}
	}
	henv, err = sc.DeleteData(SMDRelpathEthernetInterfaces, "", headers, nil)
	if err != nil {
		err = fmt.Errorf("DeleteEthernetInterfacesAll(): failed to DELETE ethernet interface(s) to SMD: %w", err)
	}

	return henv, err
}

// DeleteComponentEndpoints takes a token and one or more xnames and
// iteratively calls OchamiClient.DeleteData for each xname. This is necessary
// because SMD only allows deleting one component endpoint at a time. A slice
// of client.HTTPEnvelopes is returned containing one client.HTTPEnvelope per
// deletion, as well as an error slice containing errors corresponding to each
// deletion. The indexes of these should correspond. If an error in the
// function itself occurred, a separate error is returned. This is to
// distinguish HTTP request errors from control flow errors.
func (sc *SMDClient) DeleteComponentEndpoints(token string, xnames ...string) ([]client.HTTPEnvelope, []error, error) {
	var (
		errors  []error
		henvs   []client.HTTPEnvelope
		headers *client.HTTPHeaders
	)
	headers = client.NewHTTPHeaders()
	if token != "" {
		if err := headers.SetAuthorization(token); err != nil {
			return henvs, errors, fmt.Errorf("DeleteComponentEndpoints(): error setting token in HTTP headers: %w", err)
		}
	}
	for _, xname := range xnames {
		finalEP, err := url.JoinPath(SMDRelpathComponentEndpoints, xname)
		if err != nil {
			newErr := fmt.Errorf("DeleteComponentEndpoints(): failed join component endpoint path (%s) with xname %s: %w", SMDRelpathComponentEndpoints, xname, err)
			henvs = append(henvs, client.HTTPEnvelope{})
			errors = append(errors, newErr)
			continue
		}
		henv, err := sc.DeleteData(finalEP, "", headers, nil)
		henvs = append(henvs, henv)
		if err != nil {
			newErr := fmt.Errorf("DeleteComponentEndpoints(): failed to DELETE component endpoint %s in SMD: %w", xname, err)
			errors = append(errors, newErr)
			continue
		}
		errors = append(errors, nil)
	}

	return henvs, errors, nil
}

// DeleteComponentEndpointsAll is a wrapper function around
// OchamiClient.DeleteData that takes a token, puts it in the request headers as
// an authorization bearer, and sends it in a DELETE request to the SMD
// component endpoints endpoint. This should delete all component endpoints SMD
// knows about if the token is authorized.
func (sc *SMDClient) DeleteComponentEndpointsAll(token string) (client.HTTPEnvelope, error) {
	var (
		henv    client.HTTPEnvelope
		headers *client.HTTPHeaders
		err     error
	)

	headers = client.NewHTTPHeaders()
	if token != "" {
		if err := headers.SetAuthorization(token); err != nil {
			return henv, fmt.Errorf("DeleteComponentEndpointsAll(): error setting token in HTTP headers: %w", err)
		}
	}
	henv, err = sc.DeleteData(SMDRelpathComponentEndpoints, "", headers, nil)
	if err != nil {
		err = fmt.Errorf("DeleteComponentEndpointsAll(): failed to DELETE component endpoint(s) to SMD: %w", err)
	}

	return henv, err
}

// DeleteGroups takes a token and one or more group labels and iteratively
// calls OchamiClient.DeleteData for each label. This is necessary because SMD
// only allows deleting one group at a time. A slice of client.HTTPEnvelopes is
// returned containing one client.HTTPEnvelope per deletion, as well as an
// error slice containing errors corresponding to each deletion. The indexes of
// these should correspond. If an error in the function itself occurred, a
// separate error is returned. This is to distinguish HTTP request errors from
// control flow errors.
func (sc *SMDClient) DeleteGroups(token string, groupLabels ...string) ([]client.HTTPEnvelope, []error, error) {
	var (
		errors  []error
		henvs   []client.HTTPEnvelope
		headers *client.HTTPHeaders
	)
	headers = client.NewHTTPHeaders()
	if token != "" {
		if err := headers.SetAuthorization(token); err != nil {
			return henvs, errors, fmt.Errorf("DeleteGroups(): error setting token in HTTP headers: %w", err)
		}
	}
	for _, label := range groupLabels {
		labelPath, err := url.JoinPath(SMDRelpathGroups, label)
		if err != nil {
			newErr := fmt.Errorf("DeleteGroups(): failed join group path (%s) with group label (%s): %w", SMDRelpathGroups, label, err)
			henvs = append(henvs, client.HTTPEnvelope{})
			errors = append(errors, newErr)
			continue
		}
		henv, err := sc.DeleteData(labelPath, "", headers, nil)
		henvs = append(henvs, henv)
		if err != nil {
			newErr := fmt.Errorf("DeleteGroups(): failed to DELETE group %s in SMD: %w", label, err)
			errors = append(errors, newErr)
			continue
		}
		errors = append(errors, nil)
	}

	return henvs, errors, nil
}

// DeleteGroupMembers takes a token, group name, and one or more component IDs
// and iteratively calls OchamiClient.DeleteData for each member for the group.
// This is necessary because SMD only allows deleting one member at a time. A
// slice of client.HTTPEnvelopes is returned containing one client.HTTPEnvelope
// per deletion, as well as an error slice containing errors corresponding to
// each deletion. The indexes of these should correspond. If an error in the
// function itself occurred, a separate error is returned. This is to
// distinguish HTTP request errors from control flow errors.
func (sc *SMDClient) DeleteGroupMembers(token, group string, members ...string) ([]client.HTTPEnvelope, []error, error) {
	var (
		errors  []error
		henvs   []client.HTTPEnvelope
		headers *client.HTTPHeaders
	)
	headers = client.NewHTTPHeaders()
	if token != "" {
		if err := headers.SetAuthorization(token); err != nil {
			return henvs, errors, fmt.Errorf("DeleteGroupMembers(): error setting token in HTTP headers: %w", err)
		}
	}
	for _, member := range members {
		memberPath, err := url.JoinPath(SMDRelpathGroups, group, "members", member)
		if err != nil {
			newErr := fmt.Errorf("DeleteGroupMembers(): failed join group path (%s) with group %s and member %s: %w", SMDRelpathGroups, group, member, err)
			henvs = append(henvs, client.HTTPEnvelope{})
			errors = append(errors, newErr)
			continue
		}
		henv, err := sc.DeleteData(memberPath, "", headers, nil)
		henvs = append(henvs, henv)
		if err != nil {
			newErr := fmt.Errorf("DeleteGroupMembers(): failed to DELETE member %s from group %s in SMD: %w", member, group, err)
			errors = append(errors, newErr)
			continue
		}
		errors = append(errors, nil)
	}

	return henvs, errors, nil
}
