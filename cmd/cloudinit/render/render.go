// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package render

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/cloudinit"
)

// NewCmd constructs the canonical cloud-init render command.
func NewCmd() *cobra.Command { return newRenderCmd(false) }

// NewGroupCmd constructs the positional group/node compatibility command.
func NewGroupCmd() *cobra.Command { return newRenderCmd(true) }

func newRenderCmd(legacy bool) *cobra.Command {
	var options renderInputs
	cmd := &cobra.Command{
		Use:   "render",
		Short: "Render cloud-init config from a service or local template",
		Long: `Render cloud-init config from a service or local template.

Select exactly one template source: --template FILE or --group GROUP.
With --template, use --metadata FILE, --node NODE, or omit both for empty
metadata. With --group, omit --node to fetch the raw stored template, or provide
--node to select its impersonated template. Without --node, metadata is supplied
by --metadata or defaults to empty; stored group metadata is not merged.
With --node, --metadata replaces live rendering variables and skips that request.
--instance-data replaces metadata with a node's complete instance data.

With --full and --node, render everything the node receives: its vendor-data
includes, fetched as the node would through the impersonation API, its
user-data, and cloud-init's merging. Parts a node would skip are reported.

The internal engine renders by default. --engine cloud-init, --cloud-init PATH,
or cloud-init.delegate-render renders with a real cloud-init instead, and
--compare renders with both and fails with a diff when they differ.
--export-bundle writes the gathered inputs for rendering later with --bundle or
with cloud-init itself.

Metadata files contain YAML or JSON node metadata without a ds wrapper. Use -
for template or metadata stdin; only one source may read stdin, including
--extra-vars @-. The --format-input flag controls extra variables, not metadata.

Raw group selection does not establish membership or applicability to a node.
The endpoint is chosen solely by --node; metadata never selects or infers a node.

See ochami-cloud-init(1) for more details.`,
		Example: `  # Render entirely locally
  ochami --ignore-config cloud-init render --template template.j2 --metadata metadata.yaml

  # Render with empty metadata and extra variables
  ochami --ignore-config cloud-init render --template template.j2 --extra-vars '{"role":"compute"}'

  # Render a template from stdin
  ochami --ignore-config cloud-init render --template - --metadata metadata.yaml < template.j2

  # Render a local template with live node metadata
  ochami cloud-init render --template template.j2 --node x1000c0s0b0n0

  # Render the group template and metadata for a node
  ochami cloud-init render --group compute --node x1000c0s0b0n0

  # Render the stored group template without selecting a node
  ochami cloud-init render --group compute --metadata metadata.yaml

  # Select a group template for a node and replace rendering metadata
  ochami cloud-init render --group compute --node x1000c0s0b0n0 --metadata metadata.yaml

  # Render everything a node receives
  ochami cloud-init render --full --node x1000c0s0b0n0

  # Check the internal engine against a local cloud-init
  ochami cloud-init render --full --node x1000c0s0b0n0 --compare --cloud-init /usr/bin/cloud-init

  # Save the inputs and render them later
  ochami cloud-init render --full --node x1000c0s0b0n0 --export-bundle node.bundle --no-render
  ochami --ignore-config cloud-init render --bundle node.bundle --engine cloud-init --cloud-init-python python3`,
		Args: func(cmd *cobra.Command, args []string) error {
			if legacy {
				if err := cobra.ExactArgs(2)(cmd, args); err != nil {
					return err
				}
				if strings.TrimSpace(args[0]) == "" || strings.TrimSpace(args[1]) == "" {
					return fmt.Errorf("group name and node ID must be nonempty")
				}
			} else if err := cobra.NoArgs(cmd, args); err != nil {
				return err
			}
			return validateRenderInputs(cmd, optionsForArgs(options, args, legacy))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get runtime from context (always available since cmd/root.go injects it)
			rt, err := cli.RuntimeFromCommand(cmd)
			if err != nil {
				return err
			}
			options := optionsForArgs(options, args, legacy)

			e := engines{internal: true}
			if !legacy && !options.noRender {
				// Choose engines before gathering, so an unusable cloud-init
				// fails before any request.
				if e, err = selectEngines(cmd, rt, options); err != nil {
					return err
				}
			}
			b, err := gatherBundle(cmd, rt, options)
			if err != nil || b == nil {
				return err
			}
			if options.exportBundle != "" {
				if err := cloudinit.WriteBundle(options.exportBundle, *b); err != nil {
					return cli.Errorf(cli.CodePayload, "failed to write render bundle: %w", err)
				}
				rt.Logger.Info().Msgf("wrote render bundle to %s, which may contain sensitive node metadata", options.exportBundle)
				if options.noRender {
					return nil
				}
			}
			if b.Mode == cloudinit.BundleFull {
				return renderFull(cmd, rt, b, e)
			}
			return renderTemplate(cmd, rt, b, e)
		},
	}

	if legacy {
		cmd.Use = "render <group_name> <node_id>"
		cmd.Short = "Render a group template (compatibility form of cloud-init render)"
		cmd.Long = `Render the group template selected for a node.

This is the compatibility form of:
  ochami cloud-init render --group GROUP --node NODE

Both the node-specific group template and node metadata are fetched from the
service. Use --extra-vars to overlay rendering variables. Local template inputs,
stored group selection, and metadata files are available in cloud-init render.

See ochami-cloud-init(1) for more details.`
		cmd.Example = `  ochami cloud-init group render compute x1000c0s0b0n0
  ochami cloud-init group render compute x1000c0s0b0n0 --extra-vars '{"role":"compute"}'`
	} else {
		cmd.Flags().StringVar(&options.template, "template", "", "local template file (use - for stdin); mutually exclusive with --group")
		cmd.Flags().StringVar(&options.group, "group", "", "stored group template, or impersonated template with --node; mutually exclusive with --template")
		cmd.Flags().StringVar(&options.node, "node", "", "select an impersonated group template and, unless --metadata is supplied, live metadata")
		cmd.Flags().StringVar(&options.metadata, "metadata", "", "YAML or JSON rendering metadata file (use - for stdin)")
		cmd.Flags().StringVar(&options.instanceData, "instance-data", "", "a node's instance-data JSON to render with instead of metadata (use - for stdin)")
		cmd.Flags().BoolVar(&options.full, "full", false, "render everything a node receives: vendor-data includes, user-data, and merging; requires --node or --bundle")
		cmd.Flags().StringVar(&options.userData, "user-data", "", "with --full, user-data file to use instead of the node's (use - for stdin)")
		cmd.Flags().StringVar(&options.bundle, "bundle", "", "render a bundle written by --export-bundle instead of gathering inputs")
		cmd.Flags().StringVar(&options.exportBundle, "export-bundle", "", "also write the gathered inputs to a new render bundle directory")
		cmd.Flags().BoolVar(&options.noRender, "no-render", false, "with --export-bundle, only write the bundle")
		cmd.Flags().StringVar(&options.engine, "engine", "", "rendering engine: internal or cloud-init (default from cloud-init.delegate-render, else internal)")
		cmd.Flags().BoolVar(&options.compare, "compare", false, "render with both engines, print cloud-init's output, and fail with a diff if they differ")
		cmd.Flags().StringVar(&options.cloudInit, "cloud-init", "", "cloud-init executable, or a wrapper that accepts its arguments (selects the cloud-init engine)")
		cmd.Flags().StringVar(&options.cloudInitPython, "cloud-init-python", "", "Python interpreter that imports cloud-init, for exact full rendering")
		cmd.Flags().StringVar(&options.cloudInitSrc, "cloud-init-src", "", "cloud-init source tree to import, for testing")
		if err := cmd.Flags().MarkHidden("cloud-init-src"); err != nil {
			panic(err)
		}
		for _, name := range []string{"template", "metadata", "instance-data", "user-data", "cloud-init", "cloud-init-python"} {
			cmd.RegisterFlagCompletionFunc(name, cobra.FixedCompletions(nil, cobra.ShellCompDirectiveDefault))
		}
		for _, name := range []string{"bundle", "export-bundle", "cloud-init-src"} {
			cmd.RegisterFlagCompletionFunc(name, cobra.FixedCompletions(nil, cobra.ShellCompDirectiveFilterDirs))
		}
		cmd.RegisterFlagCompletionFunc("engine", cobra.FixedCompletions([]string{engineInternal, engineCloudInit}, cobra.ShellCompDirectiveNoFileComp))
	}
	cmd.Flags().StringP("extra-vars", "e", "", "extra variables, @file, or @- for stdin")
	cli.AddFormatInputFlag(cmd)
	cmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)
	return cmd
}

// missingVariablePattern finds undefined variables in rendered output, as
// cloud-init does to log its warning.
var missingVariablePattern = regexp.MustCompile(regexp.QuoteMeta(cloudinit.MissingVariablePrefix) + `\S+`)

// missingVariables returns the quoted names of undefined variables printed in
// out, in order of first appearance.
func missingVariables(out string) []string {
	var names []string
	seen := map[string]bool{}
	for _, match := range missingVariablePattern.FindAllString(out, -1) {
		name := "'" + strings.TrimPrefix(match, cloudinit.MissingVariablePrefix) + "'"
		if !seen[name] {
			seen[name] = true
			names = append(names, name)
		}
	}
	return names
}
