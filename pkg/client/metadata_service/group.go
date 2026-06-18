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

// AddGroups is a wrapper that calls the metadata-service client's
// CreateGroup() function, passing it context. It returns a slice of
// successfully created Group resources, a slice of per-request errors, and an
// error that is populated if an error occurred in the function itself. A nil
// resource returned without an error is reported as a per-request error.
func (msc *MetadataServiceClient) AddGroups(token string, groups []metadata_service_client.CreateGroupRequest) (groupsAdded []api.Group, errors []error, funcErr error) {
	// TODO: metadata-service client functions don't support tokens yet.
	_ = token

	// TODO: Make concurrent
	for _, g := range groups {
		ctx, cancel := context.WithTimeout(context.Background(), msc.Timeout)
		defer cancel()

		item, err := msc.Client.CreateGroup(ctx, g)
		if err != nil {
			newErr := fmt.Errorf("failed to add group %+v: %w", g, err)
			errors = append(errors, newErr)
		} else if item != nil {
			groupsAdded = append(groupsAdded, *item)
		} else {
			newErr := fmt.Errorf("group creation did not err, but was not created for: %+v", g)
			errors = append(errors, newErr)
		}
	}

	return
}

// DeleteGroups is a wrapper that calls the metadata-service client's
// DeleteGroup() function, passing it context and a list of Group UIDs to
// delete. It returns a slice of successfully deleted Group UIDs, a slice of
// per-request errors, and an error that is populated if an error occurred in the
// function itself.
func (msc *MetadataServiceClient) DeleteGroups(token string, uids []string) (groupsDeleted []string, errors []error, funcErr error) {
	// TODO: metadata-service client functions don't support tokens yet.
	_ = token

	// TODO: Make concurrent
	for _, groupUid := range uids {
		ctx, cancel := context.WithTimeout(context.Background(), msc.Timeout)
		defer cancel()

		if err := msc.Client.DeleteGroup(ctx, groupUid); err != nil {
			newErr := fmt.Errorf("failed to delete group %s: %w", groupUid, err)
			errors = append(errors, newErr)
		} else {
			groupsDeleted = append(groupsDeleted, groupUid)
		}
	}

	return
}

// GetGroup is a wrapper that calls the metadata-service client's
// GetGroup() function, passing it context and a UID. The output is a
// []byte containing the entity's group information, formatted as
// outFormat.
func (msc *MetadataServiceClient) GetGroup(token string, outFormat format.DataFormat, uid string) ([]byte, error) {
	// TODO: metadata-service client functions don't support tokens yet.
	_ = token

	ctx, cancel := context.WithTimeout(context.Background(), msc.Timeout)
	defer cancel()

	group, err := msc.Client.GetGroup(ctx, uid)
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
func (msc *MetadataServiceClient) ListGroups(token string, outFormat format.DataFormat) ([]byte, error) {
	// TODO: metadata-service client functions don't support tokens yet.
	_ = token

	ctx, cancel := context.WithTimeout(context.Background(), msc.Timeout)
	defer cancel()

	groups, err := msc.Client.GetGroups(ctx)
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
func (msc *MetadataServiceClient) PatchGroup(token string, patchFormat client.PatchMethod, uid string, data map[string]interface{}) (*api.Group, error) {
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

	item, err := msc.Client.PatchGroup(ctx, uid, outData, contentType)
	if err != nil {
		return nil, fmt.Errorf("failed to patch group for %s: %w", uid, err)
	}

	return item, nil
}

// SetGroup is a wrapper that calls the metadata-service client's
// UpdateGroup() function, passing it context. It returns the modified Group
// resource returned by metadata-service and any error.
func (msc *MetadataServiceClient) SetGroup(token string, uid string, group metadata_service_client.UpdateGroupRequest) (*api.Group, error) {
	// TODO: metadata-service client functions don't support tokens yet.
	_ = token

	ctx, cancel := context.WithTimeout(context.Background(), msc.Timeout)
	defer cancel()

	item, err := msc.Client.UpdateGroup(ctx, uid, group)
	if err != nil {
		return nil, fmt.Errorf("failed to set group %+v: %w", group, err)
	}

	return item, nil
}
