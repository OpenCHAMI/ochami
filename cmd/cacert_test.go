// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// cacert_test.go checks CA trust through the real command tree. The command
// matrix is shared with the invalid-CA test in commands_errors_test.go.

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"
)

type caCertCommand struct {
	path string
	args []string
	body string // empty selects the default JSON response
}

// caCertCommands supplies one valid invocation per network command. Routing,
// payload contents, and output formatting have their own command tests.
func caCertCommands() []caCertCommand {
	cases := []caCertCommand{
		{path: "smd group get"},
		{path: "smd group add", args: []string{"compute"}},
		{path: "smd group update", args: []string{"--description", "d", "compute"}},
		{path: "smd group delete", args: []string{"--no-confirm", "compute"}},
		{path: "smd group membership"},
		{path: "smd group member get", args: []string{"compute"}},
		{path: "smd group member add", args: []string{"compute", "x0"}},
		{path: "smd group member set", args: []string{"compute", "x0"}},
		{path: "smd group member delete", args: []string{"--no-confirm", "compute", "x0"}},
		{path: "smd iface get"},
		{path: "smd iface add", args: []string{"x0", "de:ad:be:ef:00:00", "NMN,172.16.0.1"}},
		{path: "smd iface delete", args: []string{"--no-confirm", "deadbeef0000"}},
		{path: "smd rfe get"},
		{path: "smd rfe add", args: []string{"x0", "n", "172.16.0.1", "de:ad:be:ef:00:00"}},
		{path: "smd rfe delete", args: []string{"--no-confirm", "x0"}},
		{path: "smd compep get"},
		{path: "smd compep delete", args: []string{"--no-confirm", "x0"}},
		{path: "smd component get"},
		{path: "smd component add", args: []string{"x0", "1"}},
		{path: "smd component delete", args: []string{"--no-confirm", "x0"}},
		{path: "smd service status"},
		{path: "smd status"},
		{path: "bss boot params get"},
		{path: "bss boot params add", args: []string{"--mac", "de:ad:be:ef:00:00", "--kernel", "http://kernel"}},
		{path: "bss boot params set", args: []string{"--mac", "de:ad:be:ef:00:00", "--kernel", "http://kernel"}},
		{path: "bss boot params update", args: []string{"--mac", "de:ad:be:ef:00:00", "--kernel", "http://kernel"}},
		{path: "bss boot params delete", args: []string{"--no-confirm", "--mac", "de:ad:be:ef:00:00", "--kernel", "http://kernel"}},
		{path: "bss boot image set", args: []string{"--mac", "de:ad:be:ef:00:00", "http://image"}, body: `[{"macs":["de:ad:be:ef:00:00"],"params":"quiet"}]`},
		{path: "bss boot script get", args: []string{"--mac", "de:ad:be:ef:00:00"}},
		{path: "bss hosts get"},
		{path: "bss dumpstate"},
		{path: "bss history"},
		{path: "bss service status"},
		{path: "bss service version"},
		{path: "bss status"},
		{path: "cloud-init defaults get"},
		{path: "cloud-init defaults set", args: []string{"-d", `{"cluster-name":"demo"}`}},
		{path: "cloud-init group get raw"},
		{path: "cloud-init group get config"},
		{path: "cloud-init group get meta-data"},
		{path: "cloud-init group add", args: []string{"-d", `[{"name":"compute"}]`}},
		{path: "cloud-init group set", args: []string{"-d", `[{"name":"compute"}]`}},
		{path: "cloud-init group delete", args: []string{"--no-confirm", "compute"}},
		{path: "cloud-init group render", args: []string{"compute", "x0"}, body: `{}`},
		{path: "cloud-init node get meta-data", args: []string{"x0"}},
		{path: "cloud-init node get user-data", args: []string{"x0"}},
		{path: "cloud-init node get vendor-data", args: []string{"x0"}},
		{path: "cloud-init node get group", args: []string{"x0", "compute"}},
		{path: "cloud-init node set", args: []string{"-d", `[{"id":"x0"}]`}},
		{path: "cloud-init service status"},
		{path: "cloud-init service version"},
		{path: "pcs status list"},
		{path: "pcs status show", args: []string{"x0"}, body: `{"status":[{"xname":"x0","powerState":"on"}]}`},
		{path: "pcs transition list"},
		{path: "pcs transition show", args: []string{"id"}},
		{path: "pcs transition abort", args: []string{"id"}},
		{path: "pcs transition start", args: []string{"--xname", "x0", "on"}},
		{path: "pcs transition monitor", args: []string{"id"}, body: `{"transitionStatus":"completed","taskCounts":{"total":1,"succeeded":1}}`},
		{path: "pcs service status"},
		{path: "rcs console list", body: `{"consoles":[]}`},
		{path: "rcs console show", args: []string{"x0"}},
		{path: "rcs console connect", args: []string{"x0"}},
		{path: "rcs service status"},
		{path: "discover static", args: []string{"-d", discoveryPayload}},
	}
	for _, family := range []string{"boot", "metadata"} {
		resources := []string{"bmc", "config", "node"}
		if family == "metadata" {
			resources = []string{"defaults", "group", "instance", "peer"}
		}
		for _, resource := range resources {
			prefix := family + " " + resource + " "
			payload := bootAddPayload(resource)
			if family == "metadata" {
				payload = addPayloadFor(resource)
			}
			cases = append(cases,
				caCertCommand{path: prefix + "list"},
				caCertCommand{path: prefix + "get", args: []string{"uid"}},
				caCertCommand{path: prefix + "add", args: []string{"-d", payload}},
				caCertCommand{path: prefix + "set", args: []string{"uid", "-d", payload}},
				caCertCommand{path: prefix + "patch", args: []string{"uid", "-d", `{"name":"updated"}`}},
				caCertCommand{path: prefix + "delete", args: []string{"--no-confirm", "uid"}},
			)
		}
		cases = append(cases, caCertCommand{path: family + " service status"})
	}
	return cases
}

