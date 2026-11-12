// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/openchami/ochami/internal/cli"
)

func writeRenderInput(t *testing.T, name, data string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestCloudInitRender_LocalReference verifies all cloud-init fixtures match
// exactly when templates and JSON or YAML metadata come from local files.
func TestCloudInitRender_LocalReference(t *testing.T) {
	t.Parallel()
	root := filepath.Join("..", "pkg", "cloudinit", "testdata")
	read := func(name string) []byte {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	var manifest struct {
		Fixtures []struct {
			Name, Template, Metadata, Expected string
		} `json:"fixtures"`
	}
	if err := json.Unmarshal(read("reference.json"), &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Fixtures) != 5 {
		t.Fatalf("fixture count = %d, want 5", len(manifest.Fixtures))
	}
	for _, fixture := range manifest.Fixtures {
		for _, metadataFormat := range []string{"json", "yaml"} {
			t.Run(fixture.Name+"/"+metadataFormat, func(t *testing.T) {
				t.Parallel()
				metadataPath := filepath.Join(root, fixture.Metadata)
				if metadataFormat == "yaml" {
					var node yaml.Node
					if err := yaml.Unmarshal(read(fixture.Metadata), &node); err != nil {
						t.Fatal(err)
					}
					data, err := yaml.Marshal(&node)
					if err != nil {
						t.Fatal(err)
					}
					metadataPath = writeRenderInput(t, "metadata.yaml", string(data))
				}
				res := runOchamiWithRuntime(t, "--ignore-config", "cloud-init", "render",
					"--template", filepath.Join(root, fixture.Template), "--metadata", metadataPath)
				if res.err != nil {
					t.Fatal(res.err)
				}
				if want := string(read(fixture.Expected)); res.stdout != want {
					t.Fatalf("stdout = %q, want %q", res.stdout, want)
				}
			})
		}
	}
}

// TestCloudInitRender_LocalInputSources verifies omitted metadata, empty
// templates, variable replacement, and each source's exclusive use of stdin.
func TestCloudInitRender_LocalInputSources(t *testing.T) {
	t.Parallel()
	const template = "## template: jinja\n{{ds.meta_data.hostname|default('none')}}|{{label|default('none')}}\n"
	for _, tc := range []struct {
		name, stdinSource, metadata, extra, inputFormat, want string
		emptyTemplate                                         bool
	}{
		{name: "no metadata", want: "none|none\n"},
		{name: "empty template", emptyTemplate: true},
		{name: "template stdin", stdinSource: "template", metadata: "hostname: node01", want: "node01|none\n"},
		{name: "metadata stdin", stdinSource: "metadata", metadata: "hostname: node01", want: "node01|none\n"},
		{name: "extra vars stdin", stdinSource: "extra", extra: `{"label":"compute"}`, want: "none|compute\n"},
		{name: "JSON override", metadata: "hostname: node01", extra: `{"ds":{"meta_data":{"hostname":"replacement"}},"label":"compute"}`, want: "replacement|compute\n"},
		{name: "YAML override", metadata: `{"hostname":"node01"}`, extra: "ds: {meta_data: {hostname: replacement}}\nlabel: compute", inputFormat: "yaml", want: "replacement|compute\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			source := template
			if tc.emptyTemplate {
				source = ""
			}
			templatePath := writeRenderInput(t, "template.j2", source)
			stdin := ""
			if tc.stdinSource == "template" {
				templatePath, stdin = "-", source
			}
			args := []string{"--ignore-config", "cloud-init", "render", "--template", templatePath,
				"--uri", "http://127.0.0.1:1"}
			if tc.metadata != "" {
				path := writeRenderInput(t, "metadata", tc.metadata)
				if tc.stdinSource == "metadata" {
					path, stdin = "-", tc.metadata
				}
				args = append(args, "--metadata", path)
			}
			if tc.extra != "" {
				value := tc.extra
				if tc.stdinSource == "extra" {
					value, stdin = "@-", tc.extra
				}
				args = append(args, "--extra-vars", value)
			}
			if tc.inputFormat != "" {
				args = append(args, "--format-input", tc.inputFormat)
			}
			res := runOchamiWithInputAndRuntime(t, stdin, args...)
			if res.err != nil || res.stdout != tc.want {
				t.Fatalf("stdout = %q, error = %v, want %q without error", res.stdout, res.err, tc.want)
			}
		})
	}
}

// TestCloudInitRender_NodeWarnings verifies warnings for a template cloud-init
// would use verbatim and for variables it would print as missing.
func TestCloudInitRender_NodeWarnings(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, template, want, warning string }{
		{
			name:     "missing header",
			template: "#cloud-config\nx: {{ 1 }}\n",
			want:     "#cloud-config\nx: 1\n",
			warning:  "does not begin with '## template: jinja'",
		},
		{
			name:     "undefined variables",
			template: "## template: jinja\n{{ a }} {{ ds.meta_data.b }} {{ a }}\n",
			want:     "CI_MISSING_JINJA_VAR/a CI_MISSING_JINJA_VAR/b CI_MISSING_JINJA_VAR/a\n",
			warning:  "prints as CI_MISSING_JINJA_VAR/<name>: 'a', 'b'",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			path := writeRenderInput(t, "template.j2", tc.template)
			res := runOchamiWithRuntime(t, "--ignore-config", "cloud-init", "render", "--template", path)
			if res.err != nil || !strings.Contains(res.stdout, tc.warning) || !strings.HasSuffix(res.stdout, tc.want) {
				t.Fatalf("output = %q, error = %v, want warning %q and output %q", res.stdout, res.err, tc.warning, tc.want)
			}
		})
	}
}

