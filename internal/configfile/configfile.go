// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

// Package configfile formats effective ochami configuration for display. It
// is an internal implementation detail of the CLI's read-only "config show"
// and "config cluster show" commands. Reading and editing configuration files
// is public API — see pkg/config's [config.File] — and on-disk mutation
// (add/set/rename/delete a cluster, set/unset a global key) should use that
// instead of anything in this package.
package configfile

import (
	"fmt"
	"strings"

	"github.com/knadh/koanf/providers/structs"
	"github.com/knadh/koanf/v2"
	"gopkg.in/yaml.v3"

	"github.com/openchami/ochami/pkg/config"
)

// kConfig is the strict koanf configuration used by GetConfigCluster, which
// mirrors the public loader by catching incompatible types.
var kConfig = koanf.Conf{Delim: ".", StrictMerge: true}

// ReadConfigWithDefaults is the "effective" single-file loader: it applies
// default global keys and per-cluster defaults, mirroring what the public
// loader produces for a single source. It is used by the read-only "config
// show" and "config cluster show" commands. Cluster order is preserved.
func ReadConfigWithDefaults(path string) (config.Effective, error) {
	if path == "" {
		return config.Effective{}, fmt.Errorf("no configuration file passed")
	}
	eff, _, err := config.LoadEffective([]config.Source{
		{Name: "default", Map: config.DefaultGlobalMap()},
		{Name: "file", Path: path},
	})
	return eff, err
}

// GetConfig returns the config value of key from an effective configuration.
// If key is empty, the whole config is returned. This function only retrieves
// global config options and errors if the key targets an individual cluster
// config (use GetConfigCluster for that).
func GetConfig(eff config.Effective, key string) (any, error) {
	if strings.HasPrefix(key, "clusters") && len(key) > len("clusters") {
		return nil, fmt.Errorf("cannot get individual cluster config with global get command")
	}

	if key != "" {
		return eff.Get(key), nil
	}
	return eff.Raw(), nil
}

// GetConfigString wraps GetConfig and returns a YAML string representation of
// the value of key.
func GetConfigString(eff config.Effective, key string) (string, error) {
	if strings.HasPrefix(key, "clusters.") {
		return "", fmt.Errorf("key cannot be a cluster")
	}
	return marshalConfigValue(key, eff.Get(key))
}

// GetConfigCluster returns the config value of key for a Cluster,
// returning an error if loading the config into koanf errs. If key is empty, the
// whole cluster config is returned.
func GetConfigCluster(cluster config.Cluster, key string) (interface{}, error) {
	var val interface{}
	ko := koanf.NewWithConf(kConfig)
	if err := ko.Load(structs.Provider(cluster, "koanf"), nil); err != nil {
		return nil, fmt.Errorf("failed to load cluster config: %w", err)
	}
	val = ko.Get(key)
	return val, nil
}

// GetConfigClusterString wraps GetConfigCluster and returns a YAML string
// representation of the value of key.
func GetConfigClusterString(cluster config.Cluster, key string) (string, error) {
	val, err := GetConfigCluster(cluster, key)
	if err != nil {
		return "", err
	}
	return marshalConfigValue(key, val)
}

// marshalConfigValue returns a YAML representation of val. A nil value is
// represented by an empty string to preserve the missing-key behavior of the
// config show commands instead of emitting YAML null.
func marshalConfigValue(key string, val any) (string, error) {
	if val == nil {
		return "", nil
	}
	valBytes, err := yaml.Marshal(val)
	if err != nil {
		return "", fmt.Errorf("failed to marshal value for key %q: %w", key, err)
	}
	return string(valBytes), nil
}
