// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package boot_service

import (
	"context"
	"fmt"

	boot_service_client "github.com/openchami/boot-service/pkg/client"
	"github.com/openchami/boot-service/pkg/resources/bmc"

	"github.com/OpenCHAMI/ochami/pkg/client"
	"github.com/OpenCHAMI/ochami/pkg/format"
)

// AddBMCs is a wrapper that calls the boot-service client's CreateBMC()
// function, passing it context. The output is a slice of the BMCs it created,
// each element of which corresponds to an error in an error slice, followed by
// an error that is populatd if an error occurred in the function itself.
func (bsc *BootServiceClient) AddBMCs(token string, bmcs []boot_service_client.CreateBMCRequest) (bmcsAdded []*bmc.BMC, errors []error, funcErr error) {
	// TODO: boot-service client functions don't support tokens yet.
	_ = token

	// TODO: Make concurrent
	for _, bmc := range bmcs {
		ctx, cancel := context.WithTimeout(context.Background(), bsc.Timeout)
		defer cancel()

		item, err := bsc.Client.CreateBMC(ctx, bmc)
		if err != nil {
			newErr := fmt.Errorf("failed to add bmc %+v: %w", bmc, err)
			errors = append(errors, newErr)
			bmcsAdded = append(bmcsAdded, nil)
		}
		bmcsAdded = append(bmcsAdded, item)
	}

	return
}

// DeleteBMCs is a wrapper that calls the boot-service client's DeleteBMC()
// function, passing it context and a list of bmc UIDs to delete. The output is
// a slice of BMC UIDs that got deleted, a slice of errors containing any
// errors deleting BMCs, and an error that is populated if an error in the
// function itself occurred.
func (bsc *BootServiceClient) DeleteBMCs(token string, uids []string) (bmcsDeleted []string, errors []error, funcErr error) {
	// TODO: boot-service client functions don't support tokens yet.
	_ = token

	// TODO: Make concurrent
	for _, bmcUid := range uids {
		ctx, cancel := context.WithTimeout(context.Background(), bsc.Timeout)
		defer cancel()

		if err := bsc.Client.DeleteBMC(ctx, bmcUid); err != nil {
			newErr := fmt.Errorf("failed to delete BMC %s: %w", bmcUid, err)
			errors = append(errors, newErr)
		} else {
			bmcsDeleted = append(bmcsDeleted, bmcUid)
		}
	}

	return
}

// GetBMC is a wrapper that calls the boot-service client's GetBMC() function,
// passing it context and a UID. The output is a []byte containing the entity's
// BMC information, formatted as outFormat.
func (bsc *BootServiceClient) GetBMC(token string, outFormat format.DataFormat, uid string) ([]byte, error) {
	// TODO: boot-service client functions don't support tokens yet.
	_ = token

	ctx, cancel := context.WithTimeout(context.Background(), bsc.Timeout)
	defer cancel()

	bcfg, err := bsc.Client.GetBMC(ctx, uid)
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
func (bsc *BootServiceClient) ListBMCs(token string, outFormat format.DataFormat) ([]byte, error) {
	// TODO: boot-service client functions don't support tokens yet.
	_ = token

	ctx, cancel := context.WithTimeout(context.Background(), bsc.Timeout)
	defer cancel()

	nodes, err := bsc.Client.GetBMCs(ctx)
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
func (bsc *BootServiceClient) PatchBMC(token string, patchFormat client.PatchMethod, uid string, data map[string]interface{}) (*bmc.BMC, error) {
	// TODO: boot-service client functions don't support tokens yet.
	_ = token

	ctx, cancel := context.WithTimeout(context.Background(), bsc.Timeout)
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

	item, err := bsc.Client.PatchBMC(ctx, uid, outData, contentType)
	if err != nil {
		return nil, fmt.Errorf("failed to patch BMC for %s: %w", uid, err)
	}

	return item, nil
}

// SetBMC is a wrapper that calls the boot-service client's UpdateBMC()
// function, passing it context. The output is a pointer to the BMC details that
// got updated, along with an error if one occurred.
func (bsc *BootServiceClient) SetBMC(token string, uid string, bmc boot_service_client.UpdateBMCRequest) (*bmc.BMC, error) {
	// TODO: boot-service client functions don't support tokens yet.
	_ = token

	ctx, cancel := context.WithTimeout(context.Background(), bsc.Timeout)
	defer cancel()

	item, err := bsc.Client.UpdateBMC(ctx, uid, bmc)
	if err != nil {
		return nil, fmt.Errorf("failed to set BMC %+v: %w", bmc, err)
	}

	return item, nil
}