// TestCloudInitRender_HybridMetadata verifies a local template fetches
// only node metadata and sends the configured authorization token.
func TestCloudInitRender_HybridMetadata(t *testing.T) {
	t.Parallel()
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.Method != http.MethodGet || r.URL.Path != "/admin/impersonation/node01/meta-data" {
			t.Errorf("request = %s %s, want GET /admin/impersonation/node01/meta-data", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("authorization = %q, want Bearer test-token", got)
		}
		io.WriteString(w, "hostname: node01\n")
	}))
	t.Cleanup(srv.Close)
	path := writeRenderInput(t, "template.j2", "## template: jinja\n{{ds.meta_data.hostname}}|{{role}}\n")
	res := runOchamiWithRuntime(t, "--ignore-config", "cloud-init", "render", "--template", path, "--node", "node01",
		"--uri", srv.URL, "--token", "test-token", "--extra-vars", `{"role":"compute"}`)
	if res.err != nil || res.stdout != "node01|compute\n" || requests.Load() != 1 {
		t.Fatalf("stdout = %q, error = %v, requests = %d, want node01|compute and one request", res.stdout, res.err, requests.Load())
	}
}

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
						io.WriteString(w, "## template: jinja\n{{ds.meta_data.hostname}}|{{role}}\n")
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
				stdin, want, wantMetadataRequests := "", "live-node|compute\n", int32(1)
				if metadata != "live" {
					path := writeRenderInput(t, "metadata.yaml", "hostname: replacement\n")
					if metadata == "stdin" {
						path, stdin = "-", "hostname: replacement\n"
					}
					args = append(args, "--metadata", path)
					want, wantMetadataRequests = "replacement|compute\n", 0
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

// TestCloudInitRender_StoredGroupSources verifies plain and base64 group content
// uses only the authenticated raw-group endpoint, without inferring a node or
// merging stored group metadata into the supplied or empty rendering context.
func TestCloudInitRender_StoredGroupSources(t *testing.T) {
	t.Parallel()
	for _, encoding := range []string{"plain", "base64"} {
		for _, metadata := range []string{"empty", "yaml", "json", "stdin", "extra-vars"} {
			t.Run(encoding+"/"+metadata, func(t *testing.T) {
				t.Parallel()
				content := "## template: jinja\n{{ds.meta_data.hostname|default('none')}}|{{ds.meta_data['instance-id']|default('none')}}|{{role}}\n"
				if encoding == "base64" {
					content = base64.StdEncoding.EncodeToString([]byte(content))
				}
				var requests atomic.Int32
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					requests.Add(1)
					if r.Method != http.MethodGet || r.URL.Path != "/admin/groups/compute" {
						t.Errorf("request = %s %s, want GET /admin/groups/compute", r.Method, r.URL.Path)
						http.NotFound(w, r)
						return
					}
					if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
						t.Errorf("authorization = %q, want Bearer test-token", got)
					}
					writeJSONResponse(t, w, map[string]any{
						"name":      "compute",
						"meta-data": map[string]any{"hostname": "stored-value", "instance-id": "stored-node"},
						"file":      map[string]any{"content": content, "encoding": encoding},
					})
				}))
				t.Cleanup(srv.Close)
				args := []string{"--ignore-config", "cloud-init", "render", "--group", "compute", "--uri", srv.URL, "--token", "test-token"}
				stdin, want := "", "none|none|compute\n"
				extra := `{"role":"compute"}`
				if metadata == "extra-vars" {
					extra = `{"role":"compute","ds":{"meta_data":{"hostname":"extra-host","instance-id":"extra-node"}}}`
					want = "extra-host|extra-node|compute\n"
				} else if metadata != "empty" {
					data := "hostname: file-host\ninstance-id: file-node\n"
					if metadata == "json" {
						data = `{"hostname":"file-host","instance-id":"file-node"}`
					}
					path := writeRenderInput(t, "metadata", data)
					if metadata == "stdin" {
						path, stdin = "-", data
					}
					args = append(args, "--metadata", path)
					want = "file-host|file-node|compute\n"
				}
				args = append(args, "--extra-vars", extra)
				res := runOchamiWithInputAndRuntime(t, stdin, args...)
				if res.err != nil || res.stdout != want || requests.Load() != 1 {
					t.Fatalf("stdout = %q, error = %v, requests = %d, want %q and one request", res.stdout, res.err, requests.Load(), want)
				}
			})
		}
	}
}

