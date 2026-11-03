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

// applyBootParamFlags sets each field of bp whose flag was passed to cmd.
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

// readBootParams returns the boot parameters that rt's command cmd should send:
// those in the -d payload, which can hold one boot parameters object or an
// array of them, or, without -d, one built from the flag values in f. It
// rejects a payload with no boot parameters and any malformed MAC address,
// whether it came from --mac or the payload.
func readBootParams(cmd *cobra.Command, rt *cli.Runtime, f bootParamFields) ([]bssTypes.BootParams, error) {
	var bps []bssTypes.BootParams
	if cmd.Flag("data").Changed {
		if err := cli.HandlePayloadSlice(rt, cmd, &bps); err != nil {
			return nil, err
		}
		if len(bps) == 0 {
			return nil, cli.Errorf(cli.CodeUsage, "payload contained no boot parameters")
		}
	} else {
		var bp bssTypes.BootParams
		applyBootParamFlags(cmd, &bp, f)
		bps = append(bps, bp)
	}

	// CheckMacs accepts an empty list
	for _, bp := range bps {
		if err := bp.CheckMacs(); err != nil {
			return nil, cli.Errorf(cli.CodeUsage, "invalid mac(s): %w", err)
		}
	}

	return bps, nil
}

// validateBootParamFlags validates the flags of the add, set, and update
// commands. Without -d, at least one of --xname/--nid/--mac must be specified,
// along with at least one of --kernel/--initrd/--params.
func validateBootParamFlags(cmd *cobra.Command, args []string) error {
	// With -d, the flag groups registered by markDataExclusive reject the
	// other flags
	if cmd.Flag("data").Changed {
		return nil
	}
	if !cli.AnyFlagChanged(cmd, "xname", "nid", "mac") {
		return cli.Errorf(cli.CodeUsage, "expected -d or one of --xname, --nid, or --mac")
	} else if !cli.AnyFlagChanged(cmd, "kernel", "initrd", "params") {
		return cli.Errorf(cli.CodeUsage, "specifying any of --xname, --nid, or --mac also requires specifying at least one of --kernel, --initrd, or --params")
	}

	return nil
}

// markDataExclusive marks each boot parameter flag of cmd as mutually
// exclusive with -d, so that a request comes either wholly from the payload or
// wholly from the flags.
func markDataExclusive(cmd *cobra.Command) {
	for _, f := range bootParamFlags {
		cmd.MarkFlagsMutuallyExclusive(f, "data")
	}
}
