// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package iface

import (
	"net/url"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/client"

	smd_lib "github.com/openchami/ochami/internal/cli/smd"
	"github.com/openchami/ochami/pkg/client/smd"
)

// ifaceGetOptions holds the flag values for the smd iface get command.
// This struct is used to avoid ignored errors by providing a clean interface
// for accessing flag values that are registered with the correct types.
type ifaceGetOptions struct {
	ID        string
	ByIP      bool
	Mac       []string
	IP        []string
	Net       []string
	CompID    []string
	Type      []string
	OlderThan string
	NewerThan string
}

// runCoreIfaceGetWithRuntime contains the core logic for the smd iface get command.
// It takes the parsed options and performs the actual work of getting ethernet interfaces.
// Runtime-aware version.
func runCoreIfaceGetWithRuntime(cmd *cobra.Command, opts *ifaceGetOptions, smdClient *smd.SMDClient, rt *cli.Runtime) error {
	// Deal with --id
	if opts.ID != "" {
		// This endpoint requires authentication, so a token is needed
		if err := rt.HandleToken(cmd); err != nil {
			return err
		}

		httpEnv, err := smdClient.GetEthernetInterfaceByID(cmd.Context(), opts.ID, rt.Token, opts.ByIP)
		if err != nil {
			return cli.ClassifyClientError(err, "SMD ethernet interface request by ID yielded unsuccessful HTTP response", "failed to request ethernet interfaces by ID from SMD")
		}
		if err := cli.WriteString(rt.Ios.Out(), string(httpEnv.Body)+"\n"); err != nil {
			return err
		}
		return nil
	} else if opts.ByIP {
		return cli.Errorf(cli.CodeUsage, "--by-ip can only be used with --id")
	}

	// All other cases
	qstr := ""
	if len(opts.Mac) > 0 || len(opts.IP) > 0 || len(opts.Net) > 0 || len(opts.CompID) > 0 ||
		len(opts.Type) > 0 || opts.OlderThan != "" || opts.NewerThan != "" {
		values := url.Values{}
		for _, m := range opts.Mac {
			values.Add("MACAddress", m)
		}
		for _, i := range opts.IP {
			values.Add("IPAddress", i)
		}
		for _, n := range opts.Net {
			values.Add("Network", n)
		}
		for _, c := range opts.CompID {
			values.Add("ComponentID", c)
		}
		for _, t := range opts.Type {
			values.Add("Type", t)
		}
		if opts.OlderThan != "" {
			values.Add("OlderThan", opts.OlderThan)
		}
		if opts.NewerThan != "" {
			values.Add("NewerThan", opts.NewerThan)
		}
		qstr = values.Encode()
	}

	httpEnv, err := smdClient.GetEthernetInterfaces(cmd.Context(), qstr)
	if err != nil {
		return cli.ClassifyClientError(err, "SMD ethernet interface request yielded unsuccessful HTTP response", "failed to request ethernet interfaces from SMD")
	}

	// Print output
	outBytes, err := client.FormatBody(httpEnv.Body, rt.FormatOutput)
	if err != nil {
		return cli.Errorf(cli.CodePayload, "failed to format output: %w", err)
	}
	if err := cli.WriteOutput(rt.Ios.Out(), outBytes); err != nil {
		return err
	}

	return nil
}

