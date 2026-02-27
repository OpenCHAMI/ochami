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
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/knadh/koanf/providers/structs"
	"github.com/knadh/koanf/v2"
	"gopkg.in/yaml.v3"

	"github.com/OpenCHAMI/ochami/internal/log"
)

type ServiceName string

const (
	ServiceBoot      ServiceName = ""
	ServiceBSS       ServiceName = "bss"
	ServiceCloudInit ServiceName = "cloud-init"
	ServicePCS       ServiceName = "pcs"
	ServiceSMD       ServiceName = "smd"
)

const (
	DefaultBasePathBootService = "/boot"
	DefaultBasePathBSS         = "/boot/v1"
	DefaultBasePathCloudInit   = "/cloud-init"
	DefaultBasePathPCS         = "/"
	DefaultBasePathSMD         = "/hsm/v2"

	SystemConfigFile = "/etc/ochami/config.yaml"
)

// Default configuration values if either no configuration files exist or the
// configuration files don't contain values for items that need them.
var DefaultConfig = Config{
	Log: ConfigLog{
		Format: "rfc3339",
		Level:  "warning",
	},
	Timeout: 30 * time.Second,
}

var (
	GlobalConfig   = DefaultConfig // Global config struct
	GlobalKoanf    *koanf.Koanf    // Koanf instance for gobal config struct
	UserConfigFile string

	// Koanf YAML parser provider
	configParser = kyaml.Parser()

	// Global koanf struct configuration
	kConfig = koanf.Conf{Delim: ".", StrictMerge: true}
)

// Config represents the structure of a configuration file.
type Config struct {
	Log            ConfigLog       `yaml:"log,omitempty"`
	Timeout        time.Duration   `yaml:"timeout,omitempty"`
	DefaultCluster string          `yaml:"default-cluster,omitempty"`
	Clusters       []ConfigCluster `yaml:"clusters,omitempty"`
}

// UnmarshalYAML unmarshals YAML into a Config, handling default values. For
// instance, it detects if 'timeout' is present in the YAML and, if not, assigns
// the default value.
func (c *Config) UnmarshalYAML(value *yaml.Node) error {
	type alias Config

	// If node is top-level document (DocumentNode), work with MappingNode contained within
	n := value
	if n.Kind == yaml.DocumentNode && len(n.Content) == 1 {
		n = n.Content[0]
	}

	// Detect whether config keys were explicitly set
	hasTimeout := false
	if n.Kind == yaml.MappingNode {
		// Iterate over keys to find desired one
		//
		// Order of nodes in MappingNode are key, val, key, val, ...
		for i := 0; i+1 < len(n.Content); i += 2 {
			switch n.Content[i].Value {
			case "timeout":
				// Make sure a value was passed
				if len(n.Content[i+1].Value) == 0 {
					return ErrInvalidConfigVal{
						Key:      "timeout",
						Value:    "empty value",
						Expected: "duration",
						Line:     n.Content[i].Line,
					}
				} else if _, err := time.ParseDuration(n.Content[i+1].Value); err != nil {
					return ErrInvalidConfigVal{
						Key:      "timeout",
						Value:    "invalid duration",
						Expected: "duration",
						Line:     n.Content[i].Line,
					}
				}
				// If key was found and is not empty, set our sentinal
				hasTimeout = true
				break
			}
		}
	}

	// Decode once into a alias type to avoid infinite recursion when unmarshalling
	var tmp alias
	if err := n.Decode(&tmp); err != nil {
		return err
	}

	// Set default value only if the key was not present
	if !hasTimeout {
		tmp.Timeout = 30 * time.Second
	}

	// Assign temporarily-aliased struct back to receiver
	*c = Config(tmp)

	// No errors occurred
	return nil
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
	Format string `yaml:"format,omitempty"`
	Level  string `yaml:"level,omitempty"`
}

// ConfigCluster is a "wrapper" around an individual cluster configuration. It
// contains the cluster's name, as well as the actual configuration structure.
type ConfigCluster struct {
	Name    string              `yaml:"name,omitempty"`
	Cluster ConfigClusterConfig `yaml:"cluster,omitempty"`
}

