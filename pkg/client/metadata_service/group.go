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

// GroupSpec is a wrapper around the metadata-service's GroupSpec and is used
// specifically for the simple API. For adding groups, a "name" field is
// required but is only provided in the "metadata" structure, which is outside
// of the spec and is only available in the advanced API. To get around this,
// the upstream spec is wrapped with a "name" field so bulk specs can be added
// with names specified for each without having to provide them as arguments.
type GroupSpec struct {
	Name          string `json:"name" yaml:"name"` // Mandatory for adding resource
	api.GroupSpec `yaml:",inline"`
}

// AddGroups is a wrapper that calls the metadata-service client's
// CreateGroup() function, passing it context. It returns one result per request
// in input order.
func (msc *MetadataServiceClient) AddGroups(ctx context.Context, token string, groups []metadata_service_client.CreateGroupRequest) client.BatchResult[api.Group] {
	return runBatch(ctx, msc.Timeout, groups, func(requestCtx context.Context, g metadata_service_client.CreateGroupRequest) (api.Group, error) {
		item, err := msc.Client.WithBearerToken(token).CreateGroup(requestCtx, g)
		if err != nil {
			return api.Group{}, fmt.Errorf("failed to add group %+v: %w", g, err)
		}
		return *item, nil
	})
}

// DeleteGroups is a wrapper that calls the metadata-service client's
// DeleteGroup() function, passing it context and a list of Group UIDs to
// delete. It returns one result per UID in input order.
func (msc *MetadataServiceClient) DeleteGroups(ctx context.Context, token string, uids []string) client.BatchResult[string] {
	return runBatch(ctx, msc.Timeout, uids, func(requestCtx context.Context, groupUID string) (string, error) {
		err := msc.Client.WithBearerToken(token).DeleteGroup(requestCtx, groupUID)
		if err != nil {
			return "", fmt.Errorf("failed to delete group %s: %w", groupUID, err)
		}
		return groupUID, nil
	})
}

// GetGroup is a wrapper that calls the metadata-service client's
// GetGroup() function, passing it context and a UID. The output is a
// []byte containing the entity's group information, formatted as
// outFormat.
func (msc *MetadataServiceClient) GetGroup(ctx context.Context, token string, outFormat format.DataFormat, uid string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, msc.Timeout)
	defer cancel()

	group, err := msc.Client.WithBearerToken(token).GetGroup(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("request to get group info for %s failed: %w", uid, err)
	}

	out, err := format.MarshalData(group, outFormat)
	if err != nil {
		return nil, fmt.Errorf("formatting group info for %s failed: %w", uid, err)
	}

	return out, nil
}

// ListGroups is a wrapper that calls the metadata-service client's
// GetGroups() function, passing it context. The output is a []byte
// containing the groups formatted as outFormat.
func (msc *MetadataServiceClient) ListGroups(ctx context.Context, token string, outFormat format.DataFormat) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, msc.Timeout)
	defer cancel()

	groups, err := msc.Client.WithBearerToken(token).GetGroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("request to list groups failed: %w", err)
	}

	out, err := format.MarshalData(groups, outFormat)
	if err != nil {
		return nil, fmt.Errorf("formatting groups failed: %w", err)
	}

	return out, nil
}

// PatchGroup is a wrapper that calls the metadata-service client's
// PatchGroup() function. It accepts data that represents a patch
// formatted as patchFormat and sends it as JSON to the metadata-service via a
// PATCH request for the Group identified by uid. It returns the modified Group
// resource returned by metadata-service and any error.
func (msc *MetadataServiceClient) PatchGroup(ctx context.Context, token string, patchFormat client.PatchMethod, uid string, data interface{}) (*api.Group, error) {
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

	item, err := msc.Client.WithBearerToken(token).PatchGroup(ctx, uid, outData, contentType)
	if err != nil {
		return nil, fmt.Errorf("failed to patch group for %s: %w", uid, err)
	}

	return item, nil
}

// SetGroup is a wrapper that calls the metadata-service client's
// UpdateGroup() function, passing it context. It returns the modified Group
// resource returned by metadata-service and any error.
func (msc *MetadataServiceClient) SetGroup(ctx context.Context, token string, uid string, group metadata_service_client.UpdateGroupRequest) (*api.Group, error) {
	ctx, cancel := context.WithTimeout(ctx, msc.Timeout)
	defer cancel()

	item, err := msc.Client.WithBearerToken(token).UpdateGroup(ctx, uid, group)
	if err != nil {
		return nil, fmt.Errorf("failed to set group %+v: %w", group, err)
	}

	return item, nil
}

// AddGroupSpecs is like AddGroups but calls the metadata-service client's
// simple CreateGroupSimple() function, which only sends the resource name and
// spec.
func (msc *MetadataServiceClient) AddGroupSpecs(ctx context.Context, token string, groups []GroupSpec) client.BatchResult[api.Group] {
	return runBatch(ctx, msc.Timeout, groups, func(requestCtx context.Context, g GroupSpec) (api.Group, error) {
		item, err := msc.Client.WithBearerToken(token).CreateGroupSimple(requestCtx, g.Name, g.GroupSpec)
		if err != nil {
			return api.Group{}, fmt.Errorf("failed to add group %q (%+v): %w", g.Name, g.GroupSpec, err)
		}
		return *item, nil
	})
}

// SetGroupSpec is like SetGroup but calls the metadata-service client's simple
// UpdateGroupSimple() function, which only sends the resource spec.
func (msc *MetadataServiceClient) SetGroupSpec(ctx context.Context, token string, uid string, spec api.GroupSpec) (*api.Group, error) {
	ctx, cancel := context.WithTimeout(ctx, msc.Timeout)
	defer cancel()

	item, err := msc.Client.WithBearerToken(token).UpdateGroupSimple(ctx, uid, spec)
	if err != nil {
		return nil, fmt.Errorf("failed to set group %+v: %w", spec, err)
	}

	return item, nil
}
