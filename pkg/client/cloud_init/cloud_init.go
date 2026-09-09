// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cloud_init

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/openchami/cloud-init/pkg/cistore"

	"github.com/openchami/ochami/internal/log"
	"github.com/openchami/ochami/pkg/client"
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
// an error occurred creating the embedded OchamiClient, it is returned. Behavior
// such as TLS verification and token redaction is configured via functional
// options (e.g. client.WithInsecure, client.WithShowToken).
func NewClient(baseURI string, opts ...client.Option) (*CloudInitClient, error) {
	oc, err := client.NewOchamiClient(serviceNameCloudInit, baseURI, opts...)
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
func (cic *CloudInitClient) GetAPI(ctx context.Context) (client.HTTPEnvelope, error) {
	henv, err := cic.GetData(ctx, CloudInitRelpathAPI, "", nil)
	if err != nil {
		err = fmt.Errorf("GetAPI(): error getting cloud-init API: %w", err)
	}
	return henv, err
}

// GetDefaults is a wrapper function around OchamiClient.GetData that returns
// the result of querying the cloud-init cluster-defaults endpoint.
func (cic *CloudInitClient) GetDefaults(ctx context.Context, token string) (client.HTTPEnvelope, error) {
	var (
		henv    client.HTTPEnvelope
		headers *client.HTTPHeaders
	)
	headers = client.NewHTTPHeaders()
	if token != "" {
		_ = headers.SetAuthorization(token) //nolint:errcheck // headers was allocated above and cannot be nil
	}
	henv, err := cic.GetData(ctx, CloudInitRelpathDefaults, "", headers)
	if err != nil {
		err = fmt.Errorf("GetDefaults(): error getting cloud-init cluster-defaults: %w", err)
	}
	return henv, err
}

// GetGroups is a wrapper function around OchamiClient.Getdata that returns
// group data for a list of group ids. If none are passed, all group data is
// returned.
func (cic *CloudInitClient) GetGroups(ctx context.Context, token string, ids ...string) client.BatchResult[client.HTTPEnvelope] {
	headers := client.NewHTTPHeaders()
	if token != "" {
		_ = headers.SetAuthorization(token) //nolint:errcheck // headers was allocated above and cannot be nil
	}
	if len(ids) == 0 {
		henv, err := cic.GetData(ctx, CloudInitRelpathGroups, "", headers)
		if err != nil {
			err = fmt.Errorf("GetGroups(): failed to GET all groups from cloud-init: %w", err)
		}
		return client.BatchResult[client.HTTPEnvelope]{{Value: henv, Err: err}}
	}
	return executeBatch(ctx, ids, func(ctx context.Context, id string) (client.HTTPEnvelope, error) {
		finalEP, err := url.JoinPath(CloudInitRelpathGroups, id)
		if err != nil {
			return client.HTTPEnvelope{}, fmt.Errorf("GetGroups(): failed to join base group path with ID: %w", err)
		}
		henv, err := cic.GetData(ctx, finalEP, "", headers)
		if err != nil {
			log.Logger.Debug().Err(err).Msg("failed to get group")
			return henv, fmt.Errorf("GetGroups(): failed to GET group from cloud-init: %w", err)
		}
		return henv, nil
	})
}

// GetNodeData gets the data of type dataType for each ID in the passed list (at
// least one is required). It does this by iteratively calling
// OchamiClient.GetData. Slices containing the client.HTTPEnvelope and error for
// each request is returned, along with a separate single error if a function
// error occurred.
func (cic *CloudInitClient) GetNodeData(ctx context.Context, dataType CIDataType, token string, ids ...string) (client.BatchResult[client.HTTPEnvelope], error) {
	if len(ids) == 0 {
		return nil, fmt.Errorf("GetNodeData(): at least one ID is required")
	}
	headers := client.NewHTTPHeaders()
	if token != "" {
		_ = headers.SetAuthorization(token) //nolint:errcheck // headers was allocated above and cannot be nil
	}
	return executeBatch(ctx, ids, func(ctx context.Context, id string) (client.HTTPEnvelope, error) {
		finalEP, err := url.JoinPath(CloudInitRelpathImpersonation, id, string(dataType))
		if err != nil {
			return client.HTTPEnvelope{}, fmt.Errorf("GetNodeData(): failed to join %s with ID %s and %s: %w", CloudInitRelpathImpersonation, id, dataType, err)
		}
		henv, err := cic.GetData(ctx, finalEP, "", headers)
		if err != nil {
			log.Logger.Debug().Err(err).Msg("failed to get node data")
			return henv, fmt.Errorf("GetNodeData(): failed to GET node data from cloud-init: %w", err)
		}
		return henv, nil
	}), nil
}

// GetNodeGroupData gets the {group}.yaml data for a list of group IDs (at least
// one is required) for a node that is a member of those groups. It does this by
// iteratively calling OchamiClient.GetData. Slices containing the
// client.HTTPEnvelope and error for each request are returned, along with a
// separate single error if a function error occurred.
func (cic *CloudInitClient) GetNodeGroupData(ctx context.Context, token, id string, groups ...string) (client.BatchResult[client.HTTPEnvelope], error) {
	if strings.Trim(id, " ") == "" {
		return nil, fmt.Errorf("GetNodeGroupData(): group cannot be blank")
	}
	if len(groups) == 0 {
		return nil, fmt.Errorf("GetNodeGroupData(): at least one group is required")
	}
	headers := client.NewHTTPHeaders()
	if token != "" {
		_ = headers.SetAuthorization(token) //nolint:errcheck // headers was allocated above and cannot be nil
	}
	return executeBatch(ctx, groups, func(ctx context.Context, group string) (client.HTTPEnvelope, error) {
		finalEP, err := url.JoinPath(CloudInitRelpathImpersonation, id, fmt.Sprintf("%s.yaml", group))
		if err != nil {
			return client.HTTPEnvelope{}, fmt.Errorf("GetNodeGroupData(): failed to join %s with ID %s and %s.yaml: %w", CloudInitRelpathImpersonation, id, group, err)
		}
		henv, err := cic.GetData(ctx, finalEP, "", headers)
		if err != nil {
			log.Logger.Debug().Err(err).Msg("failed to get node group data")
			return henv, fmt.Errorf("GetNodeGroupData(): failed to GET node group data from cloud-init: %w", err)
		}
		return henv, nil
	}), nil
}

// GetVersion sends a GET to cloud-init's /version endpoint.
func (cic *CloudInitClient) GetVersion(ctx context.Context) (client.HTTPEnvelope, error) {
	henv, err := cic.GetData(ctx, CloudInitRelpathVersion, "", nil)
	if err != nil {
		err = fmt.Errorf("GetVersion(): error getting cloud-init version: %w", err)
	}
	return henv, err
}

// PostDefaults is a wrapper function around OchamiClient.PostData that takes a
// cistore.ClusterDefaults and a token, puts the token in the request headers as
// an authorization bearer, marshals ciDflts as JSON and sets it as the request
// body, then passes it to Ochami.PostData.
func (cic *CloudInitClient) PostDefaults(ctx context.Context, ciDflts cistore.ClusterDefaults, token string) (client.HTTPEnvelope, error) {
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
		_ = headers.SetAuthorization(token) //nolint:errcheck // headers was allocated above and cannot be nil
	}
	henv, err = cic.PostData(ctx, CloudInitRelpathDefaults, "", headers, body)
	if err != nil {
		err = fmt.Errorf("PostDefaults(): failed to POST cluster-defaults to cloud-init: %w", err)
	}

	return henv, err
}

