// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	kyaml "github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/structs"
	"github.com/knadh/koanf/v2"
	"go.yaml.in/yaml/v3"

	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/format"
)

type ServiceName string

const (
	ServiceBoot      ServiceName = "boot-service"
	ServiceBSS       ServiceName = "bss"
	ServiceCloudInit ServiceName = "cloud-init"
	ServiceMetadata  ServiceName = "metadata-service"
	ServicePCS       ServiceName = "pcs"
	ServiceSMD       ServiceName = "smd"
	ServiceRCS       ServiceName = "rcs"
)

const (
	DefaultBasePathBootService     = "/boot-service"
	DefaultBasePathBSS             = "/boot/v1"
	DefaultBasePathCloudInit       = "/cloud-init"
	DefaultBasePathMetadataService = "/metadata-service"
	DefaultBasePathPCS             = "/"
	DefaultBasePathSMD             = "/hsm/v2"
	DefaultBasePathRCS             = "/remote-console"

	SystemConfigFile = "/etc/ochami/config.yaml"
)

// Default configuration values if either no configuration files exist or the
// configuration files don't contain values for items that need them.

var DefaultConfigMap = map[string]any{
	"log.format":            "rfc3339",
	"log.level":             "warning",
	"log.color":             "auto",
	"timeout":               "30s",
	"default-input-format":  "json",
	"default-output-format": "json",
}

var DefaultClusterConfigMap = map[string]any{
	"enable-auth": true,
}

var (
	GlobalConfig   = Config{}   // Global config struct
	GlobalKoanf    *koanf.Koanf // Koanf instance for gobal config struct
	UserConfigFile string

	// Koanf YAML parser provider
	configParser = kyaml.Parser()

	// Global koanf struct configuration
	kConfig = koanf.Conf{Delim: "."}
)

// Config represents the structure of a configuration file.
type Config struct {
	Log                 ConfigLog         `koanf:"log"`
	Timeout             time.Duration     `koanf:"timeout"`
	DefaultCluster      string            `koanf:"default-cluster"`
	DefaultInputFormat  format.DataFormat `koanf:"default-input-format"`
	DefaultOutputFormat format.DataFormat `koanf:"default-output-format"`
	Clusters            []ConfigCluster   `koanf:"clusters"`
}

// GetCluster searches for a cluster by name and returns it if it exists in the
// config. If not, an ErrUnknownCluster is returned.
func (c Config) GetCluster(name string) (ConfigCluster, error) {
	for _, cl := range c.Clusters {
		if cl.Name == name {
			return cl, nil
		}
	}
	return ConfigCluster{}, ErrUnknownCluster{ClusterName: name}
}

type ConfigLog struct {
	Format string `koanf:"format"`
	Level  string `koanf:"level"`
	Color  string `koanf:"color"`
}

// ConfigCluster is a "wrapper" around an individual cluster configuration. It
// contains the cluster's name, as well as the actual configuration structure.
type ConfigCluster struct {
	Name    string              `koanf:"name"`
	Cluster ConfigClusterConfig `koanf:"cluster"`
}

// ConfigClusterConfig is the actual structure for an individual cluster
// configuration.
type ConfigClusterConfig struct {
	URI             string                       `koanf:"uri"`
	BootService     ConfigClusterBootService     `koanf:"boot-service"`
	BSS             ConfigClusterBSS             `koanf:"bss"`
	CloudInit       ConfigClusterCloudInit       `koanf:"cloud-init"`
	MetadataService ConfigClusterMetadataService `koanf:"metadata-service"`
	PCS             ConfigClusterPCS             `koanf:"pcs"`
	SMD             ConfigClusterSMD             `koanf:"smd"`
	RCS             ConfigClusterRCS             `koanf:"rcs"`
	EnableAuth      bool                         `koanf:"enable-auth"`
}

// ConfigClusterBootService represents configuration specifically for the
// boot service.
type ConfigClusterBootService struct {
	APIVersion string `koanf:"api-version"`
	URI        string `koanf:"uri"`
}

// ConfigClusterBSS represents configuration specifically for the Boot Script
// Service.
type ConfigClusterBSS struct {
	URI string `koanf:"uri"`
}

