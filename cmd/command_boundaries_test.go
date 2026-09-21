// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	bootcmd "github.com/openchami/ochami/cmd/boot"
	bsscmd "github.com/openchami/ochami/cmd/bss"
	cloudinitcmd "github.com/openchami/ochami/cmd/cloud_init"
	configcmd "github.com/openchami/ochami/cmd/config"
	metadatacmd "github.com/openchami/ochami/cmd/metadata"
	pcscmd "github.com/openchami/ochami/cmd/pcs"
	rcscmd "github.com/openchami/ochami/cmd/rcs"
	smdcmd "github.com/openchami/ochami/cmd/smd"
	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/config"
)

func TestCommandsPropagateOutputFailures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
		args func(string) []string
	}{
		{name: "boot bmc get", body: `{}`, args: func(uri string) []string { return []string{"boot", "bmc", "get", "id", "--uri", uri, "--token", "t"} }},
		{name: "boot bmc list", body: `[]`, args: func(uri string) []string { return []string{"boot", "bmc", "list", "--uri", uri, "--token", "t"} }},
		{name: "boot config get", body: `{}`, args: func(uri string) []string {
			return []string{"boot", "config", "get", "id", "--uri", uri, "--token", "t"}
		}},
		{name: "boot config list", body: `[]`, args: func(uri string) []string { return []string{"boot", "config", "list", "--uri", uri, "--token", "t"} }},
		{name: "boot node get", body: `{}`, args: func(uri string) []string { return []string{"boot", "node", "get", "id", "--uri", uri, "--token", "t"} }},
		{name: "boot node list", body: `[]`, args: func(uri string) []string { return []string{"boot", "node", "list", "--uri", uri, "--token", "t"} }},
		{name: "bss dumpstate", body: `{}`, args: func(uri string) []string { return []string{"bss", "dumpstate", "--uri", uri} }},
		{name: "bss history", body: `[]`, args: func(uri string) []string { return []string{"bss", "history", "--uri", uri} }},
		{name: "bss hosts", body: `[]`, args: func(uri string) []string { return []string{"bss", "hosts", "get", "--uri", uri} }},
		{name: "bss script", body: `#!ipxe`, args: func(uri string) []string {
			return []string{"bss", "boot", "script", "get", "--uri", uri, "--mac", "de:ad:be:ef:00:00"}
		}},
		{name: "bss service version", body: `{"version":"1"}`, args: func(uri string) []string { return []string{"bss", "service", "version", "--uri", uri} }},
		{name: "cloud-init defaults", body: `{}`, args: func(uri string) []string { return []string{"cloud-init", "defaults", "get", "--uri", uri} }},
		{name: "cloud-init group metadata", body: `{"compute":{"name":"compute","meta-data":{"role":"worker"}}}`, args: func(uri string) []string { return []string{"cloud-init", "group", "get", "meta-data", "--uri", uri} }},
		{name: "cloud-init group raw", body: `{"compute":{"name":"compute"}}`, args: func(uri string) []string { return []string{"cloud-init", "group", "get", "raw", "--uri", uri} }},
		{name: "cloud-init node metadata", body: `hostname: node01`, args: func(uri string) []string {
			return []string{"cloud-init", "node", "get", "meta-data", "--uri", uri, "node01"}
		}},
		{name: "cloud-init node user data", body: `#cloud-config`, args: func(uri string) []string {
			return []string{"cloud-init", "node", "get", "user-data", "--uri", uri, "node01"}
		}},
		{name: "cloud-init node vendor data", body: `#cloud-config`, args: func(uri string) []string {
			return []string{"cloud-init", "node", "get", "vendor-data", "--uri", uri, "node01"}
		}},
		{name: "cloud-init service status", body: `{"version":"1"}`, args: func(uri string) []string { return []string{"cloud-init", "service", "status", "--uri", uri} }},
		{name: "cloud-init service api", body: `{"openapi":"3.0.0"}`, args: func(uri string) []string { return []string{"cloud-init", "service", "status", "--api", "--uri", uri} }},
		{name: "cloud-init service version", body: `{"version":"1"}`, args: func(uri string) []string { return []string{"cloud-init", "service", "version", "--uri", uri} }},
		{name: "metadata defaults get", body: `{}`, args: func(uri string) []string {
			return []string{"metadata", "defaults", "get", "id", "--uri", uri, "--token", "t"}
		}},
		{name: "metadata defaults list", body: `[]`, args: func(uri string) []string {
			return []string{"metadata", "defaults", "list", "--uri", uri, "--token", "t"}
		}},
		{name: "metadata group get", body: `{}`, args: func(uri string) []string {
			return []string{"metadata", "group", "get", "id", "--uri", uri, "--token", "t"}
		}},
		{name: "metadata group list", body: `[]`, args: func(uri string) []string { return []string{"metadata", "group", "list", "--uri", uri, "--token", "t"} }},
		{name: "metadata instance get", body: `{}`, args: func(uri string) []string {
			return []string{"metadata", "instance", "get", "id", "--uri", uri, "--token", "t"}
		}},
		{name: "metadata instance list", body: `[]`, args: func(uri string) []string {
			return []string{"metadata", "instance", "list", "--uri", uri, "--token", "t"}
		}},
		{name: "metadata peer get", body: `{}`, args: func(uri string) []string {
			return []string{"metadata", "peer", "get", "id", "--uri", uri, "--token", "t"}
		}},
		{name: "metadata peer list", body: `[]`, args: func(uri string) []string { return []string{"metadata", "peer", "list", "--uri", uri, "--token", "t"} }},
		{name: "pcs status list", body: `{"status":[]}`, args: func(uri string) []string { return []string{"pcs", "status", "list", "--uri", uri} }},
		{name: "pcs status show", body: `{"status":[{"xname":"node","powerState":"on"}]}`, args: func(uri string) []string { return []string{"pcs", "status", "show", "node", "--uri", uri} }},
		{name: "pcs transition list", body: `{"transitions":[]}`, args: func(uri string) []string { return []string{"pcs", "transition", "list", "--uri", uri} }},
		{name: "pcs transition show", body: `{}`, args: func(uri string) []string { return []string{"pcs", "transition", "show", "id", "--uri", uri} }},
		{name: "pcs transition start", body: `{"TransitionID":"id","Operation":"on"}`, args: func(uri string) []string {
			return []string{"pcs", "transition", "start", "--uri", uri, "--token", "t", "--xname", "x0c0s0b0n0", "on"}
		}},
		{name: "smd component get", body: `{"Components":[]}`, args: func(uri string) []string { return []string{"smd", "component", "get", "--uri", uri} }},
		{name: "smd group get", body: `[]`, args: func(uri string) []string { return []string{"smd", "group", "get", "--uri", uri} }},
		{name: "smd group membership", body: `{}`, args: func(uri string) []string { return []string{"smd", "group", "membership", "--uri", uri} }},
		{name: "smd iface get", body: `[]`, args: func(uri string) []string { return []string{"smd", "iface", "get", "--uri", uri} }},
		{name: "smd rfe get", body: `[]`, args: func(uri string) []string { return []string{"smd", "rfe", "get", "--uri", uri} }},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = io.WriteString(w, tc.body) //nolint:errcheck // response writes are observed by the client
			}))
			defer srv.Close()

			args := append([]string{"--ignore-config"}, tc.args(srv.URL)...)
			res := runOchamiWithOutputWriter(t, commandErrorWriter{}, args...)
			if res.err == nil {
				t.Fatal("expected output error, got nil")
			}
			if res.exitCode != cli.CodePayload {
				t.Errorf("exit code = %d, want %d (CodePayload): %v", res.exitCode, cli.CodePayload, res.err)
			}
			if !strings.Contains(res.err.Error(), "injected command output failure") {
				t.Errorf("error = %q, want injected writer failure", res.err)
			}
		})
	}
}

