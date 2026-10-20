// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package group

import (
	"github.com/spf13/cobra"

	render_cmd "github.com/openchami/ochami/cmd/cloud_init/render"
)

func newCmdGroupRender() *cobra.Command {
	return render_cmd.NewGroupCmd()
}
