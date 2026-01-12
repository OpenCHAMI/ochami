// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package config

import (
	"strconv"
	"strings"
)

// StringToType attempts to convert a string into a rudimentary Go type, simply
// returning as a string if failing to do so. For example, if the string is
// "true" or "false", the value will be returned as a bool.
//
// Currently-supported type conversions are:
//
// - bool
// - int
// - float
func StringToType(s string) any {
	// Try bool
	if b, err := strconv.ParseBool(strings.ToLower(s)); err == nil {
		return b
	}

	// Try integer
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i
	}

	// Try float
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}

	// Fallback to string
	return s
}
