// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package params

import (
	"github.com/openchami/bss/pkg/bssTypes"
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
)

// bootParamFlags names the flags that select components (--xname, --nid,
// --mac) and set boot parameters (--kernel, --initrd, --params).
var bootParamFlags = []string{"xname", "nid", "mac", "kernel", "initrd", "params"}

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

// validateBootParamFlags validates the flags of the add, set, and update
// commands. Without -d, at least one of --xname/--nid/--mac must be specified,
// along with at least one of --kernel/--initrd/--params.
func validateBootParamFlags(cmd *cobra.Command, args []string) error {
	if cmd.Flag("data").Changed {
		warnDataWithFlags(cmd)
	} else if !cli.AnyFlagChanged(cmd, "xname", "nid", "mac") {
		return cli.Errorf(cli.CodeUsage, "expected -d or one of --xname, --nid, or --mac")
	} else if !cli.AnyFlagChanged(cmd, "kernel", "initrd", "params") {
		return cli.Errorf(cli.CodeUsage, "specifying any of --xname, --nid, or --mac also requires specifying at least one of --kernel, --initrd, or --params")
	}

	return nil
}

// warnDataWithFlags warns if any boot parameter flags were passed to cmd along
// with -d.
func warnDataWithFlags(cmd *cobra.Command) {
	// -d/--data trumps all, ignore values of other flags if specified
	if cli.AnyFlagChanged(cmd, bootParamFlags...) {
		cli.LoggerFromCommand(cmd).Warn().Msg("raw data passed, ignoring CLI configuration")
	}
}