// TestCloudInitRender_StoredGroupReference verifies both storage encodings render
// the multi-group fixture exactly as the pinned cloud-init reference.
func TestCloudInitRender_StoredGroupReference(t *testing.T) {
	t.Parallel()
	read := func(name string) string {
		t.Helper()
		data, err := os.ReadFile(filepath.Join("..", "pkg", "cloudinit", "testdata", name))
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	for _, encoding := range []string{"plain", "base64"} {
		t.Run(encoding, func(t *testing.T) {
			t.Parallel()
			content := read("cloud-init-original.j2")
			if encoding == "base64" {
				content = base64.StdEncoding.EncodeToString([]byte(content))
			}
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/admin/groups/compute" {
					t.Errorf("unexpected request path %q", r.URL.Path)
					http.NotFound(w, r)
					return
				}
				writeJSONResponse(t, w, map[string]any{"name": "compute", "file": map[string]any{"content": content, "encoding": encoding}})
			}))
			t.Cleanup(srv.Close)
			res := runOchamiWithRuntime(t, "--ignore-config", "cloud-init", "render", "--group", "compute", "--uri", srv.URL, "--token", "t",
				"--metadata", filepath.Join("..", "pkg", "cloudinit", "testdata", "cloud-init-original.metadata.json"))
			if want := read("cloud-init-original.yaml"); res.err != nil || res.stdout != want {
				t.Fatalf("stdout = %q, error = %v, want %q without error", res.stdout, res.err, want)
			}
		})
	}
}

