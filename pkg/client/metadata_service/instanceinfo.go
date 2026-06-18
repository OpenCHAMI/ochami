// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package metadata_service

import (
	"context"
	"fmt"

	api "github.com/OpenCHAMI/metadata-service/apis/cloud-init.openchami.io/v1"
	metadata_service_client "github.com/OpenCHAMI/metadata-service/pkg/client"

	"github.com/OpenCHAMI/ochami/pkg/client"
	"github.com/OpenCHAMI/ochami/pkg/format"
)

// AddInstanceInfos is a wrapper that calls the metadata-service client's
// CreateInstanceInfo() function, passing it context. It returns a slice of
// successfully created InstanceInfo resources, a slice of per-request errors,
// and an error that is populated if an error occurred in the function itself. A
// nil resource returned without an error is reported as a per-request error.
func (msc *MetadataServiceClient) AddInstanceInfos(token string, instances []metadata_service_client.CreateInstanceInfoRequest) (instancesAdded []api.InstanceInfo, errors []error, funcErr error) {
	// TODO: metadata-service client functions don't support tokens yet.
	_ = token

	// TODO: Make concurrent
	for _, i := range instances {
		ctx, cancel := context.WithTimeout(context.Background(), msc.Timeout)
		defer cancel()

		item, err := msc.Client.CreateInstanceInfo(ctx, i)
		if err != nil {
			newErr := fmt.Errorf("failed to add instance info %+v: %w", i, err)
			errors = append(errors, newErr)
		} else if item != nil {
			instancesAdded = append(instancesAdded, *item)
		} else {
			newErr := fmt.Errorf("instance info creation did not err, but was not created for: %+v", i)
			errors = append(errors, newErr)
		}
	}

	return
}

// DeleteInstanceInfos is a wrapper that calls the metadata-service client's
// DeleteInstanceInfo() function, passing it context and a list of InstanceInfo
// UIDs to delete. It returns a slice of successfully deleted InstanceInfo UIDs,
// a slice of per-request errors, and an error that is populated if an error
// occurred in the function itself.
func (msc *MetadataServiceClient) DeleteInstanceInfos(token string, uids []string) (instancesDeleted []string, errors []error, funcErr error) {
	// TODO: metadata-service client functions don't support tokens yet.
	_ = token

	// TODO: Make concurrent
	for _, instanceUid := range uids {
		ctx, cancel := context.WithTimeout(context.Background(), msc.Timeout)
		defer cancel()

		if err := msc.Client.DeleteInstanceInfo(ctx, instanceUid); err != nil {
			newErr := fmt.Errorf("failed to delete instance info %s: %w", instanceUid, err)
			errors = append(errors, newErr)
		} else {
			instancesDeleted = append(instancesDeleted, instanceUid)
		}
	}

	return
}

// GetInstanceInfo is a wrapper that calls the metadata-service client's
// GetInstanceInfo() function, passing it context and a UID. The output is a
// []byte containing the entity's instance info, formatted as
// outFormat.
func (msc *MetadataServiceClient) GetInstanceInfo(token string, outFormat format.DataFormat, uid string) ([]byte, error) {
	// TODO: metadata-service client functions don't support tokens yet.
	_ = token

	ctx, cancel := context.WithTimeout(context.Background(), msc.Timeout)
	defer cancel()

	instance, err := msc.Client.GetInstanceInfo(ctx, uid)
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
func (msc *MetadataServiceClient) ListInstanceInfos(token string, outFormat format.DataFormat) ([]byte, error) {
	// TODO: metadata-service client functions don't support tokens yet.
	_ = token

	ctx, cancel := context.WithTimeout(context.Background(), msc.Timeout)
	defer cancel()

	instances, err := msc.Client.GetInstanceInfos(ctx)
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
func (msc *MetadataServiceClient) PatchInstanceInfo(token string, patchFormat client.PatchMethod, uid string, data map[string]interface{}) (*api.InstanceInfo, error) {
	// TODO: metadata-service client functions don't support tokens yet.
	_ = token

	ctx, cancel := context.WithTimeout(context.Background(), msc.Timeout)
	defer cancel()

	outData, err := format.MarshalData(data, format.DataFormatJson)
	if err != nil {
		return nil, fmt.Errorf("failed to convert data to JSON: %w", err)
	}

	var contentType string
	switch patchFormat {
	case client.PatchMethodRFC6902:
		contentType = "application/json-patch+json"
	case client.PatchMethodRFC7386:
		contentType = "application/merge-patch+json"
	case client.PatchMethodKeyVal:
		contentType = "application/merge-patch+json"
	default:
		return nil, fmt.Errorf("unknown patch format: %s", patchFormat)
	}

	item, err := msc.Client.PatchInstanceInfo(ctx, uid, outData, contentType)
	if err != nil {
		return nil, fmt.Errorf("failed to patch instance info for %s: %w", uid, err)
	}

	return item, nil
}

// SetInstanceInfo is a wrapper that calls the metadata-service client's
// UpdateInstanceInfo() function, passing it context. It returns the modified
// InstanceInfo resource returned by metadata-service and any error.
func (msc *MetadataServiceClient) SetInstanceInfo(token string, uid string, instance metadata_service_client.UpdateInstanceInfoRequest) (*api.InstanceInfo, error) {
	// TODO: metadata-service client functions don't support tokens yet.
	_ = token

	ctx, cancel := context.WithTimeout(context.Background(), msc.Timeout)
	defer cancel()

	item, err := msc.Client.UpdateInstanceInfo(ctx, uid, instance)
	if err != nil {
		return nil, fmt.Errorf("failed to set instance info %+v: %w", instance, err)
	}

	return item, nil
}
