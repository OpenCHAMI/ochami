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

// Engine names accepted by --engine.
const (
	engineInternal  = "internal"
	engineCloudInit = "cloud-init"
)

// renderInputs selects the sources, engine, and outputs for an invocation.
type renderInputs struct {
	template     string
	group        string
	node         string
	metadata     string
	instanceData string
	userData     string
	bundle       string
	exportBundle string
	full         bool
	noRender     bool
	compare      bool
	engine       string

	cloudInit       string
	cloudInitPython string
	cloudInitSrc    string
}

func optionsForArgs(options renderInputs, args []string, legacy bool) renderInputs {
	if legacy {
		options.group, options.node = args[0], args[1]
	}
	return options
}

// changed reports whether a flag exists on cmd and was set.
func changed(cmd *cobra.Command, name string) bool {
	f := cmd.Flag(name)
	return f != nil && f.Changed
}

// validateRenderInputs checks every flag combination before any input is
// read or requested.
func validateRenderInputs(cmd *cobra.Command, options renderInputs) error {
	for _, name := range []string{"template", "group", "node", "metadata", "instance-data", "user-data", "bundle", "export-bundle", "cloud-init", "cloud-init-python", "cloud-init-src"} {
		if f := cmd.Flag(name); f != nil && f.Changed && f.Value.String() == "" {
			return fmt.Errorf("--%s requires a nonempty value", name)
		}
	}
	if options.engine != "" && options.engine != engineInternal && options.engine != engineCloudInit {
		return fmt.Errorf("--engine must be %s or %s", engineInternal, engineCloudInit)
	}
	switch {
	case options.bundle != "":
		for _, name := range []string{"template", "group", "node", "metadata", "instance-data", "user-data", "extra-vars", "export-bundle"} {
			if changed(cmd, name) {
				return fmt.Errorf("--bundle supplies every input, so it cannot be combined with --%s", name)
			}
		}
	case options.full:
		if options.template != "" || options.group != "" {
			return fmt.Errorf("--full renders every group of a node, so it cannot be combined with --template or --group")
		}
		if options.node == "" {
			return fmt.Errorf("--full requires --node or --bundle")
		}
	default:
		if (options.template == "") == (options.group == "") {
			return fmt.Errorf("specify exactly one of --template or --group")
		}
	}
	if options.template != "" && options.metadata != "" && options.node != "" {
		return fmt.Errorf("with --template, use either --metadata or --node, not both")
	}
	if options.instanceData != "" && options.metadata != "" {
		return fmt.Errorf("--instance-data replaces metadata, so it cannot be combined with --metadata")
	}
	if options.userData != "" && !options.full {
		return fmt.Errorf("--user-data requires --full")
	}
	if options.compare && options.engine != "" {
		return fmt.Errorf("--compare runs both engines, so it cannot be combined with --engine")
	}
	if options.noRender {
		if options.exportBundle == "" {
			return fmt.Errorf("--no-render requires --export-bundle")
		}
		if options.compare || options.engine != "" {
			return fmt.Errorf("--no-render renders nothing, so it cannot be combined with --compare or --engine")
		}
	}
	stdinSources := 0
	for _, path := range []string{options.template, options.metadata, options.instanceData, options.userData} {
		if path == "-" {
			stdinSources++
		}
	}
	if f := cmd.Flag("extra-vars"); f.Changed && f.Value.String() == "@-" {
		stdinSources++
	}
	if stdinSources > 1 {
		return fmt.Errorf("only one of --template -, --metadata -, --instance-data -, --user-data -, and --extra-vars @- may read stdin")
	}
	return nil
}

// loadRenderRequest acquires local or remote sources for template mode. A nil
// request preserves the early return for empty remote group configuration.
func loadRenderRequest(cmd *cobra.Command, rt *cli.Runtime, options renderInputs) (*cloudinit.RenderRequest, error) {
	req := &cloudinit.RenderRequest{Metadata: cloudinit.Document{Format: format.DataFormatYaml}}
	if options.template != "" {
		data, err := readRenderFile(options.template, rt.Ios.InContext(cmd.Context()))
		if err != nil {
			return nil, cli.Errorf(cli.CodePayload, "failed to read template file: %w", err)
		}
		req.Template = data
	}
	if err := readVariableFiles(cmd, rt, options, req); err != nil {
		return nil, err
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
	if options.metadata != "" || options.instanceData != "" || options.node == "" {
		return req, nil
	}
	if req.Metadata.Data, err = fetchNodeData(cmd, rt, cloudInitClient, cloud_init.CloudInitMetaData, options.node); err != nil {
		return nil, err
	}
	return req, nil
}

// readVariableFiles reads --metadata and --instance-data into req.
func readVariableFiles(cmd *cobra.Command, rt *cli.Runtime, options renderInputs, req *cloudinit.RenderRequest) error {
	if options.metadata != "" {
		data, err := readRenderFile(options.metadata, rt.Ios.InContext(cmd.Context()))
		if err != nil {
			return cli.Errorf(cli.CodePayload, "failed to read metadata file: %w", err)
		}
		req.Metadata.Data = data
	}
	if options.instanceData != "" {
		data, err := readRenderFile(options.instanceData, rt.Ios.InContext(cmd.Context()))
		if err != nil {
			return cli.Errorf(cli.CodePayload, "failed to read instance data file: %w", err)
		}
		req.InstanceData = cloudinit.Document{Data: data, Format: format.DataFormatYaml}
	}
	return nil
}

// fetchNodeData gets a node's meta-data, user-data, or vendor-data through
// the impersonation API.
func fetchNodeData(cmd *cobra.Command, rt *cli.Runtime, cic *cloud_init.CloudInitClient, dataType cloud_init.CIDataType, node string) ([]byte, error) {
	results, err := cic.GetNodeData(cmd.Context(), dataType, rt.Token, node)
	if err != nil {
		return nil, cli.ClassifyClientError(err, fmt.Sprintf("failed to get cloud-init node %s", dataType), fmt.Sprintf("failed to get cloud-init node %s", dataType))
	}
	if results[0].Err != nil {
		return nil, cli.ClassifyClientError(results[0].Err, fmt.Sprintf("cloud-init node %s request yielded unsuccessful HTTP response", dataType), fmt.Sprintf("failed to get cloud-init node %s", dataType))
	}
	return results[0].Value.Body, nil
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
