// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"time"

	kyaml "github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"

	"github.com/openchami/ochami/pkg/format"
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

	// SystemConfigFile is the path to the system-wide configuration file.
	SystemConfigFile = "/etc/ochami/config.yaml"
)

// DefaultGlobalMap returns a fresh copy of the default values for global
// (non-cluster) configuration options. A new map is returned on each call so
// that callers cannot mutate shared package state.
func DefaultGlobalMap() map[string]any {
	return map[string]any{
		"log.format":            "rfc3339",
		"log.level":             "warning",
		"log.color":             "auto",
		"timeout":               "30s",
		"default-input-format":  "json",
		"default-output-format": "json",
	}
}

// DefaultClusterMap returns a fresh copy of the default values applied to each
// cluster configuration. A new map is returned on each call so that callers
// cannot mutate shared package state.
func DefaultClusterMap() map[string]any {
	return map[string]any{
		"enable-auth": true,
	}
}

// koanfConf is the strict koanf configuration used by the loaders. StrictMerge
// catches incompatible types between merged sources.
var koanfConf = koanf.Conf{Delim: ".", StrictMerge: true}

// Config represents the structure of a configuration file.
// Normally the omitempty field tag would be set, but koanf doesn't use it since
// fields are first loaded into a map[string]any for merging purposes, so
// unspecified fields simply aren't present during serialization
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

// DefaultTimeout returns the default request timeout derived from the built-in
// default configuration. It returns -1 if the default cannot be parsed, which
// should never happen for the compiled-in default.
func DefaultTimeout() time.Duration {
	to := DefaultGlobalMap()["timeout"]
	switch tot := to.(type) {
	case time.Duration:
		return tot
	case string:
		ret, err := time.ParseDuration(tot)
		if err == nil {
			return ret
		}
	}
	return -1
}

// UserConfigPath returns the path to the per-user configuration file. It
// honors $XDG_CONFIG_HOME when set, otherwise uses $HOME/.config and finally
// falls back to the current user's home directory.
func UserConfigPath() (string, error) {
	if configHome := os.Getenv("XDG_CONFIG_HOME"); configHome != "" {
		return filepath.Join(configHome, "ochami", "config.yaml"), nil
	}
	if home := os.Getenv("HOME"); home != "" {
		return filepath.Join(home, ".config", "ochami", "config.yaml"), nil
	}
	u, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("unable to fetch current user: %w", err)
	}
	return filepath.Join(u.HomeDir, ".config", "ochami", "config.yaml"), nil
}

// Source identifies a single named configuration source to be loaded and
// merged. A source may be optional, in which case it is silently skipped if the
// underlying file does not exist. This allows callers to represent optional
// system and user configuration files. All other loading failures are returned.
type Source struct {
	// Name is a human-readable identifier used in log and error messages.
	Name string
	// Path is the config file to load. It is ignored if Map is set.
	Path string
	// Map, when non-nil, is loaded instead of a file. It is used to inject
	// built-in defaults.
	Map map[string]any
	// Optional indicates that a missing file should be skipped rather than
	// treated as an error.
	Optional bool
}

// Logger is an optional sink for verbose, human-readable trace messages emitted
// while loading configuration. It is satisfied by simple printf-style loggers.
// A nil Logger disables tracing.
type Logger interface {
	Logf(format string, args ...any)
}

// loadOptions holds optional behavior for the loaders.
type loadOptions struct {
	logger Logger
}

// LoadOption customizes the behavior of the loaders.
type LoadOption func(*loadOptions)

// WithLogger attaches a Logger that receives verbose trace messages during
// loading.
func WithLogger(l Logger) LoadOption {
	return func(o *loadOptions) { o.logger = l }
}

func (o *loadOptions) logf(format string, args ...any) {
	if o.logger != nil {
		o.logger.Logf(format, args...)
	}
}

// LoadDefaults returns a Config populated solely from the built-in defaults. It
// is the equivalent of loading with no config files (e.g. --ignore-config).
func LoadDefaults(opts ...LoadOption) (Config, error) {
	_, cfg, err := loadKoanf([]Source{{Name: "default", Map: DefaultGlobalMap()}}, opts...)
	return cfg, err
}

// LoadFile returns a Config loaded from the built-in defaults merged with the
// single file at path. The file is required; a missing file is an error.
func LoadFile(path string, opts ...LoadOption) (Config, error) {
	_, cfg, err := loadKoanf([]Source{
		{Name: "default", Map: DefaultGlobalMap()},
		{Name: "file", Path: path},
	}, opts...)
	return cfg, err
}

