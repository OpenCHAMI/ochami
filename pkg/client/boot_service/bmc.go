// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package boot_service

import (
	"context"
	"fmt"

	api "github.com/openchami/boot-service/apis/boot.openchami.io/v1"
	boot_service_client "github.com/openchami/boot-service/pkg/client"

	"github.com/openchami/ochami/pkg/client"
	"github.com/openchami/ochami/pkg/format"
)

// BMCSpec is a wrapper around the boot-service's BMCSpec and is used
// specifically for the simple API. For adding BMCs, a "name" field is required
// but is only provided in the "metadata" structure, which is outside of the
// spec and is only available in the advanced API. To get around this, the
// upstream spec is wrapped with a "name" field so bulk specs can be added with
// names specified for each without having to provide them as arguments.
type BMCSpec struct {
	Name        string `json:"name" yaml:"name"` // Mandatory for adding resource
	api.BMCSpec `yaml:",inline"`
}

// AddBMCs is a wrapper that calls the boot-service client's CreateBMC()
// function, passing it context. It returns one result per request in input
// order.
func (bsc *BootServiceClient) AddBMCs(ctx context.Context, token string, bmcs []boot_service_client.CreateBMCRequest) client.BatchResult[*api.BMC] {
	return runBatch(ctx, bsc.Timeout, bmcs, func(requestCtx context.Context, bmc boot_service_client.CreateBMCRequest) (*api.BMC, error) {
		item, err := bsc.Client.WithBearerToken(token).CreateBMC(requestCtx, bmc)
		if err != nil {
			return nil, fmt.Errorf("failed to add bmc %+v: %w", bmc, err)
		}
		return item, nil
	})
}

// DeleteBMCs is a wrapper that calls the boot-service client's DeleteBMC()
// function, passing it context and a list of BMC UIDs to delete. It returns one
// result per UID in input order.
func (bsc *BootServiceClient) DeleteBMCs(ctx context.Context, token string, uids []string) client.BatchResult[string] {
	return runBatch(ctx, bsc.Timeout, uids, func(requestCtx context.Context, bmcUid string) (string, error) {
		err := bsc.Client.WithBearerToken(token).DeleteBMC(requestCtx, bmcUid)
		if err != nil {
			return "", fmt.Errorf("failed to delete BMC %s: %w", bmcUid, err)
		}
		return bmcUid, nil
	})
}

// GetBMC is a wrapper that calls the boot-service client's GetBMC() function,
// passing it context and a UID. The output is a []byte containing the entity's
// BMC information, formatted as outFormat.
func (bsc *BootServiceClient) GetBMC(ctx context.Context, token string, outFormat format.DataFormat, uid string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, bsc.Timeout)
	defer cancel()

	bcfg, err := bsc.Client.WithBearerToken(token).GetBMC(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("request to get BMC info for %s failed: %w", uid, err)
	}

	out, err := format.MarshalData(bcfg, outFormat)
	if err != nil {
		return nil, fmt.Errorf("formatting BMC info for %s failed: %w", uid, err)
	}

	return out, nil
}

// ListBMCs is a wrapper that calls the boot-service client's GetBMCs()
// function, passing it context. The output is a []byte containing a list of
// BMC formatted as outFormat.
func (bsc *BootServiceClient) ListBMCs(ctx context.Context, token string, outFormat format.DataFormat) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, bsc.Timeout)
	defer cancel()

	nodes, err := bsc.Client.WithBearerToken(token).GetBMCs(ctx)
	if err != nil {
		return nil, fmt.Errorf("request to list BMCs failed: %w", err)
	}

	out, err := format.MarshalData(nodes, outFormat)
	if err != nil {
		return nil, fmt.Errorf("formatting BMC list failed: %w", err)
	}

	return out, nil
}

// PatchBMC is a wrapper that calls the boot-service client's PatchBMC()
// function. It accepts data that represents a patch formatted as patchFormat
// and sends it as JSON to the boot-service via a PATCH request for the BMC
// identified by uid.
func (bsc *BootServiceClient) PatchBMC(ctx context.Context, token string, patchFormat client.PatchMethod, uid string, data interface{}) (*api.BMC, error) {
	ctx, cancel := context.WithTimeout(ctx, bsc.Timeout)
	defer cancel()

	outData, err := format.MarshalData(data, format.DataFormatJson)
	if err != nil {
		return nil, fmt.Errorf("failed to convert data to JSON: %w", err)
	}

	contentType, err := patchFormat.ContentType()
	if err != nil {
		return nil, err
	}

	item, err := bsc.Client.WithBearerToken(token).PatchBMC(ctx, uid, outData, contentType)
	if err != nil {
		return nil, fmt.Errorf("failed to patch BMC for %s: %w", uid, err)
	}

	return item, nil
}

// SetBMC is a wrapper that calls the boot-service client's UpdateBMC()
// function, passing it context. The output is a pointer to the BMC details that
// got updated, along with an error if one occurred.
func (bsc *BootServiceClient) SetBMC(ctx context.Context, token string, uid string, bmc boot_service_client.UpdateBMCRequest) (*api.BMC, error) {
	ctx, cancel := context.WithTimeout(ctx, bsc.Timeout)
	defer cancel()

	item, err := bsc.Client.WithBearerToken(token).UpdateBMC(ctx, uid, bmc)
	if err != nil {
		return nil, fmt.Errorf("failed to set BMC %+v: %w", bmc, err)
	}

	return item, nil
}

// AddBMCSpecs is like AddBMCs but calls the boot-service client's simple
// CreateBMCSimple() function, which only sends the resource name and spec.
func (bsc *BootServiceClient) AddBMCSpecs(ctx context.Context, token string, bmcs []BMCSpec) client.BatchResult[*api.BMC] {
	return runBatch(ctx, bsc.Timeout, bmcs, func(requestCtx context.Context, bmc BMCSpec) (*api.BMC, error) {
		item, err := bsc.Client.WithBearerToken(token).CreateBMCSimple(requestCtx, bmc.Name, bmc.BMCSpec)
		if err != nil {
			return nil, fmt.Errorf("failed to add bmc %q (%+v): %w", bmc.Name, bmc.BMCSpec, err)
		}
		return item, nil
	})
}

// SetBMCSpec is like SetBMC but calls the boot-service client's simple
// UpdateBMCSimple() function, which only sends the resource spec.
func (bsc *BootServiceClient) SetBMCSpec(ctx context.Context, token string, uid string, spec api.BMCSpec) (*api.BMC, error) {
	ctx, cancel := context.WithTimeout(ctx, bsc.Timeout)
	defer cancel()

	item, err := bsc.Client.WithBearerToken(token).UpdateBMCSimple(ctx, uid, spec)
	if err != nil {
		return nil, fmt.Errorf("failed to set BMC %+v: %w", spec, err)
	}

	return item, nil
}