// PostGroups is a wrapper function around OchamiClient.PostData that takes a
// slice of cistore.GroupData and a token, puts the token in the request headers
// as an authorization bearer, and iteratively calls OchamiClient.PostData using
// each item from the slice.
func (cic *CloudInitClient) PostGroups(ctx context.Context, ciGroups []cistore.GroupData, token string) client.BatchResult[client.HTTPEnvelope] {
	headers := client.NewHTTPHeaders()
	if token != "" {
		_ = headers.SetAuthorization(token) //nolint:errcheck // headers was allocated above and cannot be nil
	}
	return executeBatch(ctx, ciGroups, func(ctx context.Context, cig cistore.GroupData) (client.HTTPEnvelope, error) {
		body, err := json.Marshal(cig)
		if err != nil {
			return client.HTTPEnvelope{}, fmt.Errorf("PostGroups(): failed to marshal GroupData: %w", err)
		}
		henv, err := cic.PostData(ctx, CloudInitRelpathGroups, "", headers, body)
		if err != nil {
			return henv, fmt.Errorf("PostGroups(): failed to POST group(s) to cloud-init: %w", err)
		}
		return henv, nil
	})
}

// PutGroups is a wrapper function around OchamiClient.PutData that takes a
// slice of cistore.GroupData and a token, puts the token in the request
// headers as an authorization bearer, and iteratively calls
// OchamiClient.PostData using each item from the slice.
func (cic *CloudInitClient) PutGroups(ctx context.Context, ciGroups []cistore.GroupData, token string) client.BatchResult[client.HTTPEnvelope] {
	headers := client.NewHTTPHeaders()
	if token != "" {
		_ = headers.SetAuthorization(token) //nolint:errcheck // headers was allocated above and cannot be nil
	}
	return executeBatch(ctx, ciGroups, func(ctx context.Context, cig cistore.GroupData) (client.HTTPEnvelope, error) {
		if strings.Trim(cig.Name, " ") == "" {
			return client.HTTPEnvelope{}, fmt.Errorf("PutGroups(): group name cannot be blank")
		}
		finalEP, err := url.JoinPath(CloudInitRelpathGroups, cig.Name)
		if err != nil {
			return client.HTTPEnvelope{}, fmt.Errorf("PutGroups(): failed to join paths %q and %q: %w", CloudInitRelpathGroups, cig.Name, err)
		}
		body, err := json.Marshal(cig)
		if err != nil {
			return client.HTTPEnvelope{}, fmt.Errorf("PutGroups(): failed to marshal GroupData: %w", err)
		}
		henv, err := cic.PutData(ctx, finalEP, "", headers, body)
		if err != nil {
			return henv, fmt.Errorf("PutGroups(): failed to PUT group(s) to cloud-init: %w", err)
		}
		return henv, nil
	})
}

