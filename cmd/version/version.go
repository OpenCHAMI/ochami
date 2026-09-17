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
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(cli.Ios.Out(), "Version:    %s\n", version.Version)
			fmt.Fprintf(cli.Ios.Out(), "Tag:        %s\n", version.Tag)
			fmt.Fprintf(cli.Ios.Out(), "Branch:     %s\n", version.Branch)
			fmt.Fprintf(cli.Ios.Out(), "Commit:     %s\n", version.Commit)
			fmt.Fprintf(cli.Ios.Out(), "Git State:  %s\n", version.GitState)
			fmt.Fprintf(cli.Ios.Out(), "Date:       %s\n", version.Date)
			fmt.Fprintf(cli.Ios.Out(), "Go:         %s\n", version.GoVersion)
			fmt.Fprintf(cli.Ios.Out(), "Compiler:   %s\n", runtime.Compiler)
			fmt.Fprintf(cli.Ios.Out(), "Build Host: %s\n", version.BuildHost)
			fmt.Fprintf(cli.Ios.Out(), "Build User: %s\n", version.BuildUser)
		},
	}

	return versionCmd
}
