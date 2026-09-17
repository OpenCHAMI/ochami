// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

// Package configfile implements the on-disk reading, writing, and mutation of
// ochami configuration files. It is an internal implementation detail of the
// CLI's "config" commands and is intentionally not part of the public API: it
// exposes koanf and file-editing semantics that external consumers should not
// depend on. Consumers that only need to load and inspect effective
// configuration should use the public pkg/config package instead.
package configfile

import (
	"fmt"
	"os"
	"strings"

	kyaml "github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/structs"
	"github.com/knadh/koanf/v2"
	"gopkg.in/yaml.v3"

	"github.com/openchami/ochami/pkg/config"
)

var (
	// configParser is the koanf YAML parser used to read and write config
	// files.
	configParser = kyaml.Parser()

	// kConfig is the strict koanf configuration used by the "effective"
	// single-file loader (ReadConfigWithDefaults), which mirrors the
	// public loader by catching incompatible types between merged sources.
	kConfig = koanf.Conf{Delim: ".", StrictMerge: true}

	// kConfigRaw is the non-strict koanf configuration used by the
	// edit/modify loaders (ReadConfig and friends). These wholesale-replace
	// keys such as "clusters" (e.g. swapping a []interface{} for a
	// []map[string]any), which StrictMerge disallows, so strict merging
	// must stay disabled on this path.
	kConfigRaw = koanf.Conf{Delim: ".", StrictMerge: false}
)

// ReadConfig opens the config file at path and loads it into koanf without
// applying defaults or merging. It is the raw loader used by the edit/modify
// commands. If path is empty or the file cannot be loaded, an error is
// returned.
func ReadConfig(path string) (*koanf.Koanf, error) {
	if path == "" {
		return nil, fmt.Errorf("no configuration file passed")
	}

	ko := koanf.NewWithConf(kConfigRaw)
	if err := ko.Load(file.Provider(path), configParser); err != nil {
		return ko, fmt.Errorf("failed to load config file %s: %w", path, err)
	}

	return ko, nil
}

// ReadConfigWithDefaults is the "effective" single-file loader: like
// ReadConfig, but it applies default global keys and per-cluster defaults,
// mirroring what the public loader produces for a single source. It is used by
// the read-only "config show" and "config cluster show" commands. Cluster
// order is preserved.
func ReadConfigWithDefaults(path string) (*koanf.Koanf, error) {
	if path == "" {
		return nil, fmt.Errorf("no configuration file passed")
	}

	ko := koanf.NewWithConf(kConfig)

	if err := ko.Load(confmap.Provider(config.DefaultGlobalMap(), "."), nil); err != nil {
		return ko, fmt.Errorf("failed to load defaults for config file %s: %w", path, err)
	}

	// Load the file into a separate instance first so explicit null values
	// for required global scalars can be rejected with a clean error before
	// StrictMerge would otherwise fail with a cryptic type mismatch.
	fileKo := koanf.NewWithConf(kConfig)
	if err := fileKo.Load(file.Provider(path), configParser); err != nil {
		return ko, err
	}
	if err := config.CheckGlobalNulls(fileKo); err != nil {
		return ko, err
	}
	if err := ko.Merge(fileKo); err != nil {
		return ko, err
	}

	if err := applyClusterDefaults(ko); err != nil {
		return ko, err
	}

	// Validate the fully-merged (effective) config.
	if err := config.ValidateConfig(ko); err != nil {
		return ko, err
	}

	return ko, nil
}

// EffectiveKoanf builds the effective merged koanf across the given optional
// files (in ascending priority), applying default global keys and per-cluster
// defaults and preserving cluster order. Missing files are skipped. It mirrors
// what the public loader produces and is used to render the merged view for the
// "config show" commands when no specific source flag is given.
func EffectiveKoanf(paths ...string) (*koanf.Koanf, error) {
	ko := koanf.NewWithConf(kConfig)
	if err := ko.Load(confmap.Provider(config.DefaultGlobalMap(), "."), nil); err != nil {
		return ko, fmt.Errorf("failed to load config defaults: %w", err)
	}

	for _, path := range paths {
		fileKo := koanf.NewWithConf(kConfig)
		if err := fileKo.Load(file.Provider(path), configParser); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return ko, fmt.Errorf("failed to load config file %s: %w", path, err)
		}
		if err := config.CheckGlobalNulls(fileKo); err != nil {
			return ko, err
		}
		if err := ko.Merge(fileKo); err != nil {
			return ko, err
		}
	}

	if err := applyClusterDefaults(ko); err != nil {
		return ko, err
	}
	if err := config.ValidateConfig(ko); err != nil {
		return ko, err
	}

	return ko, nil
}

