// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// smd_test.go exercises representative "smd" subcommands (group, group member,
// iface, rfe, compep, service, status) end-to-end against an httptest.Server,
// asserting outbound request method/path and the resolved exit code. The
// "smd component" commands are covered separately in smd_component_test.go.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// --- groups ---

// TestSMDGroupGet verifies "smd group get" issues GET /groups.
func TestSMDGroupGet(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_, _ = w.Write([]byte(`[]`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "group", "get", "--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodGet || gotPath != "/groups" {
		t.Errorf("request = %s %s, want GET /groups", gotMethod, gotPath)
	}
}

// TestSMDGroupAddViaFlags verifies "smd group add <label>" issues POST /groups.
func TestSMDGroupAddViaFlags(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "group", "add", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"--description", "compute nodes", "compute")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodPost || gotPath != "/groups" {
		t.Errorf("request = %s %s, want POST /groups", gotMethod, gotPath)
	}
}

// TestSMDGroupDeleteNoConfirm verifies "smd group delete --no-confirm <label>"
// issues DELETE /groups/<label>.
func TestSMDGroupDeleteNoConfirm(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "group", "delete", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"--no-confirm", "compute")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodDelete || !strings.HasPrefix(gotPath, "/groups/compute") {
		t.Errorf("request = %s %s, want DELETE /groups/compute", gotMethod, gotPath)
	}
}

// TestSMDGroupMembershipGet verifies "smd group membership" issues GET
// /memberships.
func TestSMDGroupMembershipGet(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`[]`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "group", "membership", "--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotPath != "/memberships" {
		t.Errorf("path = %q, want /memberships", gotPath)
	}
}

// --- group members ---

// TestSMDGroupMemberGet verifies "smd group member get <label>" issues GET
// /groups/<label>/members.
func TestSMDGroupMemberGet(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`[]`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "group", "member", "get", "--ignore-config", "--uri", srv.URL, "--token", "t", "compute")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotPath != "/groups/compute/members" {
		t.Errorf("path = %q, want /groups/compute/members", gotPath)
	}
}

// TestSMDGroupMemberSet verifies "smd group member set <label> <comp>..." issues
// PUT /groups/<label>/members.
func TestSMDGroupMemberSet(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "group", "member", "set", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"compute", "x0c0s0b0n0")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodPut || gotPath != "/groups/compute/members" {
		t.Errorf("request = %s %s, want PUT /groups/compute/members", gotMethod, gotPath)
	}
}

// --- ethernet interfaces ---

// TestSMDIfaceGet verifies "smd iface get" issues GET
// /Inventory/EthernetInterfaces.
func TestSMDIfaceGet(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`[]`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "iface", "get", "--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotPath != "/Inventory/EthernetInterfaces" {
		t.Errorf("path = %q, want /Inventory/EthernetInterfaces", gotPath)
	}
}

// TestSMDIfaceAddViaArgs verifies "smd iface add <comp> <mac> <net,ip>" issues
// POST /Inventory/EthernetInterfaces.
func TestSMDIfaceAddViaArgs(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "iface", "add", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"x0c0s0b0n0", "de:ad:be:ef:00:00", "internal,172.16.0.1")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodPost || gotPath != "/Inventory/EthernetInterfaces" {
		t.Errorf("request = %s %s, want POST /Inventory/EthernetInterfaces", gotMethod, gotPath)
	}
}

// TestSMDIfaceDeleteNoConfirm verifies "smd iface delete --no-confirm <id>"
// issues DELETE under /Inventory/EthernetInterfaces.
func TestSMDIfaceDeleteNoConfirm(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "iface", "delete", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"--no-confirm", "deadbeef0000")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want DELETE", gotMethod)
	}
}

// --- redfish endpoints ---

// TestSMDRFEGet verifies "smd rfe get" issues GET /Inventory/RedfishEndpoints.
func TestSMDRFEGet(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{}`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "rfe", "get", "--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotPath != "/Inventory/RedfishEndpoints" {
		t.Errorf("path = %q, want /Inventory/RedfishEndpoints", gotPath)
	}
}

// TestSMDRFEAddViaArgs verifies "smd rfe add <xname> <name> <ip> <mac>" issues
// POST /Inventory/RedfishEndpoints.
func TestSMDRFEAddViaArgs(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "rfe", "add", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"x0c0s0b0", "bmc0", "172.16.0.100", "de:ad:be:ef:01:02")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodPost || gotPath != "/Inventory/RedfishEndpoints" {
		t.Errorf("request = %s %s, want POST /Inventory/RedfishEndpoints", gotMethod, gotPath)
	}
}

// --- component endpoints ---

// TestSMDCompepGet verifies "smd compep get" issues GET
// /Inventory/ComponentEndpoints.
func TestSMDCompepGet(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{}`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "compep", "get", "--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotPath != "/Inventory/ComponentEndpoints" {
		t.Errorf("path = %q, want /Inventory/ComponentEndpoints", gotPath)
	}
}

// TestSMDCompepDeleteNoConfirm verifies "smd compep delete --no-confirm <xname>"
// issues DELETE under /Inventory/ComponentEndpoints.
func TestSMDCompepDeleteNoConfirm(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "compep", "delete", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"--no-confirm", "x0c0s0b0n0")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want DELETE", gotMethod)
	}
}

// --- service / status ---

// TestSMDServiceStatus verifies "smd service status" issues a GET under /service.
func TestSMDServiceStatus(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"code":0}`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "service", "status", "--ignore-config", "--uri", srv.URL)
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.HasPrefix(gotPath, "/service") {
		t.Errorf("path = %q, want a /service path", gotPath)
	}
}

// TestSMDStatusHTTPError verifies that an unsuccessful HTTP response from the
// SMD service status endpoint resolves to CodeHTTP.
func TestSMDStatusHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "service", "status", "--ignore-config", "--uri", srv.URL)
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}