// ConfigClusterCloudInit represents configuration specifically for the
// cloud-init service.
type ConfigClusterCloudInit struct {
	URI string `koanf:"uri"`
}

// ConfigClusterMetadataService represents configuration specifically for the
// metadata service.
type ConfigClusterMetadataService struct {
	APIVersion string `koanf:"api-version"`
	URI        string `koanf:"uri"`
}

// ConfigClusterRCS represents configuration specifically for the Remote Console
// Service.
type ConfigClusterRCS struct {
	URI string `koanf:"uri"`
}

// ConfigClusterPCS represents configuration specifically for the Power Control
// Service.
type ConfigClusterPCS struct {
	URI string `koanf:"uri"`
}

// ConfigClusterSMD represents configuration specifically for the State
// Management Database service.
type ConfigClusterSMD struct {
	URI string `koanf:"uri"`
}

// MergeURIConfig takes a ConfigClusterConfig and returns a ConfigClusterConfig
// with updated values, leaving the member one unmodified. If any of the URI
// attributes are not blank in the passed ConfigClusterConfig, those attributes
// are updated in the one returned. Otherwise, the old values are left alone.
func (ccc *ConfigClusterConfig) MergeURIConfig(c ConfigClusterConfig) ConfigClusterConfig {
	compare := func(oldStr, newStr string) string {
		if newStr != "" {
			return newStr
		}
		return oldStr
	}
	newCCC := ConfigClusterConfig{URI: compare(ccc.URI, c.URI)}
	if ccc.BSS == (ConfigClusterBSS{}) {
		newCCC.BSS = ConfigClusterBSS{URI: c.BSS.URI}
	} else {
		newCCC.BSS.URI = compare(ccc.BSS.URI, c.BSS.URI)
	}
	if ccc.BootService == (ConfigClusterBootService{}) {
		newCCC.BootService = ConfigClusterBootService{URI: c.BootService.URI}
	} else {
		newCCC.BootService.URI = compare(ccc.BootService.URI, c.BootService.URI)
	}
	if ccc.CloudInit == (ConfigClusterCloudInit{}) {
		newCCC.CloudInit = ConfigClusterCloudInit{URI: c.CloudInit.URI}
	} else {
		newCCC.CloudInit.URI = compare(ccc.CloudInit.URI, c.CloudInit.URI)
	}
	if ccc.PCS == (ConfigClusterPCS{}) {
		newCCC.PCS = ConfigClusterPCS{URI: c.PCS.URI}
	} else {
		newCCC.PCS.URI = compare(ccc.PCS.URI, c.PCS.URI)
	}
	if ccc.MetadataService == (ConfigClusterMetadataService{}) {
		newCCC.MetadataService = ConfigClusterMetadataService{URI: c.MetadataService.URI}
	} else {
		newCCC.MetadataService.URI = compare(ccc.MetadataService.URI, c.MetadataService.URI)
	}
	if ccc.SMD == (ConfigClusterSMD{}) {
		newCCC.SMD = ConfigClusterSMD{URI: c.SMD.URI}
	} else {
		newCCC.SMD.URI = compare(ccc.SMD.URI, c.SMD.URI)
	}
	if ccc.RCS == (ConfigClusterRCS{}) {
		newCCC.RCS = ConfigClusterRCS{URI: c.RCS.URI}
	} else {
		newCCC.RCS.URI = compare(ccc.RCS.URI, c.RCS.URI)
	}

	return newCCC
}

