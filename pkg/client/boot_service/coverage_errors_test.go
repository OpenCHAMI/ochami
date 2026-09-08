// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package boot_service

// coverage_errors_test.go exercises the per-item error arms of the generic
// Add/Delete helpers and the error arms of the Get/List helpers by returning an
// error status from the mock server.

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	boot_service_client "github.com/openchami/boot-service/pkg/client"
	"github.com/rs/zerolog"

	"github.com/openchami/ochami/pkg/format"
)

func errClient(t *testing.T) (*BootServiceClient, *httptest.Server) {
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

func TestBootAddHelpersPerItemError(t *testing.T) {
	c, srv := errClient(t)
	defer srv.Close()

	if _, errs, err := c.AddBMCs("", []boot_service_client.CreateBMCRequest{{}}); err != nil {
		t.Fatalf("AddBMCs: unexpected common error: %v", err)
	} else if len(errs) == 0 {
		t.Error("AddBMCs: expected a per-item error")
	}
	if _, errs, err := c.AddNodes("", []boot_service_client.CreateNodeRequest{{}}); err != nil {
		t.Fatalf("AddNodes: unexpected common error: %v", err)
	} else if len(errs) == 0 {
		t.Error("AddNodes: expected a per-item error")
	}
	if _, errs, err := c.AddBootConfigs("", []boot_service_client.CreateBootConfigurationRequest{{}}); err != nil {
		t.Fatalf("AddBootConfigs: unexpected common error: %v", err)
	} else if len(errs) == 0 {
		t.Error("AddBootConfigs: expected a per-item error")
	}
}

func TestBootDeleteHelpersPerItemError(t *testing.T) {
	c, srv := errClient(t)
	defer srv.Close()

	if _, errs, err := c.DeleteBMCs("", []string{"uid"}); err != nil {
		t.Fatalf("DeleteBMCs: unexpected common error: %v", err)
	} else if len(errs) == 0 {
		t.Error("DeleteBMCs: expected a per-item error")
	}
	if _, errs, err := c.DeleteNodes("", []string{"uid"}); err != nil {
		t.Fatalf("DeleteNodes: unexpected common error: %v", err)
	} else if len(errs) == 0 {
		t.Error("DeleteNodes: expected a per-item error")
	}
	if _, errs, err := c.DeleteBootConfigs("", []string{"uid"}); err != nil {
		t.Fatalf("DeleteBootConfigs: unexpected common error: %v", err)
	} else if len(errs) == 0 {
		t.Error("DeleteBootConfigs: expected a per-item error")
	}
}

func TestBootGetListHelpersErrorArm(t *testing.T) {
	c, srv := errClient(t)
	defer srv.Close()

	if _, err := c.GetBMC("", format.DataFormatJson, "uid"); err == nil {
		t.Error("GetBMC: expected an error")
	}
	if _, err := c.GetNode("", format.DataFormatJson, "uid"); err == nil {
		t.Error("GetNode: expected an error")
	}
	if _, err := c.GetBootConfig("", format.DataFormatJson, "uid"); err == nil {
		t.Error("GetBootConfig: expected an error")
	}
	if _, err := c.ListBMCs("", format.DataFormatJson); err == nil {
		t.Error("ListBMCs: expected an error")
	}
	if _, err := c.ListNodes("", format.DataFormatJson); err == nil {
		t.Error("ListNodes: expected an error")
	}
	if _, err := c.ListBootConfigs("", format.DataFormatJson); err == nil {
		t.Error("ListBootConfigs: expected an error")
	}
}
