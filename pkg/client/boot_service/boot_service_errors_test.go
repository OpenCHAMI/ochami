// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package boot_service

// boot_service_errors_test.go exercises the per-item error arms of the generic
// Add/Delete helpers and the error arms of the Get/List helpers by returning an
// error status from the mock server.

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	api "github.com/openchami/boot-service/apis/boot.openchami.io/v1"
	boot_service_client "github.com/openchami/boot-service/pkg/client"
	"github.com/rs/zerolog"

	"github.com/openchami/ochami/pkg/client"
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

// TestBootAddHelpersPerItemError verifies create failures are represented as item errors.
func TestBootAddHelpersPerItemError(t *testing.T) {
	c, srv := errClient(t)
	defer srv.Close()

	if results := c.AddBMCs(context.Background(), "", []boot_service_client.CreateBMCRequest{{}}); !results.HasErrors() {
		t.Error("AddBMCs: expected a per-item error")
	}
	if results := c.AddNodes(context.Background(), "", []boot_service_client.CreateNodeRequest{{}}); !results.HasErrors() {
		t.Error("AddNodes: expected a per-item error")
	}
	if results := c.AddBootConfigs(context.Background(), "", []boot_service_client.CreateBootConfigurationRequest{{}}); !results.HasErrors() {
		t.Error("AddBootConfigs: expected a per-item error")
	}
}

// TestBootDeleteHelpersPerItemError verifies delete failures are represented as item errors.
func TestBootDeleteHelpersPerItemError(t *testing.T) {
	c, srv := errClient(t)
	defer srv.Close()

	if results := c.DeleteBMCs(context.Background(), "", []string{"uid"}); !results.HasErrors() {
		t.Error("DeleteBMCs: expected a per-item error")
	}
	if results := c.DeleteNodes(context.Background(), "", []string{"uid"}); !results.HasErrors() {
		t.Error("DeleteNodes: expected a per-item error")
	}
	if results := c.DeleteBootConfigs(context.Background(), "", []string{"uid"}); !results.HasErrors() {
		t.Error("DeleteBootConfigs: expected a per-item error")
	}
}

// TestBootGetListHelpersErrorArm verifies read wrappers propagate unsuccessful responses.
func TestBootGetListHelpersErrorArm(t *testing.T) {
	c, srv := errClient(t)
	defer srv.Close()

	if _, err := c.GetBMC(context.Background(), "", format.DataFormatJson, "uid"); err == nil {
		t.Error("GetBMC: expected an error")
	}
	if _, err := c.GetNode(context.Background(), "", format.DataFormatJson, "uid"); err == nil {
		t.Error("GetNode: expected an error")
	}
	if _, err := c.GetBootConfig(context.Background(), "", format.DataFormatJson, "uid"); err == nil {
		t.Error("GetBootConfig: expected an error")
	}
	if _, err := c.ListBMCs(context.Background(), "", format.DataFormatJson); err == nil {
		t.Error("ListBMCs: expected an error")
	}
	if _, err := c.ListNodes(context.Background(), "", format.DataFormatJson); err == nil {
		t.Error("ListNodes: expected an error")
	}
	if _, err := c.ListBootConfigs(context.Background(), "", format.DataFormatJson); err == nil {
		t.Error("ListBootConfigs: expected an error")
	}
}

func TestBootPatchMethodValidation(t *testing.T) {
	c, srv := errClient(t)
	defer srv.Close()
	bad := client.PatchMethod("invalid")
	for name, call := range map[string]func() error{
		"BMC": func() error { _, err := c.PatchBMC(context.Background(), "", bad, "uid", map[string]any{}); return err },
		"node": func() error {
			_, err := c.PatchNode(context.Background(), "", bad, "uid", map[string]any{})
			return err
		},
		"boot config": func() error {
			_, err := c.PatchBootConfig(context.Background(), "", bad, "uid", map[string]any{})
			return err
		},
	} {
		t.Run(name, func(t *testing.T) {
			if err := call(); err == nil {
				t.Fatal("patch accepted an invalid patch method")
			}
		})
	}
}

func TestBootWriteHelpersHTTPError(t *testing.T) {
	c, srv := errClient(t)
	defer srv.Close()
	tests := []struct {
		name string
		call func() error
	}{
		{name: "set BMC", call: func() error {
			_, err := c.SetBMC(context.Background(), "", "uid", boot_service_client.UpdateBMCRequest{})
			return err
		}},
		{name: "set node", call: func() error {
			_, err := c.SetNode(context.Background(), "", "uid", boot_service_client.UpdateNodeRequest{})
			return err
		}},
		{name: "set boot config", call: func() error {
			_, err := c.SetBootConfig(context.Background(), "", "uid", boot_service_client.UpdateBootConfigurationRequest{})
			return err
		}},
		{name: "add BMC spec", call: func() error { return c.AddBMCSpecs(context.Background(), "", []BMCSpec{{Name: "one"}})[0].Err }},
		{name: "add node spec", call: func() error { return c.AddNodeSpecs(context.Background(), "", []NodeSpec{{Name: "one"}})[0].Err }},
		{name: "add boot config spec", call: func() error {
			return c.AddBootConfigSpecs(context.Background(), "", []BootConfigSpec{{Name: "one"}})[0].Err
		}},
		{name: "set BMC spec", call: func() error { _, err := c.SetBMCSpec(context.Background(), "", "uid", api.BMCSpec{}); return err }},
		{name: "set node spec", call: func() error { _, err := c.SetNodeSpec(context.Background(), "", "uid", api.NodeSpec{}); return err }},
		{name: "set boot config spec", call: func() error {
			_, err := c.SetBootConfigSpec(context.Background(), "", "uid", api.BootConfigurationSpec{})
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
