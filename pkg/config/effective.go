// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package config

import (
	"fmt"

	"github.com/knadh/koanf/v2"
)

// Effective is a read-only view of an effective (defaults-applied, merged)
// configuration, as produced by LoadEffective and its
// LoadDefaultsEffective/LoadFileEffective/LoadMergedEffective variants. It
// composes the underlying merge engine without exposing it, so callers that
// need to inspect keys a Config's typed fields don't surface directly (for
// example "ochami config show") can do so without depending on a third-party
// merge library.
//
// The zero value of Effective is valid and behaves as an empty
// configuration.
type Effective struct {
	ko *koanf.Koanf
}

// Get returns the value at key, or nil if it isn't set. An empty key returns
// the same result as Raw.
func (e Effective) Get(key string) any {
	if e.ko == nil {
		return nil
	}
	return e.ko.Get(key)
}

// Raw returns the entire effective configuration as a nested map.
func (e Effective) Raw() map[string]any {
	if e.ko == nil {
		return nil
	}
	return e.ko.Raw()
}

// Unmarshal decodes the value at key into out. An empty key unmarshals the
// entire effective configuration.
func (e Effective) Unmarshal(key string, out any) error {
	if e.ko == nil {
		return fmt.Errorf("no configuration loaded")
	}
	return e.ko.Unmarshal(key, out)
}
