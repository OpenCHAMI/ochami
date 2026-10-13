// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

// Package config exposes koanf on purpose; see TestChecks_FlagFixtures.
package config

import "github.com/knadh/koanf/v2"

// Raw returns the koanf instance behind a configuration.
func Raw() *koanf.Koanf { return nil }
