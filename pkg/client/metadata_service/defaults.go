// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package metadata_service

import (
	"context"
	"fmt"

	api "github.com/openchami/metadata-service/apis/cloud-init.openchami.io/v1"
	metadata_service_client "github.com/openchami/metadata-service/pkg/client"

	"github.com/openchami/ochami/pkg/client"
	"github.com/openchami/ochami/pkg/format"
)

// ClusterDefaultsSpec is a wrapper around the metadata-service's
// ClusterDefaultsSpec and is used specifically for the simple API. For adding
// cluster defaults, a "name" field is required but is only provided in the
// "metadata" structure, which is outside of the spec and is only available in
// the advanced API. To get around this, the upstream spec is wrapped with a
// "name" field so bulk specs can be added with names specified for each without
// having to provide them as arguments.
type ClusterDefaultsSpec struct {
	Name                    string `json:"name" yaml:"name"` // Mandatory for adding resource
	api.ClusterDefaultsSpec `yaml:",inline"`
}

// AddDefaults is a wrapper that calls the metadata-service client's
// CreateClusterDefaults() function, passing it context. It returns one result
// per request in input order.
func (msc *MetadataServiceClient) AddDefaults(ctx context.Context, token string, defaults []metadata_service_client.CreateClusterDefaultsRequest) client.BatchResult[api.ClusterDefaults] {
	return runBatch(ctx, msc.Timeout, defaults, func(requestCtx context.Context, d metadata_service_client.CreateClusterDefaultsRequest) (api.ClusterDefaults, error) {
		item, err := msc.Client.WithBearerToken(token).CreateClusterDefaults(requestCtx, d)
		if err != nil {
			return api.ClusterDefaults{}, fmt.Errorf("failed to add cluster defaults %+v: %w", d, err)
		}
		return *item, nil
	})
}

// DeleteDefaults is a wrapper that calls the metadata-service client's
// DeleteClusterDefaults() function, passing it context and a list of cluster
// defaults UIDs to delete. It returns one result per UID in input order.
func (msc *MetadataServiceClient) DeleteDefaults(ctx context.Context, token string, uids []string) client.BatchResult[string] {
	return runBatch(ctx, msc.Timeout, uids, func(requestCtx context.Context, defaultsUID string) (string, error) {
		err := msc.Client.WithBearerToken(token).DeleteClusterDefaults(requestCtx, defaultsUID)
		if err != nil {
			return "", fmt.Errorf("failed to delete cluster defaults %s: %w", defaultsUID, err)
		}
		return defaultsUID, nil
	})
}

// GetDefaults is a wrapper that calls the metadata-service client's
// GetClusterDefaults() function, passing it context and a UID. The output is a
// []byte containing the entity's cluster defaults information, formatted as
// outFormat.
func (msc *MetadataServiceClient) GetDefaults(ctx context.Context, token string, outFormat format.DataFormat, uid string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, msc.Timeout)
	defer cancel()

	defaults, err := msc.Client.WithBearerToken(token).GetClusterDefaults(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("request to get cluster defaults info for %s failed: %w", uid, err)
	}

	out, err := format.MarshalData(defaults, outFormat)
	if err != nil {
		return nil, fmt.Errorf("formatting cluster defaults info for %s failed: %w", uid, err)
	}

	return out, nil
}

// ListDefaults is a wrapper that calls the metadata-service client's
// GetClusterDefaultss() function, passing it context. The output is a []byte
// containing the cluster defaults formatted as outFormat.
func (msc *MetadataServiceClient) ListDefaults(ctx context.Context, token string, outFormat format.DataFormat) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, msc.Timeout)
	defer cancel()

	defaults, err := msc.Client.WithBearerToken(token).GetClusterDefaultss(ctx)
	if err != nil {
		return nil, fmt.Errorf("request to list cluster defaults failed: %w", err)
	}

	out, err := format.MarshalData(defaults, outFormat)
	if err != nil {
		return nil, fmt.Errorf("formatting cluster defaults failed: %w", err)
	}

	return out, nil
}

// PatchDefaults is a wrapper that calls the metadata-service client's
// PatchClusterDefaults() function. It accepts data that represents a patch
// formatted as patchFormat and sends it as JSON to the metadata-service via a
// PATCH request for the cluster defaults identified by uid. It returns the
// modified ClusterDefaults resource returned by metadata-service and any error.
func (msc *MetadataServiceClient) PatchDefaults(ctx context.Context, token string, patchFormat client.PatchMethod, uid string, data interface{}) (*api.ClusterDefaults, error) {
	ctx, cancel := context.WithTimeout(ctx, msc.Timeout)
	defer cancel()

	outData, err := format.MarshalData(data, format.DataFormatJson)
	if err != nil {
		return nil, fmt.Errorf("failed to convert data to JSON: %w", err)
	}

	contentType, err := patchFormat.ContentType()
	if err != nil {
		return nil, err
	}

	item, err := msc.Client.WithBearerToken(token).PatchClusterDefaults(ctx, uid, outData, contentType)
	if err != nil {
		return nil, fmt.Errorf("failed to patch cluster defaults for %s: %w", uid, err)
	}

	return item, nil
}

// SetDefaults is a wrapper that calls the metadata-service client's
// UpdateClusterDefaults() function, passing it context. It returns the modified
// ClusterDefaults resource returned by metadata-service and any error.
func (msc *MetadataServiceClient) SetDefaults(ctx context.Context, token string, uid string, defaults metadata_service_client.UpdateClusterDefaultsRequest) (*api.ClusterDefaults, error) {
	ctx, cancel := context.WithTimeout(ctx, msc.Timeout)
	defer cancel()

	item, err := msc.Client.WithBearerToken(token).UpdateClusterDefaults(ctx, uid, defaults)
	if err != nil {
		return nil, fmt.Errorf("failed to set cluster defaults %+v: %w", defaults, err)
	}

	return item, nil
}

// AddDefaultsSpecs is like AddDefaults but calls the metadata-service client's
// simple CreateClusterDefaultsSimple() function, which only sends the resource
// name and spec.
func (msc *MetadataServiceClient) AddDefaultsSpecs(ctx context.Context, token string, defaults []ClusterDefaultsSpec) client.BatchResult[api.ClusterDefaults] {
	return runBatch(ctx, msc.Timeout, defaults, func(requestCtx context.Context, d ClusterDefaultsSpec) (api.ClusterDefaults, error) {
		item, err := msc.Client.WithBearerToken(token).CreateClusterDefaultsSimple(requestCtx, d.Name, d.ClusterDefaultsSpec)
		if err != nil {
			return api.ClusterDefaults{}, fmt.Errorf("failed to add cluster defaults %q (%+v): %w", d.Name, d.ClusterDefaultsSpec, err)
		}
		return *item, nil
	})
}

// SetDefaultsSpec is like SetDefaults but calls the metadata-service client's
// simple UpdateClusterDefaultsSimple() function, which only sends the resource
// spec.
func (msc *MetadataServiceClient) SetDefaultsSpec(ctx context.Context, token string, uid string, spec api.ClusterDefaultsSpec) (*api.ClusterDefaults, error) {
	ctx, cancel := context.WithTimeout(ctx, msc.Timeout)
	defer cancel()

	item, err := msc.Client.WithBearerToken(token).UpdateClusterDefaultsSimple(ctx, uid, spec)
	if err != nil {
		return nil, fmt.Errorf("failed to set cluster defaults %+v: %w", spec, err)
	}

	return item, nil
}
