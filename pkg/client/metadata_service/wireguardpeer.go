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

// AddWireGuardPeers is a wrapper that calls the metadata-service client's
// CreateWireGuardPeer() function, passing it context. The output is a slice
// of the WireGuardPeers it created, each element of which corresponds to an
// error in an error slice, followed by an error that is populated if an error
// occurred in the function itself.
func (msc *MetadataServiceClient) AddWireGuardPeers(token string, peers []metadata_service_client.CreateWireGuardPeerRequest) (peersAdded []*api.WireGuardPeer, errors []error, funcErr error) {
	// TODO: metadata-service client functions don't support tokens yet.
	_ = token

	// TODO: Make concurrent
	for _, p := range peers {
		ctx, cancel := context.WithTimeout(context.Background(), msc.Timeout)
		defer cancel()

		item, err := msc.Client.CreateWireGuardPeer(ctx, p)
		if err != nil {
			newErr := fmt.Errorf("failed to add WireGuard peer %+v: %w", p, err)
			errors = append(errors, newErr)
			peersAdded = append(peersAdded, nil)
		} else {
			peersAdded = append(peersAdded, item)
			errors = append(errors, nil)
		}
	}

	return
}

// DeleteWireGuardPeers is a wrapper that calls the metadata-service client's
// DeleteWireGuardPeer() function, passing it context and a list of WireGuard peer
// UIDs to delete. The output is a slice of WireGuard peer UIDs that
// got deleted, a slice of errors containing any errors deleting peers,
// and an error that is populated if an error in the function itself
// occurred.
func (msc *MetadataServiceClient) DeleteWireGuardPeers(token string, uids []string) (peersDeleted []string, errors []error, funcErr error) {
	// TODO: metadata-service client functions don't support tokens yet.
	_ = token

	// TODO: Make concurrent
	for _, peerUid := range uids {
		ctx, cancel := context.WithTimeout(context.Background(), msc.Timeout)
		defer cancel()

		if err := msc.Client.DeleteWireGuardPeer(ctx, peerUid); err != nil {
			newErr := fmt.Errorf("failed to delete WireGuard peer %s: %w", peerUid, err)
			errors = append(errors, newErr)
		} else {
			peersDeleted = append(peersDeleted, peerUid)
		}
	}

	return
}

// GetWireGuardPeer is a wrapper that calls the metadata-service client's
// GetWireGuardPeer() function, passing it context and a UID. The output is a
// []byte containing the entity's WireGuard peer information, formatted as
// outFormat.
func (msc *MetadataServiceClient) GetWireGuardPeer(token string, outFormat format.DataFormat, uid string) ([]byte, error) {
	// TODO: metadata-service client functions don't support tokens yet.
	_ = token

	ctx, cancel := context.WithTimeout(context.Background(), msc.Timeout)
	defer cancel()

	peer, err := msc.Client.GetWireGuardPeer(ctx, uid)
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
func (msc *MetadataServiceClient) ListWireGuardPeers(token string, outFormat format.DataFormat) ([]byte, error) {
	// TODO: metadata-service client functions don't support tokens yet.
	_ = token

	ctx, cancel := context.WithTimeout(context.Background(), msc.Timeout)
	defer cancel()

	peers, err := msc.Client.GetWireGuardPeers(ctx)
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
// PATCH request for the WireGuard peer identified by uid.
func (msc *MetadataServiceClient) PatchWireGuardPeer(token string, patchFormat client.PatchMethod, uid string, data map[string]interface{}) (*api.WireGuardPeer, error) {
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

	item, err := msc.Client.PatchWireGuardPeer(ctx, uid, outData, contentType)
	if err != nil {
		return nil, fmt.Errorf("failed to patch WireGuard peer for %s: %w", uid, err)
	}

	return item, nil
}

// SetWireGuardPeer is a wrapper that calls the metadata-service client's
// UpdateWireGuardPeer() function, passing it context. The output is a pointer
// to the WireGuard peer details that got updated, along with an error if one
// occurred.
func (msc *MetadataServiceClient) SetWireGuardPeer(token string, uid string, peer metadata_service_client.UpdateWireGuardPeerRequest) (*api.WireGuardPeer, error) {
	// TODO: metadata-service client functions don't support tokens yet.
	_ = token

	ctx, cancel := context.WithTimeout(context.Background(), msc.Timeout)
	defer cancel()

	item, err := msc.Client.UpdateWireGuardPeer(ctx, uid, peer)
	if err != nil {
		return nil, fmt.Errorf("failed to set WireGuard peer %+v: %w", peer, err)
	}

	return item, nil
}
