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
