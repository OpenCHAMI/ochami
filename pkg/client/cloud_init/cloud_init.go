// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cloud_init

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/OpenCHAMI/cloud-init/pkg/cistore"

	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client"
)

// CIDataType is an enum that represents the types of cloud-init data: user,
// meta, and vendor.
type CIDataType string

// CloudInitClient is an OchamiClient that has its BasePath configured to the
// one that the cloud-init service uses.
type CloudInitClient struct {
	*client.OchamiClient
}

const (
	serviceNameCloudInit = "cloud-init"

	CloudInitRelpathAPI           = "/openapi.json"
	CloudInitRelpathDefaults      = "/admin/cluster-defaults"
	CloudInitRelpathGroups        = "/admin/groups"
	CloudInitRelpathImpersonation = "/admin/impersonation"
	CloudInitRelpathInstanceInfo  = "/admin/instance-info"
	CloudInitRelpathVersion       = "/version"
)

// The different types of cloud-init data.
const (
	CloudInitUserData   CIDataType = "user-data"
	CloudInitMetaData   CIDataType = "meta-data"
	CloudInitVendorData CIDataType = "vendor-data"
)

// CIGroupDataMapToSlice converts a map of cistore.GroupData to a slice of
// cistore.GroupData.
//
// When GroupData is returned when fetching all groups, a map is returned keyed
// on the name. It can be easier and more consistent to have this be a list of
// GroupData instead,
func CIGroupDataMapToSlice(gMap map[string]cistore.GroupData) (gSlice []cistore.GroupData) {
	for _, group := range gMap {
		gSlice = append(gSlice, group)
	}
	return
}

// DecodeCloudConfig returns the bytes of a passed cistore.CloudConfigFile.Content.
// If the Encoding is base64, the bytes are base64-decoded before being
// returned.
func DecodeCloudConfig(ccf cistore.CloudConfigFile) ([]byte, error) {
	switch ccf.Encoding {
	case "plain":
		return ccf.Content, nil
	case "base64":
		contentBytes := make([]byte, base64.StdEncoding.EncodedLen(len(ccf.Content)))
		if n, err := base64.StdEncoding.Decode(contentBytes, ccf.Content); err != nil {
			return []byte{}, fmt.Errorf("failed to base64 decode cloud config (read %d bytes): %w", n, err)
		}
		return contentBytes, nil
	default:
		return []byte{}, fmt.Errorf("unknown encoding for cloud-config: %s", ccf.Encoding)
	}
}

// NewClient takes a baseURI and returns a pointer to a new CloudInitClient. If
// an error occurred creating the embedded OchamiClient, it is returned. If
// insecure is true, TLS certificates will not be verified.
func NewClient(baseURI string, insecure bool) (*CloudInitClient, error) {
	oc, err := client.NewOchamiClient(serviceNameCloudInit, baseURI, insecure)
	if err != nil {
		return nil, fmt.Errorf("failed to create OchamiClient for %s: %w", serviceNameCloudInit, err)
	}
	cic := &CloudInitClient{
		OchamiClient: oc,
	}

	return cic, err
}

// GetAPI sends a GET to cloud-init's /openapi.json endpoint to retrieve the
// OpenAPI specification.
func (cic *CloudInitClient) GetAPI() (client.HTTPEnvelope, error) {
	henv, err := cic.GetData(CloudInitRelpathAPI, "", nil)
	if err != nil {
		err = fmt.Errorf("GetAPI(): error getting cloud-init API: %w", err)
	}
	return henv, err
}

// GetDefaults is a wrapper function around OchamiClient.GetData that returns
// the result of querying the cloud-init cluster-defaults endpoint.
func (cic *CloudInitClient) GetDefaults(token string) (client.HTTPEnvelope, error) {
	var (
		henv    client.HTTPEnvelope
		headers *client.HTTPHeaders
	)
	headers = client.NewHTTPHeaders()
	if token != "" {
		if err := headers.SetAuthorization(token); err != nil {
			return henv, fmt.Errorf("GetDefaults(): error setting token in HTTP headers: %w", err)
		}
	}
	henv, err := cic.GetData(CloudInitRelpathDefaults, "", headers)
	if err != nil {
		err = fmt.Errorf("GetDefaults(): error getting cloud-init cluster-defaults: %w", err)
	}
	return henv, err
}