// GetServiceBaseURI returns a URI string for the service identified by svcName
// based on URI values set in the ConfigClusterConfig. At least one of URI or
// the URI for a service must be set in the ConfigClusterConfig, otherwise an
// ErrMissingURI error is returned. If svcName is unknown, an ErrUnknownService
// is returned. If the cluster URI is invalid or the service URI is invalid, an
// ErrInvalidURI or ErrInvalidServiceURI is returned, respectively.
//
// The cluster URI must be an absolute URI: proto://host[:port][/path]
// The service URI can be a relative path (/path) or an absolute URI.
func (ccc *ConfigClusterConfig) GetServiceBaseURI(svcName ServiceName) (string, error) {
	var (
		serviceBaseURI string
		uri            *url.URL
	)
	// If the cluster's URI is set, parse and verify it.
	if ccc.URI != "" {
		var err error
		uri, err = url.Parse(ccc.URI)
		if err != nil {
			return "", ErrInvalidURI{Err: err}
		}
		if uri.Opaque != "" || uri.Scheme == "" || uri.Host == "" {
			return "", ErrInvalidURI{Err: fmt.Errorf("unknown URI format (must be \"proto://host[:port][/path]\")")}
		}
		serviceBaseURI = uri.String()
	}

	// Parse service URI for ConfigClusterConfig field based on passed
	// ServiceName.
	var svcURI *url.URL
	var err error
	switch svcName {
	case ServiceBoot:
		if ccc.URI == "" && ccc.BootService.URI == "" {
			return "", ErrMissingURI{Service: svcName}
		}
		if ccc.BootService.URI != "" {
			svcURI, err = url.Parse(ccc.BootService.URI)
		} else {
			svcURI, err = url.Parse(DefaultBasePathBootService)
		}
	case ServiceBSS:
		if ccc.URI == "" && ccc.BSS.URI == "" {
			return "", ErrMissingURI{Service: svcName}
		}
		if ccc.BSS.URI != "" {
			svcURI, err = url.Parse(ccc.BSS.URI)
		} else {
			svcURI, err = url.Parse(DefaultBasePathBSS)
		}
	case ServiceCloudInit:
		if ccc.URI == "" && ccc.CloudInit.URI == "" {
			return "", ErrMissingURI{Service: svcName}
		}
		if ccc.CloudInit.URI != "" {
			svcURI, err = url.Parse(ccc.CloudInit.URI)
		} else {
			svcURI, err = url.Parse(DefaultBasePathCloudInit)
		}
	case ServiceMetadata:
		if ccc.URI == "" && ccc.MetadataService.URI == "" {
			return "", ErrMissingURI{Service: svcName}
		}
		if ccc.MetadataService.URI != "" {
			svcURI, err = url.Parse(ccc.MetadataService.URI)
		} else {
			svcURI, err = url.Parse(DefaultBasePathMetadataService)
		}
	case ServicePCS:
		if ccc.URI == "" && ccc.PCS.URI == "" {
			return "", ErrMissingURI{Service: svcName}
		}
		if ccc.PCS.URI != "" {
			svcURI, err = url.Parse(ccc.PCS.URI)
		} else {
			svcURI, err = url.Parse(DefaultBasePathPCS)
		}
	case ServiceSMD:
		if ccc.URI == "" && ccc.SMD.URI == "" {
			return "", ErrMissingURI{Service: svcName}
		}
		if ccc.SMD.URI != "" {
			svcURI, err = url.Parse(ccc.SMD.URI)
		} else {
			svcURI, err = url.Parse(DefaultBasePathSMD)
		}
	case ServiceRCS:
		if ccc.URI == "" && ccc.RCS.URI == "" {
			return "", ErrMissingURI{Service: svcName}
		}
		if ccc.RCS.URI != "" {
			svcURI, err = url.Parse(ccc.RCS.URI)
		} else {
			svcURI, err = url.Parse(DefaultBasePathRCS)
		}
	default:
		return "", ErrUnknownService{Service: string(svcName)}
	}
	if err != nil {
		return "", ErrInvalidServiceURI{Service: svcName, Err: err}
	}

	// Once parsed (if not nil), verify that the service URI is either a
	// valid absolute URI or a valid relative path.
	if svcURI != nil {
		if svcURI.IsAbs() {
			// Service URI is an absolute URI. Override API URI.
			if svcURI.Opaque != "" || svcURI.Scheme == "" {
				return "", ErrInvalidServiceURI{Service: svcName, Err: fmt.Errorf("unknown URI format (must be \"/path\" or \"proto://host[:port][/path]\")")}
			}
			serviceBaseURI = svcURI.String()
		} else if svcURI.Path != "" {
			// Service URI is a relative path. Append it to API URI.
			var newURI *url.URL
			if uri != nil {
				newURI = uri.JoinPath(svcURI.Path)
			} else {
				return "", ErrInvalidServiceURI{Service: svcName, Err: fmt.Errorf("%s.uri is a relative path but cluster.uri not set", svcName)}
			}
			serviceBaseURI = newURI.String()
		} else {
			return "", ErrInvalidServiceURI{Service: svcName, Err: fmt.Errorf("%s.uri is neither an absolute URI nor has a path component", svcName)}
		}
	}

	return serviceBaseURI, nil
}

