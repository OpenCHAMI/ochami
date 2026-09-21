// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package config

import (
	"fmt"
	"strconv"
	"time"

	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/v2"
)

// coerceBool attempts to interpret v as a boolean, tolerating string values for
// backward compatibility (e.g. a value written as the string "true" rather than
// the YAML boolean true). It returns the boolean value and true on success, or
// false and false if v cannot be interpreted as a boolean.
func coerceBool(v any) (bool, bool) {
	switch b := v.(type) {
	case bool:
		return b, true
	case string:
		parsed, err := strconv.ParseBool(b)
		if err != nil {
			return false, false
		}
		return parsed, true
	default:
		return false, false
	}
}

// requiredGlobalScalars lists the global scalar keys that must never be
// explicitly null in a config source. These correspond to required values with
// defaults in DefaultGlobalMap.
var requiredGlobalScalars = []string{
	"log.format",
	"log.level",
	"log.color",
	"timeout",
	"default-input-format",
	"default-output-format",
}

// CheckGlobalNulls returns an ErrInvalidConfigVal if any required global scalar
// key exists in ko but is explicitly null or an empty string. This is checked
// per-source before merging so that a clean validation error is surfaced
// instead of the cryptic type-mismatch error StrictMerge would otherwise
// produce.
//
// It is a low-level helper exposed for the internal config-file tooling and is
// not intended for general external use.
func CheckGlobalNulls(ko *koanf.Koanf) error {
	for _, key := range requiredGlobalScalars {
		if ko.Exists(key) {
			val := ko.Get(key)
			if val == nil {
				return ErrInvalidConfigVal{Key: key, Value: "null", Expected: "non-null value"}
			}
			if s, ok := val.(string); ok && s == "" {
				return ErrInvalidConfigVal{Key: key, Value: "empty string", Expected: "non-empty value"}
			}
		}
	}
	return nil
}

// ValidateConfig performs semantic validation on a fully-merged (effective)
// koanf instance. It rejects explicitly-null required values and type-incorrect
// values for known keys. For boolean cluster keys such as enable-auth, string
// values are coerced for backward tolerance.
//
// It returns an ErrInvalidConfigVal describing the first problem encountered, or
// nil if the config is valid. It is a low-level helper exposed for the internal
// config-file tooling and is not intended for general external use.
func ValidateConfig(ko *koanf.Koanf) error {
	if err := CheckGlobalNulls(ko); err != nil {
		return err
	}

	// timeout must be a valid, non-null duration string.
	if ko.Exists("timeout") {
		v := ko.Get("timeout")
		if s, ok := v.(string); ok {
			if _, err := time.ParseDuration(s); err != nil {
				return ErrInvalidConfigVal{Key: "timeout", Value: s, Expected: "duration string (e.g. \"30s\")"}
			}
		}
	}

	// Validate per-cluster enable-auth values, coercing strings.
	var clusters []map[string]any
	if err := ko.Unmarshal("clusters", &clusters); err != nil {
		return fmt.Errorf("unable to unmarshal clusters for validation: %w", err)
	}
	for _, c := range clusters {
		name, _ := c["name"].(string)
		cl, ok := c["cluster"].(map[string]any)
		if !ok {
			continue
		}
		if v, exists := cl["enable-auth"]; exists {
			if v == nil {
				return ErrInvalidConfigVal{
					Key:      fmt.Sprintf("clusters[%s].cluster.enable-auth", name),
					Value:    "null",
					Expected: "boolean",
				}
			}
			if _, ok := coerceBool(v); !ok {
				return ErrInvalidConfigVal{
					Key:      fmt.Sprintf("clusters[%s].cluster.enable-auth", name),
					Value:    fmt.Sprintf("%v", v),
					Expected: "boolean",
				}
			}
		}
	}

	return nil
}

// normalizeClusterBools coerces string boolean values (e.g. "true") for known
// boolean cluster keys into actual booleans within a cluster's "cluster"
// sub-map, mutating it in place. This is applied before the config is merged so
// that StrictMerge does not fail on a string-vs-bool mismatch against the
// DefaultClusterMap default.
//
// If a known boolean key is present but explicitly null or otherwise not
// coercible to a boolean, an ErrInvalidConfigVal is returned so a clean error
// is surfaced instead of the cryptic type-mismatch error from StrictMerge.
func normalizeClusterBools(name string, cluster map[string]any) error {
	if v, ok := cluster["enable-auth"]; ok {
		key := fmt.Sprintf("clusters[%s].cluster.enable-auth", name)
		if v == nil {
			return ErrInvalidConfigVal{Key: key, Value: "null", Expected: "boolean"}
		}
		b, ok := coerceBool(v)
		if !ok {
			return ErrInvalidConfigVal{Key: key, Value: fmt.Sprintf("%v", v), Expected: "boolean"}
		}
		cluster["enable-auth"] = b
	}
	return nil
}

// ClusterAccumulator merges cluster configurations by name across multiple
// sources while preserving the order in which cluster names are first seen.
// This provides deterministic output regardless of Go's map iteration order.
//
// It is a low-level helper exposed for the internal config-file tooling and is
// not intended for general external use.
type ClusterAccumulator struct {
	conf   koanf.Conf              // koanf configuration for per-cluster instances
	order  []string                // cluster names in first-seen order
	byName map[string]*koanf.Koanf // per-cluster merged koanf instance
}

// NewClusterAccumulator returns an initialized ClusterAccumulator that builds
// per-cluster koanf instances using conf.
func NewClusterAccumulator(conf koanf.Conf) *ClusterAccumulator {
	return &ClusterAccumulator{conf: conf, byName: map[string]*koanf.Koanf{}}
}

// Add merges a single cluster's config (the "cluster" sub-map) into the
// accumulator under the given name, applying DefaultClusterMap the first time a
// name is seen. Later calls for the same name merge on top of earlier ones
// (higher-priority sources should be added last).
func (ca *ClusterAccumulator) Add(name string, cluster map[string]any) error {
	if ca.byName[name] == nil {
		ca.order = append(ca.order, name)
		ca.byName[name] = koanf.NewWithConf(ca.conf)
		if err := ca.byName[name].Load(confmap.Provider(DefaultClusterMap(), "."), nil); err != nil {
			return fmt.Errorf("unable to load default cluster config: %w", err)
		}
	}
	// Coerce string booleans (e.g. "true") into real booleans so that
	// StrictMerge does not fail merging against the typed defaults in
	// DefaultClusterMap. This also rejects null/invalid boolean values with
	// a clean error.
	if err := normalizeClusterBools(name, cluster); err != nil {
		return err
	}
	if err := ca.byName[name].Load(confmap.Provider(cluster, ""), nil); err != nil {
		return fmt.Errorf("unable to merge cluster '%s': %w", name, err)
	}
	return nil
}

// Slice returns the accumulated clusters as a slice of maps suitable for
// koanf.Set("clusters", ...), in first-seen order.
func (ca *ClusterAccumulator) Slice() []map[string]any {
	clusterSlice := make([]map[string]any, 0, len(ca.order))
	for _, name := range ca.order {
		clusterSlice = append(clusterSlice, map[string]any{
			"name":    name,
			"cluster": ca.byName[name].Raw(),
		})
	}
	return clusterSlice
}
