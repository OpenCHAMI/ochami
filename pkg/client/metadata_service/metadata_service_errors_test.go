// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package metadata_service

// metadata_service_errors_test.go exercises the per-item error arms of the generic
// Add/Set/Delete/List helpers by returning error statuses from the mock server,
// and the format/marshal error arms of the getters.

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	api "github.com/openchami/metadata-service/apis/cloud-init.openchami.io/v1"
	metadata_service_client "github.com/openchami/metadata-service/pkg/client"
	"github.com/rs/zerolog"

	"github.com/openchami/ochami/pkg/client"
	"github.com/openchami/ochami/pkg/format"
)

// errServer returns a client pointed at a server that fails every request.
func errServer(t *testing.T) (*MetadataServiceClient, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	c, err := NewClient(srv.URL, 5*time.Second, "", zerolog.New(io.Discard))
	if err != nil {
		srv.Close()
		t.Fatalf("failed to create client: %v", err)
	}
	return c, srv
}

// TestAddHelpersPerItemError verifies create failures are represented as item errors.
func TestAddHelpersPerItemError(t *testing.T) {
	c, srv := errServer(t)
	defer srv.Close()

	if results := c.AddGroups(context.Background(), "", []metadata_service_client.CreateGroupRequest{{}}); !results.HasErrors() {
		t.Error("AddGroups: expected a per-item error")
	}
	if results := c.AddDefaults(context.Background(), "", []metadata_service_client.CreateClusterDefaultsRequest{{}}); !results.HasErrors() {
		t.Error("AddDefaults: expected a per-item error")
	}
	if results := c.AddInstanceInfos(context.Background(), "", []metadata_service_client.CreateInstanceInfoRequest{{}}); !results.HasErrors() {
		t.Error("AddInstanceInfos: expected a per-item error")
	}
	if results := c.AddWireGuardPeers(context.Background(), "", []metadata_service_client.CreateWireGuardPeerRequest{{}}); !results.HasErrors() {
		t.Error("AddWireGuardPeers: expected a per-item error")
	}
}

// TestDeleteHelpersPerItemError verifies delete failures are represented as item errors.
func TestDeleteHelpersPerItemError(t *testing.T) {
	c, srv := errServer(t)
	defer srv.Close()

	if results := c.DeleteGroups(context.Background(), "", []string{"uid"}); !results.HasErrors() {
		t.Error("DeleteGroups: expected a per-item error")
	}
	if results := c.DeleteDefaults(context.Background(), "", []string{"uid"}); !results.HasErrors() {
		t.Error("DeleteDefaults: expected a per-item error")
	}
	if results := c.DeleteInstanceInfos(context.Background(), "", []string{"uid"}); !results.HasErrors() {
		t.Error("DeleteInstanceInfos: expected a per-item error")
	}
	if results := c.DeleteWireGuardPeers(context.Background(), "", []string{"uid"}); !results.HasErrors() {
		t.Error("DeleteWireGuardPeers: expected a per-item error")
	}
}

// TestGetHelpersErrorArm verifies individual read wrappers propagate unsuccessful responses.
func TestGetHelpersErrorArm(t *testing.T) {
	c, srv := errServer(t)
	defer srv.Close()

	if _, err := c.GetGroup(context.Background(), "", format.DataFormatJson, "uid"); err == nil {
		t.Error("GetGroup: expected an error")
	}
	if _, err := c.GetDefaults(context.Background(), "", format.DataFormatJson, "uid"); err == nil {
		t.Error("GetDefaults: expected an error")
	}
	if _, err := c.GetInstanceInfo(context.Background(), "", format.DataFormatJson, "uid"); err == nil {
		t.Error("GetInstanceInfo: expected an error")
	}
	if _, err := c.GetWireGuardPeer(context.Background(), "", format.DataFormatJson, "uid"); err == nil {
		t.Error("GetWireGuardPeer: expected an error")
	}
}

