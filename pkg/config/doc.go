// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

// Package config provides loading, merging, and inspection of ochami
// configuration for the ochami CLI and for external tools.
//
// The package is deliberately state-independent: the loaders return a fresh
// [Config] value and never read or mutate package-global state. This makes it
// safe to build multiple independent configurations concurrently and easy to
// use in tests.
//
// # Loading configuration
//
// Most callers want either the standard CLI precedence (built-in defaults, then
// the optional system file, then the optional user file) or a single explicit
// file:
//
//	// Standard precedence (system + user, both optional):
//	cfg, err := config.LoadMerged()
//
//	// A single explicit file (required):
//	cfg, err := config.LoadFile("/path/to/config.yaml")
//
//	// Only the built-in defaults (e.g. --ignore-config):
//	cfg, err := config.LoadDefaults()
//
// For full control over the sources and their precedence, use [Load] with a
// slice of [Source]. Built-in defaults are always applied first (lowest
// priority); later sources win on conflicts. Cluster configurations are merged
// by name with per-cluster defaults applied, and cluster order is preserved by
// first appearance.
//
// # Resolving service URIs
//
// Given a [Config], select a cluster with [Config.GetCluster] and resolve the
// base URI for a particular service with
// [ConfigClusterConfig.GetServiceBaseURI]:
//
//	cluster, err := cfg.GetCluster("foobar")
//	if err != nil {
//		// handle ErrUnknownCluster
//	}
//	uri, err := cluster.Cluster.GetServiceBaseURI(config.ServiceSMD)
//
// # Errors
//
// Loading and resolution return typed errors (for example [ErrUnknownCluster],
// [ErrMissingURI], and [ErrInvalidConfigVal]) that callers may match with
// errors.As.
//
// On-disk editing of configuration files (used by the "ochami config" commands)
// is intentionally not part of this package; it lives in an internal package
// because it exposes file-format and koanf details that external consumers
// should not depend on.
package config