func TestAuthenticatedCommandsRejectMissingTokenBeforeRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		new  func() *cobra.Command
		args []string
	}{
		{name: "boot bmc add", new: bootcmd.NewCmd, args: []string{"bmc", "add", "-d", `{}`}},
		{name: "boot bmc delete", new: bootcmd.NewCmd, args: []string{"bmc", "delete", "--no-confirm", "id"}},
		{name: "boot bmc get", new: bootcmd.NewCmd, args: []string{"bmc", "get", "id"}},
		{name: "boot bmc list", new: bootcmd.NewCmd, args: []string{"bmc", "list"}},
		{name: "boot bmc patch", new: bootcmd.NewCmd, args: []string{"bmc", "patch", "id", "-d", `{}`}},
		{name: "boot bmc set", new: bootcmd.NewCmd, args: []string{"bmc", "set", "id", "-d", `{}`}},
		{name: "boot config add", new: bootcmd.NewCmd, args: []string{"config", "add", "-d", `{}`}},
		{name: "boot config delete", new: bootcmd.NewCmd, args: []string{"config", "delete", "--no-confirm", "id"}},
		{name: "boot config get", new: bootcmd.NewCmd, args: []string{"config", "get", "id"}},
		{name: "boot config list", new: bootcmd.NewCmd, args: []string{"config", "list"}},
		{name: "boot config patch", new: bootcmd.NewCmd, args: []string{"config", "patch", "id", "-d", `{}`}},
		{name: "boot config set", new: bootcmd.NewCmd, args: []string{"config", "set", "id", "-d", `{}`}},
		{name: "boot node add", new: bootcmd.NewCmd, args: []string{"node", "add", "-d", `{}`}},
		{name: "boot node delete", new: bootcmd.NewCmd, args: []string{"node", "delete", "--no-confirm", "id"}},
		{name: "boot node get", new: bootcmd.NewCmd, args: []string{"node", "get", "id"}},
		{name: "boot node list", new: bootcmd.NewCmd, args: []string{"node", "list"}},
		{name: "boot node patch", new: bootcmd.NewCmd, args: []string{"node", "patch", "id", "-d", `{}`}},
		{name: "boot node set", new: bootcmd.NewCmd, args: []string{"node", "set", "id", "-d", `{}`}},

		{name: "bss image set", new: bsscmd.NewCmd, args: []string{"boot", "image", "set", "--mac", "de:ad:be:ef:00:00", "image"}},
		{name: "bss params add", new: bsscmd.NewCmd, args: []string{"boot", "params", "add", "-d", `{}`}},
		{name: "bss params delete", new: bsscmd.NewCmd, args: []string{"boot", "params", "delete", "--no-confirm", "-d", `{}`}},
		{name: "bss params get", new: bsscmd.NewCmd, args: []string{"boot", "params", "get"}},
		{name: "bss params set", new: bsscmd.NewCmd, args: []string{"boot", "params", "set", "-d", `{}`}},
		{name: "bss params update", new: bsscmd.NewCmd, args: []string{"boot", "params", "update", "-d", `{}`}},

		{name: "cloud-init group add", new: cloudinitcmd.NewCmd, args: []string{"group", "add", "-d", `[]`}},
		{name: "cloud-init group delete", new: cloudinitcmd.NewCmd, args: []string{"group", "delete", "--no-confirm", "group"}},
		{name: "cloud-init group config", new: cloudinitcmd.NewCmd, args: []string{"group", "get", "config"}},
		{name: "cloud-init group metadata", new: cloudinitcmd.NewCmd, args: []string{"group", "get", "meta-data"}},
		{name: "cloud-init group raw", new: cloudinitcmd.NewCmd, args: []string{"group", "get", "raw"}},
		{name: "cloud-init group render", new: cloudinitcmd.NewCmd, args: []string{"group", "render", "group", "node"}},
		{name: "cloud-init group set", new: cloudinitcmd.NewCmd, args: []string{"group", "set", "-d", `[]`}},
		{name: "cloud-init node group", new: cloudinitcmd.NewCmd, args: []string{"node", "get", "group", "node", "group"}},
		{name: "cloud-init node metadata", new: cloudinitcmd.NewCmd, args: []string{"node", "get", "meta-data", "node"}},
		{name: "cloud-init node user data", new: cloudinitcmd.NewCmd, args: []string{"node", "get", "user-data", "node"}},
		{name: "cloud-init node vendor data", new: cloudinitcmd.NewCmd, args: []string{"node", "get", "vendor-data", "node"}},
		{name: "cloud-init node set", new: cloudinitcmd.NewCmd, args: []string{"node", "set", "-d", `[]`}},

		{name: "metadata defaults add", new: metadatacmd.NewCmd, args: []string{"defaults", "add", "-d", `{}`}},
		{name: "metadata defaults delete", new: metadatacmd.NewCmd, args: []string{"defaults", "delete", "--no-confirm", "id"}},
		{name: "metadata defaults get", new: metadatacmd.NewCmd, args: []string{"defaults", "get", "id"}},
		{name: "metadata defaults list", new: metadatacmd.NewCmd, args: []string{"defaults", "list"}},
		{name: "metadata defaults patch", new: metadatacmd.NewCmd, args: []string{"defaults", "patch", "id", "-d", `{}`}},
		{name: "metadata defaults set", new: metadatacmd.NewCmd, args: []string{"defaults", "set", "id", "-d", `{}`}},
		{name: "metadata group add", new: metadatacmd.NewCmd, args: []string{"group", "add", "-d", `{}`}},
		{name: "metadata group delete", new: metadatacmd.NewCmd, args: []string{"group", "delete", "--no-confirm", "id"}},
		{name: "metadata group get", new: metadatacmd.NewCmd, args: []string{"group", "get", "id"}},
		{name: "metadata group list", new: metadatacmd.NewCmd, args: []string{"group", "list"}},
		{name: "metadata group patch", new: metadatacmd.NewCmd, args: []string{"group", "patch", "id", "-d", `{}`}},
		{name: "metadata group set", new: metadatacmd.NewCmd, args: []string{"group", "set", "id", "-d", `{}`}},
		{name: "metadata instance add", new: metadatacmd.NewCmd, args: []string{"instance", "add", "-d", `{}`}},
		{name: "metadata instance delete", new: metadatacmd.NewCmd, args: []string{"instance", "delete", "--no-confirm", "id"}},
		{name: "metadata instance get", new: metadatacmd.NewCmd, args: []string{"instance", "get", "id"}},
		{name: "metadata instance list", new: metadatacmd.NewCmd, args: []string{"instance", "list"}},
		{name: "metadata instance patch", new: metadatacmd.NewCmd, args: []string{"instance", "patch", "id", "-d", `{}`}},
		{name: "metadata instance set", new: metadatacmd.NewCmd, args: []string{"instance", "set", "id", "-d", `{}`}},
		{name: "metadata peer add", new: metadatacmd.NewCmd, args: []string{"peer", "add", "-d", `{}`}},
		{name: "metadata peer delete", new: metadatacmd.NewCmd, args: []string{"peer", "delete", "--no-confirm", "id"}},
		{name: "metadata peer get", new: metadatacmd.NewCmd, args: []string{"peer", "get", "id"}},
		{name: "metadata peer list", new: metadatacmd.NewCmd, args: []string{"peer", "list"}},
		{name: "metadata peer patch", new: metadatacmd.NewCmd, args: []string{"peer", "patch", "id", "-d", `{}`}},
		{name: "metadata peer set", new: metadatacmd.NewCmd, args: []string{"peer", "set", "id", "-d", `{}`}},

		{name: "smd compep get", new: smdcmd.NewCmd, args: []string{"compep", "get"}},
		{name: "smd component add", new: smdcmd.NewCmd, args: []string{"component", "add", "node", "1"}},
		{name: "smd group add", new: smdcmd.NewCmd, args: []string{"group", "add", "group"}},
		{name: "smd group get", new: smdcmd.NewCmd, args: []string{"group", "get"}},
		{name: "smd group membership", new: smdcmd.NewCmd, args: []string{"group", "membership"}},
		{name: "smd iface add", new: smdcmd.NewCmd, args: []string{"iface", "add", "node", "de:ad:be:ef:00:00", "NMN,172.16.0.1"}},
		{name: "smd rfe add", new: smdcmd.NewCmd, args: []string{"rfe", "add", "bmc", "bmc.example", "172.16.0.1", "de:ad:be:ef:00:00"}},
		{name: "smd rfe get", new: smdcmd.NewCmd, args: []string{"rfe", "get"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			requestMade := false
			srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				requestMade = true
			}))
			defer srv.Close()

			var output bytes.Buffer
			rt := cli.NewTestRuntime(strings.NewReader(""), &output, &output)
			rt.WithConfig(config.Config{
				DefaultCluster: "secure",
				Clusters: []config.ConfigCluster{{
					Name: "secure",
					Cluster: config.ConfigClusterConfig{
						URI:        srv.URL,
						EnableAuth: true,
					},
				}},
			})

			cmd := tc.new()
			if cmd.PersistentFlags().Lookup("cluster") == nil {
				cmd.PersistentFlags().String("cluster", "", "test cluster")
			}
			if cmd.PersistentFlags().Lookup("cluster-uri") == nil {
				cmd.PersistentFlags().String("cluster-uri", "", "test cluster URI")
			}
			cmd.SetContext(cli.ContextWithRuntime(t.Context(), rt))
			cmd.SetOut(&output)
			cmd.SetErr(&output)
			cmd.SetArgs(append(tc.args, "--cluster", "secure", "--uri", srv.URL))

			err := cmd.Execute()
			if err == nil {
				t.Fatal("expected missing-token error, got nil")
			}
			if cli.ExitCode(err) != cli.CodeAuth {
				t.Errorf("exit code = %d, want %d (CodeAuth): %v", cli.ExitCode(err), cli.CodeAuth, err)
			}
			if requestMade {
				t.Error("service received a request despite missing authentication")
			}
		})
	}
}