// RemoveFromSlice removes an element from a slice and returns the resulting
// slice. The element to be removed is identified by its index in the slice.
func RemoveFromSlice[T any](slice []T, index int) []T {
	slice[len(slice)-1], slice[index] = slice[index], slice[len(slice)-1]
	return slice[:len(slice)-1]
}

// LoadGlobalConfigMerged populates the GlobalConfig Config structure and
// GlobalKoanf structure with a configuration that is a merge of, in ascending
// order of priority (higher is more priority:
//
// 1. DefaultConfig
// 2. System config file (/etc/ochami/config.yaml)
// 3. User config file (~/.config/ochami/config.yaml)
//
// If any of the system or user config file fails to load, it is skipped in the
// merging.
func LoadGlobalConfigMerged() error {
	log.EarlyLogger.BasicLog("early verbose log messages activated")

	var err error
	k := koanf.NewWithConf(kConfig)
	GlobalKoanf = k

	UserConfigFile, err = getUserConfigPath()
	if err != nil {
		return err
	}

	type configLoader struct {
		name     string
		provider koanf.Provider
		parser   koanf.Parser
	}

	configsToLoad := []configLoader{
		{"default", confmap.Provider(DefaultConfigMap, "."), nil},
		// {"system", file.Provider(SystemConfigFile), configParser},
		{"system", file.Provider("doc/config.example.yaml"), configParser},
		{"user", file.Provider(UserConfigFile), configParser},
	}

	// For merging purposes, maps name to key-value pairs
	clusterMap := map[string]*koanf.Koanf{}
	for _, c := range configsToLoad {
		k2 := koanf.NewWithConf(kConfig)
		err = k2.Load(c.provider, c.parser)
		if errors.Is(err, os.ErrNotExist) { // This an error we can ignore
			log.EarlyLogger.BasicLogf("config '%s' not found, skipping", c.name)
		} else if err != nil { // If it gets here something has actually gone wrong
			return fmt.Errorf("unable to load config '%s': %w", c.name, err)
		} else { // Good to go
			log.EarlyLogger.BasicLogf("successfully loaded key-value pairs from config '%s':", c.name)
			for _, k := range k2.Keys() {
				log.EarlyLogger.BasicLogf("\t%s -> %v", k, k2.Get(k))
			}

			// Merge clusters separately
			var kClusterSlice []map[string]any
			err = k2.Unmarshal("clusters", &kClusterSlice)
			if err != nil {
				return fmt.Errorf("unable to unmarshal cluster configs from config '%s': %w", c.name, err)
			}

			for i, cluster := range kClusterSlice {
				name, ok := cluster["name"].(string)
				if !ok || name == "" {
					return fmt.Errorf("cluster #%d from config '%s' is missing a name", i, c.name)
				}
				if clusterMap[name] == nil {
					clusterMap[name] = koanf.NewWithConf(kConfig)
					err = clusterMap[name].Load(confmap.Provider(DefaultClusterConfigMap, "."), nil)
					if err != nil {
						return fmt.Errorf("unable to load default cluster config: %w", err)
					}
				}
				err = clusterMap[name].Load(confmap.Provider(cluster["cluster"].(map[string]any), ""), nil)
				if err != nil {
					return fmt.Errorf("unable to merge cluster '%s' from config '%s': %w", name, c.name, err)
				}
			}

			// Finish the job
			err = k.Merge(k2)
			if err != nil {
				return fmt.Errorf("unable to merge config '%s': %w", c.name, err)
			} else {
				log.EarlyLogger.BasicLogf("successfully merged config '%s'", c.name)
			}
		}
	}

	// add merged clusters back to primary koanf instance
	clusterSlice := make([]map[string]any, 0, len(clusterMap))
	for k, v := range clusterMap {
		clusterSlice = append(clusterSlice, map[string]any{
			"name":    k,
			"cluster": v.Raw(),
		})
	}
	k.Set("clusters", clusterSlice)

	// Marshalling to the global config variable
	err = k.Unmarshal("", &GlobalConfig)
	if err != nil {
		return fmt.Errorf("unable to unmarshal merged config: %w", err)
	}

	log.EarlyLogger.BasicLogf("final config:")
	for _, key := range k.Keys() {
		log.EarlyLogger.BasicLogf("\t%s -> %v", key, k.Get(key))
	}
	return nil
}