// TestCloudInitRender_EmptyStoredGroup verifies a group without template content
// succeeds with a warning and does not attempt to fetch node metadata.
func TestCloudInitRender_EmptyStoredGroup(t *testing.T) {
	t.Parallel()
	for _, body := range []string{`{"name":"compute"}`, `{"name":"compute","file":{"content":"","encoding":"plain"}}`} {
		t.Run(body, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/admin/groups/compute" {
					t.Errorf("unexpected request path %q", r.URL.Path)
				}
				io.WriteString(w, body)
			}))
			t.Cleanup(srv.Close)
			res := runOchamiWithRuntime(t, "--ignore-config", "cloud-init", "render", "--group", "compute", "--uri", srv.URL, "--token", "t")
			if res.err != nil || !strings.Contains(res.stdout, "stored cloud-config for group compute was empty") {
				t.Fatalf("output = %q, error = %v, want empty-template warning without error", res.stdout, res.err)
			}
		})
	}
}

// fullRenderServer serves a node's data and group templates through the
// impersonation API and counts requests by path.
func fullRenderServer(t *testing.T, counts map[string]*atomic.Int32) *httptest.Server {
	t.Helper()
	responses := map[string]string{
		"/admin/impersonation/node01/vendor-data":  "#include\nhttp://ci.example/cloud-init/base.yaml\nhttp://ci.example/cloud-init/compute.yaml\nhttp://ci.example/other.txt\n",
		"/admin/impersonation/node01/user-data":    "#cloud-config\n",
		"/admin/impersonation/node01/meta-data":    "instance-id: i-1\nlocal-hostname: node01.example\n",
		"/admin/impersonation/node01/base.yaml":    "#cloud-config\nruncmd: [base]\nntp: {enabled: true}\n",
		"/admin/impersonation/node01/compute.yaml": "## template: jinja\n#cloud-config\nhostname: {{ v1.local_hostname }}\nruncmd: [compute]\nrole: {{ role }}\n",
	}
	for path := range responses {
		counts[path] = &atomic.Int32{}
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := responses[r.URL.Path]
		if !ok {
			t.Errorf("unexpected request path %q", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		counts[r.URL.Path].Add(1)
		io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// TestCloudInitRender_Full verifies that --full fetches a node's data and
// includes through impersonation, merges the parts as cloud-init does, and
// reports includes it cannot resolve.
func TestCloudInitRender_Full(t *testing.T) {
	t.Parallel()
	counts := map[string]*atomic.Int32{}
	srv := fullRenderServer(t, counts)
	res := runOchamiWithRuntime(t, "--ignore-config", "cloud-init", "render", "--full", "--node", "node01",
		"--uri", srv.URL, "--token", "test-token", "--extra-vars", `{"role":"compute"}`)
	if res.err != nil {
		t.Fatalf("error = %v, output:\n%s", res.err, res.stdout)
	}
	for _, want := range []string{
		"# from 2 files\n",
		"# vendor-data part-002 (http://ci.example/cloud-init/compute.yaml)\n",
		"hostname: node01\n",
		"role: compute\n",
		"runcmd:\n    - compute\n",
		"ntp:\n    enabled: true\n",
		"http://ci.example/other.txt is unresolved",
	} {
		if !strings.Contains(res.stdout, want) {
			t.Errorf("output missing %q:\n%s", want, res.stdout)
		}
	}
	for path, count := range counts {
		if count.Load() != 1 {
			t.Errorf("%s requested %d times, want 1", path, count.Load())
		}
	}
}

// TestCloudInitRender_FullOverrides verifies that --user-data and
// --instance-data replace the node's user-data and metadata requests.
func TestCloudInitRender_FullOverrides(t *testing.T) {
	t.Parallel()
	counts := map[string]*atomic.Int32{}
	srv := fullRenderServer(t, counts)
	userData := writeRenderInput(t, "user-data", "#cloud-config\nhostname: from-user\nvendor_data: {disabled_handlers: []}\n")
	instanceData := writeRenderInput(t, "instance-data.json", `{"v1": {"local_hostname": "given"}, "role": "given-role"}`)
	res := runOchamiWithRuntime(t, "--ignore-config", "cloud-init", "render", "--full", "--node", "node01",
		"--uri", srv.URL, "--token", "test-token", "--user-data", userData, "--instance-data", instanceData)
	if res.err != nil {
		t.Fatalf("error = %v, output:\n%s", res.err, res.stdout)
	}
	if !strings.Contains(res.stdout, "hostname: from-user\n") || !strings.Contains(res.stdout, "role: given-role\n") {
		t.Fatalf("output does not use the replacements:\n%s", res.stdout)
	}
	if counts["/admin/impersonation/node01/user-data"].Load() != 0 || counts["/admin/impersonation/node01/meta-data"].Load() != 0 {
		t.Fatal("replaced data was requested")
	}
}

// TestCloudInitRender_BundleRoundTrip verifies that an exported bundle
// renders offline to the same output as the live render.
func TestCloudInitRender_BundleRoundTrip(t *testing.T) {
	t.Parallel()
	counts := map[string]*atomic.Int32{}
	srv := fullRenderServer(t, counts)
	dir := filepath.Join(t.TempDir(), "bundle")
	live := []string{"--ignore-config", "cloud-init", "render", "--full", "--node", "node01", "--uri", srv.URL, "--token", "test-token", "--extra-vars", `{"role":"compute"}`}
	want := runOchamiWithRuntime(t, live...)
	exported := runOchamiWithRuntime(t, append(live, "--export-bundle", dir, "--no-render")...)
	if exported.err != nil || exported.stdout != "" {
		t.Fatalf("export: error = %v, output = %q, want no output", exported.err, exported.stdout)
	}
	for _, name := range []string{"manifest.json", "instance-data.json", "vendor-data", "user-data", "ochami_render.py", "run.sh", "README"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("bundle file %s: %v", name, err)
		}
	}
	got := runOchamiWithRuntime(t, "--ignore-config", "cloud-init", "render", "--bundle", dir)
	if got.err != nil || got.stdout != want.stdout {
		t.Fatalf("bundle output = %q, error = %v, want %q", got.stdout, got.err, want.stdout)
	}
}

// renderFakeCloudInit is a cloud-init stand-in for devel render that prints
// the template without its header, followed by SUFFIX when set.
const renderFakeCloudInit = `#!/bin/sh
if [ "$1" = "--version" ]; then echo "cloud-init 99.1"; exit 0; fi
tail -n +2 template
[ -n "${SUFFIX:-}" ] && echo "$SUFFIX"
exit 0
`

func writeRenderFake(t *testing.T, suffix string) string {
	t.Helper()
	script := renderFakeCloudInit
	if suffix != "" {
		script = strings.Replace(script, "#!/bin/sh\n", "#!/bin/sh\nSUFFIX="+suffix+"\n", 1)
	}
	path := filepath.Join(t.TempDir(), "cloud-init")
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil { //nolint:gosec // the fake must be executable
		t.Fatal(err)
	}
	return path
}

// TestCloudInitRender_Engines verifies engine selection from flags, the
// environment, and configuration, with fallback only when cloud-init was not
// requested by a flag.
func TestCloudInitRender_Engines(t *testing.T) {
	t.Parallel()
	template := writeRenderInput(t, "template.j2", "## template: jinja\nhost: {{ 'internal' }}\n")
	fake := writeRenderFake(t, "")
	missing := filepath.Join(t.TempDir(), "missing")
	config := func(t *testing.T, path string, delegate bool) string {
		t.Helper()
		return writeTempConfig(t, "cloud-init:\n  path: "+path+"\n  delegate-render: "+strconv.FormatBool(delegate)+"\n")
	}
	env := func(values map[string]string) cli.Environment {
		return cli.EnvironmentFunc(func(key string) (string, bool) {
			v, ok := values[key]
			return v, ok
		})
	}
	internal := "host: internal\n"
	delegated := "host: {{ 'internal' }}\n"
	for _, tc := range []struct {
		name    string
		args    []string
		env     map[string]string
		want    string
		warning string
	}{
		{name: "default internal", args: []string{"--ignore-config"}, want: internal},
		{name: "flag selects cloud-init", args: []string{"--ignore-config", "--cloud-init", fake}, want: delegated},
		{name: "engine flag with environment path", args: []string{"--ignore-config", "--engine", "cloud-init"}, env: map[string]string{"OCHAMI_CLOUD_INIT_PATH": fake}, want: delegated},
		{name: "environment delegates", args: []string{"--ignore-config"}, env: map[string]string{"OCHAMI_CLOUD_INIT_PATH": fake, "OCHAMI_CLOUD_INIT_DELEGATE_RENDER": "true"}, want: delegated},
		{name: "configuration delegates", args: []string{"--config", config(t, fake, true)}, want: delegated},
		{name: "configuration path only", args: []string{"--config", config(t, fake, false)}, want: internal},
		{name: "engine flag overrides configuration", args: []string{"--config", config(t, fake, true), "--engine", "internal"}, want: internal},
		{name: "environment overrides configuration", args: []string{"--config", config(t, fake, true)}, env: map[string]string{"OCHAMI_CLOUD_INIT_DELEGATE_RENDER": "false"}, want: internal},
		{name: "configured cloud-init falls back", args: []string{"--config", config(t, missing, true)}, want: internal, warning: "rendering with ochami's internal engine"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			args := append(append([]string{}, tc.args...), "cloud-init", "render", "--template", template)
			res := runOchamiWithRuntimeEnv(t, env(tc.env), args...)
			if res.err != nil {
				t.Fatalf("error = %v, output:\n%s", res.err, res.stdout)
			}
			if !strings.HasSuffix(res.stdout, tc.want) || !strings.Contains(res.stdout, tc.warning) {
				t.Fatalf("output = %q, want %q with warning %q", res.stdout, tc.want, tc.warning)
			}
		})
	}
}

// TestCloudInitRender_Compare verifies that --compare prints cloud-init's
// output and fails with a diff when the engines differ.
func TestCloudInitRender_Compare(t *testing.T) {
	t.Parallel()
	template := writeRenderInput(t, "template.j2", "## template: jinja\nhost: node\n")
	res := runOchamiWithRuntime(t, "--ignore-config", "cloud-init", "render", "--template", template, "--compare", "--cloud-init", writeRenderFake(t, ""))
	if res.err != nil || res.stdout != "host: node\n" {
		t.Fatalf("matching: output = %q, error = %v", res.stdout, res.err)
	}
	res = runOchamiWithRuntime(t, "--ignore-config", "cloud-init", "render", "--template", template, "--compare", "--cloud-init", writeRenderFake(t, "extra"))
	if res.exitCode != cli.CodeGeneric {
		t.Fatalf("differing: exit code = %d, error = %v, want %d (%s)", res.exitCode, res.err, cli.CodeGeneric, cli.CodeName(cli.CodeGeneric))
	}
	for _, want := range []string{"host: node\nextra\n", "--- internal\n+++ cloud-init 99.1\n", "+extra\n"} {
		if !strings.Contains(res.stdout, want) {
			t.Errorf("differing output missing %q:\n%s", want, res.stdout)
		}
	}
	if !strings.Contains(res.err.Error(), "differs from cloud-init 99.1") {
		t.Errorf("error = %v, want it to name the difference", res.err)
	}
}

// TestCloudInitRender_CompatibilityIgnoresDelegation verifies that the legacy
// group render form always uses the internal engine.
func TestCloudInitRender_CompatibilityIgnoresDelegation(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/admin/impersonation/node01/compute.yaml":
			io.WriteString(w, "## template: jinja\nhost: {{ 'internal' }}\n")
		case "/admin/impersonation/node01/meta-data":
			io.WriteString(w, "{}\n")
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	cfg := writeTempConfig(t, "cloud-init:\n  path: /nonexistent/cloud-init\n  delegate-render: true\n")
	res := runOchamiWithRuntime(t, "--config", cfg, "cloud-init", "group", "render", "compute", "node01", "--uri", srv.URL, "--token", "test-token")
	if res.err != nil || res.stdout != "host: internal\n" {
		t.Fatalf("output = %q, error = %v, want internal rendering without warnings", res.stdout, res.err)
	}
}