// TestListHelpersErrorArm verifies collection read wrappers propagate unsuccessful responses.
func TestListHelpersErrorArm(t *testing.T) {
	c, srv := errServer(t)
	defer srv.Close()

	if _, err := c.ListGroups(context.Background(), "", format.DataFormatJson); err == nil {
		t.Error("ListGroups: expected an error")
	}
	if _, err := c.ListDefaults(context.Background(), "", format.DataFormatJson); err == nil {
		t.Error("ListDefaults: expected an error")
	}
	if _, err := c.ListInstanceInfos(context.Background(), "", format.DataFormatJson); err == nil {
		t.Error("ListInstanceInfos: expected an error")
	}
	if _, err := c.ListWireGuardPeers(context.Background(), "", format.DataFormatJson); err == nil {
		t.Error("ListWireGuardPeers: expected an error")
	}
}

func TestMetadataPatchMethodValidation(t *testing.T) {
	c, srv := errServer(t)
	defer srv.Close()
	bad := client.PatchMethod("invalid")
	tests := []struct {
		name string
		call func() error
	}{
		{name: "group", call: func() error {
			_, err := c.PatchGroup(context.Background(), "", bad, "uid", map[string]any{})
			return err
		}},
		{name: "defaults", call: func() error {
			_, err := c.PatchDefaults(context.Background(), "", bad, "uid", map[string]any{})
			return err
		}},
		{name: "instance info", call: func() error {
			_, err := c.PatchInstanceInfo(context.Background(), "", bad, "uid", map[string]any{})
			return err
		}},
		{name: "wireguard peer", call: func() error {
			_, err := c.PatchWireGuardPeer(context.Background(), "", bad, "uid", map[string]any{})
			return err
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.call(); err == nil {
				t.Fatal("patch accepted an invalid patch method")
			}
		})
	}
}

func TestMetadataWriteHelpersHTTPError(t *testing.T) {
	c, srv := errServer(t)
	defer srv.Close()
	tests := []struct {
		name string
		call func() error
	}{
		{name: "set group", call: func() error {
			_, err := c.SetGroup(context.Background(), "", "uid", metadata_service_client.UpdateGroupRequest{})
			return err
		}},
		{name: "set defaults", call: func() error {
			_, err := c.SetDefaults(context.Background(), "", "uid", metadata_service_client.UpdateClusterDefaultsRequest{})
			return err
		}},
		{name: "set instance info", call: func() error {
			_, err := c.SetInstanceInfo(context.Background(), "", "uid", metadata_service_client.UpdateInstanceInfoRequest{})
			return err
		}},
		{name: "set wireguard peer", call: func() error {
			_, err := c.SetWireGuardPeer(context.Background(), "", "uid", metadata_service_client.UpdateWireGuardPeerRequest{})
			return err
		}},
		{name: "add group spec", call: func() error { return c.AddGroupSpecs(context.Background(), "", []GroupSpec{{Name: "one"}})[0].Err }},
		{name: "add defaults spec", call: func() error {
			return c.AddDefaultsSpecs(context.Background(), "", []ClusterDefaultsSpec{{Name: "one"}})[0].Err
		}},
		{name: "add instance info spec", call: func() error {
			return c.AddInstanceInfoSpecs(context.Background(), "", []InstanceInfoSpec{{Name: "one"}})[0].Err
		}},
		{name: "add wireguard peer spec", call: func() error {
			return c.AddWireGuardPeerSpecs(context.Background(), "", []WireGuardPeerSpec{{Name: "one"}})[0].Err
		}},
		{name: "set group spec", call: func() error { _, err := c.SetGroupSpec(context.Background(), "", "uid", api.GroupSpec{}); return err }},
		{name: "set defaults spec", call: func() error {
			_, err := c.SetDefaultsSpec(context.Background(), "", "uid", api.ClusterDefaultsSpec{})
			return err
		}},
		{name: "set instance info spec", call: func() error {
			_, err := c.SetInstanceInfoSpec(context.Background(), "", "uid", api.InstanceInfoSpec{})
			return err
		}},
		{name: "set wireguard peer spec", call: func() error {
			_, err := c.SetWireGuardPeerSpec(context.Background(), "", "uid", api.WireGuardPeerSpec{})
			return err
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.call(); err == nil {
				t.Fatal("call returned nil error")
			}
		})
	}
}
