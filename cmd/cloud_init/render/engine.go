// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package render

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/cloudinit"

	cloud_init_lib "github.com/openchami/ochami/internal/cli/cloud_init"
)

// Environment variables that configure cloud-init delegation, matching the
// cloud-init.path, cloud-init.python, and cloud-init.delegate-render keys.
const (
	envCloudInitPath   = "OCHAMI_CLOUD_INIT_PATH"
	envCloudInitPython = "OCHAMI_CLOUD_INIT_PYTHON"
	envDelegateRender  = "OCHAMI_CLOUD_INIT_DELEGATE_RENDER"
)

// engines is the outcome of engine selection.
type engines struct {
	internal  bool
	delegate  *cloud_init_lib.Delegate
	compare   bool
	reference string
}

// selectEngines chooses the engines for an invocation. Flags take precedence
// over environment variables, which take precedence over configuration. When
// cloud-init was requested by a flag and cannot run, it is an error. When it
// came only from the environment or configuration, ochami warns and uses its
// internal engine.
func selectEngines(cmd *cobra.Command, rt *cli.Runtime, options renderInputs) (engines, error) {
	lookup := func(flag, env, cfg string) string {
		if flag != "" {
			return flag
		}
		if rt.Env != nil {
			if v, ok := rt.Env.LookupEnv(env); ok && v != "" {
				return v
			}
		}
		return cfg
	}
	d := &cloud_init_lib.Delegate{
		Path:   lookup(options.cloudInit, envCloudInitPath, rt.Config.CloudInit.Path),
		Python: lookup(options.cloudInitPython, envCloudInitPython, rt.Config.CloudInit.Python),
		Src:    options.cloudInitSrc,
		Warn:   func(w string) { rt.Logger.Warn().Msgf("cloud-init: %s", w) },
	}
	delegateDefault := rt.Config.CloudInit.DelegateRender
	if rt.Env != nil {
		if v, ok := rt.Env.LookupEnv(envDelegateRender); ok && v != "" {
			parsed, err := strconv.ParseBool(v)
			if err != nil {
				return engines{}, cli.Errorf(cli.CodeConfig, "%s must be a boolean, got %q", envDelegateRender, v)
			}
			delegateDefault = parsed
		}
	}
	locationFlag := options.cloudInit != "" || options.cloudInitPython != "" || options.cloudInitSrc != ""

	var useCloudInit, explicit bool
	switch {
	case options.compare:
		useCloudInit, explicit = true, true
	case options.engine == engineInternal:
		useCloudInit = false
	case options.engine == engineCloudInit:
		useCloudInit, explicit = true, true
	case locationFlag:
		useCloudInit, explicit = true, true
	default:
		useCloudInit = delegateDefault
	}
	if !useCloudInit {
		return engines{internal: true}, nil
	}

	fallback := func(reason error) (engines, error) {
		if explicit {
			return engines{}, cli.Errorf(cli.CodeConfig, "cannot render with cloud-init: %w", reason)
		}
		rt.Logger.Warn().Msgf("rendering with ochami's internal engine, because cloud-init cannot run: %v", reason)
		return engines{internal: true}, nil
	}
	if !d.Configured() {
		return fallback(fmt.Errorf("no cloud-init is configured: set --cloud-init, %s, or cloud-init.path", envCloudInitPath))
	}
	if err := d.Probe(cmd.Context()); err != nil {
		return fallback(err)
	}
	reference := "cloud-init"
	if v := d.Version(); v != "" {
		reference += " " + v
	}
	return engines{internal: options.compare, delegate: d, compare: options.compare, reference: reference}, nil
}

// writeOutput writes rendered output to standard output.
func writeOutput(rt *cli.Runtime, out string) error {
	n, err := io.WriteString(rt.Ios.Out(), out)
	if err == nil && n != len(out) {
		err = io.ErrShortWrite
	}
	if err != nil {
		return cli.Errorf(cli.CodePayload, "failed to write rendered output: %w", err)
	}
	return nil
}

// renderTemplate renders a template-mode bundle with the selected engines.
func renderTemplate(cmd *cobra.Command, rt *cli.Runtime, b *cloudinit.Bundle, e engines) error {
	if len(strings.TrimSpace(string(b.Template))) > 0 && !cloudinit.HasJinjaHeader(b.Template) {
		rt.Logger.Warn().Msg("template does not begin with '## template: jinja': cloud-init on a node will use it verbatim without rendering")
	}
	var internalOut string
	if e.internal {
		out, err := cloudinit.Render(cmd.Context(), cloudinit.RenderRequest{Template: b.Template, InstanceData: cloudinit.Document{Data: b.InstanceData}})
		if err != nil {
			return cli.Errorf(cli.CodePayload, "%s: %w", renderErrorMessage(err), err)
		}
		if names := missingVariables(out); len(names) > 0 {
			rt.Logger.Warn().Msgf("could not render template variables, which cloud-init prints as %s<name>: %s",
				cloudinit.MissingVariablePrefix, strings.Join(names, ", "))
		}
		internalOut = out
	}
	if e.delegate == nil {
		return writeOutput(rt, internalOut)
	}
	out, err := e.delegate.RenderTemplate(cmd.Context(), b.Template, b.InstanceData)
	if err != nil {
		return cli.Errorf(cli.CodePayload, "failed to render template with %s: %w", e.reference, err)
	}
	if err := writeOutput(rt, out); err != nil {
		return err
	}
	if e.compare {
		return reportComparison(rt, e.reference, cloud_init_lib.UnifiedDiff("internal", e.reference, internalOut, out))
	}
	return nil
}