func TestLeafCommandsRequireInvocationRuntime(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		new  func() *cobra.Command
		args []string
	}{
		{name: "boot bmc add", new: bootcmd.NewCmd, args: []string{"bmc", "add", "-d", `{}`}},
		{name: "boot bmc delete", new: bootcmd.NewCmd, args: []string{"bmc", "delete", "--no-confirm", "id"}},
		{name: "boot bmc get", new: bootcmd.NewCmd, args: []string{"bmc", "get", "id"}},
		{name: "boot bmc list", new: bootcmd.NewCmd, args: []string{"bmc", "list"}},
		{name: "boot bmc patch", new: bootcmd.NewCmd, args: []string{"bmc", "patch", "id", "-d", `{}`}},
		{name: "boot bmc set", new: bootcmd.NewCmd, args: []string{"bmc", "set", "id", "-d", `{}`}},
		{name: "boot config add", new: bootcmd.NewCmd, args: []string{"config", "add", "-d", `{}`}},
		{name: "boot config delete", new: bootcmd.NewCmd, args: []string{"config", "delete", "--no-confirm", "id"}},
		{name: "boot config get", new: bootcmd.NewCmd, args: []string{"config", "get", "id"}},
		{name: "boot config list", new: bootcmd.NewCmd, args: []string{"config", "list"}},
		{name: "boot config patch", new: bootcmd.NewCmd, args: []string{"config", "patch", "id", "-d", `{}`}},
		{name: "boot config set", new: bootcmd.NewCmd, args: []string{"config", "set", "id", "-d", `{}`}},
		{name: "boot node add", new: bootcmd.NewCmd, args: []string{"node", "add", "-d", `{}`}},
		{name: "boot node delete", new: bootcmd.NewCmd, args: []string{"node", "delete", "--no-confirm", "id"}},
		{name: "boot node get", new: bootcmd.NewCmd, args: []string{"node", "get", "id"}},
		{name: "boot node list", new: bootcmd.NewCmd, args: []string{"node", "list"}},
		{name: "boot node patch", new: bootcmd.NewCmd, args: []string{"node", "patch", "id", "-d", `{}`}},
		{name: "boot node set", new: bootcmd.NewCmd, args: []string{"node", "set", "id", "-d", `{}`}},
		{name: "boot service status", new: bootcmd.NewCmd, args: []string{"service", "status"}},

		{name: "bss image set", new: bsscmd.NewCmd, args: []string{"boot", "image", "set", "--mac", "de:ad:be:ef:00:00", "image"}},
		{name: "bss params add", new: bsscmd.NewCmd, args: []string{"boot", "params", "add", "-d", `{}`}},
		{name: "bss params delete", new: bsscmd.NewCmd, args: []string{"boot", "params", "delete", "--no-confirm", "-d", `{}`}},
		{name: "bss params get", new: bsscmd.NewCmd, args: []string{"boot", "params", "get"}},
		{name: "bss params set", new: bsscmd.NewCmd, args: []string{"boot", "params", "set", "-d", `{}`}},
		{name: "bss params update", new: bsscmd.NewCmd, args: []string{"boot", "params", "update", "-d", `{}`}},
		{name: "bss script get", new: bsscmd.NewCmd, args: []string{"boot", "script", "get", "--mac", "de:ad:be:ef:00:00"}},
		{name: "bss dumpstate", new: bsscmd.NewCmd, args: []string{"dumpstate"}},
		{name: "bss history", new: bsscmd.NewCmd, args: []string{"history"}},
		{name: "bss hosts", new: bsscmd.NewCmd, args: []string{"hosts", "get"}},
		{name: "bss service status", new: bsscmd.NewCmd, args: []string{"service", "status"}},
		{name: "bss service version", new: bsscmd.NewCmd, args: []string{"service", "version"}},
		{name: "bss deprecated status", new: bsscmd.NewCmd, args: []string{"status"}},

		{name: "cloud-init defaults get", new: cloudinitcmd.NewCmd, args: []string{"defaults", "get"}},
		{name: "cloud-init defaults set", new: cloudinitcmd.NewCmd, args: []string{"defaults", "set", "-d", `{}`}},
		{name: "cloud-init group add", new: cloudinitcmd.NewCmd, args: []string{"group", "add", "-d", `[]`}},
		{name: "cloud-init group delete", new: cloudinitcmd.NewCmd, args: []string{"group", "delete", "--no-confirm", "group"}},
		{name: "cloud-init group config", new: cloudinitcmd.NewCmd, args: []string{"group", "get", "config"}},
		{name: "cloud-init group metadata", new: cloudinitcmd.NewCmd, args: []string{"group", "get", "meta-data"}},
		{name: "cloud-init group raw", new: cloudinitcmd.NewCmd, args: []string{"group", "get", "raw"}},
		{name: "cloud-init group render", new: cloudinitcmd.NewCmd, args: []string{"group", "render", "group", "node"}},
		{name: "cloud-init group set", new: cloudinitcmd.NewCmd, args: []string{"group", "set", "-d", `[]`}},
		{name: "cloud-init node group", new: cloudinitcmd.NewCmd, args: []string{"node", "get", "group", "node", "group"}},
		{name: "cloud-init node metadata", new: cloudinitcmd.NewCmd, args: []string{"node", "get", "meta-data", "node"}},
		{name: "cloud-init node user data", new: cloudinitcmd.NewCmd, args: []string{"node", "get", "user-data", "node"}},
		{name: "cloud-init node vendor data", new: cloudinitcmd.NewCmd, args: []string{"node", "get", "vendor-data", "node"}},
		{name: "cloud-init node set", new: cloudinitcmd.NewCmd, args: []string{"node", "set", "-d", `[]`}},
		{name: "cloud-init service status", new: cloudinitcmd.NewCmd, args: []string{"service", "status"}},
		{name: "cloud-init service version", new: cloudinitcmd.NewCmd, args: []string{"service", "version"}},

		{name: "config set", new: configcmd.NewCmd, args: []string{"set", "log.level", "debug"}},
		{name: "config show", new: configcmd.NewCmd, args: []string{"show"}},
		{name: "config unset", new: configcmd.NewCmd, args: []string{"unset", "log.level"}},
		{name: "config cluster delete", new: configcmd.NewCmd, args: []string{"cluster", "delete", "cluster"}},
		{name: "config cluster set", new: configcmd.NewCmd, args: []string{"cluster", "set", "cluster", "cluster.uri", "https://example.com"}},
		{name: "config cluster show", new: configcmd.NewCmd, args: []string{"cluster", "show", "cluster"}},
		{name: "config cluster unset", new: configcmd.NewCmd, args: []string{"cluster", "unset", "cluster", "cluster.uri"}},

		{name: "metadata defaults add", new: metadatacmd.NewCmd, args: []string{"defaults", "add", "-d", `{}`}},
		{name: "metadata defaults delete", new: metadatacmd.NewCmd, args: []string{"defaults", "delete", "--no-confirm", "id"}},
		{name: "metadata defaults get", new: metadatacmd.NewCmd, args: []string{"defaults", "get", "id"}},
		{name: "metadata defaults list", new: metadatacmd.NewCmd, args: []string{"defaults", "list"}},
		{name: "metadata defaults patch", new: metadatacmd.NewCmd, args: []string{"defaults", "patch", "id", "-d", `{}`}},
		{name: "metadata defaults set", new: metadatacmd.NewCmd, args: []string{"defaults", "set", "id", "-d", `{}`}},
		{name: "metadata group add", new: metadatacmd.NewCmd, args: []string{"group", "add", "-d", `{}`}},
		{name: "metadata group delete", new: metadatacmd.NewCmd, args: []string{"group", "delete", "--no-confirm", "id"}},
		{name: "metadata group get", new: metadatacmd.NewCmd, args: []string{"group", "get", "id"}},
		{name: "metadata group list", new: metadatacmd.NewCmd, args: []string{"group", "list"}},
		{name: "metadata group patch", new: metadatacmd.NewCmd, args: []string{"group", "patch", "id", "-d", `{}`}},
		{name: "metadata group set", new: metadatacmd.NewCmd, args: []string{"group", "set", "id", "-d", `{}`}},
		{name: "metadata instance add", new: metadatacmd.NewCmd, args: []string{"instance", "add", "-d", `{}`}},
		{name: "metadata instance delete", new: metadatacmd.NewCmd, args: []string{"instance", "delete", "--no-confirm", "id"}},
		{name: "metadata instance get", new: metadatacmd.NewCmd, args: []string{"instance", "get", "id"}},
		{name: "metadata instance list", new: metadatacmd.NewCmd, args: []string{"instance", "list"}},
		{name: "metadata instance patch", new: metadatacmd.NewCmd, args: []string{"instance", "patch", "id", "-d", `{}`}},
		{name: "metadata instance set", new: metadatacmd.NewCmd, args: []string{"instance", "set", "id", "-d", `{}`}},
		{name: "metadata peer add", new: metadatacmd.NewCmd, args: []string{"peer", "add", "-d", `{}`}},
		{name: "metadata peer delete", new: metadatacmd.NewCmd, args: []string{"peer", "delete", "--no-confirm", "id"}},
		{name: "metadata peer get", new: metadatacmd.NewCmd, args: []string{"peer", "get", "id"}},
		{name: "metadata peer list", new: metadatacmd.NewCmd, args: []string{"peer", "list"}},
		{name: "metadata peer patch", new: metadatacmd.NewCmd, args: []string{"peer", "patch", "id", "-d", `{}`}},
		{name: "metadata peer set", new: metadatacmd.NewCmd, args: []string{"peer", "set", "id", "-d", `{}`}},
		{name: "metadata service status", new: metadatacmd.NewCmd, args: []string{"service", "status"}},

		{name: "pcs service status", new: pcscmd.NewCmd, args: []string{"service", "status"}},
		{name: "pcs status list", new: pcscmd.NewCmd, args: []string{"status", "list"}},
		{name: "pcs status show", new: pcscmd.NewCmd, args: []string{"status", "show", "node"}},
		{name: "pcs transition abort", new: pcscmd.NewCmd, args: []string{"transition", "abort", "id"}},
		{name: "pcs transition list", new: pcscmd.NewCmd, args: []string{"transition", "list"}},
		{name: "pcs transition monitor", new: pcscmd.NewCmd, args: []string{"transition", "monitor", "id"}},
		{name: "pcs transition show", new: pcscmd.NewCmd, args: []string{"transition", "show", "id"}},
		{name: "pcs transition start", new: pcscmd.NewCmd, args: []string{"transition", "start", "--xname", "node", "on"}},

		{name: "rcs console list", new: rcscmd.NewCmd, args: []string{"console", "list"}},
		{name: "rcs console show", new: rcscmd.NewCmd, args: []string{"console", "show", "node"}},
		{name: "rcs service status", new: rcscmd.NewCmd, args: []string{"service", "status"}},

		{name: "smd compep get", new: smdcmd.NewCmd, args: []string{"compep", "get"}},
		{name: "smd component add", new: smdcmd.NewCmd, args: []string{"component", "add", "node", "1"}},
		{name: "smd component get", new: smdcmd.NewCmd, args: []string{"component", "get"}},
		{name: "smd group add", new: smdcmd.NewCmd, args: []string{"group", "add", "group"}},
		{name: "smd group get", new: smdcmd.NewCmd, args: []string{"group", "get"}},
		{name: "smd group membership", new: smdcmd.NewCmd, args: []string{"group", "membership"}},
		{name: "smd iface add", new: smdcmd.NewCmd, args: []string{"iface", "add", "node", "de:ad:be:ef:00:00", "NMN,172.16.0.1"}},
		{name: "smd iface get", new: smdcmd.NewCmd, args: []string{"iface", "get"}},
		{name: "smd rfe add", new: smdcmd.NewCmd, args: []string{"rfe", "add", "bmc", "bmc.example", "172.16.0.1", "de:ad:be:ef:00:00"}},
		{name: "smd rfe get", new: smdcmd.NewCmd, args: []string{"rfe", "get"}},
		{name: "smd service status", new: smdcmd.NewCmd, args: []string{"service", "status"}},
		{name: "smd deprecated status", new: smdcmd.NewCmd, args: []string{"status"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cmd := tc.new()
			if cmd.PersistentFlags().Lookup("config") == nil {
				cmd.PersistentFlags().String("config", "", "test config path")
			}
			if cmd.PersistentFlags().Lookup("system") == nil {
				cmd.PersistentFlags().Bool("system", false, "test system config")
			}
			if cmd.PersistentFlags().Lookup("user") == nil {
				cmd.PersistentFlags().Bool("user", false, "test user config")
			}
			var output bytes.Buffer
			cmd.SetOut(&output)
			cmd.SetErr(&output)
			cmd.SetArgs(tc.args)

			err := cmd.Execute()
			if err == nil {
				t.Fatal("expected missing-runtime error, got nil")
			}
			if cli.ExitCode(err) != cli.CodeConfig {
				t.Errorf("exit code = %d, want %d (CodeConfig): %v", cli.ExitCode(err), cli.CodeConfig, err)
			}
			if !strings.Contains(err.Error(), "CLI runtime is unavailable") {
				t.Errorf("error = %q, want runtime-unavailable context", err)
			}
		})
	}
}

