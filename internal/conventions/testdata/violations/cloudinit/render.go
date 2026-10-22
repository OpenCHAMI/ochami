// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

// Package cloudinit deliberately exposes third-party rendering types.
package cloudinit

import (
	gj "github.com/jryberg/gojinja"
	"gopkg.in/yaml.v3"
)

// New exposes the rendering engine deliberately.
func New() *gj.Environment { return nil }

// Document exposes YAML nodes deliberately.
type Document struct {
	Node yaml.Node
}
