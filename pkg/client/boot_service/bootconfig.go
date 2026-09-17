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

// BootConfigSpec is a wrapper around the boot-service's BootConfigurationSpec
// and is used specifically for the simple API. For adding boot configurations,
// a "name" field is required but is only provided in the "metadata" structure,
// which is outside of the spec and is only available in the advanced API. To get
// around this, the upstream spec is wrapped with a "name" field so bulk specs
// can be added with names specified for each without having to provide them as
// arguments.
type BootConfigSpec struct {
	Name                      string `json:"name" yaml:"name"` // Mandatory for adding resource
	api.BootConfigurationSpec `yaml:",inline"`
}

// AddBootConfigs is a wrapper that calls the boot-service client's
// CreateBootConfiguration() function, passing it context. It returns one result
// per request in input order.
func (bsc *BootServiceClient) AddBootConfigs(ctx context.Context, token string, bootCfgs []boot_service_client.CreateBootConfigurationRequest) client.BatchResult[*api.BootConfiguration] {
	return runBatch(ctx, bsc.Timeout, bootCfgs, func(requestCtx context.Context, bootCfg boot_service_client.CreateBootConfigurationRequest) (*api.BootConfiguration, error) {
		item, err := bsc.Client.WithBearerToken(token).CreateBootConfiguration(requestCtx, bootCfg)
		if err != nil {
			return nil, fmt.Errorf("failed to add boot configuration %+v: %w", bootCfg, err)
		}
		return item, nil
	})
}

// DeleteBootConfigs is a wrapper that calls the boot-service client's
// DeleteBootConfiguration() function, passing it context and a list of boot
// config UIDs to delete. It returns one result per UID in input order.
func (bsc *BootServiceClient) DeleteBootConfigs(ctx context.Context, token string, uids []string) client.BatchResult[string] {
	return runBatch(ctx, bsc.Timeout, uids, func(requestCtx context.Context, bcfgUID string) (string, error) {
		err := bsc.Client.WithBearerToken(token).DeleteBootConfiguration(requestCtx, bcfgUID)
		if err != nil {
			return "", fmt.Errorf("failed to delete boot config %s: %w", bcfgUID, err)
		}
		return bcfgUID, nil
	})
}

// GetBootConfig is a wrapper that calls the boot-service client's
// GetBootConfiguration() function, passing it context and a UID. The output is
// a []byte containing the entity's boot configuration, formatted as outFormat.
func (bsc *BootServiceClient) GetBootConfig(ctx context.Context, token string, outFormat format.DataFormat, uid string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, bsc.Timeout)
	defer cancel()

	bcfg, err := bsc.Client.WithBearerToken(token).GetBootConfiguration(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("request to get boot configuration for %s failed: %w", uid, err)
	}

	out, err := format.MarshalData(bcfg, outFormat)
	if err != nil {
		return nil, fmt.Errorf("formatting boot configuration for %s failed: %w", uid, err)
	}

	return out, nil
}

// ListBootConfigs is a wrapper that calls the boot-service client's
// GetBootConfigurations() function, passing it context. The output is a []byte
// containing a list of boot configurations formatted as outFormat.
func (bsc *BootServiceClient) ListBootConfigs(ctx context.Context, token string, outFormat format.DataFormat) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, bsc.Timeout)
	defer cancel()

	bcfgs, err := bsc.Client.WithBearerToken(token).GetBootConfigurations(ctx)
	if err != nil {
		return nil, fmt.Errorf("request to list boot configurations failed: %w", err)
	}

	out, err := format.MarshalData(bcfgs, outFormat)
	if err != nil {
		return nil, fmt.Errorf("formatting boot configuration list failed: %w", err)
	}

	return out, nil
}

// PatchBootConfig is a wrapper that calls the boot-service client's
// PatchBootConfiguration() function. It accepts data that represents a patch
// formatted as patchFormat and sends it as JSON to the boot-service via a PATCH
// request for the boot configuration identified by uid.
func (bsc *BootServiceClient) PatchBootConfig(ctx context.Context, token string, patchFormat client.PatchMethod, uid string, data interface{}) (*api.BootConfiguration, error) {
	// TODO: boot-service client functions don't support tokens yet.
	_ = token

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

	item, err := bsc.Client.WithBearerToken(token).PatchBootConfiguration(ctx, uid, outData, contentType)
	if err != nil {
		return nil, fmt.Errorf("failed to patch boot configuration for %s: %w", uid, err)
	}

	return item, nil
}

// SetBootConfig is a wrapper that calls the boot-service client's
// UpdateBootConfiguration() function, passing it context. The output is a
// pointer to the boot configuration that got updated, along with an error if
// one occurred.
func (bsc *BootServiceClient) SetBootConfig(ctx context.Context, token string, uid string, bootCfg boot_service_client.UpdateBootConfigurationRequest) (*api.BootConfiguration, error) {
	ctx, cancel := context.WithTimeout(ctx, bsc.Timeout)
	defer cancel()

	item, err := bsc.Client.WithBearerToken(token).UpdateBootConfiguration(ctx, uid, bootCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to set boot configuration %+v: %w", bootCfg, err)
	}

	return item, nil
}

// AddBootConfigSpecs is like AddBootConfigs but calls the boot-service
// client's simple CreateBootConfigurationSimple() function, which only sends
// the resource name and spec.
func (bsc *BootServiceClient) AddBootConfigSpecs(ctx context.Context, token string, bootCfgs []BootConfigSpec) client.BatchResult[*api.BootConfiguration] {
	return runBatch(ctx, bsc.Timeout, bootCfgs, func(requestCtx context.Context, bootCfg BootConfigSpec) (*api.BootConfiguration, error) {
		item, err := bsc.Client.WithBearerToken(token).CreateBootConfigurationSimple(requestCtx, bootCfg.Name, bootCfg.BootConfigurationSpec)
		if err != nil {
			return nil, fmt.Errorf("failed to add boot configuration %q (%+v): %w", bootCfg.Name, bootCfg.BootConfigurationSpec, err)
		}
		return item, nil
	})
}

// SetBootConfigSpec is like SetBootConfig but calls the boot-service client's
// simple UpdateBootConfigurationSimple() function, which only sends the
// resource spec.
func (bsc *BootServiceClient) SetBootConfigSpec(ctx context.Context, token string, uid string, spec api.BootConfigurationSpec) (*api.BootConfiguration, error) {
	ctx, cancel := context.WithTimeout(ctx, bsc.Timeout)
	defer cancel()

	item, err := bsc.Client.WithBearerToken(token).UpdateBootConfigurationSimple(ctx, uid, spec)
	if err != nil {
		return nil, fmt.Errorf("failed to set boot configuration %+v: %w", spec, err)
	}

	return item, nil
}
