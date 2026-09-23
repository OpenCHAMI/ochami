// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// smd_test.go exercises representative "smd" subcommands (group, group member,
// iface, rfe, compep, service, status) end-to-end against an httptest.Server,
// asserting outbound request method/path and the resolved exit code. The
// "smd component" commands are covered separately in smd_component_test.go.
// HTTP-failure cases are covered in smd_errors_test.go.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// --- groups ---

// TestSMDGroupGet_Success verifies "smd group get" issues GET /groups.
func TestSMDGroupGet_Success(t *testing.T) {

	t.Parallel()

	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_, _ = w.Write([]byte(`[]`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "group", "get", "--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodGet || gotPath != "/groups" {
		t.Errorf("request = %s %s, want GET /groups", gotMethod, gotPath)
	}
}

// TestSMDGroupAdd_ViaFlags verifies "smd group add <label>" issues POST /groups.
func TestSMDGroupAdd_ViaFlags(t *testing.T) {

	t.Parallel()

	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "group", "add", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"--description", "compute nodes", "compute")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodPost || gotPath != "/groups" {
		t.Errorf("request = %s %s, want POST /groups", gotMethod, gotPath)
	}
}

// TestSMDGroupDelete_NoConfirm verifies "smd group delete --no-confirm <label>"
// issues DELETE /groups/<label>.
func TestSMDGroupDelete_NoConfirm(t *testing.T) {

	t.Parallel()

	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "group", "delete", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"--no-confirm", "compute")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodDelete || !strings.HasPrefix(gotPath, "/groups/compute") {
		t.Errorf("request = %s %s, want DELETE /groups/compute", gotMethod, gotPath)
	}
}

// TestSMDGroupMembership_Get verifies "smd group membership" issues GET
// /memberships.
func TestSMDGroupMembership_Get(t *testing.T) {

	t.Parallel()

	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`[]`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "group", "membership", "--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotPath != "/memberships" {
		t.Errorf("path = %q, want /memberships", gotPath)
	}
}

// --- group members ---

// TestSMDGroupMemberGet_Success verifies "smd group member get <label>" issues GET
// /groups/<label>/members.
func TestSMDGroupMemberGet_Success(t *testing.T) {

	t.Parallel()

	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`[]`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "group", "member", "get", "--ignore-config", "--uri", srv.URL, "--token", "t", "compute")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotPath != "/groups/compute/members" {
		t.Errorf("path = %q, want /groups/compute/members", gotPath)
	}
}

// TestSMDGroupMemberSet_Success verifies "smd group member set <label> <comp>..." issues
// PUT /groups/<label>/members.
func TestSMDGroupMemberSet_Success(t *testing.T) {

	t.Parallel()

	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "group", "member", "set", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"compute", "x0c0s0b0n0")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodPut || gotPath != "/groups/compute/members" {
		t.Errorf("request = %s %s, want PUT /groups/compute/members", gotMethod, gotPath)
	}
}

// --- ethernet interfaces ---

// TestSMDIfaceGet_Success verifies "smd iface get" issues GET
// /Inventory/EthernetInterfaces.
func TestSMDIfaceGet_Success(t *testing.T) {

	t.Parallel()

	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`[]`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "iface", "get", "--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotPath != "/Inventory/EthernetInterfaces" {
		t.Errorf("path = %q, want /Inventory/EthernetInterfaces", gotPath)
	}
}

// TestSMDIfaceAdd_ViaArgs verifies "smd iface add <comp> <mac> <net,ip>" issues
// POST /Inventory/EthernetInterfaces.
func TestSMDIfaceAdd_ViaArgs(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "iface", "add", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"x0c0s0b0n0", "de:ad:be:ef:00:00", "internal,172.16.0.1")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodPost || gotPath != "/Inventory/EthernetInterfaces" {
		t.Errorf("request = %s %s, want POST /Inventory/EthernetInterfaces", gotMethod, gotPath)
	}
}

// TestSMDIfaceDelete_NoConfirm verifies "smd iface delete --no-confirm <id>"
// issues DELETE under /Inventory/EthernetInterfaces.
func TestSMDIfaceDelete_NoConfirm(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "iface", "delete", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"--no-confirm", "deadbeef0000")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want DELETE", gotMethod)
	}
}

// --- redfish endpoints ---

// TestSMDRFEGet_Success verifies "smd rfe get" issues GET /Inventory/RedfishEndpoints.
func TestSMDRFEGet_Success(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{}`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "rfe", "get", "--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotPath != "/Inventory/RedfishEndpoints" {
		t.Errorf("path = %q, want /Inventory/RedfishEndpoints", gotPath)
	}
}

// TestSMDRFEAdd_ViaArgs verifies "smd rfe add <xname> <name> <ip> <mac>" issues
// POST /Inventory/RedfishEndpoints.
func TestSMDRFEAdd_ViaArgs(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "rfe", "add", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"x0c0s0b0", "bmc0", "172.16.0.100", "de:ad:be:ef:01:02")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodPost || gotPath != "/Inventory/RedfishEndpoints" {
		t.Errorf("request = %s %s, want POST /Inventory/RedfishEndpoints", gotMethod, gotPath)
	}
}

// --- component endpoints ---

// TestSMDCompepGet_Success verifies "smd compep get" issues GET
// /Inventory/ComponentEndpoints.
func TestSMDCompepGet_Success(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{}`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "compep", "get", "--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotPath != "/Inventory/ComponentEndpoints" {
		t.Errorf("path = %q, want /Inventory/ComponentEndpoints", gotPath)
	}
}

// TestSMDCompepDelete_NoConfirm verifies "smd compep delete --no-confirm <xname>"
// issues DELETE under /Inventory/ComponentEndpoints.
func TestSMDCompepDelete_NoConfirm(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "compep", "delete", "--ignore-config", "--uri", srv.URL, "--token", "t",
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

	res := runOchamiWithRuntime(t, "smd", "service", "status", "--ignore-config", "--uri", srv.URL)
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.HasPrefix(gotPath, "/service") {
		t.Errorf("path = %q, want a /service path", gotPath)
	}
}

// TestSMDDeprecatedStatus_Success verifies the deprecated "smd status" issues GET
// /service/ready.
func TestSMDDeprecatedStatus_Success(t *testing.T) {
	t.Parallel()

	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "smd", "status", "--uri", srv.URL)
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotPath != "/service/ready" {
		t.Errorf("path = %q, want /service/ready", gotPath)
	}
}