// GetGroups is a wrapper function around OchamiClient.Getdata that returns
// group data for a list of group ids. If none are passed, all group data is
// returned.
func (cic *CloudInitClient) GetGroups(token string, ids ...string) ([]client.HTTPEnvelope, []error, error) {
	var (
		errors  []error
		headers *client.HTTPHeaders
		henvs   []client.HTTPEnvelope
	)
	headers = client.NewHTTPHeaders()
	if token != "" {
		if err := headers.SetAuthorization(token); err != nil {
			return henvs, errors, fmt.Errorf("GetGroups(): error setting token in HTTP headers: %w", err)
		}
	}
	if len(ids) == 0 {
		henv, err := cic.GetData(CloudInitRelpathGroups, "", headers)
		henvs = append(henvs, henv)
		if err != nil {
			newErr := fmt.Errorf("GetGroups(): failed to GET all groups from cloud-init: %w", err)
			errors = append(errors, newErr)
		} else {
			errors = append(errors, nil)
		}
	} else {
		for _, id := range ids {
			var henv client.HTTPEnvelope
			finalEP, err := url.JoinPath(CloudInitRelpathGroups, id)
			if err != nil {
				newErr := fmt.Errorf("GetGroups(): failed to join base group path with ID: %w", err)
				errors = append(errors, newErr)
				henvs = append(henvs, henv)
				continue
			}
			henv, err = cic.GetData(finalEP, "", headers)
			henvs = append(henvs, henv)
			if err != nil {
				newErr := fmt.Errorf("GetGroups(): failed to GET group from cloud-init: %w", err)
				log.Logger.Debug().Err(err).Msg("failed to get group")
				errors = append(errors, newErr)
				continue
			}
			errors = append(errors, nil)
		}
	}

	return henvs, errors, nil
}

// GetNodeData gets the data of type dataType for each ID in the passed list (at
// least one is required). It does this by iteratively calling
// OchamiClient.GetData. Slices containing the client.HTTPEnvelope and error for
// each request is returned, along with a separate single error if a function
// error occurred.
func (cic *CloudInitClient) GetNodeData(dataType CIDataType, token string, ids ...string) ([]client.HTTPEnvelope, []error, error) {
	var (
		errors  []error
		headers *client.HTTPHeaders
		henvs   []client.HTTPEnvelope
	)
	if len(ids) == 0 {
		return henvs, errors, fmt.Errorf("GetNodeData(): at least one ID is required")
	}
	headers = client.NewHTTPHeaders()
	if token != "" {
		if err := headers.SetAuthorization(token); err != nil {
			return henvs, errors, fmt.Errorf("GetNodeData(): error setting token in HTTP headers: %w", err)
		}
	}
	for _, id := range ids {
		var henv client.HTTPEnvelope
		finalEP, err := url.JoinPath(CloudInitRelpathImpersonation, id, string(dataType))
		if err != nil {
			newErr := fmt.Errorf("GetNodeData(): failed to join %s with ID %s and %s: %w", CloudInitRelpathImpersonation, id, dataType, err)
			errors = append(errors, newErr)
			henvs = append(henvs, henv)
			continue
		}
		henv, err = cic.GetData(finalEP, "", headers)
		henvs = append(henvs, henv)
		if err != nil {
			newErr := fmt.Errorf("GetNodeData(): failed to GET node data from cloud-init: %w", err)
			log.Logger.Debug().Err(err).Msg("failed to get node data")
			errors = append(errors, newErr)
			continue
		}
		errors = append(errors, nil)
	}

	return henvs, errors, nil
}

// GetNodeGroupData gets the {group}.yaml data for a list of group IDs (at least
// one is required) for a node that is a member of those groups. It does this by
// iteratively calling OchamiClient.GetData. Slices containing the
// client.HTTPEnvelope and error for each request are returned, along with a
// separate single error if a function error occurred.
func (cic *CloudInitClient) GetNodeGroupData(token, id string, groups ...string) ([]client.HTTPEnvelope, []error, error) {
	var (
		errors  []error
		headers *client.HTTPHeaders
		henvs   []client.HTTPEnvelope
	)
	if strings.Trim(id, " ") == "" {
		return henvs, errors, fmt.Errorf("GetNodeGroupData(): group cannot be blank")
	}
	if len(groups) == 0 {
		return henvs, errors, fmt.Errorf("GetNodeGroupData(): at least one group is required")
	}
	headers = client.NewHTTPHeaders()
	if token != "" {
		if err := headers.SetAuthorization(token); err != nil {
			return henvs, errors, fmt.Errorf("GetNodeGroupData(): error setting token in HTTP headers: %w", err)
		}
	}
	for _, group := range groups {
		var henv client.HTTPEnvelope
		finalEP, err := url.JoinPath(CloudInitRelpathImpersonation, id, fmt.Sprintf("%s.yaml", group))
		if err != nil {
			newErr := fmt.Errorf("GetNodeGroupData(): failed to join %s with ID %s and %s.yaml: %w", CloudInitRelpathImpersonation, id, group, err)
			errors = append(errors, newErr)
			henvs = append(henvs, henv)
			continue
		}
		henv, err = cic.GetData(finalEP, "", headers)
		henvs = append(henvs, henv)
		if err != nil {
			newErr := fmt.Errorf("GetNodeGroupData(): failed to GET node group data from cloud-init: %w", err)
			log.Logger.Debug().Err(err).Msg("failed to get node group data")
			errors = append(errors, newErr)
			continue
		}
		errors = append(errors, nil)
	}

	return henvs, errors, nil
}