// LoadGlobalConfigFromFile reads a YAML configuration at path and loads it into
// the GlobalConfig Config structure.
func LoadGlobalConfigFromFile(path string) error {
	log.EarlyLogger.BasicLog("early verbose log messages activated")

	ko := koanf.NewWithConf(kConfig)
	err := ko.Load(file.Provider(path), configParser)
	if errors.Is(err, os.ErrNotExist) { // This an error we can ignore
		log.EarlyLogger.BasicLogf("config '%s' not found, skipping", path)
	} else if err != nil { // If it gets here something has actually gone wrong
		return fmt.Errorf("unable to load config '%s': %w", path, err)
	} else { // Good to go
		log.EarlyLogger.BasicLogf("successfully loaded key-value pairs from config '%s':", path)
		for _, k := range ko.Keys() {
			log.EarlyLogger.BasicLogf("\t%s -> %v", k, ko.Get(k))
		}
	}

	// No error occurred
	return nil
}

// ModifyConfig modifies a single key in a config file. It does this by opening
// the config file and loading it into a koanf instance, using koanf to modify
// the key with the new value, unmarshalling the config into a config struct,
// then writing the config back out to the file. If an error occurs during this
// process or a config error occurs (e.g. there is a key specified that doesn't
// exist in the config struct or an invalid key was specified), an error is
// returned. Otherwise, nil is returned.
func ModifyConfig(path, key string, value interface{}) error {
	// Open file for writing
	ko, err := ReadConfig(path)
	if err != nil {
		return fmt.Errorf("failed to read %s for modification: %w", path, err)
	}

	// Perform modification
	if err := ko.Set(key, value); err != nil {
		return fmt.Errorf("failed to set key %s to value %v: %w", key, value, err)
	}

	// Write file back to file
	if err := WriteConfig(path, ko); err != nil {
		return fmt.Errorf("failed to write modified config to %s: %w", path, err)
	}

	return nil
}

// ModifyConfigCluster sets or modifies a single key for a single cluster,
// identified by name, in a config file located at path. If dflt is true,
// default-cluster is set to the specified cluster. If cluster does not already
// exist, it is added. If key is "name", the cluster is renamed but setting the
// name to an existing cluster name is not allowed. If the default cluster's
// name is changed, default-cluster is set to the new name, regardless of dflt.
//
// This function works similarly to ModifyConfig in that it loads the
// configuration into a koanf instance, sets the key, then unmarhalls back into
// a struct, where it can be written back to the config file.
func ModifyConfigCluster(path, cluster, key string, dflt bool, value any) error {
	// Open file for writing
	ko, err := ReadConfig(path)
	if err != nil {
		return fmt.Errorf("failed to read %s for modification: %w", path, err)
	}

	var clusters []map[string]any
	err = ko.Unmarshal("clusters", &clusters)

	// Make sure that if setting the cluster name, a cluster with that name
	// doesn't already exist.
	if key == "name" {
		if err != nil {
			return fmt.Errorf("unable to unmarshal clusters: %w", err)
		}
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
		clusters[cidx]["name"] = cluster
	}

	kc := koanf.NewWithConf(kConfig)
	err = kc.Load(confmap.Provider(clusters[cidx], ""), nil)
	if err != nil {
		return fmt.Errorf("unable to load cluster '%s' from config '%s': %w", cluster, path, err)
	}

	err = kc.Set(key, value)
	if err != nil {
		return fmt.Errorf("unable to modify config value '%s' in cluster '%s': %w", key, cluster, err)
	}

	clusters[cidx] = kc.Raw()
	ko.Set("clusters", clusters)

	defaultCluster := ko.String("default-cluster")

	// If default is set, set default-cluster to cluster name.
	// Also do it if the default-cluster was renamed (to reflect the new name)
	if dflt || (key == "name" && defaultCluster == cluster) {
		// If key was "name", set default-cluster to "name"
		// instead of cluster specified in arg.
		if key == "name" {
			s, ok := value.(string)
			if ok {
				err = ko.Set("default-cluster", s)
			} else {
				err = fmt.Errorf("value '%v' is not a string", value)
			}
		} else {
			err = ko.Set("default-cluster", cluster)
		}
		if err != nil {
			return fmt.Errorf("failed to set default-cluster: %w", err)
		}
	}

	// Write modified config back to file
	if err := WriteConfig(path, ko); err != nil {
		return fmt.Errorf("failed to write modified config to %s: %w", path, err)
	}

	return nil
}

