// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cli

import (
	"errors"
	"time"

	"github.com/spf13/pflag"
)

// positiveDuration is a pflag.Value for a time.Duration that must be greater
// than zero. Its Type is "duration", like pflag's own duration flags, so the
// flag still reads back with FlagSet.GetDuration.
type positiveDuration time.Duration

func (d *positiveDuration) Set(s string) error {
	v, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	if v <= 0 {
		return errors.New(`must be a positive duration (e.g. "30s")`)
	}
	*d = positiveDuration(v)
	return nil
}

func (d *positiveDuration) String() string { return time.Duration(*d).String() }

func (d *positiveDuration) Type() string { return "duration" }

// AddPositiveDurationFlag defines a duration flag on fs, like fs.Duration,
// that rejects values of zero or less. A rejected value is a flag error, so
// it resolves to CodeUsage like any other invalid flag value.
func AddPositiveDurationFlag(fs *pflag.FlagSet, name string, value time.Duration, usage string) {
	d := positiveDuration(value)
	fs.Var(&d, name, usage)
}
