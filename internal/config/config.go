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

	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/go-viper/mapstructure/v2"
	kyaml "github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/structs"
	"github.com/knadh/koanf/v2"
	"gopkg.in/yaml.v3"
)

type ServiceName string

const ProgName = "ochami"

const (
	ServiceBSS       ServiceName = "bss"
	ServiceCloudInit ServiceName = "cloud-init"
	ServicePCS       ServiceName = "pcs"
	ServiceSMD       ServiceName = "smd"
)

const (
	DefaultBasePathBSS       = "/boot/v1"
	DefaultBasePathCloudInit = "/cloud-init"
	DefaultBasePathPCS       = "/"
	DefaultBasePathSMD       = "/hsm/v2"
)

// Default configuration values if either no configuration files exist or the
// configuration files don't contain values for items that need them.
var DefaultConfig = Config{
	Log: ConfigLog{
		Format: "rfc3339",
		Level:  "warning",
	},
}

var (
	GlobalConfig     = DefaultConfig // Global config struct
	GlobalKoanf      *koanf.Koanf    // Koanf instance for gobal config struct
	UserConfigFile   string
	SystemConfigFile = "/etc/ochami/config.yaml"

	// Since logging isn't set up until after config is read, this variable
	// allows more verbose printing if true for more verbose logging
	// pre-config parsing.
	EarlyVerbose bool

	configParser = kyaml.Parser() // Koanf YAML parser provider

	// Global koanf struct configuration
	kConfig = koanf.Conf{Delim: ".", StrictMerge: true}

	// koanf unmarshal config used in unmarshalling function
	kUnmarshalConf = koanf.UnmarshalConf{
		Tag: "yaml", // Tag for determining mapping to struct members
		DecoderConfig: &mapstructure.DecoderConfig{
			ErrorUnused: true,          // Err if unknown keys found
			Result:      &GlobalConfig, // Unmarshal to global config
		},
	}
)

