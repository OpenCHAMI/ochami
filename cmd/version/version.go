// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package version

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/internal/version"
)

func NewCmd() *cobra.Command {
	// versionCmd represents the version command
	var versionCmd = &cobra.Command{
		Use:     "version",
		Args:    cobra.NoArgs,
		Short:   "Print detailed version to stdout and exit",
		Example: `  ochami version`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get runtime from context (always available since cmd/root.go injects it)
			rt, err := cli.RuntimeFromCommand(cmd)
			if err != nil {
				return err
			}
			return cli.WriteString(rt.Ios.Out(), fmt.Sprintf(
				"Version:    %s\nTag:        %s\nBranch:     %s\nCommit:     %s\nGit State:  %s\nDate:       %s\nGo:         %s\nCompiler:   %s\nBuild Host: %s\nBuild User: %s\n",
				version.Version, version.Tag, version.Branch, version.Commit, version.GitState,
				version.Date, version.GoVersion, runtime.Compiler, version.BuildHost, version.BuildUser,
			))
		},
	}

	return versionCmd
}
