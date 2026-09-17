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

// NodeSpec is a wrapper around the boot-service's NodeSpec and is used
// specifically for the simple API. For adding Nodes, a "name" field is required
// but is only provided in the "metadata" structure, which is outside of the
// spec and is only available in the advanced API. To get around this, the
// upstream spec is wrapped with a "name" field so bulk specs can be added with
// names specified for each without having to provide them as arguments.
type NodeSpec struct {
	Name         string `json:"name" yaml:"name"` // Mandatory for adding resource
	api.NodeSpec `yaml:",inline"`
}

// AddNodes is a wrapper that calls the boot-service client's CreateNode()
// function, passing it context. It returns one result per request in input
// order.
func (bsc *BootServiceClient) AddNodes(ctx context.Context, token string, nodes []boot_service_client.CreateNodeRequest) client.BatchResult[*api.Node] {
	return runBatch(ctx, bsc.Timeout, nodes, func(requestCtx context.Context, node boot_service_client.CreateNodeRequest) (*api.Node, error) {
		item, err := bsc.Client.WithBearerToken(token).CreateNode(requestCtx, node)
		if err != nil {
			return nil, fmt.Errorf("failed to add node %+v: %w", node, err)
		}
		return item, nil
	})
}

// DeleteNodes is a wrapper that calls the boot-service client's DeleteNode()
// function, passing it context and a list of node UIDs to delete. It returns one
// result per UID in input order.
func (bsc *BootServiceClient) DeleteNodes(ctx context.Context, token string, uids []string) client.BatchResult[string] {
	return runBatch(ctx, bsc.Timeout, uids, func(requestCtx context.Context, nodeUID string) (string, error) {
		err := bsc.Client.WithBearerToken(token).DeleteNode(requestCtx, nodeUID)
		if err != nil {
			return "", fmt.Errorf("failed to delete node %s: %w", nodeUID, err)
		}
		return nodeUID, nil
	})
}

// GetNode is a wrapper that calls the boot-service client's GetNode() function,
// passing it context and a UID. The output is a []byte containing the entity's
// node information, formatted as outFormat.
func (bsc *BootServiceClient) GetNode(ctx context.Context, token string, outFormat format.DataFormat, uid string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, bsc.Timeout)
	defer cancel()

	bcfg, err := bsc.Client.WithBearerToken(token).GetNode(ctx, uid)
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
func (bsc *BootServiceClient) ListNodes(ctx context.Context, token string, outFormat format.DataFormat) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, bsc.Timeout)
	defer cancel()

	nodes, err := bsc.Client.WithBearerToken(token).GetNodes(ctx)
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
func (bsc *BootServiceClient) PatchNode(ctx context.Context, token string, patchFormat client.PatchMethod, uid string, data interface{}) (*api.Node, error) {
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

	item, err := bsc.Client.WithBearerToken(token).PatchNode(ctx, uid, outData, contentType)
	if err != nil {
		return nil, fmt.Errorf("failed to patch node for %s: %w", uid, err)
	}

	return item, nil
}

// SetNode is a wrapper that calls the boot-service client's UpdateNode()
// function, passing it context. The output is a pointer to the node
// details that got updated, along with an error if one occurred.
func (bsc *BootServiceClient) SetNode(ctx context.Context, token string, uid string, node boot_service_client.UpdateNodeRequest) (*api.Node, error) {
	ctx, cancel := context.WithTimeout(ctx, bsc.Timeout)
	defer cancel()

	item, err := bsc.Client.WithBearerToken(token).UpdateNode(ctx, uid, node)
	if err != nil {
		return nil, fmt.Errorf("failed to set node %+v: %w", node, err)
	}

	return item, nil
}

// AddNodeSpecs is like AddNodes but calls the boot-service client's simple
// CreateNodeSimple() function, which only sends the resource name and spec.
func (bsc *BootServiceClient) AddNodeSpecs(ctx context.Context, token string, nodes []NodeSpec) client.BatchResult[*api.Node] {
	return runBatch(ctx, bsc.Timeout, nodes, func(requestCtx context.Context, node NodeSpec) (*api.Node, error) {
		item, err := bsc.Client.WithBearerToken(token).CreateNodeSimple(requestCtx, node.Name, node.NodeSpec)
		if err != nil {
			return nil, fmt.Errorf("failed to add node %q (%+v): %w", node.Name, node.NodeSpec, err)
		}
		return item, nil
	})
}

// SetNodeSpec is like SetNode but calls the boot-service client's simple
// UpdateNodeSimple() function, which only sends the resource spec.
func (bsc *BootServiceClient) SetNodeSpec(ctx context.Context, token string, uid string, spec api.NodeSpec) (*api.Node, error) {
	ctx, cancel := context.WithTimeout(ctx, bsc.Timeout)
	defer cancel()

	item, err := bsc.Client.WithBearerToken(token).UpdateNodeSimple(ctx, uid, spec)
	if err != nil {
		return nil, fmt.Errorf("failed to set node %+v: %w", spec, err)
	}

	return item, nil
}
