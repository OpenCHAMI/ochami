// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025-2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cli

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/log"
	"github.com/openchami/ochami/pkg/client"
	"github.com/openchami/ochami/pkg/discover"
	"github.com/openchami/ochami/pkg/format"
)

var FileExistsError = errors.New("file exists")

// PrintUsageHandleError prints a command's usage, returning an error rather
// than terminating the process when usage output fails.
func PrintUsageHandleError(cmd *cobra.Command) error {
	if err := cmd.Usage(); err != nil {
		return Errorf(CodeGeneric, "failed to print usage: %w", err)
	}
	return nil
}

// PrintUsage is a Cobra RunE adapter for metacommands whose only direct action
// is to display their usage.
func PrintUsage(cmd *cobra.Command, args []string) error {
	return PrintUsageHandleError(cmd)
}

// LogHelpHint emits the "see '<cmd> --help'" hint for cmd.
func LogHelpHint(cmd *cobra.Command) {
	log.Logger.Error().Msgf("see '%s --help' for long command help", cmd.CommandPath())
}

// CompletionFormatData completes values accepted by format.DataFormat.
func CompletionFormatData(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	help := make([]string, 0, len(format.DataFormatHelp))
	for value, description := range format.DataFormatHelp {
		help = append(help, fmt.Sprintf("%s\t%s", value, description))
	}
	return help, cobra.ShellCompDirectiveDefault
}

// AddFormatInputFlag registers invocation-local input format storage on cmd.
// The FlagSet owns the value for the lifetime of the command tree.
func AddFormatInputFlag(cmd *cobra.Command) {
	value := format.DataFormatJson
	cmd.Flags().VarP(&value, "format-input", "f", "format of input payload data (json,json-pretty,yaml)")
}

// AddFormatOutputFlag registers invocation-local output format storage on cmd.
// The FlagSet owns the value for the lifetime of the command tree.
func AddFormatOutputFlag(cmd *cobra.Command) {
	value := format.DataFormatJson
	cmd.Flags().VarP(&value, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")
}

// ApplyFormatFlags copies explicitly changed, command-owned format values into
// the invocation runtime. Unchanged flags leave configuration defaults intact.
func ApplyFormatFlags(cmd *cobra.Command, rt *Runtime) error {
	if cmd == nil || rt == nil {
		return Errorf(CodeConfig, "cannot apply format flags without a command runtime")
	}
	if flag := cmd.Flag("format-input"); flag != nil && flag.Changed {
		if err := rt.FormatInput.Set(flag.Value.String()); err != nil {
			return Errorf(CodeUsage, "invalid input format: %w", err)
		}
	}
	if flag := cmd.Flag("format-output"); flag != nil && flag.Changed {
		if err := rt.FormatOutput.Set(flag.Value.String()); err != nil {
			return Errorf(CodeUsage, "invalid output format: %w", err)
		}
	}
	return nil
}

// CompletionDiscoveryVersion completes discovery file format versions.
func CompletionDiscoveryVersion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	help := make([]string, 0, len(discover.DiscoveryVersionHelp))
	for value, description := range discover.DiscoveryVersionHelp {
		help = append(help, fmt.Sprintf("%d\t%s", value, description))
	}
	return help, cobra.ShellCompDirectiveDefault
}

// CompletionPatchMethod completes supported HTTP patch methods.
func CompletionPatchMethod(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	help := make([]string, 0, len(client.PatchMethodHelp))
	for value, description := range client.PatchMethodHelp {
		help = append(help, fmt.Sprintf("%s\t%s", value, description))
	}
	return help, cobra.ShellCompDirectiveDefault
}
