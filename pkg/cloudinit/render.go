// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cloudinit

import (
	"context"
	"fmt"

	gj "github.com/jryberg/gojinja"
	"github.com/jryberg/gojinja/pkg/lexer"
	"github.com/jryberg/gojinja/pkg/runtime"

	"github.com/openchami/ochami/pkg/format"
)

// Document is an in-memory JSON or YAML mapping. Empty data and null represent
// an empty mapping. An empty Format defaults to JSON; json-pretty also accepts JSON.
type Document struct {
	Data   []byte
	Format format.DataFormat
}

// RenderRequest contains a template and the documents used to construct its
// variables. Metadata becomes ds.meta_data. ExtraVars replaces top-level values,
// including ds when supplied, without recursively merging mappings.
type RenderRequest struct {
	Template  []byte
	Metadata  Document
	ExtraVars Document
}

// RenderStage identifies the operation that failed during rendering.
type RenderStage string

const (
	// StageMetadata identifies metadata decoding errors.
	StageMetadata RenderStage = "metadata"
	// StageExtraVars identifies extra-variable decoding errors.
	StageExtraVars RenderStage = "extra-vars"
	// StageInitialize identifies engine initialization errors.
	StageInitialize RenderStage = "initialize"
	// StageCompile identifies template compilation errors.
	StageCompile RenderStage = "compile"
	// StageExecute identifies template execution errors, including cancellation.
	StageExecute RenderStage = "execute"
)

// RenderError describes a failed rendering operation and retains its cause.
type RenderError struct {
	Stage RenderStage
	Err   error
}

// Error describes the rendering stage and its underlying error.
func (e *RenderError) Error() string {
	return fmt.Sprintf("cloud-init rendering (%s): %v", e.Stage, e.Err)
}

// Unwrap returns the underlying error for errors.Is and errors.As.
func (e *RenderError) Unwrap() error { return e.Err }

// Render renders a cloud-init template using the documented Python Jinja2
// reference profile. It preserves input mapping order and never modifies req.
// Independent calls may run concurrently. On error the returned string is empty.
// A nil context is invalid, as with other context-taking Go APIs.
func Render(ctx context.Context, req RenderRequest) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", &RenderError{Stage: StageExecute, Err: err}
	}
	metadata, err := decodeDocument(req.Metadata)
	if err != nil {
		return "", &RenderError{Stage: StageMetadata, Err: err}
	}
	extra, err := decodeDocument(req.ExtraVars)
	if err != nil {
		return "", &RenderError{Stage: StageExtraVars, Err: err}
	}
	ds := gj.NewOrderedDict()
	ds.Set("meta_data", metadata)
	vars := gj.NewOrderedDict()
	vars.Set("ds", ds)
	for _, key := range extra.Keys() {
		value, _ := extra.Get(key)
		vars.Set(key, value)
	}

	whitespace := lexer.DefaultOptions()
	whitespace.TrimBlocks = false
	whitespace.LstripBlocks = false
	whitespace.KeepTrailingNewline = false
	whitespace.NewlineSequence = "\n"
	env, err := gj.New(gj.WithUnsafe(), gj.WithAutoescape(gj.AutoescapeNever{}),
		gj.WithUndefined(runtime.NewBase), gj.WithLexerOptions(whitespace))
	if err != nil {
		return "", &RenderError{Stage: StageInitialize, Err: err}
	}
	tpl, err := env.FromString(string(req.Template))
	if err != nil {
		return "", &RenderError{Stage: StageCompile, Err: err}
	}
	out, err := tpl.RenderContext(ctx, vars)
	if err != nil {
		return "", &RenderError{Stage: StageExecute, Err: err}
	}
	if err := ctx.Err(); err != nil {
		return "", &RenderError{Stage: StageExecute, Err: err}
	}
	return out, nil
}
