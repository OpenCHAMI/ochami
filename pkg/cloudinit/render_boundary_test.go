// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cloudinit_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/openchami/ochami/pkg/cloudinit"
	"github.com/openchami/ochami/pkg/format"
)

// TestRender_Values verifies cloud-init's variable construction (sorted keys,
// operator aliases, version key promotion), scalar types, overrides, and its
// header, whitespace, sandbox, and undefined profile through the public API.
func TestRender_Values(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, template, metadata, extra, want string
		inputFormat                           format.DataFormat
	}{
		{name: "nested mappings sorted", template: `{% for k,v in ds.meta_data.items() %}{{k}}:{% for x in v %}{{x}};{% endfor %}|{% endfor %}`, metadata: `{"z":{"b":1,"a":2},"a":{"z":1}}`, want: "a:z;|z:a;b;|"},
		{name: "mappings in lists sorted", template: `{% for d in ds.meta_data.l %}{% for k in d %}{{k}};{% endfor %}{% endfor %}`, metadata: `{"l":[{"z":1,"a":2}]}`, want: "a;z;"},
		{name: "JSON duplicate last value", template: `{% for k,v in ds.meta_data.items() %}{{k}}={{v}};{% endfor %}`, metadata: `{"z":1,"a":2,"z":3}`, want: "a=2;z=3;"},
		{name: "operator aliases", template: `{{ds.meta_data.local_hostname}}|{{ds.meta_data.v1_0}}|{% for k in ds.meta_data %}{{k}},{% endfor %}`, metadata: `{"local-hostname":"n1","v1.0":"x","a":1}`, want: "n1|x|a,local-hostname,local_hostname,v1.0,v1_0,"},
		{name: "version keys promoted", template: `{{local_hostname}}|{{cloud_name}}|{{v1.cloud_name}}`, extra: `{"v1":{"local-hostname":"n1","cloud_name":"nocloud"}}`, want: "n1|nocloud|nocloud"},
		{name: "numeric types", template: `{{ds.meta_data.i}}|{{ds.meta_data.f}}|{{ds.meta_data.i is integer}}|{{ds.meta_data.f is float}}|{{ds.meta_data.e is float}}`, metadata: `{"i":9007199254740993,"f":1.0,"e":1e2}`, want: "9007199254740993|1.0|True|True|True"},
		{name: "YAML values", template: `{{ ds.meta_data.s }}|{{ds.meta_data.n is none}}|{{ds.meta_data.b}}|{{ds.meta_data.i}}|{{ds.meta_data.f}}`, metadata: "s: café\nn: null\nb: true\ni: 42\nf: 1.0\n", inputFormat: format.DataFormatYaml, want: "café|True|True|42|1.0"},
		{name: "YAML quoted identifier", template: `{{ds.meta_data.x}}`, metadata: "x: '18446744073709551616'", inputFormat: format.DataFormatYaml, want: "18446744073709551616"},
		{name: "YAML explicit string identifier", template: `{{ds.meta_data.x}}`, metadata: "x: !!str 18446744073709551616", inputFormat: format.DataFormatYaml, want: "18446744073709551616"},
		{name: "YAML explicit float", template: `{{ds.meta_data.x}}`, metadata: "x: !!float 1", inputFormat: format.DataFormatYaml, want: "1.0"},
		{name: "YAML leading zero string", template: `{{ds.meta_data.x}}`, metadata: "x: 09", inputFormat: format.DataFormatYaml, want: "09"},
		{name: "YAML 1.1 values", template: `{{ds.meta_data.flag}}|{{ds.meta_data.mode}}|{{ds.meta_data.sex}}|{{ds.meta_data.date}}`, metadata: "flag: on\nmode: 0644\nsex: 1:20\ndate: 2001-12-14\n", inputFormat: format.DataFormatYaml, want: "True|420|80|Warning: redacted unserializable type <class 'datetime.date'>"},
		{name: "YAML binary", template: `{{ds.meta_data.b}}`, metadata: "b: !!binary aGk=\n", inputFormat: format.DataFormatYaml, want: "ci-b64:aGk="},
		{name: "YAML duplicate key", template: `{{ds.meta_data.a}}`, metadata: "a: 1\na: 2\n", inputFormat: format.DataFormatYaml, want: "2"},
		{name: "YAML scalar keys", template: `{{ds.meta_data["1"]}}|{{ds.meta_data["true"]}}|{{ds.meta_data["null"]}}`, metadata: "1: x\nyes: y\n~: z\n", inputFormat: format.DataFormatYaml, want: "x|y|z"},
		{name: "YAML merge values", template: `{% for k,v in ds.meta_data.result.items() %}{{k}}={{v}};{% endfor %}`, metadata: "base: &base {z: 1, a: 2}\nresult:\n  before: 0\n  <<: *base\n  z: 9\n  after: 3\n", inputFormat: format.DataFormatYaml, want: "a=2;after=3;before=0;z=9;"},
		{name: "YAML merge sequence", template: `{% for k,v in ds.meta_data.result.items() %}{{k}}={{v}};{% endfor %}`, metadata: "result: {a: 9, <<: [{z: 1, a: 2}, {z: 3, b: 4}]}\n", inputFormat: format.DataFormatYaml, want: "a=9;b=4;z=1;"},
		{name: "YAML sequence alias", template: `{{ds.meta_data.b|join(',')}}`, metadata: "a: &a [z, a]\nb: *a\n", inputFormat: format.DataFormatYaml, want: "z,a"},
		{name: "extra variables replace ds", template: `{{ds.meta_data.hostname}}|{{ds.other|default('missing')}}`, metadata: `{"hostname":"original"}`, extra: `{"ds":{"meta_data":{"hostname":"override"}}}`, want: "override|missing"},
		{name: "extra variables sorted", template: `{% for k in group %}{{k}};{% endfor %}`, extra: "group: {z: 1, a: 2}\n", inputFormat: format.DataFormatYaml, want: "a;z;"},
		{name: "empty documents", template: `{{ ds.meta_data|length }}|{{missing|default('fallback')}}`, want: "0|fallback"},
		{name: "null documents", template: `{{ds.meta_data|length}}`, metadata: "null", extra: "null", want: "0"},
		{name: "pretty JSON", template: `{{ds.meta_data.x}}`, metadata: "{\n  \"x\": 1\n}", inputFormat: format.DataFormatJsonPretty, want: "1"},
		{name: "autoescape disabled", template: `{{ds.meta_data.text}}`, metadata: `{"text":"<&\"'é>"}`, want: "<&\"'é>"},
		{name: "missing variable text", template: `a{{missing}}b{{ds.meta_data.key}}{% if missing %}bad{% endif %}`, want: "aCI_MISSING_JINJA_VAR/missingbCI_MISSING_JINJA_VAR/key"},
		{name: "missing variable concatenated", template: `{{'a' ~ missing ~ 'b'}}|{{missing|default('d')}}|{{missing|length}}`, want: "aCI_MISSING_JINJA_VAR/missingb|d|0"},
		{name: "sandboxed attribute", template: `{{ ''.__class__ }}`, want: "CI_MISSING_JINJA_VAR/__class__"},
		{name: "whitespace", template: "a\r\n  {% if true %}\r\nb\r\n  {% endif %}\r\n", want: "a\n  b\n  \n"},
		{name: "final newline restored", template: "{% if true %}a\n{% endif %}\n", want: "a\n\n"},
		{name: "jinja header removed", template: "## Template: Jinja \n#cloud-config\nx: {{ 1 }}\n", want: "#cloud-config\nx: 1\n"},
		{name: "header without body", template: "## template: jinja"},
		{name: "empty template"},
		{name: "namespace mutation", template: `{% set ns=namespace(xs=[]) %}{% for i in range(3) %}{% do ns.xs.extend([i]) %}{% endfor %}{{ns.xs|join(',')}}`, want: "0,1,2"},
		{name: "loop control text", template: `{% raw %}{% break %}{% endraw %}`, want: "{% break %}"},
		{name: "no host environment", template: `{{env is undefined}}`, want: "True"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := cloudinit.RenderRequest{
				Template:  []byte(tc.template),
				Metadata:  cloudinit.Document{Data: []byte(tc.metadata), Format: tc.inputFormat},
				ExtraVars: cloudinit.Document{Data: []byte(tc.extra), Format: tc.inputFormat},
			}
			out, err := cloudinit.Render(t.Context(), req)
			if err != nil {
				t.Fatal(err)
			}
			if out != tc.want {
				t.Fatalf("output = %q, want %q", out, tc.want)
			}
			if string(req.Template) != tc.template || string(req.Metadata.Data) != tc.metadata || string(req.ExtraVars.Data) != tc.extra {
				t.Fatal("Render modified caller input")
			}
		})
	}
}