// ConfigClusterConfig is the actual structure for an individual cluster
// configuration.
type ConfigClusterConfig struct {
	URI         string                   `yaml:"uri,omitempty"`
	BootService ConfigClusterBootService `yaml:"boot-service,omitempty"`
	BSS         ConfigClusterBSS         `yaml:"bss,omitempty"`
	CloudInit   ConfigClusterCloudInit   `yaml:"cloud-init,omitempty"`
	PCS         ConfigClusterPCS         `yaml:"pcs,omitempty"`
	SMD         ConfigClusterSMD         `yaml:"smd,omitempty"`
	EnableAuth  bool                     `yaml:"enable-auth"`
}

// UnmarshalYAML unmarshals YAML into a ConfigClusterConfig, handling default
// values. For instance, it detects if 'enable-auth' is present in the YAML and,
// if not, assigns a default value of true.
func (c *ConfigClusterConfig) UnmarshalYAML(value *yaml.Node) error {
	type alias ConfigClusterConfig

	// If node is top-level document (DocumentNode), work with MappingNode contained within
	n := value
	if n.Kind == yaml.DocumentNode && len(n.Content) == 1 {
		n = n.Content[0]
	}

	// Detect whether config keys were explicitly set
	hasEnableAuth := false
	if n.Kind == yaml.MappingNode {
		// Iterate over keys to find desired one
		//
		// Order of nodes in MappingNode are key, val, key, val, ...
		for i := 0; i+1 < len(n.Content); i += 2 {
			switch n.Content[i].Value {
			case "enable-auth":
				// Make sure a value was passed
				if len(n.Content[i+1].Value) == 0 {
					return ErrInvalidConfigVal{
						Key:      "enable-auth",
						Value:    "empty value",
						Expected: "true or false",
						Line:     n.Content[i].Line,
					}
				}
				// If key was found and is not empty, set our sentinel
				hasEnableAuth = true
				break
			}
		}
	}

	// Decode once into a alias type to avoid infinite recursion when unmarshalling
	var tmp alias
	if err := n.Decode(&tmp); err != nil {
		return err
	}

	// Set default value only if the key was not present
	if !hasEnableAuth {
		tmp.EnableAuth = true
	}

	// Assign temporarily-aliased struct back to receiver
	*c = ConfigClusterConfig(tmp)

	// No errors occurred
	return nil
}

// ConfigClusterBootService represents configuration specifically for the
// boot service.
type ConfigClusterBootService struct {
	APIVersion string `yaml:"api-version,omitempty"`
	URI        string `yaml:"uri,omitempty"`
}

// ConfigClusterBSS represents configuration specifically for the Boot Script
// Service.
type ConfigClusterBSS struct {
	URI string `yaml:"uri,omitempty"`
}

// ConfigClusterCloudInit represents configuration specifically for the
// cloud-init service.
type ConfigClusterCloudInit struct {
	URI string `yaml:"uri,omitempty"`
}

// ConfigClusterPCS represents configuration specifically for the Power Control
// Service.
type ConfigClusterPCS struct {
	URI string `yaml:"uri,omitempty"`
}