func TestCommandTransportFailures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		wantCode int
		args     func(string) []string
	}{
		{
			name:     "cloud-init defaults get",
			wantCode: cli.CodeNetwork,
			args: func(uri string) []string {
				return []string{"--ignore-config", "cloud-init", "defaults", "get", "--uri", uri}
			},
		},
		{
			name:     "cloud-init defaults set",
			wantCode: cli.CodeNetwork,
			args: func(uri string) []string {
				return []string{"--ignore-config", "cloud-init", "defaults", "set", "--uri", uri, "-d", `{}`}
			},
		},
		{
			name:     "cloud-init group add",
			wantCode: cli.CodeHTTP,
			args: func(uri string) []string {
				return []string{"--ignore-config", "cloud-init", "group", "add", "--uri", uri, "--token", "t", "-d", `[{"name":"compute"}]`}
			},
		},
		{
			name:     "cloud-init group set",
			wantCode: cli.CodeHTTP,
			args: func(uri string) []string {
				return []string{"--ignore-config", "cloud-init", "group", "set", "--uri", uri, "--token", "t", "-d", `[{"name":"compute"}]`}
			},
		},
		{
			name:     "cloud-init group delete",
			wantCode: cli.CodeHTTP,
			args: func(uri string) []string {
				return []string{"--ignore-config", "cloud-init", "group", "delete", "--uri", uri, "--token", "t", "--no-confirm", "compute"}
			},
		},
		{
			name:     "cloud-init node set",
			wantCode: cli.CodeHTTP,
			args: func(uri string) []string {
				return []string{"--ignore-config", "cloud-init", "node", "set", "--uri", uri, "--token", "t", "-d", `[{"id":"x0c0s0b0n0"}]`}
			},
		},
		{
			name:     "cloud-init node metadata",
			wantCode: cli.CodeHTTP,
			args: func(uri string) []string {
				return []string{"--ignore-config", "cloud-init", "node", "get", "meta-data", "--uri", uri, "--token", "t", "x0c0s0b0n0"}
			},
		},
		{
			name:     "cloud-init node user data",
			wantCode: cli.CodeHTTP,
			args: func(uri string) []string {
				return []string{"--ignore-config", "cloud-init", "node", "get", "user-data", "--uri", uri, "--token", "t", "x0c0s0b0n0"}
			},
		},
		{
			name:     "cloud-init node vendor data",
			wantCode: cli.CodeHTTP,
			args: func(uri string) []string {
				return []string{"--ignore-config", "cloud-init", "node", "get", "vendor-data", "--uri", uri, "--token", "t", "x0c0s0b0n0"}
			},
		},
		{
			name:     "cloud-init node group",
			wantCode: cli.CodeHTTP,
			args: func(uri string) []string {
				return []string{"--ignore-config", "cloud-init", "node", "get", "group", "--uri", uri, "--token", "t", "x0c0s0b0n0", "compute"}
			},
		},
		{
			name:     "cloud-init service version",
			wantCode: cli.CodeNetwork,
			args: func(uri string) []string {
				return []string{"--ignore-config", "cloud-init", "service", "version", "--uri", uri}
			},
		},
		{
			name:     "cloud-init service api",
			wantCode: cli.CodeHTTP,
			args: func(uri string) []string {
				return []string{"--ignore-config", "cloud-init", "service", "status", "--api", "--uri", uri}
			},
		},
		{
			name:     "bss dumpstate",
			wantCode: cli.CodeNetwork,
			args: func(uri string) []string {
				return []string{"--ignore-config", "bss", "dumpstate", "--uri", uri}
			},
		},
		{
			name:     "bss hosts",
			wantCode: cli.CodeNetwork,
			args: func(uri string) []string {
				return []string{"--ignore-config", "bss", "hosts", "get", "--uri", uri}
			},
		},
		{
			name:     "bss history",
			wantCode: cli.CodeNetwork,
			args: func(uri string) []string {
				return []string{"--ignore-config", "bss", "history", "--uri", uri}
			},
		},
		{
			name:     "bss script",
			wantCode: cli.CodeNetwork,
			args: func(uri string) []string {
				return []string{"--ignore-config", "bss", "boot", "script", "get", "--uri", uri, "--mac", "de:ad:be:ef:00:00"}
			},
		},
		{
			name:     "bss service status",
			wantCode: cli.CodeNetwork,
			args: func(uri string) []string {
				return []string{"--ignore-config", "bss", "service", "status", "--uri", uri}
			},
		},
		{
			name:     "bss service version",
			wantCode: cli.CodeNetwork,
			args: func(uri string) []string {
				return []string{"--ignore-config", "bss", "service", "version", "--uri", uri}
			},
		},
		{
			name:     "pcs transition list",
			wantCode: cli.CodeNetwork,
			args: func(uri string) []string {
				return []string{"--ignore-config", "pcs", "transition", "list", "--uri", uri}
			},
		},
		{
			name:     "pcs transition show",
			wantCode: cli.CodeNetwork,
			args: func(uri string) []string {
				return []string{"--ignore-config", "pcs", "transition", "show", "--uri", uri, "transition-id"}
			},
		},
		{
			name:     "pcs transition abort",
			wantCode: cli.CodeNetwork,
			args: func(uri string) []string {
				return []string{"--ignore-config", "pcs", "transition", "abort", "--uri", uri, "transition-id"}
			},
		},
		{
			name:     "pcs transition start",
			wantCode: cli.CodeNetwork,
			args: func(uri string) []string {
				return []string{"--ignore-config", "pcs", "transition", "start", "--uri", uri, "--xname", "x0c0s0b0n0", "on"}
			},
		},
		{
			name:     "pcs status list",
			wantCode: cli.CodeNetwork,
			args: func(uri string) []string {
				return []string{"--ignore-config", "pcs", "status", "list", "--uri", uri}
			},
		},
		{
			name:     "pcs status show",
			wantCode: cli.CodeNetwork,
			args: func(uri string) []string {
				return []string{"--ignore-config", "pcs", "status", "show", "--uri", uri, "x0c0s0b0n0"}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			res := runOchamiWithRuntime(t, tc.args(closedTestServerURL(t))...)
			if res.err == nil {
				t.Fatal("expected a transport error, got nil")
			}
			if res.exitCode != tc.wantCode {
				t.Errorf("exit code = %d, want %d: %v", res.exitCode, tc.wantCode, res.err)
			}
		})
	}
}