// TestRender_InvalidInputs verifies structured stages and empty results for
// malformed input documents, compilation failures, and execution failures.
func TestRender_InvalidInputs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, data string
		format     format.DataFormat
	}{
		{name: "JSON trailing object", data: "{}{}"},
		{name: "JSON trailing garbage", data: "{} trailing"},
		{name: "JSON malformed", data: "{"},
		{name: "JSON array", data: "[]"},
		{name: "JSON string", data: `"x"`},
		{name: "JSON oversized integer", data: `{"x":9223372036854775808}`},
		{name: "JSON negative oversized integer", data: `{"x":-9223372036854775809}`},
		{name: "JSON oversized float", data: `{"x":1e9999}`},
		{name: "YAML array", data: "[]", format: format.DataFormatYaml},
		{name: "YAML malformed", data: "x: [", format: format.DataFormatYaml},
		{name: "YAML multiple documents", data: "{}\n---\n{}", format: format.DataFormatYaml},
		{name: "YAML cyclic alias", data: "x: &x [*x]", format: format.DataFormatYaml},
		{name: "YAML unknown alias", data: "x: *missing", format: format.DataFormatYaml},
		{name: "YAML sequence key", data: "? [a, b]\n: x", format: format.DataFormatYaml},
		{name: "YAML tab separator", data: "x:\ta", format: format.DataFormatYaml},
		{name: "YAML value indicator", data: "x: =", format: format.DataFormatYaml},
		{name: "YAML invalid date", data: "x: 2001-02-30", format: format.DataFormatYaml},
		{name: "YAML custom scalar tag", data: "x: !custom abc", format: format.DataFormatYaml},
		{name: "YAML custom mapping tag", data: "x: !custom {}", format: format.DataFormatYaml},
		{name: "YAML custom sequence tag", data: "x: !custom []", format: format.DataFormatYaml},
		{name: "YAML invalid merge", data: "x: {<<: 1}", format: format.DataFormatYaml},
		{name: "YAML oversized integer", data: "x: 9223372036854775808", format: format.DataFormatYaml},
		{name: "YAML integer beyond uint64", data: "x: 18446744073709551616", format: format.DataFormatYaml},
		{name: "YAML integer beyond float64", data: "x: " + strings.Repeat("9", 400), format: format.DataFormatYaml},
		{name: "unsupported format", data: "{}", format: "toml"},
	}
	for _, tc := range tests {
		for _, stage := range []cloudinit.RenderStage{cloudinit.StageMetadata, cloudinit.StageExtraVars} {
			t.Run(tc.name+"/"+string(stage), func(t *testing.T) {
				t.Parallel()
				req := cloudinit.RenderRequest{Template: []byte("prefix")}
				doc := cloudinit.Document{Data: []byte(tc.data), Format: tc.format}
				if stage == cloudinit.StageMetadata {
					req.Metadata = doc
				} else {
					req.ExtraVars = doc
				}
				assertRenderError(t, req, stage)
			})
		}
	}
	t.Run("compile", func(t *testing.T) {
		t.Parallel()
		assertRenderError(t, cloudinit.RenderRequest{Template: []byte("prefix{% if")}, cloudinit.StageCompile)
	})
	for name, template := range map[string]string{
		"break":        "{% for i in range(2) %}{% break %}{% endfor %}",
		"continue":     "{% for i in range(2) %}{% continue %}{% endfor %}",
		"basic header": "## template: basic\n$x",
	} {
		t.Run("compile/"+name, func(t *testing.T) {
			t.Parallel()
			assertRenderError(t, cloudinit.RenderRequest{Template: []byte(template)}, cloudinit.StageCompile)
		})
	}
	for name, template := range map[string]string{
		"include":               "prefix{% include 'missing' %}",
		"undefined attribute":   "{{ missing.attr }}",
		"undefined arithmetic":  "{{ missing + 1 }}",
		"sandboxed chain":       "{{ ''.__class__.__mro__ }}",
		"transformed undefined": "{{ missing|upper }}",
	} {
		t.Run("execute/"+name, func(t *testing.T) {
			t.Parallel()
			assertRenderError(t, cloudinit.RenderRequest{Template: []byte(template)}, cloudinit.StageExecute)
		})
	}
}

