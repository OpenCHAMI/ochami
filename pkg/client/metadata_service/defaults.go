// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package metadata_service

import (
	"context"
	"fmt"

	//api "github.com/openchami/metadata-service/apis/metadata.openchami.io/v1"
	//metadata_service_client "github.com/openchami/metadata-service/pkg/client"

	//"github.com/OpenCHAMI/ochami/pkg/client"
	"github.com/OpenCHAMI/ochami/pkg/format"
)

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
