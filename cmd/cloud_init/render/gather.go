// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package render

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/internal/version"
	"github.com/openchami/ochami/pkg/client/cloud_init"
	"github.com/openchami/ochami/pkg/cloudinit"
	"github.com/openchami/ochami/pkg/format"

	cloud_init_lib "github.com/openchami/ochami/internal/cli/cloud_init"
)

// createdBy names this ochami in bundles it writes.
func createdBy() string { return "ochami " + version.Version }

// renderErrorMessage describes a rendering error by its stage.
func renderErrorMessage(err error) string {
	var renderErr *cloudinit.RenderError
	if errors.As(err, &renderErr) {
		switch renderErr.Stage {
		case cloudinit.StageMetadata:
			return "failed to decode node metadata into template variables"
		case cloudinit.StageInstanceData:
			return "failed to decode instance data"
		case cloudinit.StageExtraVars:
			return "unable to read extra variable data or file"
		case cloudinit.StageInitialize, cloudinit.StageCompile:
			return "failed to create template"
		}
	}
	return "failed to render template"
}

// instanceDataFor builds instance-data.json for a request and the
// invocation's extra variables.
func instanceDataFor(cmd *cobra.Command, rt *cli.Runtime, req *cloudinit.RenderRequest) ([]byte, error) {
	if cmd.Flag("extra-vars").Changed {
		extraVars, err := readExtraVars(cmd.Flag("extra-vars").Value.String(), rt.Ios.InContext(cmd.Context()))
		if err != nil {
			return nil, cli.Errorf(cli.CodePayload, "unable to read extra variable data or file: %w", err)
		}
		req.ExtraVars = cloudinit.Document{Data: extraVars, Format: rt.FormatInput}
	}
	data, err := cloudinit.InstanceDataJSON(*req)
	if err != nil {
		return nil, cli.Errorf(cli.CodePayload, "%s: %w", renderErrorMessage(err), err)
	}
	return data, nil
}

// gatherBundle collects every input for the invocation into a bundle. A nil
// bundle means there is nothing to render.
func gatherBundle(cmd *cobra.Command, rt *cli.Runtime, options renderInputs) (*cloudinit.Bundle, error) {
	if options.bundle != "" {
		b, err := cloudinit.LoadBundle(options.bundle)
		if err != nil {
			return nil, cli.Errorf(cli.CodePayload, "failed to load render bundle: %w", err)
		}
		if options.full && b.Mode != cloudinit.BundleFull {
			return nil, cli.Errorf(cli.CodeUsage, "--full requires a full render bundle, but %s renders a template", options.bundle)
		}
		return &b, nil
	}
	if options.full {
		return gatherFull(cmd, rt, options)
	}
	req, err := loadRenderRequest(cmd, rt, options)
	if err != nil || req == nil {
		return nil, err
	}
	data, err := instanceDataFor(cmd, rt, req)
	if err != nil {
		return nil, err
	}
	return &cloudinit.Bundle{
		Mode: cloudinit.BundleTemplate, CreatedBy: createdBy(), Node: options.node, Group: options.group,
		Template: req.Template, InstanceData: data,
	}, nil
}

// gatherFull fetches a node's user-data, vendor-data, metadata, and every
// include through the impersonation API.
func gatherFull(cmd *cobra.Command, rt *cli.Runtime, options renderInputs) (*cloudinit.Bundle, error) {
	req := &cloudinit.RenderRequest{Metadata: cloudinit.Document{Format: format.DataFormatYaml}}
	if err := readVariableFiles(cmd, rt, options, req); err != nil {
		return nil, err
	}
	b := &cloudinit.Bundle{Mode: cloudinit.BundleFull, CreatedBy: createdBy(), Node: options.node}
	if options.userData != "" {
		data, err := readRenderFile(options.userData, rt.Ios.InContext(cmd.Context()))
		if err != nil {
			return nil, cli.Errorf(cli.CodePayload, "failed to read user-data file: %w", err)
		}
		b.UserData = data
	}

	cic, err := cloud_init_lib.GetClient(cmd, rt)
	if err != nil {
		return nil, err
	}
	if err := rt.HandleToken(cmd); err != nil {
		return nil, err
	}
	if b.VendorData, err = fetchNodeData(cmd, rt, cic, cloud_init.CloudInitVendorData, options.node); err != nil {
		return nil, err
	}
	if options.userData == "" {
		if b.UserData, err = fetchNodeData(cmd, rt, cic, cloud_init.CloudInitUserData, options.node); err != nil {
			return nil, err
		}
	}
	if options.metadata == "" && options.instanceData == "" {
		if req.Metadata.Data, err = fetchNodeData(cmd, rt, cic, cloud_init.CloudInitMetaData, options.node); err != nil {
			return nil, err
		}
	}
	if b.InstanceData, err = instanceDataFor(cmd, rt, req); err != nil {
		return nil, err
	}

	rec := &includeRecorder{cmd: cmd, rt: rt, cic: cic, node: options.node, seen: map[string]bool{}}
	for _, data := range [][]byte{b.UserData, b.VendorData} {
		// A failed include stops cloud-init, and the recorder keeps it, so
		// the error itself needs no handling here.
		_ = cloudinit.CollectIncludes(cmd.Context(), data, rec.fetch) //nolint:errcheck // recorded in rec.includes
	}
	if err := cmd.Context().Err(); err != nil {
		return nil, err
	}
	b.Includes = rec.includes
	return b, nil
}

// includeRecorder fetches includes the way a node would, through the
// impersonation API, and records what each produced.
type includeRecorder struct {
	cmd      *cobra.Command
	rt       *cli.Runtime
	cic      *cloud_init.CloudInitClient
	node     string
	seen     map[string]bool
	includes []cloudinit.BundleInclude
}

// groupFromURL returns the group a vendor-data include names. OpenCHAMI
// includes are {base}/{group}.yaml, and a node is identified by its address,
// so only the group name is meaningful from an admin host.
func groupFromURL(raw string) (string, bool) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", false
	}
	base := path.Base(u.Path)
	group, ok := strings.CutSuffix(base, ".yaml")
	if !ok || group == "" || strings.ContainsAny(group, "/") {
		return "", false
	}
	return group, true
}

func (r *includeRecorder) fetch(ctx context.Context, raw string) ([]byte, error) {
	record := func(inc cloudinit.BundleInclude) {
		if !r.seen[raw] {
			r.seen[raw] = true
			r.includes = append(r.includes, inc)
		}
	}
	group, ok := groupFromURL(raw)
	if !ok {
		msg := "the URL does not name a group template ({group}.yaml)"
		record(cloudinit.BundleInclude{URL: raw, Status: cloudinit.IncludeUnresolved, Message: msg})
		return nil, fmt.Errorf("%s: %s: %w", raw, msg, cloudinit.ErrIncludeUnresolved)
	}
	results, err := r.cic.GetNodeGroupData(ctx, r.rt.Token, r.node, group)
	if err == nil {
		err = results[0].Err
	}
	if err != nil {
		inc := cloudinit.BundleInclude{URL: raw, Status: cloudinit.IncludeFailed, Message: err.Error()}
		if results != nil {
			inc.Code = results[0].Value.StatusCode
		}
		record(inc)
		return nil, err
	}
	body := results[0].Value.Body
	record(cloudinit.BundleInclude{URL: raw, Status: cloudinit.IncludeResolved, Content: body})
	return body, nil
}
