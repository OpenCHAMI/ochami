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

// AddDefaults is a wrapper that calls the metadata-service client's
// CreateClusterDefaults() function, passing it context. It returns a slice of
// successfully created ClusterDefaults resources, a slice of per-request errors,
// and an error that is populated if an error occurred in the function itself. A
// nil resource returned without an error is reported as a per-request error.
func (msc *MetadataServiceClient) AddDefaults(token string, defaults []metadata_service_client.CreateClusterDefaultsRequest) (defaultsAdded []api.ClusterDefaults, errors []error, funcErr error) {
	// TODO: Make concurrent
	for _, d := range defaults {
		ctx, cancel := context.WithTimeout(context.Background(), msc.Timeout)
		defer cancel()

		item, err := msc.Client.WithBearerToken(token).CreateClusterDefaults(ctx, d)
		if err != nil {
			newErr := fmt.Errorf("failed to add cluster defaults %+v: %w", d, err)
			errors = append(errors, newErr)
		} else if item != nil {
			defaultsAdded = append(defaultsAdded, *item)
		} else {
			newErr := fmt.Errorf("cluster defaults creation did not err, but was not created for: %+v", d)
			errors = append(errors, newErr)
		}
	}

	return
}

// DeleteDefaults is a wrapper that calls the metadata-service client's
// DeleteClusterDefaults() function, passing it context and a list of cluster
// defaults UIDs to delete. It returns a slice of successfully deleted cluster
// defaults UIDs, a slice of per-request errors, and an error that is populated
// if an error occurred in the function itself.
func (msc *MetadataServiceClient) DeleteDefaults(token string, uids []string) (defaultsDeleted []string, errors []error, funcErr error) {
	// TODO: Make concurrent
	for _, defaultsUid := range uids {
		ctx, cancel := context.WithTimeout(context.Background(), msc.Timeout)
		defer cancel()

		if err := msc.Client.WithBearerToken(token).DeleteClusterDefaults(ctx, defaultsUid); err != nil {
			newErr := fmt.Errorf("failed to delete cluster defaults %s: %w", defaultsUid, err)
			errors = append(errors, newErr)
		} else {
			defaultsDeleted = append(defaultsDeleted, defaultsUid)
		}
	}

	return
}

// GetDefaults is a wrapper that calls the metadata-service client's
// GetClusterDefaults() function, passing it context and a UID. The output is a
// []byte containing the entity's cluster defaults information, formatted as
// outFormat.
func (msc *MetadataServiceClient) GetDefaults(token string, outFormat format.DataFormat, uid string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), msc.Timeout)
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
func (msc *MetadataServiceClient) ListDefaults(token string, outFormat format.DataFormat) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), msc.Timeout)
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
func (msc *MetadataServiceClient) PatchDefaults(token string, patchFormat client.PatchMethod, uid string, data map[string]interface{}) (*api.ClusterDefaults, error) {
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

	item, err := msc.Client.WithBearerToken(token).PatchClusterDefaults(ctx, uid, outData, contentType)
	if err != nil {
		return nil, fmt.Errorf("failed to patch cluster defaults for %s: %w", uid, err)
	}

	return item, nil
}

// SetDefaults is a wrapper that calls the metadata-service client's
// UpdateClusterDefaults() function, passing it context. It returns the modified
// ClusterDefaults resource returned by metadata-service and any error.
func (msc *MetadataServiceClient) SetDefaults(token string, uid string, defaults metadata_service_client.UpdateClusterDefaultsRequest) (*api.ClusterDefaults, error) {
	ctx, cancel := context.WithTimeout(context.Background(), msc.Timeout)
	defer cancel()

	item, err := msc.Client.WithBearerToken(token).UpdateClusterDefaults(ctx, uid, defaults)
	if err != nil {
		return nil, fmt.Errorf("failed to set cluster defaults %+v: %w", defaults, err)
	}

	return item, nil
}
