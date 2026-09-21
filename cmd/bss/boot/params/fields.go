// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package params

import (
	"github.com/openchami/bss/pkg/bssTypes"
	"github.com/spf13/cobra"
)

// bootParamFields holds the values of the flags that select components and set
// boot parameters, which the add, set, update, and delete commands share.
type bootParamFields struct {
	Xname  []string
	Mac    []string
	Nid    []int32
	Kernel string
	Initrd string
	Params string
}

// readBootParamFlags returns the values of the boot parameter flags passed to
// cmd, leaving the fields of flags that weren't passed empty. Each flag is
// registered on cmd with the matching type, so the Get* calls can't fail and
// their errors are ignored.
func readBootParamFlags(cmd *cobra.Command) bootParamFields {
	var f bootParamFields
	if cmd.Flag("xname").Changed {
		f.Xname, _ = cmd.Flags().GetStringSlice("xname")
	}
	if cmd.Flag("mac").Changed {
		f.Mac, _ = cmd.Flags().GetStringSlice("mac")
	}
	if cmd.Flag("nid").Changed {
		f.Nid, _ = cmd.Flags().GetInt32Slice("nid")
	}
	if cmd.Flag("kernel").Changed {
		f.Kernel, _ = cmd.Flags().GetString("kernel")
	}
	if cmd.Flag("initrd").Changed {
		f.Initrd, _ = cmd.Flags().GetString("initrd")
	}
	if cmd.Flag("params").Changed {
		f.Params, _ = cmd.Flags().GetString("params")
	}
	return f
}

// applyBootParamFlags sets each field of bp whose flag was passed to cmd, so
// the flags override the matching fields of a payload already read into bp.
func applyBootParamFlags(cmd *cobra.Command, bp *bssTypes.BootParams, f bootParamFields) {
	if cmd.Flag("xname").Changed {
		bp.Hosts = f.Xname
	}
	if cmd.Flag("mac").Changed {
		bp.Macs = f.Mac
	}
	if cmd.Flag("nid").Changed {
		bp.Nids = f.Nid
	}
	if cmd.Flag("kernel").Changed {
		bp.Kernel = f.Kernel
	}
	if cmd.Flag("initrd").Changed {
		bp.Initrd = f.Initrd
	}
	if cmd.Flag("params").Changed {
		bp.Params = f.Params
	}
}
