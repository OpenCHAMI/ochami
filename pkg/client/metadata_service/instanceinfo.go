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

// InstanceInfoSpec is a wrapper around the metadata-service's InstanceInfoSpec
// and is used specifically for the simple API. For adding instance info, a
// "name" field is required but is only provided in the "metadata" structure,
// which is outside of the spec and is only available in the advanced API. To get
// around this, the upstream spec is wrapped with a "name" field so bulk specs
// can be added with names specified for each without having to provide them as
// arguments.
type InstanceInfoSpec struct {
	Name                 string `json:"name" yaml:"name"` // Mandatory for adding resource
	api.InstanceInfoSpec `yaml:",inline"`
}

// AddInstanceInfos is a wrapper that calls the metadata-service client's
// CreateInstanceInfo() function, passing it context. It returns one result per
// request in input order.
func (msc *MetadataServiceClient) AddInstanceInfos(ctx context.Context, token string, instances []metadata_service_client.CreateInstanceInfoRequest) client.BatchResult[api.InstanceInfo] {
	return runBatch(ctx, msc.Timeout, instances, func(requestCtx context.Context, instance metadata_service_client.CreateInstanceInfoRequest) (api.InstanceInfo, error) {
		item, err := msc.Client.WithBearerToken(token).CreateInstanceInfo(requestCtx, instance)
		if err != nil {
			return api.InstanceInfo{}, fmt.Errorf("failed to add instance info %+v: %w", instance, err)
		}
		return *item, nil
	})
}

// DeleteInstanceInfos is a wrapper that calls the metadata-service client's
// DeleteInstanceInfo() function, passing it context and a list of InstanceInfo
// UIDs to delete. It returns one result per UID in input order.
func (msc *MetadataServiceClient) DeleteInstanceInfos(ctx context.Context, token string, uids []string) client.BatchResult[string] {
	return runBatch(ctx, msc.Timeout, uids, func(requestCtx context.Context, instanceUID string) (string, error) {
		err := msc.Client.WithBearerToken(token).DeleteInstanceInfo(requestCtx, instanceUID)
		if err != nil {
			return "", fmt.Errorf("failed to delete instance info %s: %w", instanceUID, err)
		}
		return instanceUID, nil
	})
}

// GetInstanceInfo is a wrapper that calls the metadata-service client's
// GetInstanceInfo() function, passing it context and a UID. The output is a
// []byte containing the entity's instance info, formatted as
// outFormat.
func (msc *MetadataServiceClient) GetInstanceInfo(ctx context.Context, token string, outFormat format.DataFormat, uid string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, msc.Timeout)
	defer cancel()

	instance, err := msc.Client.WithBearerToken(token).GetInstanceInfo(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("request to get instance info for %s failed: %w", uid, err)
	}

	out, err := format.MarshalData(instance, outFormat)
	if err != nil {
		return nil, fmt.Errorf("formatting instance info for %s failed: %w", uid, err)
	}

	return out, nil
}

// ListInstanceInfos is a wrapper that calls the metadata-service client's
// GetInstanceInfos() function, passing it context. The output is a []byte
// containing the instance infos formatted as outFormat.
func (msc *MetadataServiceClient) ListInstanceInfos(ctx context.Context, token string, outFormat format.DataFormat) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, msc.Timeout)
	defer cancel()

	instances, err := msc.Client.WithBearerToken(token).GetInstanceInfos(ctx)
	if err != nil {
		return nil, fmt.Errorf("request to list instance infos failed: %w", err)
	}

	out, err := format.MarshalData(instances, outFormat)
	if err != nil {
		return nil, fmt.Errorf("formatting instance infos failed: %w", err)
	}

	return out, nil
}

// PatchInstanceInfo is a wrapper that calls the metadata-service client's
// PatchInstanceInfo() function. It accepts data that represents a patch
// formatted as patchFormat and sends it as JSON to the metadata-service via a
// PATCH request for the InstanceInfo identified by uid. It returns the modified
// InstanceInfo resource returned by metadata-service and any error.
func (msc *MetadataServiceClient) PatchInstanceInfo(ctx context.Context, token string, patchFormat client.PatchMethod, uid string, data interface{}) (*api.InstanceInfo, error) {
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

	item, err := msc.Client.WithBearerToken(token).PatchInstanceInfo(ctx, uid, outData, contentType)
	if err != nil {
		return nil, fmt.Errorf("failed to patch instance info for %s: %w", uid, err)
	}

	return item, nil
}

// SetInstanceInfo is a wrapper that calls the metadata-service client's
// UpdateInstanceInfo() function, passing it context. It returns the modified
// InstanceInfo resource returned by metadata-service and any error.
func (msc *MetadataServiceClient) SetInstanceInfo(ctx context.Context, token string, uid string, instance metadata_service_client.UpdateInstanceInfoRequest) (*api.InstanceInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, msc.Timeout)
	defer cancel()

	item, err := msc.Client.WithBearerToken(token).UpdateInstanceInfo(ctx, uid, instance)
	if err != nil {
		return nil, fmt.Errorf("failed to set instance info %+v: %w", instance, err)
	}

	return item, nil
}

// AddInstanceInfoSpecs is like AddInstanceInfos but calls the metadata-service
// client's simple CreateInstanceInfoSimple() function, which only sends the
// resource name and spec.
func (msc *MetadataServiceClient) AddInstanceInfoSpecs(ctx context.Context, token string, instances []InstanceInfoSpec) client.BatchResult[api.InstanceInfo] {
	return runBatch(ctx, msc.Timeout, instances, func(requestCtx context.Context, instance InstanceInfoSpec) (api.InstanceInfo, error) {
		item, err := msc.Client.WithBearerToken(token).CreateInstanceInfoSimple(requestCtx, instance.Name, instance.InstanceInfoSpec)
		if err != nil {
			return api.InstanceInfo{}, fmt.Errorf("failed to add instance info %q (%+v): %w", instance.Name, instance.InstanceInfoSpec, err)
		}
		return *item, nil
	})
}

// SetInstanceInfoSpec is like SetInstanceInfo but calls the metadata-service
// client's simple UpdateInstanceInfoSimple() function, which only sends the
// resource spec.
func (msc *MetadataServiceClient) SetInstanceInfoSpec(ctx context.Context, token string, uid string, spec api.InstanceInfoSpec) (*api.InstanceInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, msc.Timeout)
	defer cancel()

	item, err := msc.Client.WithBearerToken(token).UpdateInstanceInfoSimple(ctx, uid, spec)
	if err != nil {
		return nil, fmt.Errorf("failed to set instance info %+v: %w", spec, err)
	}

	return item, nil
}
