// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package metadata_service

import (
	"context"
	"fmt"

	api "github.com/OpenCHAMI/metadata-service/apis/cloud-init.openchami.io/v1"
	metadata_service_client "github.com/OpenCHAMI/metadata-service/pkg/client"

	"github.com/OpenCHAMI/ochami/pkg/format"
)

// AddDefaults is a wrapper that calls the metadata-service client's
// CreateClusterDefaults() function, passing it context. The output is a slice
// of the ClusterDefaults it created, each element of which corresponds to an
// error in an error slice, followed by an error that is populatd if an error
// occurred in the function itself.
func (msc *MetadataServiceClient) AddDefaults(token string, defaults []metadata_service_client.CreateClusterDefaultsRequest) (defaultsAdded []*api.ClusterDefaults, errors []error, funcErr error) {
	// TODO: metadata-service client functions don't support tokens yet.
	_ = token

	// TODO: Make concurrent
	for _, d := range defaults {
		ctx, cancel := context.WithTimeout(context.Background(), msc.Timeout)
		defer cancel()

		item, err := msc.Client.CreateClusterDefaults(ctx, d)
		if err != nil {
			newErr := fmt.Errorf("failed to add cluster defaults %+v: %w", d, err)
			errors = append(errors, newErr)
			defaultsAdded = append(defaultsAdded, nil)
		}
		defaultsAdded = append(defaultsAdded, item)
	}

	return
}

// ListDefaults is a wrapper that calls the metadata-service client's
// GetClusterDefaultss() function, passing it context. The output is a []byte
// containing the cluster defaults formatted as outFormat.
func (msc *MetadataServiceClient) ListDefaults(token string, outFormat format.DataFormat) ([]byte, error) {
	// TODO: metadata-service client functions don't support tokens yet.
	_ = token

	ctx, cancel := context.WithTimeout(context.Background(), msc.Timeout)
	defer cancel()

	defaults, err := msc.Client.GetClusterDefaultss(ctx)
	if err != nil {
		return nil, fmt.Errorf("request to list cluster defaults failed: %w", err)
	}

	out, err := format.MarshalData(defaults, outFormat)
	if err != nil {
		return nil, fmt.Errorf("formatting cluster defaults failed: %w", err)
	}

	return out, nil
}

// SetDefaults is a wrapper that calls the metadata-service client's
// UpdateClusterDefaults() function, passing it context. The output is a pointer
// to the cluster defaults details that got updated, along with an error if one
// occurred.
func (msc *MetadataServiceClient) SetDefaults(token string, uid string, defaults metadata_service_client.UpdateClusterDefaultsRequest) (*api.ClusterDefaults, error) {
	// TODO: metadata-service client functions don't support tokens yet.
	_ = token

	ctx, cancel := context.WithTimeout(context.Background(), msc.Timeout)
	defer cancel()

	item, err := msc.Client.UpdateClusterDefaults(ctx, uid, defaults)
	if err != nil {
		return nil, fmt.Errorf("failed to set cluster defaults %+v: %w", defaults, err)
	}

	return item, nil
}
