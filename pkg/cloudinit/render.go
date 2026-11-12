// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cloudinit

import (
	"context"
	"fmt"
	"strings"

	gj "github.com/jryberg/gojinja"

	"github.com/openchami/ochami/pkg/format"
)

// Document is an in-memory JSON or YAML mapping. Empty data and null represent
// an empty mapping. An empty Format defaults to JSON; json-pretty also accepts JSON.
type Document struct {
	Data   []byte
	Format format.DataFormat
}

// RenderRequest contains a template and the documents used to construct its
// variables.
//
// Metadata becomes ds.meta_data, and the standardized v1 keys that cloud-init's
// NoCloud datasource derives from metadata are added, such as v1.instance_id
// and v1.local_hostname. Keys that only a booted node knows, such as
// v1.distro, are absent.
//
// InstanceData, when its Data is nonempty, is used as the complete instance
// data instead, for example a node's instance-data-sensitive.json. It cannot be
// combined with Metadata.
//
// ExtraVars replaces top-level keys of the instance data, including ds and v1
// when supplied, without recursively merging mappings.
type RenderRequest struct {
	Template     []byte
	Metadata     Document
	InstanceData Document
	ExtraVars    Document
}

// RenderStage identifies the operation that failed during rendering.
type RenderStage string

const (
	// StageMetadata identifies metadata decoding errors.
	StageMetadata RenderStage = "metadata"
	// StageExtraVars identifies extra-variable decoding errors.
	StageExtraVars RenderStage = "extra-vars"
	// StageInstanceData identifies instance-data decoding errors.
	StageInstanceData RenderStage = "instance-data"
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

// Render renders a cloud-init template using cloud-init's Jinja profile. It
// never modifies req. Independent calls may run concurrently. On error the
// returned string is empty. A nil context is invalid, as with other
// context-taking Go APIs.
func Render(ctx context.Context, req RenderRequest) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", &RenderError{Stage: StageExecute, Err: err}
	}
	data, err := buildInstanceData(req)
	if err != nil {
		return "", err
	}
	template := string(req.Template)
	body, headed, err := templateBody(template)
	if err != nil {
		return "", &RenderError{Stage: StageCompile, Err: err}
	}

	missing := newMissingVariables()
	opts := lexerOptions()
	env, err := gj.New(gj.WithAutoescape(gj.AutoescapeNever{}),
		gj.WithUndefined(missing.factory), gj.WithLexerOptions(opts))
	if err != nil {
		return "", &RenderError{Stage: StageInitialize, Err: err}
	}
	tpl, err := env.FromString(body)
	if err == nil {
		err = rejectLoopControls(body, opts)
	}
	if err != nil {
		if headed {
			err = headerLineOffset(err, template)
		}
		return "", &RenderError{Stage: StageCompile, Err: err}
	}
	out, err := tpl.RenderContext(ctx, convertInstanceData(data))
	if err != nil {
		return "", &RenderError{Stage: StageExecute, Err: err}
	}
	if err := ctx.Err(); err != nil {
		return "", &RenderError{Stage: StageExecute, Err: err}
	}
	if out, err = missing.replace(out); err != nil {
		return "", &RenderError{Stage: StageExecute, Err: err}
	}
	// cloud-init restores the final newline that Jinja2 removes.
	if strings.HasSuffix(body, "\n") {
		out += "\n"
	}
	return out, nil
}

// InstanceDataJSON returns the instance data that Render builds for req,
// before cloud-init's variable conversion, serialized as cloud-init writes
// instance-data.json. Template is ignored. The result can be passed to
// cloud-init devel render or used as InstanceData.
func InstanceDataJSON(req RenderRequest) ([]byte, error) {
	data, err := buildInstanceData(req)
	if err != nil {
		return nil, err
	}
	return []byte(pyJSONDumps(data)), nil
}