func TestResourceCommandTransportFailures(t *testing.T) {
	t.Parallel()

	for _, resource := range bootTypes {
		resource := resource
		t.Run("boot/"+resource, func(t *testing.T) {
			t.Parallel()
			uri := closedTestServerURL(t)
			tests := [][]string{
				{"boot", resource, "add", "--uri", uri, "--token", "t", "-d", bootAddPayload(resource)},
				{"boot", resource, "get", "some-uid", "--uri", uri, "--token", "t"},
				{"boot", resource, "patch", "some-uid", "--uri", uri, "--token", "t", "-d", bootAddPayload(resource)},
				{"boot", resource, "set", "some-uid", "--uri", uri, "--token", "t", "-d", bootAddPayload(resource)},
				{"boot", resource, "delete", "some-uid", "--uri", uri, "--token", "t", "--no-confirm"},
			}
			for _, args := range tests {
				res := runOchamiWithRuntime(t, append([]string{"--ignore-config"}, args...)...)
				if res.err == nil || res.exitCode == cli.CodeSuccess {
					t.Errorf("args %v: result = (err %v, exit %d), want transport failure", args, res.err, res.exitCode)
				}
			}
		})
	}

	for _, resource := range metadataTypes {
		resource := resource
		t.Run("metadata/"+resource, func(t *testing.T) {
			t.Parallel()
			uri := closedTestServerURL(t)
			tests := [][]string{
				{"metadata", resource, "add", "--uri", uri, "--token", "t", "-d", addPayloadFor(resource)},
				{"metadata", resource, "get", "some-uid", "--uri", uri, "--token", "t"},
				{"metadata", resource, "patch", "some-uid", "--uri", uri, "--token", "t", "-d", addPayloadFor(resource)},
				{"metadata", resource, "set", "some-uid", "--uri", uri, "--token", "t", "-d", addPayloadFor(resource)},
				{"metadata", resource, "delete", "some-uid", "--uri", uri, "--token", "t", "--no-confirm"},
			}
			for _, args := range tests {
				res := runOchamiWithRuntime(t, append([]string{"--ignore-config"}, args...)...)
				if res.err == nil || res.exitCode == cli.CodeSuccess {
					t.Errorf("args %v: result = (err %v, exit %d), want transport failure", args, res.err, res.exitCode)
				}
			}
		})
	}
}