// DeleteConfig deletes a key from a config file. It does this by reading in the
// config file at path and loading it into a koanf instance, then using that
// koanf instance to delete the key. It then unmarshals the config to a config
// struct and writes it back out to the config file. If an error in this process
// occurs or there is an error in the config (e.g. the key was not found), then
// an error is returned. Otherwise, nil is returned.
func DeleteConfig(path, key string) error {
	// Open file for writing
	ko, err := ReadConfig(path)
	if err != nil {
		return fmt.Errorf("failed to read %s for deletion: %w", path, err)
	}

	ko.Delete(key)

	// Write modified config back to file
	if err := WriteConfig(path, ko); err != nil {
		return fmt.Errorf("failed to write modified config to %s: %w", path, err)
	}

	return nil
}

// DeleteConfigCluster deletes a key from the specified cluster from a config
// file. It does by loading the cluster config into a koanf instance, deleting
// the key, then unmarshalling it back into a ConfigCluster struct before
// writing the config back to the config file. An error is thrown if the cluster
// doesn't exist or "name" is the key.
func DeleteConfigCluster(path, cluster, key string) error {
	// Open file for writing
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
			ck := koanf.NewWithConf(kConfig)
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

	ko.Set("clusters", clusters)

	// Write modified config back to file
	if err := WriteConfig(path, ko); err != nil {
		return fmt.Errorf("failed to write modified config to %s: %w", path, err)
	}

	return nil
}

// GetConfig returns the config value of key for a Config struct, returning an
// error if loading the config into koanf errs. If key is empty, the whole
// config is returned. This function _only_ retrieves global config options and
// errs if the key begins with "clusters*" ("*" is one or more characters), i.e.
// an individual cluster config is trying to be retrieved. To get an individual
// cluster config, use GetConfigCluster.
func GetConfig(ko *koanf.Koanf, key string) (any, error) {
	// Do not try to get individual cluster config. Use GetConfigCluster for
	// that.
	if strings.HasPrefix(key, "clusters") && len(key) > len("clusters") {
		return nil, fmt.Errorf("cannot get individual cluster config with global get command")
	}

	// Load config into koanf so the key can be used to get config.
	var val any
	if key != "" {
		val = ko.Get(key)
	} else {
		val = ko.Raw()
	}
	return val, nil
}