// ConfigClusterSMD represents configuration specifically for the State
// Management Database service.
type ConfigClusterSMD struct {
	URI string `yaml:"uri,omitempty"`
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
	if ccc.SMD == (ConfigClusterSMD{}) {
		newCCC.SMD = ConfigClusterSMD{URI: c.SMD.URI}
	} else {
		newCCC.SMD.URI = compare(ccc.SMD.URI, c.SMD.URI)
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

// unmarshalKoanfYAML is a helper function that unmarshals data in ko into out
// using parser p. The reason this function exists is because koanf.Unmarshal
// and koanf.MarshalWithConf use mapstructure for their unmarshalling, which
// means that custom unmarshal functions defined for out do not get run.
// unmarshalKoanfYAML ensures this happens by first marshalling the data in ko
// into YAML, then using the YAML unmarshaller to unmarshal into out.
func unmarshalKoanfYAML(ko *koanf.Koanf, out interface{}) error {
	yamlBytes, err := ko.Marshal(kyaml.Parser())
	if err != nil {
		return fmt.Errorf("failed to marshal config data into YAML: %w", err)
	}
	if err := yaml.Unmarshal(yamlBytes, out); err != nil {
		return fmt.Errorf("failed to unmarshal YAML bytes into config: %w", err)
	}
	return nil
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

	// Generate user config path: ~/.config/ochami/config.yaml
	user, err := user.Current()
	if err != nil {
		return fmt.Errorf("unable to fetch current user: %w", err)
	}
	UserConfigFile = filepath.Join(user.HomeDir, ".config", "ochami", "config.yaml")

	// Read config from each file in slice
	type FileCfgMap struct {
		File string
		Cfg  Config
	}
	cfgsToCheck := []FileCfgMap{
		{File: SystemConfigFile},
		{File: UserConfigFile},
	}
	// Default config is first in list so it is loaded first (so that when
	// other configs get merged, unset values are set to the default).
	cfgsLoaded := []FileCfgMap{{File: "default", Cfg: DefaultConfig}}
	for _, cfg := range cfgsToCheck {
		// Read bytes of config file
		cfgBytes, err := os.ReadFile(cfg.File)
		if errors.Is(err, os.ErrNotExist) {
			log.EarlyLogger.BasicLogf("config file %s not found, skipping", cfg.File)
			continue
		} else if err != nil {
			log.EarlyLogger.BasicLogf("failed to load config file %s: %v", cfg.File, err)
			log.EarlyLogger.BasicLogf("skipping config file %s", cfg.File)
			continue
		}

		// Generate config struct from bytes via parser
		_, c, err := GenerateConfigFromBytes(cfgBytes)
		if err != nil {
			return fmt.Errorf("failed to parse config bytes: %w", err)
		}

		// Add local config struct to slice of loaded configs
		cfg.Cfg = c
		cfgsLoaded = append(cfgsLoaded, cfg)
	}

	// Create a parser and merge configs into it:
	//
	//   1. Default config
	//   2. System config
	//   3. User config
	//
	ko := koanf.NewWithConf(kConfig)
	for _, cfgLoaded := range cfgsLoaded {
		if cfgLoaded.File == "default" {
			log.EarlyLogger.BasicLogf("starting with default config")
			if err := MergeConfigIntoParser(ko, cfgLoaded.Cfg); err != nil {
				return fmt.Errorf("failed to load default config: %w", err)
			}
		} else {
			log.EarlyLogger.BasicLogf("merging in config from %s", cfgLoaded.File)
			if err := MergeConfigIntoParser(ko, cfgLoaded.Cfg); err != nil {
				return fmt.Errorf("failed to merge config: %w", err)
			}
		}
	}

	// Set the merged parser as the global parser
	GlobalKoanf = ko

	// Unmarshal merged config from global parser into global config struct.
	// koanf.UnMarshalWithConf won't unmarshal into the global config struct
	// so we copy it, unmarshal into the copy, then set the copy as the
	// global config.
	c := GlobalConfig
	if err := unmarshalKoanfYAML(ko, &c); err != nil {
		return fmt.Errorf("failed to read merged config: %w", err)
	}
	GlobalConfig = c

	log.EarlyLogger.BasicLog("config files, if any, have been merged")

	return nil
}

// MergeConfigIntoParser take a Config and merges it into the parser k. This can
// be done iteratively to incorporate multiple Configs into one parser.
func MergeConfigIntoParser(k *koanf.Koanf, cfg Config) error {
	if k == nil {
		return fmt.Errorf("koanf object cannot be nil")
	}
	return k.Load(structs.Provider(cfg, "yaml"), nil, koanf.WithMergeFunc(mergeConfig))
}

// LoadGlobalConfigFromFile reads a YAML configuration at path and loads it into
// the GlobalConfig Config structure.
func LoadGlobalConfigFromFile(path string) error {
	log.EarlyLogger.BasicLog("early verbose log messages activated")

	// Read bytes of config file
	log.EarlyLogger.BasicLogf("reading config bytes from config file %q", path)
	cfgBytes, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config bytes from file %q: %w", path, err)
	}
	log.EarlyLogger.BasicLog("successfully read config bytes")

	// Generate parser struct and config struct from bytes
	log.EarlyLogger.BasicLogf("parsing config bytes")
	k, cfg, err := GenerateConfigFromBytes(cfgBytes)
	if err != nil {
		return fmt.Errorf("failed to parse config bytes: %w", err)
	}
	log.EarlyLogger.BasicLog("successfully parsed config bytes")

	// Set results as global for later reference/modification
	GlobalKoanf = k
	GlobalConfig = cfg

	// No error occurred
	return nil
}

// GenerateConfigFromBytes takes a byte slice and parses it into a koanf
// structure as YAML (returning an error if this fails), then unmarshals it into
// a Config structure. Both the *koanf.Koanf and Config are returned.
func GenerateConfigFromBytes(b []byte) (*koanf.Koanf, Config, error) {
	// Initialize global parser structure
	k := koanf.NewWithConf(kConfig)

	// Initialize config to default config
	cfg := DefaultConfig

	// Load bytes into parser structure
	if err := k.Load(rawbytes.Provider(b), configParser); err != nil {
		return k, cfg, fmt.Errorf("failed to load config bytes: %w", err)
	}

	// Unmarshal the YAML directly from the input bytes into config struct,
	// using the custom YAML unmarshaller. The koanf unmarshaller does not
	// call the custom UnmarshalYAML since it uses mapstructure under the
	// hood.
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return k, cfg, fmt.Errorf("failed to unmarshal YAML bytes into config: %w", err)
	}

	// Return parser structure and config structure
	return k, cfg, nil
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
	cfg, err := ReadConfig(path)
	if err != nil {
		return fmt.Errorf("failed to read %s for modification: %w", path, err)
	}

	// Perform modification
	ko := koanf.NewWithConf(kConfig)
	if err := ko.Load(structs.Provider(cfg, "yaml"), nil); err != nil {
		return fmt.Errorf("failed to load config from %s: %w", path, err)
	}
	if err := ko.Set(key, value); err != nil {
		return fmt.Errorf("failed to set key %s to value %v: %w", key, value, err)
	}
	var modCfg Config
	if err := unmarshalKoanfYAML(ko, &modCfg); err != nil {
		return fmt.Errorf("failed to read modified config: %w", err)
	}

	// Write file back to file
	if err := WriteConfig(path, modCfg); err != nil {
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
func ModifyConfigCluster(path, cluster, key string, dflt bool, value interface{}) error {
	// Open file for writing
	cfg, err := ReadConfig(path)
	if err != nil {
		return fmt.Errorf("failed to read %s for modification: %w", path, err)
	}

	// Make sure that if setting the cluster name, a cluster with that name
	// doesn't already exist.
	if key == "name" {
		for _, cl := range cfg.Clusters {
			if cl.Name == value.(string) {
				return fmt.Errorf("cluster with name %q already exists", cl.Name)
			}
		}
	}

	// Determine if a new cluster needs to be added or an existing cluster
	// needs to be modified.
	var clusterToMod *ConfigCluster
	newCluster := true
	for cidx, cl := range cfg.Clusters {
		if cl.Name == cluster || (key == "name" && cl.Name == value.(string)) {
			// Existing cluster found, set pointer to it
			clusterToMod = &(cfg.Clusters[cidx])
			newCluster = false
			break
		}
	}
	ko := koanf.NewWithConf(kConfig)
	if newCluster {
		// Adding a new cluster; create it and append to list
		nCl := ConfigCluster{Name: cluster}
		if err := ko.Load(structs.Provider(nCl, "yaml"), nil); err != nil {
			return fmt.Errorf("failed to load config for new cluster %s: %w", cluster, err)
		}

		// Modify key for new cluster
		if err := ko.Set(key, value); err != nil {
			return fmt.Errorf("failed to set key %s to value %v for new cluster %s: %w", key, value, cluster, err)
		}
		if err := unmarshalKoanfYAML(ko, &nCl); err != nil {
			return fmt.Errorf("failed to read modified config for new cluster %s: %w", cluster, err)
		}

		// Add new cluster to cluster list
		cfg.Clusters = append(cfg.Clusters, nCl)
	} else {
		// Modifying existing cluster; modify directly in cluster list
		// Make sure there is a cluster to modify
		if clusterToMod == nil {
			return fmt.Errorf("unknown error finding existing cluster %s in %s", cluster, path)
		}
		if err := ko.Load(structs.Provider(*clusterToMod, "yaml"), nil); err != nil {
			return fmt.Errorf("failed to load config for existing cluster %s: %w", cluster, err)
		}

		// Modify key for existing cluster
		if err := ko.Set(key, value); err != nil {
			return fmt.Errorf("failed to set key %s to value %v for existing cluster %s: %w", key, value, cluster, err)
		}
		if err := unmarshalKoanfYAML(ko, &clusterToMod); err != nil {
			return fmt.Errorf("failed to read modified config for existing cluster %s: %w", cluster, err)
		}
	}

	// If default is set, set default-cluster to cluster name.
	if dflt {
		if key == "name" {
			// If key was "name", set default-cluster to "name"
			// instead of cluster specified in arg.
			cfg.DefaultCluster = value.(string)
		} else {
			// If any other key, set default-cluster to cluster
			// specified in arg.
			cfg.DefaultCluster = cluster
		}
	} else if cfg.DefaultCluster == cluster && key == "name" {
		// Even if default is not set, if the current default cluster
		// matches cluster specified in arg and key is "name", change
		// default-cluster to the new name after changing the cluster
		// name so it doesn't point to a non-existent cluster.
		cfg.DefaultCluster = value.(string)
	}

	// Write modified config back to file
	if err := WriteConfig(path, cfg); err != nil {
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
	cfg, err := ReadConfig(path)
	if err != nil {
		return fmt.Errorf("failed to read %s for deletion: %w", path, err)
	}

	// Perform deletion
	ko := koanf.NewWithConf(kConfig)
	if err := ko.Load(structs.Provider(cfg, "yaml"), nil); err != nil {
		return fmt.Errorf("failed to load config from %s: %w", path, err)
	}

	ko.Delete(key)

	var modCfg Config
	if err := unmarshalKoanfYAML(ko, &modCfg); err != nil {
		return fmt.Errorf("failed to unset key %s from config for %s: %w", key, path, err)
	}

	// Write modified config back to file
	if err := WriteConfig(path, modCfg); err != nil {
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
	cfg, err := ReadConfig(path)
	if err != nil {
		return fmt.Errorf("failed to read %s for modification: %w", path, err)
	}

	if key == "name" {
		return fmt.Errorf("cannot unset name of cluster")
	}

	// Find cluster to modify
	var clusterToMod *ConfigCluster
	for cidx, cl := range cfg.Clusters {
		if cl.Name == cluster {
			clusterToMod = &(cfg.Clusters[cidx])
			break
		}
	}
	if clusterToMod == nil {
		return fmt.Errorf("cluster %q not found", cluster)
	}

	// Perform deletion
	ko := koanf.NewWithConf(kConfig)
	if err := ko.Load(structs.Provider(*clusterToMod, "yaml"), nil); err != nil {
		return fmt.Errorf("failed to load config for cluster %s: %w", cluster, err)
	}
	ko.Delete(key)

	// Write modified config back out to struct
	var tmpCluster ConfigCluster
	if err := unmarshalKoanfYAML(ko, &tmpCluster); err != nil {
		return fmt.Errorf("failed to read modified cluster data: %w", err)
	}
	*clusterToMod = tmpCluster

	// Write modified config back to file
	if err := WriteConfig(path, cfg); err != nil {
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
func GetConfig(cfg Config, key string) (interface{}, error) {
	// Do not try to get individual cluster config. Use GetConfigCluster for
	// that.
	if strings.HasPrefix(key, "clusters") && len(key) > len("clusters") {
		return nil, fmt.Errorf("cannot get individual cluster config with global get command")
	}

	// Load config into koanf so the key can be used to get config.
	var val interface{}
	ko := koanf.NewWithConf(kConfig)
	if err := ko.Load(structs.Provider(cfg, "yaml"), nil); err != nil {
		return nil, fmt.Errorf("failed to load global config: %w", err)
	}
	if key != "" {
		val = ko.Get(key)
	} else {
		// No key specified, return whole config
		if err := unmarshalKoanfYAML(ko, &val); err != nil {
			return nil, fmt.Errorf("failed to read config data: %w", err)
		}
	}
	return val, nil
}

// GetConfigFromFile is like GetConfig except that it reads the config from the
// file at path instead of a Config struct.
func GetConfigFromFile(path, key string) (interface{}, error) {
	// Read in config file
	cfg, err := ReadConfig(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	return GetConfig(cfg, key)
}

// GetConfigString wraps GetConfig and returns a string representation of the
// value of key, using format to determine how to marshal the value.
// Currently-supported formats are yaml, json, and json-pretty.
func GetConfigString(cfg Config, key, format string) (string, error) {
	val, err := GetConfig(cfg, key)
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

// GetConfigStringFromFile is like GetConfigString except that it wraps
// GetConfigFromFile.
func GetConfigStringFromFile(path, key, format string) (string, error) {
	// Read in config file
	cfg, err := ReadConfig(path)
	if err != nil {
		return "", fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	return GetConfigString(cfg, key, format)
}

// GetConfigCluster returns the config value of key for a ConfigCluster struct,
// returning an error if loading the config into koanf errs. If key is empty,
// the whole config is returned. This function _only_ retrieves confiog options
// for a cluster. To get global config, use GetConfig.
func GetConfigCluster(cluster ConfigCluster, key string) (interface{}, error) {
	// Load config into koanf so the key can be used to get config.
	var val interface{}
	ko := koanf.NewWithConf(kConfig)
	if err := ko.Load(structs.Provider(cluster, "yaml"), nil); err != nil {
		return nil, fmt.Errorf("failed to load cluster config: %w", err)
	}
	if key != "" {
		val = ko.Get(key)
	} else {
		// No key specified, return whole config
		if err := unmarshalKoanfYAML(ko, &val); err != nil {
			return nil, fmt.Errorf("failed to read cluster config: %w", err)
		}
	}
	return val, nil
}

// GetConfigClusterFromFile is like GetConfigCluster except that it reads the
// config from the file at path instead of a ConfigCluster struct.
func GetConfigClusterFromFile(path, cluster, key string) (interface{}, error) {
	// Read in config file
	cfg, err := ReadConfig(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	for _, cl := range cfg.Clusters {
		if cl.Name == cluster {
			return GetConfigCluster(cl, key)
		}
	}
	return nil, fmt.Errorf("cluster %q not found in %s", cluster, path)
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
func ReadConfig(path string) (Config, error) {
	var cfg Config
	if path == "" {
		return cfg, fmt.Errorf("no configuration file passed")
	}
	log.Logger.Debug().Msgf("reading config file: %s", path)

	// Load config file into koanf to check for errors
	ko := koanf.NewWithConf(kConfig)
	if err := ko.Load(file.Provider(path), configParser); err != nil {
		return cfg, fmt.Errorf("failed to load config file %s: %w", path, err)
	}

	// Unmarshal koanf data into config struct
	if err := unmarshalKoanfYAML(ko, &cfg); err != nil {
		return cfg, fmt.Errorf("failed to read config data: %w", err)
	}

	return cfg, nil
}

// WriteConfig takes a path and config file format and writes the current viper
// configuration to the file pointed to by path in the format specified. If path
// is empty, an error is returned. WriteConfig accepts any config file types
// that viper accepts. If format is empty, the format is guessed by the config
// file's file extension. If there is no file extension and format is empty,
// YAML is used.
func WriteConfig(path string, cfg Config) error {
	if path == "" {
		return fmt.Errorf("no configuration file path passed")
	}
	log.Logger.Debug().Msgf("writing config file: %s", path)

	c, err := yaml.Marshal(cfg)
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

// mergeConfig is the handler function that handles merging koanf
// configurations. It is a wrapper around MergeMaps, which performs the actual
// merging of the data structures.
func mergeConfig(src, dst map[string]interface{}) error {
	// "name" is key used to identify each cluster config in config's
	// cluster list.
	return MergeMaps(src, dst, "name")
}