// applyClusterDefaults rewrites the "clusters" key of ko into a merged,
// default-applied, order-preserving slice. It is shared by the effective
// loaders.
func applyClusterDefaults(ko *koanf.Koanf) error {
	var kClusterSlice []map[string]any
	if err := ko.Unmarshal("clusters", &kClusterSlice); err != nil {
		return fmt.Errorf("unable to unmarshal cluster configs: %w", err)
	}

	clusterAcc := config.NewClusterAccumulator(kConfig)
	for i, cluster := range kClusterSlice {
		name, ok := cluster["name"].(string)
		if !ok || name == "" {
			return fmt.Errorf("cluster #%d is missing a name", i)
		}
		switch cls := cluster["cluster"].(type) {
		case map[string]any:
			if err := clusterAcc.Add(name, cls); err != nil {
				return fmt.Errorf("unable to merge cluster '%s': %w", name, err)
			}
		case nil:
			if err := clusterAcc.Add(name, map[string]any{}); err != nil {
				return fmt.Errorf("unable to merge cluster '%s': %w", name, err)
			}
		default:
			return fmt.Errorf("cluster '%s' is not a map", name)
		}
	}

	// Replace the clusters key with the merged, default-applied, ordered
	// slice. Delete first because StrictMerge disallows overwriting the
	// existing []interface{} value with a []map[string]any via Set.
	ko.Delete("clusters")
	if err := ko.Set("clusters", clusterAcc.Slice()); err != nil {
		return fmt.Errorf("unable to set clusters: %w", err)
	}
	return nil
}

// WriteConfig marshals the koanf instance to YAML and writes it to path,
// preserving the file's existing mode if it exists.
func WriteConfig(path string, k *koanf.Koanf) error {
	if path == "" {
		return fmt.Errorf("no configuration file path passed")
	}

	c, err := k.Marshal(configParser)
	if err != nil {
		return fmt.Errorf("failed to marshal config for writing: %w", err)
	}

	// Get mode if file exists
	var fmode os.FileMode = 0o644
	if finfo, err := os.Stat(path); err == nil {
		fmode = finfo.Mode()
	}

	if err := os.WriteFile(path, c, fmode); err != nil {
		return fmt.Errorf("failed to write config to file %s: %w", path, err)
	}

	return nil
}

// ModifyConfig modifies a single key in a config file. It reads the config into
// koanf, sets the key, and writes it back. An error is returned if reading,
// setting, or writing fails.
func ModifyConfig(path, key string, value interface{}) error {
	ko, err := ReadConfig(path)
	if err != nil {
		return fmt.Errorf("failed to read %s for modification: %w", path, err)
	}

	if err := ko.Set(key, value); err != nil {
		return fmt.Errorf("failed to set key %s to value %v: %w", key, value, err)
	}

	if err := WriteConfig(path, ko); err != nil {
		return fmt.Errorf("failed to write modified config to %s: %w", path, err)
	}

	return nil
}

// ModifyConfigCluster sets or modifies a single key for a single cluster,
// identified by name, in the config file at path. If dflt is true,
// default-cluster is set to the specified cluster. If the cluster does not
// already exist, it is added. If key is "name", the cluster is renamed but
// setting the name to an existing cluster name is not allowed. If the default
// cluster's name is changed, default-cluster is set to the new name, regardless
// of dflt.
func ModifyConfigCluster(path, cluster, key string, dflt bool, value any) error {
	ko, err := ReadConfig(path)
	if err != nil {
		return fmt.Errorf("failed to read %s for modification: %w", path, err)
	}

	delim := ko.Delim()
	if strings.Contains(cluster, delim) {
		return fmt.Errorf("cluster name '%s' contains delimiter character '%s'", cluster, delim)
	}

	var clusters []map[string]any
	err = ko.Unmarshal("clusters", &clusters)
	if err != nil {
		return fmt.Errorf("unable to unmarshal clusters: %w", err)
	}

	// Make sure that if setting the cluster name, a cluster with that name
	// doesn't already exist.
	if key == "name" {
		for _, cl := range clusters {
			if cl["name"] == value.(string) {
				return fmt.Errorf("cluster with name %q already exists", cl["name"])
			}
		}
	}

	// Determine if a new cluster needs to be added or an existing cluster
	// needs to be modified.
	cidx := -1
	for i, cl := range clusters {
		if cl["name"] == cluster {
			cidx = i
			break
		}
	}

	// Using -1 as a sentinel value to indicate creation is required
	if cidx == -1 {
		cidx = len(clusters)
		clusters = append(clusters, map[string]any{})
		if key == "name" {
			clusters[cidx]["name"] = value
		} else {
			clusters[cidx]["name"] = cluster
		}
	}

	kc := koanf.NewWithConf(kConfigRaw)
	err = kc.Load(confmap.Provider(clusters[cidx], ""), nil)
	if err != nil {
		return fmt.Errorf("unable to load cluster '%s' from config '%s': %w", cluster, path, err)
	}

	err = kc.Set(key, value)
	if err != nil {
		return fmt.Errorf("unable to modify config value '%s' in cluster '%s': %w", key, cluster, err)
	}

	clusters[cidx] = kc.Raw()
	err = ko.Set("clusters", clusters)
	if err != nil {
		return fmt.Errorf("unable to re-set clusters: %w", err)
	}

	defaultCluster := ko.String("default-cluster")

	// If default is set, set default-cluster to cluster name.
	// Also do it if the default-cluster was renamed (to reflect the new name)
	if dflt || (key == "name" && defaultCluster == cluster) {
		// If key was "name", set default-cluster to "name"
		// instead of cluster specified in arg.
		if key == "name" {
			s, ok := value.(string)
			if !ok || s == "" {
				err = fmt.Errorf("value '%v' is not a string or is an empty string", value)
			} else {
				err = ko.Set("default-cluster", s)
			}
		} else {
			err = ko.Set("default-cluster", cluster)
		}
		if err != nil {
			return fmt.Errorf("failed to set default-cluster: %w", err)
		}
	}

	if err := WriteConfig(path, ko); err != nil {
		return fmt.Errorf("failed to write modified config to %s: %w", path, err)
	}

	return nil
}

