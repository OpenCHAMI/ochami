// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package version

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/version"
)

func NewCmd() *cobra.Command {
	// versionCmd represents the version command
	var versionCmd = &cobra.Command{
		Use:     "version",
		Args:    cobra.NoArgs,
		Short:   "Print detailed version to stdout and exit",
		Example: `  ochami version`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Version:    %s\n", version.Version)
			fmt.Printf("Tag:        %s\n", version.Tag)
			fmt.Printf("Branch:     %s\n", version.Branch)
			fmt.Printf("Commit:     %s\n", version.Commit)
			fmt.Printf("Git State:  %s\n", version.GitState)
			fmt.Printf("Date:       %s\n", version.Date)
			fmt.Printf("Go:         %s\n", version.GoVersion)
			fmt.Printf("Compiler:   %s\n", runtime.Compiler)
			fmt.Printf("Build Host: %s\n", version.BuildHost)
			fmt.Printf("Build User: %s\n", version.BuildUser)
		},
	}

	return versionCmd
}