func newCmdIfaceGet() *cobra.Command {
	// ifaceGetCmd represents the "smd iface get" command
	var ifaceGetCmd = &cobra.Command{
		Use:     "get",
		Aliases: []string{"list"},
		Args:    cobra.NoArgs,
		Short:   "Get some or all ethernet interfaces",
		Long: `Get some or all ethernet interfaces optionally based on filter(s). If no options are
passed, all ethernet interfaces are returned. Optionally, options can be passed to limit the
ethernet interfaces returned.

See ochami-smd(1) for more details.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get runtime from context (always available since cmd/root.go injects it)
			rt, err := cli.RuntimeFromCommand(cmd)
			if err != nil {
				return err
			}

			// Create client to use for requests with runtime
			smdClient, err := smd_lib.GetClientWithRuntime(cmd, rt)
			if err != nil {
				return err
			}

			// Extract options from flags
			// Since flags are registered with the correct types on this command,
			// these Get* calls cannot fail, so we ignore errors with explicit comments
			opts := &ifaceGetOptions{}
			if cmd.Flag("id").Changed {
				opts.ID, _ = cmd.Flags().GetString("id") //nolint:errcheck // Flag registered with matching type, error impossible
			}
			if cmd.Flag("by-ip").Changed {
				opts.ByIP = true
			}
			if cmd.Flag("mac").Changed {
				opts.Mac, _ = cmd.Flags().GetStringSlice("mac") //nolint:errcheck // Flag registered with matching type, error impossible
			}
			if cmd.Flag("ip").Changed {
				opts.IP, _ = cmd.Flags().GetStringSlice("ip") //nolint:errcheck // Flag registered with matching type, error impossible
			}
			if cmd.Flag("net").Changed {
				opts.Net, _ = cmd.Flags().GetStringSlice("net") //nolint:errcheck // Flag registered with matching type, error impossible
			}
			if cmd.Flag("comp-id").Changed {
				opts.CompID, _ = cmd.Flags().GetStringSlice("comp-id") //nolint:errcheck // Flag registered with matching type, error impossible
			}
			if cmd.Flag("type").Changed {
				opts.Type, _ = cmd.Flags().GetStringSlice("type") //nolint:errcheck // Flag registered with matching type, error impossible
			}
			if cmd.Flag("older-than").Changed {
				opts.OlderThan, _ = cmd.Flags().GetString("older-than") //nolint:errcheck // Flag registered with matching type, error impossible
			}
			if cmd.Flag("newer-than").Changed {
				opts.NewerThan, _ = cmd.Flags().GetString("newer-than") //nolint:errcheck // Flag registered with matching type, error impossible
			}

			return runCoreIfaceGetWithRuntime(cmd, opts, smdClient, rt)
		},
	}

	// Create flags
	ifaceGetCmd.Flags().StringP("id", "i", "", "get an ethernet interface by its ID")
	ifaceGetCmd.Flags().Bool("by-ip", false, "get all IP addresses for an ethernet interface (used with --id)")
	ifaceGetCmd.Flags().StringSliceP("mac", "m", []string{}, "filter ethernet interfaces by mac address")
	ifaceGetCmd.Flags().StringSlice("ip", []string{}, "filter ethernet interfaces by IP address")
	ifaceGetCmd.Flags().StringSlice("net", []string{}, "filter ethernet interfaces by IP on given network")
	ifaceGetCmd.Flags().StringSlice("comp-id", []string{}, "filter ethernet interfaces by component ID")
	ifaceGetCmd.Flags().StringSlice("type", []string{}, "filter ethernet interfaces by type")
	ifaceGetCmd.Flags().String("older-than", "", "filter ethernet interfaces by update time older than specified time (RFC3339-formatted)")
	ifaceGetCmd.Flags().String("newer-than", "", "filter ethernet interfaces by update time older than specified time (RFC3339-formatted)")

	cli.AddFormatOutputFlag(ifaceGetCmd)
	ifaceGetCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)
	ifaceGetCmd.MarkFlagsMutuallyExclusive("id", "mac")
	ifaceGetCmd.MarkFlagsMutuallyExclusive("id", "ip")
	ifaceGetCmd.MarkFlagsMutuallyExclusive("id", "net")
	ifaceGetCmd.MarkFlagsMutuallyExclusive("id", "comp-id")
	ifaceGetCmd.MarkFlagsMutuallyExclusive("id", "type")
	ifaceGetCmd.MarkFlagsMutuallyExclusive("id", "older-than")
	ifaceGetCmd.MarkFlagsMutuallyExclusive("id", "newer-than")
	ifaceGetCmd.MarkFlagsMutuallyExclusive("by-ip", "mac")
	ifaceGetCmd.MarkFlagsMutuallyExclusive("by-ip", "ip")
	ifaceGetCmd.MarkFlagsMutuallyExclusive("by-ip", "net")
	ifaceGetCmd.MarkFlagsMutuallyExclusive("by-ip", "comp-id")
	ifaceGetCmd.MarkFlagsMutuallyExclusive("by-ip", "type")
	ifaceGetCmd.MarkFlagsMutuallyExclusive("by-ip", "older-than")
	ifaceGetCmd.MarkFlagsMutuallyExclusive("by-ip", "newer-than")

	return ifaceGetCmd
}