// GetVersion sends a GET to cloud-init's /version endpoint.
func (cic *CloudInitClient) GetVersion() (client.HTTPEnvelope, error) {
	henv, err := cic.GetData(CloudInitRelpathVersion, "", nil)
	if err != nil {
		err = fmt.Errorf("GetVersion(): error getting cloud-init version: %w", err)
	}
	return henv, err
}

// PostDefaults is a wrapper function around OchamiClient.PostData that takes a
// cistore.ClusterDefaults and a token, puts the token in the request headers as
// an authorization bearer, marshals ciDflts as JSON and sets it as the request
// body, then passes it to Ochami.PostData.
func (cic *CloudInitClient) PostDefaults(ciDflts cistore.ClusterDefaults, token string) (client.HTTPEnvelope, error) {
	var (
		henv    client.HTTPEnvelope
		headers *client.HTTPHeaders
		body    client.HTTPBody
		err     error
	)
	if body, err = json.Marshal(ciDflts); err != nil {
		return henv, fmt.Errorf("PostDefaults(): failed to marshal ClusterDefaults: %w", err)
	}
	headers = client.NewHTTPHeaders()
	if token != "" {
		if err := headers.SetAuthorization(token); err != nil {
			return henv, fmt.Errorf("PostDefaults(): error setting token in HTTP headers: %w", err)
		}
	}
	henv, err = cic.PostData(CloudInitRelpathDefaults, "", headers, body)
	if err != nil {
		err = fmt.Errorf("PostDefaults(): failed to POST cluster-defaults to cloud-init: %w", err)
	}

	return henv, err
}

// PostGroups is a wrapper function around OchamiClient.PostData that takes a
// slice of cistore.GroupData and a token, puts the token in the request headers
// as an authorization bearer, and iteratively calls OchamiClient.PostData using
// each item from the slice.
func (cic *CloudInitClient) PostGroups(ciGroups []cistore.GroupData, token string) ([]client.HTTPEnvelope, []error, error) {
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
	for _, cig := range ciGroups {
		var body client.HTTPBody
		var err error
		if body, err = json.Marshal(cig); err != nil {
			newErr := fmt.Errorf("PostGroups(): failed to marshal GroupData: %w", err)
			errors = append(errors, newErr)
			henvs = append(henvs, client.HTTPEnvelope{})
			continue
		}
		henv, err := cic.PostData(CloudInitRelpathGroups, "", headers, body)
		henvs = append(henvs, henv)
		if err != nil {
			newErr := fmt.Errorf("PostGroups(): failed to POST group(s) to cloud-init: %w", err)
			errors = append(errors, newErr)
			continue
		}
		errors = append(errors, nil)
	}

	return henvs, errors, nil
}

// PutGroups is a wrapper function around OchamiClient.PutData that takes a
// slice of cistore.GroupData and a token, puts the token in the request
// headers as an authorization bearer, and iteratively calls
// OchamiClient.PostData using each item from the slice.
func (cic *CloudInitClient) PutGroups(ciGroups []cistore.GroupData, token string) ([]client.HTTPEnvelope, []error, error) {
	var (
		errors  []error
		henvs   []client.HTTPEnvelope
		headers *client.HTTPHeaders
	)
	headers = client.NewHTTPHeaders()
	if token != "" {
		if err := headers.SetAuthorization(token); err != nil {
			return henvs, errors, fmt.Errorf("PutGroups(): error setting token in HTTP headers: %w", err)
		}
	}
	for _, cig := range ciGroups {
		var (
			body    client.HTTPBody
			err     error
			finalEP string
		)
		if strings.Trim(cig.Name, " ") == "" {
			newErr := fmt.Errorf("PutGroups(): group name cannot be blank")
			errors = append(errors, newErr)
			henvs = append(henvs, client.HTTPEnvelope{})
			continue
		}
		if finalEP, err = url.JoinPath(CloudInitRelpathGroups, cig.Name); err != nil {
			newErr := fmt.Errorf("PutGroups(): failed to join paths %q and %q: %w", CloudInitRelpathGroups, cig.Name, err)
			errors = append(errors, newErr)
			henvs = append(henvs, client.HTTPEnvelope{})
			continue
		}
		if body, err = json.Marshal(cig); err != nil {
			newErr := fmt.Errorf("PutGroups(): failed to marshal GroupData: %w", err)
			errors = append(errors, newErr)
			henvs = append(henvs, client.HTTPEnvelope{})
			continue
		}
		henv, err := cic.PutData(finalEP, "", headers, body)
		henvs = append(henvs, henv)
		if err != nil {
			newErr := fmt.Errorf("PutGroups(): failed to PUT group(s) to cloud-init: %w", err)
			errors = append(errors, newErr)
			continue
		}
		errors = append(errors, nil)
	}

	return henvs, errors, nil
}

