// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package boot_service

import (
	"context"
	"fmt"

	boot_service_client "github.com/openchami/boot-service/pkg/client"
	"github.com/openchami/boot-service/pkg/resources/node"

	"github.com/OpenCHAMI/ochami/pkg/client"
	"github.com/OpenCHAMI/ochami/pkg/format"
)

// AddNodes is a wrapper that calls the boot-service client's CreateNode()
// function, passing it context. The output is a slice of the nodes it created,
// each element of which corresponds to an error in an error slice, followed by
// an error that is populatd if an error occurred in the function itself.
func (bsc *BootServiceClient) AddNodes(token string, nodes []boot_service_client.CreateNodeRequest) (nodesAdded []*node.Node, errors []error, funcErr error) {
	// TODO: boot-service client functions don't support tokens yet.
	_ = token

	// TODO: Make concurrent
	for _, node := range nodes {
		ctx, cancel := context.WithTimeout(context.Background(), bsc.Timeout)
		defer cancel()

		item, err := bsc.Client.CreateNode(ctx, node)
		if err != nil {
			newErr := fmt.Errorf("failed to add node %+v: %w", node, err)
			errors = append(errors, newErr)
			nodesAdded = append(nodesAdded, nil)
		}
		nodesAdded = append(nodesAdded, item)
	}

	return
}

// DeleteNodes is a wrapper that calls the boot-service client's DeleteNode()
// function, passing it context and a list of node UIDs to delete. The output is
// a slice of node UIDs that got deleted, a slice of errors containing any
// errors deleting nodes, and an error that is populated if an error in the
// function itself occurred.
func (bsc *BootServiceClient) DeleteNodes(token string, uids []string) (nodesDeleted []string, errors []error, funcErr error) {
	// TODO: boot-service client functions don't support tokens yet.
	_ = token

	// TODO: Make concurrent
	for _, nodeUid := range uids {
		ctx, cancel := context.WithTimeout(context.Background(), bsc.Timeout)
		defer cancel()

		if err := bsc.Client.DeleteNode(ctx, nodeUid); err != nil {
			newErr := fmt.Errorf("failed to delete node %s: %w", nodeUid, err)
			errors = append(errors, newErr)
		} else {
			nodesDeleted = append(nodesDeleted, nodeUid)
		}
	}

	return
}

// GetNode is a wrapper that calls the boot-service client's GetNode() function,
// passing it context and a UID. The output is a []byte containing the entity's
// node information, formatted as outFormat.
func (bsc *BootServiceClient) GetNode(token string, outFormat format.DataFormat, uid string) ([]byte, error) {
	// TODO: boot-service client functions don't support tokens yet.
	_ = token

	ctx, cancel := context.WithTimeout(context.Background(), bsc.Timeout)
	defer cancel()

	bcfg, err := bsc.Client.GetNode(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("request to get node info for %s failed: %w", uid, err)
	}

	out, err := format.MarshalData(bcfg, outFormat)
	if err != nil {
		return nil, fmt.Errorf("formatting node info for %s failed: %w", uid, err)
	}

	return out, nil
}

// ListNodes is a wrapper that calls the boot-service client's GetNodes()
// function, passing it context. The output is a []byte containing a list of
// nodes formatted as outFormat.
func (bsc *BootServiceClient) ListNodes(token string, outFormat format.DataFormat) ([]byte, error) {
	// TODO: boot-service client functions don't support tokens yet.
	_ = token

	ctx, cancel := context.WithTimeout(context.Background(), bsc.Timeout)
	defer cancel()

	nodes, err := bsc.Client.GetNodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("request to list nodes failed: %w", err)
	}

	out, err := format.MarshalData(nodes, outFormat)
	if err != nil {
		return nil, fmt.Errorf("formatting node list failed: %w", err)
	}

	return out, nil
}

// PatchNode is a wrapper that calls the boot-service client's PatchNode()
// function. It accepts data that represents a patch formatted as patchFormat
// and sends it as JSON to the boot-service via a PATCH request for the node
// identified by uid.
func (bsc *BootServiceClient) PatchNode(token string, patchFormat client.PatchMethod, uid string, data map[string]interface{}) (*node.Node, error) {
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

	item, err := bsc.Client.PatchNode(ctx, uid, outData, contentType)
	if err != nil {
		return nil, fmt.Errorf("failed to patch node for %s: %w", uid, err)
	}

	return item, nil
}

// SetNode is a wrapper that calls the boot-service client's UpdateNode()
// function, passing it context. The output is a pointer to the node
// details that got updated, along with an error if one occurred.
func (bsc *BootServiceClient) SetNode(token string, uid string, node boot_service_client.UpdateNodeRequest) (*node.Node, error) {
	// TODO: boot-service client functions don't support tokens yet.
	_ = token

	ctx, cancel := context.WithTimeout(context.Background(), bsc.Timeout)
	defer cancel()

	item, err := bsc.Client.UpdateNode(ctx, uid, node)
	if err != nil {
		return nil, fmt.Errorf("failed to set node %+v: %w", node, err)
	}

	return item, nil
}