// TestRender_HeaderLineNumbers verifies syntax errors report lines of the
// original template after its header line is removed, as cloud-init does.
func TestRender_HeaderLineNumbers(t *testing.T) {
	t.Parallel()
	for name, template := range map[string]string{
		"syntax": "## template: jinja\na\n{% if %}\n",
		"break":  "## template: jinja\na\n{% for i in x %}{% break %}{% endfor %}\n",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := cloudinit.Render(t.Context(), cloudinit.RenderRequest{Template: []byte(template)})
			if err == nil || !strings.Contains(err.Error(), "line 3") {
				t.Fatalf("error = %v, want a syntax error on line 3", err)
			}
		})
	}
}

// TestHasJinjaHeader verifies which parts cloud-init classifies as Jinja
// templates.
func TestHasJinjaHeader(t *testing.T) {
	t.Parallel()
	for template, want := range map[string]bool{
		"## template: jinja\n#cloud-config\n": true,
		"## Template: JINJA\n":                true,
		"\n  ## template: jinja\n":            true,
		"#template: jinja\n#cloud-config\n":   false,
		"##template: jinja\n":                 false,
		"#cloud-config\n## template: jinja\n": false,
		"":                                    false,
	} {
		if got := cloudinit.HasJinjaHeader([]byte(template)); got != want {
			t.Errorf("HasJinjaHeader(%q) = %t, want %t", template, got, want)
		}
	}
}