// PutInstanceInfo sends a PUT to cloud-init for each instance info in
// instanceInfoList, using the "id" field to determine which node to use.
func (cic *CloudInitClient) PutInstanceInfo(instanceInfoList []cistore.OpenCHAMIInstanceInfo, token string) ([]client.HTTPEnvelope, []error, error) {
	var (
		errors  []error
		henvs   []client.HTTPEnvelope
		headers *client.HTTPHeaders
	)
	headers = client.NewHTTPHeaders()
	if token != "" {
		if err := headers.SetAuthorization(token); err != nil {
			return henvs, errors, fmt.Errorf("PutInstanceInfo(): error setting token in HTTP headers: %w", err)
		}
	}
	if len(instanceInfoList) == 0 {
		return henvs, errors, fmt.Errorf("PutInstanceInfo(): at least one instance info is required")
	}
	for _, instanceInfo := range instanceInfoList {
		var (
			body    client.HTTPBody
			err     error
			finalEP string
		)
		if strings.Trim(instanceInfo.ID, " ") == "" {
			newErr := fmt.Errorf("PutInstanceInfo(): id cannot be blank")
			errors = append(errors, newErr)
			henvs = append(henvs, client.HTTPEnvelope{})
			continue
		}
		if finalEP, err = url.JoinPath(CloudInitRelpathInstanceInfo, instanceInfo.ID); err != nil {
			newErr := fmt.Errorf("PutInstanceInfo(): failed to join paths %q and %q: %w", CloudInitRelpathInstanceInfo, instanceInfo.ID, err)
			errors = append(errors, newErr)
			henvs = append(henvs, client.HTTPEnvelope{})
			continue
		}
		if body, err = json.Marshal(instanceInfo); err != nil {
			newErr := fmt.Errorf("PutInstanceInfo(): failed to marshal instance info data: %w", err)
			errors = append(errors, newErr)
			henvs = append(henvs, client.HTTPEnvelope{})
			continue
		}
		henv, err := cic.PutData(finalEP, "", headers, body)
		henvs = append(henvs, henv)
		if err != nil {
			newErr := fmt.Errorf("PutInstanceInfo(): failed to PUT instance info for %q to cloud-init: %w", instanceInfo.ID, err)
			errors = append(errors, newErr)
			continue
		}
		errors = append(errors, nil)
	}

	return henvs, errors, nil
}

// DeleteGroups takes a token and group names and iteratively calls
// OchamiClient.DeleteData for each group. The iteration is necessary as the
// delete endpoint only allows deleting one group at a time. A slice of
// client.HTTPEnvelopes is returned, containing one per attempted deletion. Any
// corresponding errors are also returned. If an error in the function itself
// occurs, an additional error is returned in order to distinguish HTTP request
// errors from control flow errors.
func (cic *CloudInitClient) DeleteGroups(token string, groups ...string) ([]client.HTTPEnvelope, []error, error) {
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
	for _, group := range groups {
		finalEP, err := url.JoinPath(CloudInitRelpathGroups, group)
		if err != nil {
			newErr := fmt.Errorf("DeleteGroups(): failed join %q with %q: %w", CloudInitRelpathGroups, group, err)
			henvs = append(henvs, client.HTTPEnvelope{})
			errors = append(errors, newErr)
			continue
		}
		henv, err := cic.DeleteData(finalEP, "", headers, nil)
		henvs = append(henvs, henv)
		if err != nil {
			newErr := fmt.Errorf("DeleteGroups(): failed to DELETE group %s in cloud-init: %w", group, err)
			errors = append(errors, newErr)
			continue
		}
		errors = append(errors, nil)
	}

	return henvs, errors, nil
}
