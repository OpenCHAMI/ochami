// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package metadata_service

// coverage_errors_test.go exercises the per-item error arms of the generic
// Add/Set/Delete/List helpers by returning error statuses from the mock server,
// and the format/marshal error arms of the getters.

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	metadata_service_client "github.com/openchami/metadata-service/pkg/client"
	"github.com/rs/zerolog"

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

func TestAddHelpersPerItemError(t *testing.T) {
	c, srv := errServer(t)
	defer srv.Close()

	if _, errs, _ := c.AddGroups("", []metadata_service_client.CreateGroupRequest{{}}); len(errs) == 0 {
		t.Error("AddGroups: expected a per-item error")
	}
	if _, errs, _ := c.AddDefaults("", []metadata_service_client.CreateClusterDefaultsRequest{{}}); len(errs) == 0 {
		t.Error("AddDefaults: expected a per-item error")
	}
	if _, errs, _ := c.AddInstanceInfos("", []metadata_service_client.CreateInstanceInfoRequest{{}}); len(errs) == 0 {
		t.Error("AddInstanceInfos: expected a per-item error")
	}
	if _, errs, _ := c.AddWireGuardPeers("", []metadata_service_client.CreateWireGuardPeerRequest{{}}); len(errs) == 0 {
		t.Error("AddWireGuardPeers: expected a per-item error")
	}
}

func TestDeleteHelpersPerItemError(t *testing.T) {
	c, srv := errServer(t)
	defer srv.Close()

	if _, errs, _ := c.DeleteGroups("", []string{"uid"}); len(errs) == 0 {
		t.Error("DeleteGroups: expected a per-item error")
	}
	if _, errs, _ := c.DeleteDefaults("", []string{"uid"}); len(errs) == 0 {
		t.Error("DeleteDefaults: expected a per-item error")
	}
	if _, errs, _ := c.DeleteInstanceInfos("", []string{"uid"}); len(errs) == 0 {
		t.Error("DeleteInstanceInfos: expected a per-item error")
	}
	if _, errs, _ := c.DeleteWireGuardPeers("", []string{"uid"}); len(errs) == 0 {
		t.Error("DeleteWireGuardPeers: expected a per-item error")
	}
}

func TestGetHelpersErrorArm(t *testing.T) {
	c, srv := errServer(t)
	defer srv.Close()

	if _, err := c.GetGroup("", format.DataFormatJson, "uid"); err == nil {
		t.Error("GetGroup: expected an error")
	}
	if _, err := c.GetDefaults("", format.DataFormatJson, "uid"); err == nil {
		t.Error("GetDefaults: expected an error")
	}
	if _, err := c.GetInstanceInfo("", format.DataFormatJson, "uid"); err == nil {
		t.Error("GetInstanceInfo: expected an error")
	}
	if _, err := c.GetWireGuardPeer("", format.DataFormatJson, "uid"); err == nil {
		t.Error("GetWireGuardPeer: expected an error")
	}
}

func TestListHelpersErrorArm(t *testing.T) {
	c, srv := errServer(t)
	defer srv.Close()

	if _, err := c.ListGroups("", format.DataFormatJson); err == nil {
		t.Error("ListGroups: expected an error")
	}
	if _, err := c.ListDefaults("", format.DataFormatJson); err == nil {
		t.Error("ListDefaults: expected an error")
	}
	if _, err := c.ListInstanceInfos("", format.DataFormatJson); err == nil {
		t.Error("ListInstanceInfos: expected an error")
	}
	if _, err := c.ListWireGuardPeers("", format.DataFormatJson); err == nil {
		t.Error("ListWireGuardPeers: expected an error")
	}
}