// TestCACert_AllNetworkCommands verifies every network leaf command has a CA
// test invocation, so adding a command without trust coverage fails the suite.
func TestCACert_AllNetworkCommands(t *testing.T) {
	t.Parallel()
	root := NewRootCmd()
	cases := make(map[string]bool)
	for _, tc := range caCertCommands() {
		if cases[tc.path] {
			t.Errorf("duplicate CA test invocation for %s", tc.path)
		}
		cases[tc.path] = true
		cmd, rest, err := root.Find(strings.Fields(tc.path))
		if err != nil || len(rest) != 0 || cmd.HasSubCommands() || cmd.Flag("uri") == nil {
			t.Errorf("CA test path %q does not select a network leaf command", tc.path)
		}
	}
	var walk func(*cobra.Command)
	walk = func(cmd *cobra.Command) {
		if !cmd.HasSubCommands() && cmd.Flag("uri") != nil {
			path := strings.TrimPrefix(cmd.CommandPath(), root.Name()+" ")
			if !cases[path] {
				t.Errorf("network command %q needs an invocation in caCertCommands", path)
			}
		}
		for _, child := range cmd.Commands() {
			walk(child)
		}
	}
	walk(root)
}

// TestCACert_TrustsServer verifies every network command rejects an untrusted
// TLS server without --cacert and completes a request using the supplied CA.
func TestCACert_TrustsServer(t *testing.T) {
	t.Parallel()
	for _, tc := range caCertCommands() {
		t.Run(tc.path, func(t *testing.T) {
			t.Parallel()
			var requests atomic.Int32
			consoleDone := make(chan struct{})
			srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if websocket.IsWebSocketUpgrade(r) {
					defer close(consoleDone)
					upgrader := websocket.Upgrader{}
					conn, err := upgrader.Upgrade(w, r, nil)
					if err != nil {
						t.Errorf("upgrade console connection: %v", err)
						return
					}
					defer conn.Close()
					if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
						t.Errorf("set console close deadline: %v", err)
						return
					}
					if err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")); err != nil {
						t.Errorf("close console connection: %v", err)
						return
					}
					// Keep the server open while the client sends its WebSocket
					// close reply and TLS close notification. Closing immediately
					// can make the client's TLS close fail with a broken pipe.
					if _, err := io.Copy(io.Discard, conn.UnderlyingConn()); err != nil {
						t.Errorf("wait for console client to close: %v", err)
					}
					return
				}
				w.Header().Set("Content-Type", "application/json")
				if tc.path == "pcs service status" {
					w.WriteHeader(http.StatusNoContent)
					return
				}
				body := tc.body
				if body == "" {
					body = "null"
				}
				_, _ = io.WriteString(w, body)
			}))
			srv.Config.ErrorLog = log.New(io.Discard, "", 0) // Expected untrusted TLS handshakes.
			srv.StartTLS()
			defer srv.Close()
			caPath := filepath.Join(t.TempDir(), "ca.pem")
			caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: srv.Certificate().Raw})
			if err := os.WriteFile(caPath, caPEM, 0o600); err != nil {
				t.Fatal(err)
			}
			args := append(strings.Fields(tc.path), tc.args...)
			args = append(args, "--ignore-config", "--uri", srv.URL, "--token", "t")
			res := runOchamiWithRuntime(t, args...)
			var unknownCA x509.UnknownAuthorityError
			if !errors.As(res.err, &unknownCA) {
				t.Fatalf("without --cacert: error = %v, want an unknown certificate authority", res.err)
			}
			if requests.Load() != 0 {
				t.Fatal("server received a request without a trusted CA")
			}
			res = runOchamiWithRuntime(t, append(args, "--cacert", caPath)...)
			if res.err != nil {
				t.Fatalf("with --cacert: unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
			if requests.Load() == 0 {
				t.Error("server received no request with a trusted CA")
			}
			if tc.path == "rcs console show" || tc.path == "rcs console connect" {
				// Hijacked connections are not awaited by httptest.Server.Close.
				<-consoleDone
			}
		})
	}
}
