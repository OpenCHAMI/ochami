// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cli

import "github.com/openchami/ochami/internal/log"

// earlyLogger adapts log.EarlyLogger to the config.Logger interface so that
// verbose configuration tracing continues to honor the --verbose flag.
type earlyLogger struct {
	logger log.BasicLogger
}

func (l earlyLogger) Logf(format string, args ...any) {
	l.logger.BasicLogf(format, args...)
}