// Load returns a Config produced by merging the built-in defaults with the
// provided sources, in ascending order of priority (later sources win). Each
// source may be a file or an in-memory map and may be marked optional. Cluster
// configurations are merged by name with per-cluster defaults applied, and
// cluster order is preserved by first appearance. The effective configuration
// is validated before being returned.
//
// Load does not read or mutate any package-global state; each call is
// independent, making it safe to build multiple independent configurations.
func Load(sources []Source, opts ...LoadOption) (Config, error) {
	// Always apply built-in defaults first (lowest priority).
	all := make([]Source, 0, len(sources)+1)
	all = append(all, Source{Name: "default", Map: DefaultGlobalMap()})
	all = append(all, sources...)
	_, cfg, err := loadKoanf(all, opts...)
	return cfg, err
}

// LoadMerged returns a Config produced from the standard ochami configuration
// precedence: built-in defaults, then the optional system config file, then the
// optional user config file. Missing system/user files are skipped.
func LoadMerged(opts ...LoadOption) (Config, error) {
	userPath, err := UserConfigPath()
	if err != nil {
		return Config{}, err
	}
	return Load([]Source{
		{Name: "system", Path: SystemConfigFile, Optional: true},
		{Name: "user", Path: userPath, Optional: true},
	}, opts...)
}

// loadKoanf merges the given sources into an effective koanf instance and
// unmarshals it into a Config. The first source is expected to carry the
// built-in defaults. It returns the koanf instance (for callers that need the
// raw effective representation) and the unmarshaled Config.
func loadKoanf(sources []Source, opts ...LoadOption) (*koanf.Koanf, Config, error) {
	var o loadOptions
	for _, opt := range opts {
		opt(&o)
	}
	o.logf("early verbose log messages activated")

	parser := kyaml.Parser()
	k := koanf.NewWithConf(koanfConf)
	clusterAcc := NewClusterAccumulator(koanfConf)

	for _, src := range sources {
		k2 := koanf.NewWithConf(koanfConf)
		var err error
		if src.Map != nil {
			err = k2.Load(confmap.Provider(src.Map, "."), nil)
		} else {
			err = k2.Load(file.Provider(src.Path), parser)
			if errors.Is(err, os.ErrNotExist) {
				if src.Optional {
					o.logf("config '%s' not found, skipping", src.Name)
					continue
				}
				return nil, Config{}, fmt.Errorf("unable to load config '%s': %w", src.Name, err)
			}
		}
		if err != nil {
			return nil, Config{}, fmt.Errorf("unable to load config '%s': %w", src.Name, err)
		}

		o.logf("successfully loaded key-value pairs from config '%s':", src.Name)
		for _, key := range k2.Keys() {
			o.logf("\t%s -> %v", key, k2.Get(key))
		}

		// Reject explicit null values for required global scalars before
		// StrictMerge can turn them into a less useful type mismatch.
		if err := CheckGlobalNulls(k2); err != nil {
			return nil, Config{}, fmt.Errorf("invalid config '%s': %w", src.Name, err)
		}

		var clusters []map[string]any
		if err := k2.Unmarshal("clusters", &clusters); err != nil {
			return nil, Config{}, fmt.Errorf("unable to unmarshal cluster configs from config '%s': %w", src.Name, err)
		}
		for i, cluster := range clusters {
			name, ok := cluster["name"].(string)
			if !ok || name == "" {
				return nil, Config{}, fmt.Errorf("cluster #%d from config '%s' is missing a name", i, src.Name)
			}
			switch cls := cluster["cluster"].(type) {
			case map[string]any:
				if err := clusterAcc.Add(name, cls); err != nil {
					return nil, Config{}, fmt.Errorf("unable to merge cluster '%s' from config '%s': %w", name, src.Name, err)
				}
			case nil:
				if err := clusterAcc.Add(name, map[string]any{}); err != nil {
					return nil, Config{}, fmt.Errorf("unable to merge cluster '%s' from config '%s': %w", name, src.Name, err)
				}
			default:
				return nil, Config{}, fmt.Errorf("unable to merge cluster '%s' from config '%s': value is not a map[string]any type", name, src.Name)
			}
		}

		if err := k.Merge(k2); err != nil {
			return nil, Config{}, fmt.Errorf("unable to merge config '%s': %w", src.Name, err)
		}
		o.logf("successfully merged config '%s'", src.Name)
	}

	// Replace the raw source cluster list with the merged, default-applied,
	// ordered representation. Delete first because StrictMerge rejects the
	// []interface{} to []map[string]any replacement.
	k.Delete("clusters")
	if err := k.Set("clusters", clusterAcc.Slice()); err != nil {
		return nil, Config{}, fmt.Errorf("unable to set merged clusters: %w", err)
	}
	if err := ValidateConfig(k); err != nil {
		return nil, Config{}, fmt.Errorf("invalid merged config: %w", err)
	}

	o.logf("final config:")
	for _, key := range k.Keys() {
		o.logf("\t%s -> %v", key, k.Get(key))
	}

	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, Config{}, fmt.Errorf("unable to unmarshal merged config: %w", err)
	}
	return k, cfg, nil
}
