// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// TestCloudInitRender_GroupSources verifies both command forms select the node's
// group template, with metadata replacement available only on the canonical form.
func TestCloudInitRender_GroupSources(t *testing.T) {
	t.Parallel()
	for _, form := range []string{"canonical", "compatibility"} {
		sources := []string{"live"}
		if form == "canonical" {
			sources = append(sources, "file", "stdin")
		}
		for _, metadata := range sources {
			t.Run(form+"/"+metadata, func(t *testing.T) {
				t.Parallel()
				var groupRequests, metadataRequests atomic.Int32
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Method != http.MethodGet || r.Header.Get("Authorization") != "Bearer test-token" {
						t.Errorf("request = %s with authorization %q, want authenticated GET", r.Method, r.Header.Get("Authorization"))
					}
					switch r.URL.Path {
					case "/admin/impersonation/node01/compute.yaml":
						groupRequests.Add(1)
						io.WriteString(w, "{{ds.meta_data.hostname}}|{{role}}\n")
					case "/admin/impersonation/node01/meta-data":
						metadataRequests.Add(1)
						if metadata != "live" {
							t.Error("metadata endpoint requested despite replacement metadata")
							w.WriteHeader(http.StatusInternalServerError)
							return
						}
						io.WriteString(w, "hostname: live-node\n")
					default:
						t.Errorf("unexpected request path %q", r.URL.Path)
						http.NotFound(w, r)
					}
				}))
				t.Cleanup(srv.Close)
				args := []string{"--ignore-config", "cloud-init", "render", "--group", "compute", "--node", "node01"}
				if form == "compatibility" {
					args = []string{"--ignore-config", "cloud-init", "group", "render", "compute", "node01"}
				}
				args = append(args, "--uri", srv.URL, "--token", "test-token", "--extra-vars", `{"role":"compute"}`)
				stdin, want, wantMetadataRequests := "", "live-node|compute", int32(1)
				if metadata != "live" {
					path := writeRenderInput(t, "metadata.yaml", "hostname: replacement\n")
					if metadata == "stdin" {
						path, stdin = "-", "hostname: replacement\n"
					}
					args = append(args, "--metadata", path)
					want, wantMetadataRequests = "replacement|compute", 0
				}
				res := runOchamiWithInputAndRuntime(t, stdin, args...)
				if res.err != nil || res.stdout != want {
					t.Fatalf("stdout = %q, error = %v, want %q without error", res.stdout, res.err, want)
				}
				if groupRequests.Load() != 1 || metadataRequests.Load() != wantMetadataRequests {
					t.Fatalf("group, metadata requests = %d, %d, want 1, %d", groupRequests.Load(), metadataRequests.Load(), wantMetadataRequests)
				}
			})
		}
	}
}

// TestCloudInitRender_CompatibilityHelp verifies legacy help points to the
// canonical command and accepts only the positional group/node form.
func TestCloudInitRender_CompatibilityHelp(t *testing.T) {
	t.Parallel()
	res := runOchamiWithRuntime(t, "--ignore-config", "cloud-init", "group", "render", "--help")
	if res.err != nil || !strings.Contains(res.stdout, "ochami cloud-init render --group GROUP --node NODE") {
		t.Fatalf("help = %q, error = %v, want canonical invocation", res.stdout, res.err)
	}
	res = runOchamiWithRuntime(t, "--ignore-config", "cloud-init", "group", "render", "--template", "missing")
	if res.err == nil || res.exitCode != cli.CodeUsage {
		t.Fatalf("error = %v, exit code = %d, want %d (%s)", res.err, res.exitCode, cli.CodeUsage, cli.CodeName(cli.CodeUsage))
	}
}

// TestCloudInitGroupRender_LegacyInputs verifies the original command rejects
// new source-selection flags and missing or blank arguments as CodeUsage.
func TestCloudInitGroupRender_LegacyInputs(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		args []string
	}{
		{name: "template flag", args: []string{"compute", "node01", "--template", "missing"}},
		{name: "metadata flag", args: []string{"compute", "node01", "--metadata", "missing"}},
		{name: "metadata stdin flag", args: []string{"compute", "node01", "--metadata", "-"}},
		{name: "group flag", args: []string{"compute", "node01", "--group", "compute"}},
		{name: "node flag", args: []string{"compute", "node01", "--node", "node01"}},
		{name: "no arguments"},
		{name: "missing node", args: []string{"compute"}},
		{name: "empty node", args: []string{"compute", ""}},
		{name: "blank node", args: []string{"compute", " "}},
		{name: "empty group", args: []string{"", "node01"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			args := append([]string{"--ignore-config", "cloud-init", "group", "render"}, tc.args...)
			res := runOchamiWithRuntime(t, args...)
			if res.err == nil || res.exitCode != cli.CodeUsage {
				t.Fatalf("error = %v, exit code = %d, want %d (%s)", res.err, res.exitCode, cli.CodeUsage, cli.CodeName(cli.CodeUsage))
			}
		})
	}
}
