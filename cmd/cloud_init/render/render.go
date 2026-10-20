// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package render

import (
	"errors"
	"fmt"
	"io"
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
  ochami cloud-init render --group compute --node x1000c0s0b0n0 --metadata metadata.yaml`,
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

			req, err := loadRenderRequest(cmd, rt, optionsForArgs(options, args, legacy))
			if err != nil {
				return err
			}
			if req == nil {
				return nil
			}
			var extraVars []byte
			if cmd.Flag("extra-vars").Changed {
				extraVars, err = readExtraVars(cmd.Flag("extra-vars").Value.String(), rt.Ios.In())
				if err != nil {
					return cli.Errorf(cli.CodePayload, "unable to read extra variable data or file: %w", err)
				}
			}

			req.ExtraVars = cloudinit.Document{Data: extraVars, Format: rt.FormatInput}
			out, err := cloudinit.Render(cmd.Context(), *req)
			if err != nil {
				message := "failed to render template"
				var renderErr *cloudinit.RenderError
				if errors.As(err, &renderErr) {
					switch renderErr.Stage {
					case cloudinit.StageMetadata:
						message = "failed to decode node metadata into template variables"
					case cloudinit.StageExtraVars:
						message = "unable to read extra variable data or file"
					case cloudinit.StageInitialize, cloudinit.StageCompile:
						message = "failed to create template"
					}
				}
				return cli.Errorf(cli.CodePayload, "%s: %w", message, err)
			}
			n, err := io.WriteString(rt.Ios.Out(), out)
			if err == nil && n != len(out) {
				err = io.ErrShortWrite
			}
			if err != nil {
				return cli.Errorf(cli.CodePayload, "failed to write rendered template: %w", err)
			}

			return nil
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
		cmd.RegisterFlagCompletionFunc("template", cobra.FixedCompletions(nil, cobra.ShellCompDirectiveDefault))
		cmd.RegisterFlagCompletionFunc("metadata", cobra.FixedCompletions(nil, cobra.ShellCompDirectiveDefault))
	}
	cmd.Flags().StringP("extra-vars", "e", "", "extra variables, @file, or @- for stdin")
	cli.AddFormatInputFlag(cmd)
	cmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)
	return cmd
}
