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
// A caller that also needs to inspect keys a [Config]'s typed fields don't
// surface directly (for example rendering the full effective configuration,
// as "ochami config show" does) can use the Effective-returning variant of
// each loader instead ([LoadEffective], [LoadDefaultsEffective],
// [LoadFileEffective], [LoadMergedEffective]), which return an [Effective]
// view alongside the [Config]:
//
//	eff, cfg, err := config.LoadMergedEffective()
//	val := eff.Get("log.level")
//
// [Effective] composes the merge engine internally without exposing a
// third-party type.
//
// # Editing configuration files
//
// [File] represents a single on-disk configuration file opened for reading
// and editing — the public equivalent of what the "ochami config" and "ochami
// config cluster" commands do. Unlike the loaders above, a [File] never
// applies defaults, so editing never bakes resolved default values into the
// file:
//
//	f, err := config.OpenFile("/path/to/config.yaml")
//	err = f.SetClusterKey("foobar", "cluster.uri", "https://foobar.openchami.cluster")
//	err = f.SetDefaultCluster("foobar")
//
// Each mutating method saves the file on its own; to combine several changes
// into one write, make them inside [File.Update]:
//
//	err = f.Update(func(f *config.File) error {
//		if err := f.SetClusterKey("foobar", "cluster.uri", "https://foobar.openchami.cluster"); err != nil {
//			return err
//		}
//		return f.SetDefaultCluster("foobar")
//	})
//
// # Resolving service URIs
//
// Given a [Config], select a cluster with [Config.GetCluster] and resolve the
// base URI for a particular service with
// [ClusterConfig.GetServiceBaseURI]:
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
// CLI-specific formatting of configuration values for display (used by the
// "ochami config show" commands) lives in an internal package rather than
// here, since it's specific to that command's output conventions rather than
// being generally useful.
package config
