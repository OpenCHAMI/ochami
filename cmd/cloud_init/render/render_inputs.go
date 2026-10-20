// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package render

import (
	"encoding/json"
	"fmt"

	"github.com/openchami/cloud-init/pkg/cistore"
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/client/cloud_init"
	"github.com/openchami/ochami/pkg/cloudinit"
	"github.com/openchami/ochami/pkg/format"

	cloud_init_lib "github.com/openchami/ochami/internal/cli/cloud_init"
)

// renderInputs selects the template and metadata sources for an invocation.
type renderInputs struct {
	template string
	group    string
	node     string
	metadata string
}

func optionsForArgs(options renderInputs, args []string, legacy bool) renderInputs {
	if legacy {
		options.group, options.node = args[0], args[1]
	}
	return options
}

func validateRenderInputs(cmd *cobra.Command, options renderInputs) error {
	for _, name := range []string{"template", "group", "node", "metadata"} {
		if f := cmd.Flag(name); f != nil && f.Changed && f.Value.String() == "" {
			return fmt.Errorf("--%s requires a nonempty value", name)
		}
	}
	if (options.template == "") == (options.group == "") {
		return fmt.Errorf("specify exactly one of --template or --group")
	}
	if options.template != "" && options.metadata != "" && options.node != "" {
		return fmt.Errorf("with --template, use either --metadata or --node, not both")
	}
	stdinSources := 0
	for _, path := range []string{options.template, options.metadata} {
		if path == "-" {
			stdinSources++
		}
	}
	if f := cmd.Flag("extra-vars"); f.Changed && f.Value.String() == "@-" {
		stdinSources++
	}
	if stdinSources > 1 {
		return fmt.Errorf("only one of --template -, --metadata -, and --extra-vars @- may read stdin")
	}
	return nil
}

// loadRenderRequest acquires local or remote sources. A nil request preserves
// the early return for empty remote group configuration.
func loadRenderRequest(cmd *cobra.Command, rt *cli.Runtime, options renderInputs) (*cloudinit.RenderRequest, error) {
	req := &cloudinit.RenderRequest{Metadata: cloudinit.Document{Format: format.DataFormatYaml}}
	if options.template != "" {
		data, err := readRenderFile(options.template, rt.Ios.In())
		if err != nil {
			return nil, cli.Errorf(cli.CodePayload, "failed to read template file: %w", err)
		}
		req.Template = data
	}
	if options.metadata != "" {
		data, err := readRenderFile(options.metadata, rt.Ios.In())
		if err != nil {
			return nil, cli.Errorf(cli.CodePayload, "failed to read metadata file: %w", err)
		}
		req.Metadata.Data = data
	}
	if options.group == "" && options.node == "" {
		return req, nil
	}

	cloudInitClient, err := cloud_init_lib.GetClient(cmd, rt)
	if err != nil {
		return nil, err
	}
	if err := rt.HandleToken(cmd); err != nil {
		return nil, err
	}
	if options.group != "" {
		if options.node == "" {
			req.Template, err = loadStoredGroupTemplate(cmd, rt, cloudInitClient, options.group)
		} else {
			results, getErr := cloudInitClient.GetNodeGroupData(cmd.Context(), rt.Token, options.node, options.group)
			if getErr != nil {
				return nil, cli.ClassifyClientError(getErr, "failed to get cloud-init group", "failed to get cloud-init group")
			}
			if results[0].Err != nil {
				return nil, cli.ClassifyClientError(results[0].Err, "cloud-init group request yielded unsuccessful HTTP response", "failed to get cloud-init group")
			}
			req.Template = results[0].Value.Body
		}
		if err != nil {
			return nil, err
		}
		if len(req.Template) == 0 {
			if options.node == "" {
				rt.Logger.Warn().Msgf("stored cloud-config for group %s was empty, cannot render", options.group)
			} else {
				rt.Logger.Warn().Msgf("cloud-config for group %s was empty, cannot render for node %s", options.group, options.node)
			}
			return nil, nil
		}
	}
	if options.metadata != "" || options.node == "" {
		return req, nil
	}
	results, err := cloudInitClient.GetNodeData(cmd.Context(), cloud_init.CloudInitMetaData, rt.Token, options.node)
	if err != nil {
		return nil, cli.ClassifyClientError(err, "failed to get cloud-init node meta-data", "failed to get cloud-init node meta-data")
	}
	if results[0].Err != nil {
		return nil, cli.ClassifyClientError(results[0].Err, "cloud-init node meta-data request yielded unsuccessful HTTP response", "failed to get cloud-init node meta-data")
	}
	req.Metadata.Data = results[0].Value.Body
	return req, nil
}

// loadStoredGroupTemplate decodes raw group content without impersonating a node.
func loadStoredGroupTemplate(cmd *cobra.Command, rt *cli.Runtime, cic *cloud_init.CloudInitClient, group string) ([]byte, error) {
	results := cic.GetGroups(cmd.Context(), rt.Token, group)
	if results[0].Err != nil {
		return nil, cli.ClassifyClientError(results[0].Err, "stored cloud-init group request yielded unsuccessful HTTP response", "failed to get stored cloud-init group")
	}
	var stored *cistore.GroupData
	if err := json.Unmarshal(results[0].Value.Body, &stored); err != nil {
		return nil, cli.Errorf(cli.CodePayload, "failed to decode stored cloud-init group: %w", err)
	}
	if stored == nil {
		return nil, cli.Errorf(cli.CodePayload, "stored cloud-init group response was null")
	}
	if len(stored.File.Content) == 0 {
		return nil, nil
	}
	content, err := cloud_init.DecodeCloudConfig(stored.File)
	if err != nil {
		return nil, cli.Errorf(cli.CodePayload, "failed to decode stored cloud-config for %s: %w", group, err)
	}
	return content, nil
}