func assertRenderError(t *testing.T, req cloudinit.RenderRequest, stage cloudinit.RenderStage) {
	t.Helper()
	out, err := cloudinit.Render(t.Context(), req)
	var renderErr *cloudinit.RenderError
	if !errors.As(err, &renderErr) || renderErr.Stage != stage || renderErr.Err == nil {
		t.Fatalf("error = %v, want RenderError stage %s with a cause", err, stage)
	}
	if out != "" || !errors.Is(renderErr, renderErr.Err) || renderErr.Error() == "" {
		t.Fatalf("output = %q, error = %v; want empty output and wrapped error", out, err)
	}
}

// TestRender_Cancellation verifies cancellation before and during execution
// returns no output and preserves the context error for errors.Is.
func TestRender_Cancellation(t *testing.T) {
	t.Parallel()
	for _, during := range []bool{false, true} {
		t.Run(fmt.Sprintf("during=%t", during), func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if during {
				go func() {
					select {
					case <-time.After(10 * time.Millisecond):
						cancel()
					case <-ctx.Done():
					}
				}()
			} else {
				cancel()
			}
			out, err := cloudinit.Render(ctx, cloudinit.RenderRequest{
				Template: []byte("prefix{% for i in range(100000) %}{% for j in range(100000) %}x{% endfor %}{% endfor %}"),
			})
			var renderErr *cloudinit.RenderError
			if out != "" || !errors.Is(err, context.Canceled) || !errors.As(err, &renderErr) || renderErr.Stage != cloudinit.StageExecute {
				t.Fatalf("output length = %d, error = %v, want empty output and wrapped cancellation", len(out), err)
			}
		})
	}
}

// TestRender_ConcurrentCalls verifies calls sharing request bytes get independent
// mutable template variables and preserve their caller's documents.
func TestRender_ConcurrentCalls(t *testing.T) {
	t.Parallel()
	req := cloudinit.RenderRequest{
		Template: []byte(`{% do ds.meta_data.update({'x': ds.meta_data.x + 1}) %}{{ds.meta_data.x}}`),
		Metadata: cloudinit.Document{Data: []byte(`{"x":1}`)},
	}
	var wg sync.WaitGroup
	for range 16 {
		wg.Go(func() {
			out, err := cloudinit.Render(t.Context(), req)
			if err != nil || out != "2" {
				t.Errorf("output = %q, error = %v, want 2", out, err)
			}
		})
	}
	wg.Wait()
	if string(req.Metadata.Data) != `{"x":1}` {
		t.Fatal("Render modified shared metadata")
	}
}