func TestResourceCommandsRejectMalformedSuccessBodies(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{`) //nolint:errcheck // malformed body is the test fixture
	}))
	defer srv.Close()

	for _, resource := range bootTypes {
		resource := resource
		for _, verb := range []string{"get", "list"} {
			verb := verb
			t.Run("boot/"+resource+"/"+verb, func(t *testing.T) {
				args := []string{"--ignore-config", "boot", resource, verb}
				if verb == "get" {
					args = append(args, "some-uid")
				}
				args = append(args, "--uri", srv.URL, "--token", "t")
				res := runOchamiWithRuntime(t, args...)
				if res.err == nil || res.exitCode != cli.CodeNetwork {
					t.Errorf("result = (err %v, exit %d), want CodeNetwork", res.err, res.exitCode)
				}
				if !strings.Contains(res.err.Error(), "failed to unmarshal response") {
					t.Errorf("error = %q, want malformed-response context", res.err)
				}
			})
		}
	}

	for _, resource := range metadataTypes {
		resource := resource
		for _, verb := range []string{"get", "list"} {
			verb := verb
			t.Run("metadata/"+resource+"/"+verb, func(t *testing.T) {
				args := []string{"--ignore-config", "metadata", resource, verb}
				if verb == "get" {
					args = append(args, "some-uid")
				}
				args = append(args, "--uri", srv.URL, "--token", "t")
				res := runOchamiWithRuntime(t, args...)
				if res.err == nil || res.exitCode != cli.CodeNetwork {
					t.Errorf("result = (err %v, exit %d), want CodeNetwork", res.err, res.exitCode)
				}
				if !strings.Contains(res.err.Error(), "failed to unmarshal response") {
					t.Errorf("error = %q, want malformed-response context", res.err)
				}
			})
		}
	}
}
