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

// WireGuardPeerSpec is a wrapper around the metadata-service's
// WireGuardPeerSpec and is used specifically for the simple API. For adding
// WireGuard peers, a "name" field is required but is only provided in the
// "metadata" structure, which is outside of the spec and is only available in
// the advanced API. To get around this, the upstream spec is wrapped with a
// "name" field so bulk specs can be added with names specified for each without
// having to provide them as arguments.
type WireGuardPeerSpec struct {
	Name                  string `json:"name" yaml:"name"` // Mandatory for adding resource
	api.WireGuardPeerSpec `yaml:",inline"`
}

// AddWireGuardPeers is a wrapper that calls the metadata-service client's
// CreateWireGuardPeer() function, passing it context. It returns one result per
// request in input order.
func (msc *MetadataServiceClient) AddWireGuardPeers(ctx context.Context, token string, peers []metadata_service_client.CreateWireGuardPeerRequest) client.BatchResult[api.WireGuardPeer] {
	return runBatch(ctx, msc.Timeout, peers, func(requestCtx context.Context, peer metadata_service_client.CreateWireGuardPeerRequest) (api.WireGuardPeer, error) {
		item, err := msc.Client.WithBearerToken(token).CreateWireGuardPeer(requestCtx, peer)
		if err != nil {
			return api.WireGuardPeer{}, fmt.Errorf("failed to add WireGuard peer %+v: %w", peer, err)
		}
		return *item, nil
	})
}

// DeleteWireGuardPeers is a wrapper that calls the metadata-service client's
// DeleteWireGuardPeer() function, passing it context and a list of
// WireGuardPeer UIDs to delete. It returns one result per UID in input order.
func (msc *MetadataServiceClient) DeleteWireGuardPeers(ctx context.Context, token string, uids []string) client.BatchResult[string] {
	return runBatch(ctx, msc.Timeout, uids, func(requestCtx context.Context, peerUID string) (string, error) {
		err := msc.Client.WithBearerToken(token).DeleteWireGuardPeer(requestCtx, peerUID)
		if err != nil {
			return "", fmt.Errorf("failed to delete WireGuard peer %s: %w", peerUID, err)
		}
		return peerUID, nil
	})
}

// GetWireGuardPeer is a wrapper that calls the metadata-service client's
// GetWireGuardPeer() function, passing it context and a UID. The output is a
// []byte containing the entity's WireGuard peer information, formatted as
// outFormat.
func (msc *MetadataServiceClient) GetWireGuardPeer(ctx context.Context, token string, outFormat format.DataFormat, uid string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, msc.Timeout)
	defer cancel()

	peer, err := msc.Client.WithBearerToken(token).GetWireGuardPeer(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("request to get WireGuard peer info for %s failed: %w", uid, err)
	}

	out, err := format.MarshalData(peer, outFormat)
	if err != nil {
		return nil, fmt.Errorf("formatting WireGuard peer info for %s failed: %w", uid, err)
	}

	return out, nil
}

// ListWireGuardPeers is a wrapper that calls the metadata-service client's
// GetWireGuardPeers() function, passing it context. The output is a []byte
// containing the WireGuard peers formatted as outFormat.
func (msc *MetadataServiceClient) ListWireGuardPeers(ctx context.Context, token string, outFormat format.DataFormat) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, msc.Timeout)
	defer cancel()

	peers, err := msc.Client.WithBearerToken(token).GetWireGuardPeers(ctx)
	if err != nil {
		return nil, fmt.Errorf("request to list WireGuard peers failed: %w", err)
	}

	out, err := format.MarshalData(peers, outFormat)
	if err != nil {
		return nil, fmt.Errorf("formatting WireGuard peers failed: %w", err)
	}

	return out, nil
}

// PatchWireGuardPeer is a wrapper that calls the metadata-service client's
// PatchWireGuardPeer() function. It accepts data that represents a patch
// formatted as patchFormat and sends it as JSON to the metadata-service via a
// PATCH request for the WireGuardPeer identified by uid. It returns the modified
// WireGuardPeer resource returned by metadata-service and any error.
func (msc *MetadataServiceClient) PatchWireGuardPeer(ctx context.Context, token string, patchFormat client.PatchMethod, uid string, data interface{}) (*api.WireGuardPeer, error) {
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

	item, err := msc.Client.WithBearerToken(token).PatchWireGuardPeer(ctx, uid, outData, contentType)
	if err != nil {
		return nil, fmt.Errorf("failed to patch WireGuard peer for %s: %w", uid, err)
	}

	return item, nil
}

// SetWireGuardPeer is a wrapper that calls the metadata-service client's
// UpdateWireGuardPeer() function, passing it context. It returns the modified
// WireGuardPeer resource returned by metadata-service and any error.
func (msc *MetadataServiceClient) SetWireGuardPeer(ctx context.Context, token string, uid string, peer metadata_service_client.UpdateWireGuardPeerRequest) (*api.WireGuardPeer, error) {
	ctx, cancel := context.WithTimeout(ctx, msc.Timeout)
	defer cancel()

	item, err := msc.Client.WithBearerToken(token).UpdateWireGuardPeer(ctx, uid, peer)
	if err != nil {
		return nil, fmt.Errorf("failed to set WireGuard peer %+v: %w", peer, err)
	}

	return item, nil
}

// AddWireGuardPeerSpecs is like AddWireGuardPeers but calls the
// metadata-service client's simple CreateWireGuardPeerSimple() function, which
// only sends the resource name and spec.
func (msc *MetadataServiceClient) AddWireGuardPeerSpecs(ctx context.Context, token string, peers []WireGuardPeerSpec) client.BatchResult[api.WireGuardPeer] {
	return runBatch(ctx, msc.Timeout, peers, func(requestCtx context.Context, peer WireGuardPeerSpec) (api.WireGuardPeer, error) {
		item, err := msc.Client.WithBearerToken(token).CreateWireGuardPeerSimple(requestCtx, peer.Name, peer.WireGuardPeerSpec)
		if err != nil {
			return api.WireGuardPeer{}, fmt.Errorf("failed to add WireGuard peer %q (%+v): %w", peer.Name, peer.WireGuardPeerSpec, err)
		}
		return *item, nil
	})
}

// SetWireGuardPeerSpec is like SetWireGuardPeer but calls the metadata-service
// client's simple UpdateWireGuardPeerSimple() function, which only sends the
// resource spec.
func (msc *MetadataServiceClient) SetWireGuardPeerSpec(ctx context.Context, token string, uid string, spec api.WireGuardPeerSpec) (*api.WireGuardPeer, error) {
	ctx, cancel := context.WithTimeout(ctx, msc.Timeout)
	defer cancel()

	item, err := msc.Client.WithBearerToken(token).UpdateWireGuardPeerSimple(ctx, uid, spec)
	if err != nil {
		return nil, fmt.Errorf("failed to set WireGuard peer %+v: %w", spec, err)
	}

	return item, nil
}