// GetConfigFromFile is like GetConfig except that it reads the config from the
// file at path instead of a Config struct.
func GetConfigFromFile(path, key string) (any, error) {
	// Read in config file
	ko, err := ReadConfig(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	return GetConfig(ko, key)
}

// GetConfigString wraps GetConfig and returns a string representation of the
// value of key, using format to determine how to marshal the value.
// Currently-supported formats are yaml, json, and json-pretty.
func GetConfigString(ko *koanf.Koanf, key, format string) (string, error) {
	if strings.HasPrefix(key, "clusters.") {
		return "", fmt.Errorf("key cannot be a cluster")
	}
	// val, err := GetConfig(ko, key)
	val := ko.Get(key)
	if val == nil {
		return "", nil
	}
	switch val.(type) {
	case map[string]interface{}, []interface{}:
		var err error
		var valBytes []byte
		switch format {
		case "yaml":
			valBytes, err = yaml.Marshal(val)
		case "json":
			valBytes, err = json.Marshal(val)
		case "json-pretty":
			valBytes, err = json.MarshalIndent(val, "", "\t")
		default:
			return "", fmt.Errorf("unknown format: %s", format)
		}
		if err != nil {
			return "", fmt.Errorf("failed to marshal value for key %q: %w", key, err)
		}
		return string(valBytes), nil
	default:
		return fmt.Sprintf("%v", val), nil
	}
}

// GetConfigStringFromFile is like GetConfigString except that it wraps
// GetConfigFromFile.
func GetConfigStringFromFile(path, key, format string) (string, error) {
	// Read in config file
	ko, err := ReadConfig(path)
	if err != nil {
		return "", fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	return GetConfigString(ko, key, format)
}

// GetConfigCluster returns the config value of key for a ConfigCluster struct,
// returning an error if loading the config into koanf errs. If key is empty,
// the whole config is returned. This function _only_ retrieves config options
// for a cluster. To get global config, use GetConfig.
func GetConfigCluster(cluster ConfigCluster, key string) (interface{}, error) {
	// Load config into koanf so the key can be used to get config.
	var val interface{}
	ko := koanf.NewWithConf(kConfig)
	if err := ko.Load(structs.Provider(cluster, "koanf"), nil); err != nil {
		return nil, fmt.Errorf("failed to load cluster config: %w", err)
	}
	val = ko.Get(key)
	return val, nil
}

// GetConfigClusterString wraps GetConfigCluster and returns a string
// representation of the value of key, using format to determine how to marshal
// the value. Currently-supported formats are yaml, json, and json-pretty.
func GetConfigClusterString(cluster ConfigCluster, key, format string) (string, error) {
	val, err := GetConfigCluster(cluster, key)
	if err != nil {
		return "", err
	}
	if val == nil {
		return "", nil
	}
	switch val.(type) {
	case map[string]interface{}, []interface{}:
		var err error
		var valBytes []byte
		switch format {
		case "yaml":
			valBytes, err = yaml.Marshal(val)
		case "json":
			valBytes, err = json.Marshal(val)
		case "json-pretty":
			valBytes, err = json.MarshalIndent(val, "", "\t")
		default:
			return "", fmt.Errorf("unknown format: %s", format)
		}
		if err != nil {
			return "", fmt.Errorf("failed to marshal value for key %q: %w", key, err)
		}
		return string(valBytes), nil
	default:
		return fmt.Sprintf("%v", val), nil
	}
}

// ReadConfig opens the config file at path and loads it into koanf to check for
// errors, then unmarshals the config into a Config struct and returns it. If an
// error in this process occurs or there is an error in the config, an error is
// returned.
func ReadConfig(path string) (*koanf.Koanf, error) {
	if path == "" {
		return nil, fmt.Errorf("no configuration file passed")
	}
	log.Logger.Debug().Msgf("reading config file: %s", path)

	// Load config file into koanf to check for errors
	ko := koanf.NewWithConf(kConfig)
	if err := ko.Load(file.Provider(path), configParser); err != nil {
		return ko, fmt.Errorf("failed to load config file %s: %w", path, err)
	}

	// Unmarshal koanf data into config struct
	if err := ko.Load(file.Provider(path), configParser); err != nil {
		return ko, fmt.Errorf("failed to read config data: %w", err)
	}

	return ko, nil
}

// WriteConfig takes a path and config file format and writes the current viper
// configuration to the file pointed to by path in the format specified. If path
// is empty, an error is returned. WriteConfig accepts any config file types
// that viper accepts. If format is empty, the format is guessed by the config
// file's file extension. If there is no file extension and format is empty,
// YAML is used.
func WriteConfig(path string, k *koanf.Koanf) error {
	if path == "" {
		return fmt.Errorf("no configuration file path passed")
	}
	log.Logger.Debug().Msgf("writing config file: %s", path)

	c, err := k.Marshal(configParser)
	if err != nil {
		return fmt.Errorf("failed to marshal config for writing: %w", err)
	}

	// Get mode if file exists
	var fmode os.FileMode = 0o644
	if finfo, err := os.Stat(path); err == nil {
		fmode = finfo.Mode()
	}

	// Write config file
	if err := os.WriteFile(path, c, fmode); err != nil {
		return fmt.Errorf("failed to write config to file %s: %w", path, err)
	}
	log.Logger.Info().Msgf("wrote config to %s", path)

	return nil
}

func getUserConfigPath() (string, error) {
	// Generate user config path: ~/.config/ochami/config.yaml
	user, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("unable to fetch current user: %w", err)
	}
	return filepath.Join(user.HomeDir, ".config", "ochami", "config.yaml"), nil
}
