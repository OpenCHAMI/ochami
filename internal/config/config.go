package config

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/go-viper/mapstructure/v2"
	kyaml "github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/structs"
	"github.com/knadh/koanf/v2"
	"gopkg.in/yaml.v3"
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

type ConfigCluster struct {
	Name    string              `yaml:"name,omitempty"`
	Cluster ConfigClusterConfig `yaml:"cluster,omitempty"`
}

type ConfigClusterConfig struct {
	BaseURI string `yaml:"base-uri,omitempty"`
}

const ProgName = "ochami"

// Default configuration values if either no configuration files exist or the
// configuration files don't contain values for items that need them.
var DefaultConfig = Config{
	Log: ConfigLog{
		Format: "json",
		Level:  "warning",
	},
}

var (
	GlobalConfig = DefaultConfig // Global config struct
	GlobalKoanf  *koanf.Koanf    // Koanf instance for gobal config struct

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

func earlyLog(arg ...interface{}) {
	if EarlyVerbose {
		fmt.Fprintf(os.Stderr, "%s: ", ProgName)
		fmt.Fprintln(os.Stderr, arg...)
	}
}

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

// LoadConfig takes a path and config file format and reads in the file pointed
// to by path, loading it as a configuration file using viper. If path is empty,
// an error is returned. LoadConfig accepts any config file types that viper
// accepts. If format is specified (not empty), its value is used as the
// configuration format. If format is empty, the format is guessed by the config
// file's file extension. If there is no file extension or format is empty, YAML
// format is used.
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
	var userConfigFile string
	user, err := user.Current()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: unable to fetch current user: %v\n", ProgName, err)
		os.Exit(1)
	}
	userConfigFile = filepath.Join(user.HomeDir, ".config", "ochami", "config.yaml")

	// Read config from each file in slice
	type FileCfgMap struct {
		File string
		Cfg  Config
	}
	cfgsToCheck := []FileCfgMap{
		FileCfgMap{File: "/etc/ochami/config.yaml"}, // System config
		FileCfgMap{File: userConfigFile},            // User config
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

// WriteConfig takes a path and config file format and writes the current viper
// configuration to the file pointed to by path in the format specified. If path
// is empty, an error is returned. WriteConfig accepts any config file types
// that viper accepts. If format is empty, the format is guessed by the config
// file's file extension. If there is no file extension and format is empty,
// YAML is used.
func WriteConfig(path string) error {
	if path == "" {
		return fmt.Errorf("no configuration file path passed")
	}
	log.Logger.Debug().Msgf("writing config file: %s", path)

	c, err := yaml.Marshal(GlobalConfig)
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

func mergeConfig(src, dst map[string]interface{}) error {
	// "name" is key used to identify each cluster config in config's
	// cluster list.
	return MergeMaps(src, dst, "name")
}
