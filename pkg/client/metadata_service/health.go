// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package metadata_service

import (
	"context"
	"fmt"

	"github.com/OpenCHAMI/ochami/pkg/format"
)

// GetHealth is a wrapper that calls the metadata-service client's GetHealth()
// function, passing it context. The output is a []byte containing the response
// from the health endpoint formatted as outFormat.
func (msc *MetadataServiceClient) GetHealth(outFormat format.DataFormat) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), msc.Timeout)
	defer cancel()

	health, err := msc.Client.GetHealth(ctx)
	if err != nil {
		return nil, fmt.Errorf("request to get health data failed: %w", err)
	}

	out, err := format.MarshalData(health, outFormat)
	if err != nil {
		return nil, fmt.Errorf("formatting health info failed: %w", err)
	}

	return out, nil
}
