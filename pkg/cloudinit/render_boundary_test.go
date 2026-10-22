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

// TestRender_Values verifies ordering, scalar types, overrides, and the fixed
// whitespace and undefined profile through the public API.
func TestRender_Values(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, template, metadata, extra, want string
		inputFormat                           format.DataFormat
	}{
		{name: "nested JSON order", template: `{% for k,v in ds.meta_data.items() %}{{k}}:{% for x in v %}{{x}};{% endfor %}|{% endfor %}`, metadata: `{"z":{"b":1,"a":2},"a":{"z":1}}`, want: "z:b;a;|a:z;|"},
		{name: "JSON duplicate position", template: `{% for k,v in ds.meta_data.items() %}{{k}}={{v}};{% endfor %}`, metadata: `{"z":1,"a":2,"z":3}`, want: "z=3;a=2;"},
		{name: "numeric types", template: `{{ds.meta_data.i}}|{{ds.meta_data.f}}|{{ds.meta_data.i is integer}}|{{ds.meta_data.f is float}}|{{ds.meta_data.e is float}}`, metadata: `{"i":9007199254740993,"f":1.0,"e":1e2}`, want: "9007199254740993|1.0|True|True|True"},
		{name: "YAML values", template: `{{ ds.meta_data.s }}|{{ds.meta_data.n is none}}|{{ds.meta_data.b}}|{{ds.meta_data.i}}|{{ds.meta_data.f}}`, metadata: "s: café\nn: null\nb: true\ni: 42\nf: 1.0\n", inputFormat: format.DataFormatYaml, want: "café|True|True|42|1.0"},
		{name: "YAML quoted identifier", template: `{{ds.meta_data.x}}`, metadata: "x: '18446744073709551616'", inputFormat: format.DataFormatYaml, want: "18446744073709551616"},
		{name: "YAML explicit string identifier", template: `{{ds.meta_data.x}}`, metadata: "x: !!str 18446744073709551616", inputFormat: format.DataFormatYaml, want: "18446744073709551616"},
		{name: "YAML explicit float", template: `{{ds.meta_data.x}}`, metadata: "x: !!float 1", inputFormat: format.DataFormatYaml, want: "1.0"},
		{name: "YAML legacy decimal", template: `{{ds.meta_data.x}}`, metadata: "x: 09", inputFormat: format.DataFormatYaml, want: "9.0"},
		{name: "YAML merge order", template: `{% for k,v in ds.meta_data.result.items() %}{{k}}={{v}};{% endfor %}`, metadata: "base: &base {z: 1, a: 2}\nresult:\n  before: 0\n  <<: *base\n  z: 9\n  after: 3\n", inputFormat: format.DataFormatYaml, want: "before=0;z=9;a=2;after=3;"},
		{name: "YAML merge sequence", template: `{% for k,v in ds.meta_data.result.items() %}{{k}}={{v}};{% endfor %}`, metadata: "result: {a: 9, <<: [{z: 1, a: 2}, {z: 3, b: 4}]}\n", inputFormat: format.DataFormatYaml, want: "a=9;z=1;b=4;"},
		{name: "YAML sequence alias", template: `{{ds.meta_data.b|join(',')}}`, metadata: "a: &a [z, a]\nb: *a\n", inputFormat: format.DataFormatYaml, want: "z,a"},
		{name: "extra variables replace ds", template: `{{ds.meta_data.hostname}}|{{ds.other|default('missing')}}`, metadata: `{"hostname":"original"}`, extra: `{"ds":{"meta_data":{"hostname":"override"}}}`, want: "override|missing"},
		{name: "extra variables YAML order", template: `{% for k in group %}{{k}};{% endfor %}`, extra: "group: {z: 1, a: 2}\n", inputFormat: format.DataFormatYaml, want: "z;a;"},
		{name: "empty documents", template: `{{ ds.meta_data|length }}|{{missing|default('fallback')}}`, want: "0|fallback"},
		{name: "null documents", template: `{{ds.meta_data|length}}`, metadata: "null", extra: "null", want: "0"},
		{name: "pretty JSON", template: `{{ds.meta_data.x}}`, metadata: "{\n  \"x\": 1\n}", inputFormat: format.DataFormatJsonPretty, want: "1"},
		{name: "autoescape disabled", template: `{{ds.meta_data.text}}`, metadata: `{"text":"<&\"'é>"}`, want: "<&\"'é>"},
		{name: "default undefined", template: `a{{missing}}b{% if missing %}bad{% endif %}`, want: "ab"},
		{name: "whitespace", template: "a\r\n  {% if true %}\r\nb\r\n  {% endif %}\r\n", want: "a\n  \nb\n  "},
		{name: "empty template"},
		{name: "namespace mutation", template: `{% set ns=namespace(xs=[]) %}{% for i in range(3) %}{% do ns.xs.extend([i]) %}{% endfor %}{{ns.xs|join(',')}}`, want: "0,1,2"},
		{name: "loop controls", template: `{% for i in range(4) %}{% if i == 1 %}{% continue %}{% endif %}{% if i == 3 %}{% break %}{% endif %}{{i}}{% endfor %}`, want: "02"},
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
		{name: "YAML duplicate key", data: "a: 1\na: 2", format: format.DataFormatYaml},
		{name: "YAML nested duplicate key", data: "x: {a: 1, a: 2}", format: format.DataFormatYaml},
		{name: "YAML cyclic alias", data: "x: &x [*x]", format: format.DataFormatYaml},
		{name: "YAML unknown alias", data: "x: *missing", format: format.DataFormatYaml},
		{name: "YAML numeric key", data: "1: x", format: format.DataFormatYaml},
		{name: "YAML sequence key", data: "? [a, b]\n: x", format: format.DataFormatYaml},
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
	t.Run("execute", func(t *testing.T) {
		t.Parallel()
		assertRenderError(t, cloudinit.RenderRequest{Template: []byte("prefix{% include 'missing' %}")}, cloudinit.StageExecute)
	})
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
