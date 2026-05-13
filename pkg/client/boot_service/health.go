// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package boot_service

import (
	"context"
	"fmt"

	"github.com/OpenCHAMI/ochami/pkg/format"
)

// GetHealth is a wrapper that calls the boot-service client's GetHealth()
// function, passing it context. The output is a []byte containing the
// response from the health endpoint formatted as outFormat.
func (bsc *BootServiceClient) GetHealth(outFormat format.DataFormat) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), bsc.Timeout)
	defer cancel()

	health, err := bsc.Client.GetHealth(ctx)
	if err != nil {
		return nil, fmt.Errorf("request to get health data failed: %w", err)
	}

	out, err := format.MarshalData(health, outFormat)
	if err != nil {
		return nil, fmt.Errorf("formatting health info failed: %w", err)
	}

	return out, nil
}