// PutInstanceInfo sends a PUT to cloud-init for each instance info in
// instanceInfoList, using the "id" field to determine which node to use.
func (cic *CloudInitClient) PutInstanceInfo(ctx context.Context, instanceInfoList []cistore.OpenCHAMIInstanceInfo, token string) (client.BatchResult[client.HTTPEnvelope], error) {
	headers := client.NewHTTPHeaders()
	if token != "" {
		_ = headers.SetAuthorization(token) //nolint:errcheck // headers was allocated above and cannot be nil
	}
	if len(instanceInfoList) == 0 {
		return nil, fmt.Errorf("PutInstanceInfo(): at least one instance info is required")
	}
	return executeBatch(ctx, instanceInfoList, func(ctx context.Context, instanceInfo cistore.OpenCHAMIInstanceInfo) (client.HTTPEnvelope, error) {
		if strings.Trim(instanceInfo.ID, " ") == "" {
			return client.HTTPEnvelope{}, fmt.Errorf("PutInstanceInfo(): id cannot be blank")
		}
		finalEP, err := url.JoinPath(CloudInitRelpathInstanceInfo, instanceInfo.ID)
		if err != nil {
			return client.HTTPEnvelope{}, fmt.Errorf("PutInstanceInfo(): failed to join paths %q and %q: %w", CloudInitRelpathInstanceInfo, instanceInfo.ID, err)
		}
		body, err := json.Marshal(instanceInfo)
		if err != nil {
			return client.HTTPEnvelope{}, fmt.Errorf("PutInstanceInfo(): failed to marshal instance info data: %w", err)
		}
		henv, err := cic.PutData(ctx, finalEP, "", headers, body)
		if err != nil {
			return henv, fmt.Errorf("PutInstanceInfo(): failed to PUT instance info for %q to cloud-init: %w", instanceInfo.ID, err)
		}
		return henv, nil
	}), nil
}

// DeleteGroups takes a token and group names and iteratively calls
// OchamiClient.DeleteData for each group. The iteration is necessary as the
// delete endpoint only allows deleting one group at a time. A slice of
// client.HTTPEnvelopes is returned, containing one per attempted deletion. Any
// corresponding errors are also returned. If an error in the function itself
// occurs, an additional error is returned in order to distinguish HTTP request
// errors from control flow errors.
func (cic *CloudInitClient) DeleteGroups(ctx context.Context, token string, groups ...string) client.BatchResult[client.HTTPEnvelope] {
	headers := client.NewHTTPHeaders()
	if token != "" {
		_ = headers.SetAuthorization(token) //nolint:errcheck // headers was allocated above and cannot be nil
	}
	return executeBatch(ctx, groups, func(ctx context.Context, group string) (client.HTTPEnvelope, error) {
		finalEP, err := url.JoinPath(CloudInitRelpathGroups, group)
		if err != nil {
			return client.HTTPEnvelope{}, fmt.Errorf("DeleteGroups(): failed join %q with %q: %w", CloudInitRelpathGroups, group, err)
		}
		henv, err := cic.DeleteData(ctx, finalEP, "", headers, nil)
		if err != nil {
			return henv, fmt.Errorf("DeleteGroups(): failed to DELETE group %s in cloud-init: %w", group, err)
		}
		return henv, nil
	})
}

func executeBatch[T any](ctx context.Context, items []T, execute func(context.Context, T) (client.HTTPEnvelope, error)) client.BatchResult[client.HTTPEnvelope] {
	results := make(client.BatchResult[client.HTTPEnvelope], len(items))
	for i, item := range items {
		if err := ctx.Err(); err != nil {
			for ; i < len(results); i++ {
				results[i].Err = err
			}
			break
		}
		value, err := execute(ctx, item)
		results[i] = client.Result[client.HTTPEnvelope]{Value: value, Err: err}
	}
	return results
}