// DeleteConfig deletes a key from a config file. An error is returned if the
// key is not found or if reading or writing fails.
func DeleteConfig(path, key string) error {
	ko, err := ReadConfig(path)
	if err != nil {
		return fmt.Errorf("failed to read %s for deletion: %w", path, err)
	}

	if !ko.Exists(key) {
		return fmt.Errorf("key '%s' does not exist", key)
	}

	ko.Delete(key)

	if err := WriteConfig(path, ko); err != nil {
		return fmt.Errorf("failed to write modified config to %s: %w", path, err)
	}

	return nil
}

// DeleteConfigCluster deletes a key from the specified cluster in a config file.
// An error is returned if the cluster doesn't exist, if "name" is the key, or if
// reading or writing fails.
func DeleteConfigCluster(path, cluster, key string) error {
	ko, err := ReadConfig(path)
	if err != nil {
		return fmt.Errorf("failed to read %s for modification: %w", path, err)
	}

	if key == "name" {
		return fmt.Errorf("cannot unset name of cluster")
	}

	delim := ko.Delim()
	if strings.Contains(cluster, delim) {
		return fmt.Errorf("cluster name '%s' contains delimiter character '%s'", cluster, delim)
	}

	var clusters []map[string]any
	err = ko.Unmarshal("clusters", &clusters)
	if err != nil {
		return fmt.Errorf("unable to unmarshal clusters: %w", err)
	}

	found := false
	for i := 0; i < len(clusters); i++ {
		if clusters[i]["name"] == cluster {
			ck := koanf.NewWithConf(kConfigRaw)
			err = ck.Load(confmap.Provider(clusters[i], ""), nil)
			if err != nil {
				return fmt.Errorf("unable to load cluster config from map: %w", err)
			}
			if !ck.Exists(key) {
				return fmt.Errorf("key '%s' doesn't exist", key)
			}
			ck.Delete(key)
			clusters[i] = ck.Raw()
			found = true
		}
	}

	if !found {
		return fmt.Errorf("cluster '%s' doesn't exist", cluster)
	}

	if err := ko.Set("clusters", clusters); err != nil {
		return fmt.Errorf("unable to re-set clusters: %w", err)
	}

	if err := WriteConfig(path, ko); err != nil {
		return fmt.Errorf("failed to write modified config to %s: %w", path, err)
	}

	return nil
}

// GetConfig returns the config value of key from a koanf instance. If key is
// empty, the whole config is returned. This function only retrieves global
// config options and errors if the key targets an individual cluster config
// (use GetConfigCluster for that).
func GetConfig(ko *koanf.Koanf, key string) (any, error) {
	if strings.HasPrefix(key, "clusters") && len(key) > len("clusters") {
		return nil, fmt.Errorf("cannot get individual cluster config with global get command")
	}

	var val any
	if key != "" {
		val = ko.Get(key)
	} else {
		val = ko.Raw()
	}
	return val, nil
}

// GetConfigFromFile is like GetConfig except that it reads the config from the
// file at path.
func GetConfigFromFile(path, key string) (any, error) {
	ko, err := ReadConfig(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	return GetConfig(ko, key)
}

// GetConfigString wraps GetConfig and returns a YAML string representation of
// the value of key.
func GetConfigString(ko *koanf.Koanf, key string) (string, error) {
	if strings.HasPrefix(key, "clusters.") {
		return "", fmt.Errorf("key cannot be a cluster")
	}
	return marshalConfigValue(key, ko.Get(key))
}

// GetConfigStringFromFile is like GetConfigString except that it wraps
// GetConfigFromFile.
func GetConfigStringFromFile(path, key string) (string, error) {
	ko, err := ReadConfig(path)
	if err != nil {
		return "", fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	return GetConfigString(ko, key)
}

// GetConfigCluster returns the config value of key for a ConfigCluster,
// returning an error if loading the config into koanf errs. If key is empty, the
// whole cluster config is returned.
func GetConfigCluster(cluster config.ConfigCluster, key string) (interface{}, error) {
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
func GetConfigClusterString(cluster config.ConfigCluster, key string) (string, error) {
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