// Config represents the structure of a configuration file.
type Config struct {
	Log            ConfigLog       `yaml:"log,omitempty"`
	DefaultCluster string          `yaml:"default-cluster,omitempty"`
	Clusters       []ConfigCluster `yaml:"clusters,omitempty"`
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
	URI       string                 `yaml:"uri,omitempty"`
	BSS       ConfigClusterBSS       `yaml:"bss,omitempty"`
	CloudInit ConfigClusterCloudInit `yaml:"cloud-init,omitempty"`
	PCS       ConfigClusterPCS       `yaml:"pcs,omitempty"`
	SMD       ConfigClusterSMD       `yaml:"smd,omitempty"`
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
		newCCC.BSS.URI = compare(ccc.CloudInit.URI, c.CloudInit.URI)
	}
	if ccc.PCS == (ConfigClusterPCS{}) {
		newCCC.PCS = ConfigClusterPCS{URI: c.PCS.URI}
	} else {
		newCCC.PCS.URI = compare(ccc.PCS.URI, c.PCS.URI)
	}
	if ccc.SMD == (ConfigClusterSMD{}) {
		newCCC.SMD = ConfigClusterSMD{URI: c.PCS.URI}
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

// earlyLog is a primitive log function that works like fmt.Fprintln, printing
// to standard error only if EarlyVerbose is true.
func earlyLog(arg ...interface{}) {
	if EarlyVerbose {
		fmt.Fprintf(os.Stderr, "%s: ", ProgName)
		fmt.Fprintln(os.Stderr, arg...)
	}
}

// earlyLogf is like earlyLog, except it accepts a format string. It works like
// fmt.Fprintf.
func earlyLogf(fstr string, arg ...interface{}) {
	if EarlyVerbose {
		fmt.Fprintf(os.Stderr, "%s: ", ProgName)
		fmt.Fprintf(os.Stderr, fstr+"\n", arg...)
	}
}

// RemoveFromSlice removes an element from a slice and returns the resulting
// slice. The element to be removed is identified by its index in the slice.
func RemoveFromSlice[T any](slice []T, index int) []T {
	slice[len(slice)-1], slice[index] = slice[index], slice[len(slice)-1]
	return slice[:len(slice)-1]
}

// LoadConfig takes a path to a config file and reads the contents of the file,
// using koanf to load and unmarshal it into the global config struct. If there
// is an error in this process or there is a config error (e.g. there is a key
// specified that doesn't exist in the config struct), an error is returned.
// Otherwise, nil is returned.
func LoadConfig(path string) error {
	earlyLog("early verbose log messages activated")

	// Initialize global koanf structure
	GlobalKoanf = koanf.NewWithConf(kConfig)

	// If a config file was specified, load it alone. Do not try to merge
	// its config with any other configuration.
	if path != "" {
		earlyLogf("using passed config file %s", path)
		earlyLogf("parsing %s", path)
		if err := GlobalKoanf.Load(file.Provider(path), configParser); err != nil {
			return fmt.Errorf("failed to load specified config file %s: %w", path, err)
		}
		earlyLog("unmarshalling config into config struct")
		if err := GlobalKoanf.UnmarshalWithConf("", nil, kUnmarshalConf); err != nil {
			return fmt.Errorf("failed to unmarshal config from file %s: %w", path, err)
		}
		return nil
	}
	// Otherwise, we merge the config from the system and user config files.
	earlyLog("no config file specified on command line, attempting to merge configs")

	// Generate user config path: ~/.config/ochami/config.yaml
	user, err := user.Current()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: unable to fetch current user: %v\n", ProgName, err)
		os.Exit(1)
	}
	UserConfigFile = filepath.Join(user.HomeDir, ".config", "ochami", "config.yaml")

	// Read config from each file in slice
	type FileCfgMap struct {
		File string
		Cfg  Config
	}
	cfgsToCheck := []FileCfgMap{
		FileCfgMap{File: SystemConfigFile},
		FileCfgMap{File: UserConfigFile},
	}
	var cfgsLoaded []FileCfgMap
	for _, cfg := range cfgsToCheck {
		// Create koanf struct to load config from this file into
		ko := koanf.NewWithConf(kConfig)

		// Create config struct to unmarshal config from this file into
		var c Config

		// Copy global koanf unmarshal config, but unmarshal into config
		// struct we made above
		umc := kUnmarshalConf
		umc.DecoderConfig.Result = &c

		// Load config file into koanf struct
		earlyLogf("attempting to load config file: %s", cfg.File)
		err := ko.Load(file.Provider(cfg.File), configParser)
		if errors.Is(err, os.ErrNotExist) {
			earlyLogf("config file %s not found, skipping", cfg.File)
			continue
		} else if err != nil {
			return fmt.Errorf("failed to load config file %s: %w", cfg.File, err)
		}

		// Unmarshal loaded config into local config struct to lint
		// (i.e. check for unknown keys, etc).
		if err := ko.UnmarshalWithConf("", nil, umc); err != nil {
			return fmt.Errorf("failed to unmarshal config from %s: %w", cfg.File, err)
		}

		// Add local config struct to slice of loaded configs
		cfg.Cfg = c
		cfgsLoaded = append(cfgsLoaded, cfg)
	}

	// Merge loaded configs into global config. If none loaded, use default
	// config (set above).
	for _, cfgLoaded := range cfgsLoaded {
		earlyLogf("merging in config from %s", cfgLoaded.File)
		if err := GlobalKoanf.Load(structs.Provider(cfgLoaded.Cfg, "yaml"), nil, koanf.WithMergeFunc(mergeConfig)); err != nil {
			return fmt.Errorf("failed to merge configs into global config: %w", err)
		}
	}

	// Unmarshal merged config from Koanf into global config struct.
	// koanf.UnMarshalWithConf won't unmarshal into the global config struct
	// so we copy it, unmarhsl into the copy, then set the copy as the
	// global config.
	c := GlobalConfig
	kuc := kUnmarshalConf
	kuc.DecoderConfig.Result = &c
	if err := GlobalKoanf.UnmarshalWithConf("", nil, kuc); err != nil {
		return fmt.Errorf("failed to unmarshal global config into struct: %w", err)
	}
	GlobalConfig = c

	earlyLog("config files, if any, have been merged")

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
	kuc := kUnmarshalConf
	kuc.DecoderConfig.Result = &modCfg
	if err := ko.UnmarshalWithConf("", nil, kuc); err != nil {
		return fmt.Errorf("failed to modify config for %s: %w", path, err)
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
	kuc := kUnmarshalConf
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
		kuc.DecoderConfig.Result = &nCl
		if err := ko.UnmarshalWithConf("", nil, kuc); err != nil {
			return fmt.Errorf("failed to modify config for new cluster %s: %w", cluster, err)
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
		kuc.DecoderConfig.Result = clusterToMod
		if err := ko.UnmarshalWithConf("", nil, kuc); err != nil {
			return fmt.Errorf("failed to modify config for existing cluster %s: %w", cluster, err)
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
	kuc := kUnmarshalConf
	kuc.DecoderConfig.Result = &modCfg
	if err := ko.UnmarshalWithConf("", nil, kuc); err != nil {
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
	var tmpCluster ConfigCluster
	kuc := kUnmarshalConf
	kuc.DecoderConfig.Result = &tmpCluster
	if err := ko.UnmarshalWithConf("", nil, kuc); err != nil {
		return fmt.Errorf("failed to unset key %s from config for cluster %s: %w", key, cluster, err)
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
		kuc := kUnmarshalConf
		kuc.DecoderConfig.Result = &val
		if err := ko.UnmarshalWithConf("", nil, kuc); err != nil {
			return nil, fmt.Errorf("failed to unmarshal config from struct: %w", err)
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
		kuc := kUnmarshalConf
		kuc.DecoderConfig.Result = &val
		if err := ko.UnmarshalWithConf("", nil, kuc); err != nil {
			return nil, fmt.Errorf("failed to unmarshal cluster config from struct: %w", err)
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

	ko := koanf.NewWithConf(kConfig)
	if err := ko.Load(file.Provider(path), configParser); err != nil {
		return cfg, fmt.Errorf("failed to load config file %s: %w", path, err)
	}
	kuc := kUnmarshalConf
	kuc.DecoderConfig.Result = &cfg
	if err := ko.UnmarshalWithConf("", nil, kuc); err != nil {
		return cfg, fmt.Errorf("failed to unmarshal config from %s: %w", path, err)
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
