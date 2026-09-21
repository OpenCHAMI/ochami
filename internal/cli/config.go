// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cli

import (
	"github.com/openchami/ochami/internal/log"
	"github.com/openchami/ochami/pkg/config"
)

// earlyLogger adapts log.EarlyLogger to the config.Logger interface so that
// verbose configuration tracing continues to honor the --verbose flag.
type earlyLogger struct {
	logger log.BasicLogger
}

func (l earlyLogger) Logf(format string, args ...any) {
	l.logger.BasicLogf(format, args...)
}

// configLoadOpts returns the LoadOptions a Runtime's config loaders should
// use. A logger is attached only when --verbose is actually on, so that
// pkg/config's trace path (which enumerates and formats every config key) is
// skipped entirely on the common, non-verbose invocation instead of being
// built and then discarded.
func (rt *Runtime) configLoadOpts() []config.LoadOption {
	if !rt.EarlyVerbose {
		return nil
	}
	return []config.LoadOption{
		config.WithLogger(earlyLogger{logger: log.NewBasicLogger(rt.Ios.Err(), rt.EarlyVerbose, "ochami")}),
	}
}