// renderFull renders a full-mode bundle with the selected engines.
func renderFull(cmd *cobra.Command, rt *cli.Runtime, b *cloudinit.Bundle, e engines) error {
	var internal cloudinit.FullResult
	if e.internal {
		result, err := cloudinit.RenderFull(cmd.Context(), b.RenderFullRequest(nil))
		if err != nil {
			return cli.Errorf(cli.CodePayload, "failed to render node configuration: %w", err)
		}
		internal = result
		if e.delegate == nil {
			reportParts(rt, result.Parts, result.UserFailed, result.VendorFailed, result.VendorDisabled)
			return writeOutput(rt, result.Text())
		}
	}
	out, err := e.delegate.RenderFull(cmd.Context(), *b)
	if err != nil {
		return cli.Errorf(cli.CodePayload, "failed to render node configuration with %s: %w", e.reference, err)
	}
	if out.MergedByOchami {
		rt.Logger.Warn().Msgf("no Python interpreter that imports cloud-init was found, so %s rendered the templates and ochami merged the parts; set --cloud-init-python for an exact result", e.reference)
	}
	reportParts(rt, out.Parts, out.UserFailed, out.VendorFailed, out.VendorDisabled)
	if err := writeOutput(rt, out.Text); err != nil {
		return err
	}
	if !e.compare {
		return nil
	}
	diff, err := compareFull(internal, out, e.reference)
	if err != nil {
		return cli.Errorf(cli.CodePayload, "failed to compare results: %w", err)
	}
	return reportComparison(rt, e.reference, diff)
}

// compareFull diffs the merged configuration and part statuses from both
// engines.
func compareFull(internal cloudinit.FullResult, out cloud_init_lib.FullOutput, reference string) (string, error) {
	normalize := func(v any) (any, error) {
		data, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		var n any
		return n, json.Unmarshal(data, &n)
	}
	a, err := normalize(internal.Comparable())
	if err != nil {
		return "", err
	}
	b, err := normalize(out.Comparable)
	if err != nil {
		return "", err
	}
	var diff strings.Builder
	if !reflect.DeepEqual(a, b) {
		textA, err := cloudinit.ComparableText(a)
		if err != nil {
			return "", err
		}
		textB, err := cloudinit.ComparableText(b)
		if err != nil {
			return "", err
		}
		diff.WriteString(cloud_init_lib.UnifiedDiff("internal (merged configuration)", reference+" (merged configuration)", textA, textB))
	}
	statuses := func(parts []cloudinit.Part, failedUser, failedVendor string) string {
		var b strings.Builder
		for _, p := range parts {
			if p.Name != "" {
				fmt.Fprintf(&b, "%s %s: %s\n", p.Data, p.Name, p.Status)
			}
		}
		if failedUser != "" {
			b.WriteString("user-data: processing aborted\n")
		}
		if failedVendor != "" {
			b.WriteString("vendor-data: processing aborted\n")
		}
		return b.String()
	}
	for _, p := range internal.Parts {
		if p.Status == cloudinit.PartUnsupported || p.Status == cloudinit.PartUnresolved {
			// Part numbering differs where only cloud-init processes the
			// data or where the content is unknown, so compare merged
			// configuration only.
			return diff.String(), nil
		}
	}
	diff.WriteString(cloud_init_lib.UnifiedDiff("internal (parts)", reference+" (parts)",
		statuses(internal.Parts, internal.UserFailed, internal.VendorFailed),
		statuses(out.Parts, out.UserFailed, out.VendorFailed)))
	return diff.String(), nil
}

// reportComparison prints a difference and makes the command fail.
func reportComparison(rt *cli.Runtime, reference, diff string) error {
	if diff == "" {
		rt.Logger.Info().Msgf("ochami's internal engine matches %s", reference)
		return nil
	}
	if _, err := io.WriteString(rt.Ios.Err(), diff); err != nil {
		return cli.Errorf(cli.CodeGeneric, "failed to write comparison: %w", err)
	}
	return cli.Errorf(cli.CodeGeneric, "ochami's internal engine differs from %s", reference)
}

// reportParts warns about parts a node would not merge.
func reportParts(rt *cli.Runtime, parts []cloudinit.Part, userFailed, vendorFailed string, vendorDisabled bool) {
	for _, p := range parts {
		label := p.Source
		if p.Name != "" && p.Source != p.Data {
			label = fmt.Sprintf("%s %s (%s)", p.Data, p.Name, p.Source)
		} else if p.Name != "" {
			label = fmt.Sprintf("%s %s", p.Data, p.Name)
		}
		suffix := ""
		if p.Message != "" {
			suffix = ": " + p.Message
		}
		switch {
		case p.Status == cloudinit.PartMerged:
		case p.Status == cloudinit.PartDropped && p.Data == "user-data" && strings.Contains(p.Message, "empty cloud config"):
			// OpenCHAMI serves an empty #cloud-config as user-data, which
			// cloud-init skips on every node.
			rt.Logger.Info().Msgf("%s has no cloud-config to merge", label)
		case p.Status == cloudinit.PartScript || p.Status == cloudinit.PartBoothook:
			rt.Logger.Info().Msgf("%s is a %s, which cloud-init runs rather than merges", label, p.Status)
		default:
			rt.Logger.Warn().Msgf("%s is %s%s", label, p.Status, suffix)
		}
		for _, w := range p.Warnings {
			rt.Logger.Warn().Msgf("%s: %s", label, w)
		}
	}
	if userFailed != "" {
		rt.Logger.Warn().Msgf("cloud-init on a node would abort processing user-data and vendor-data: %s", userFailed)
	}
	if vendorFailed != "" {
		rt.Logger.Warn().Msgf("cloud-init on a node would abort processing vendor-data: %s", vendorFailed)
	}
	if vendorDisabled {
		rt.Logger.Warn().Msg("user-data disables vendor-data (vendor_data.enabled), so cloud-init ignores it")
	}
}
