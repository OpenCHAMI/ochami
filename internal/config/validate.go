// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package config

import (
	"fmt"
	"strconv"
	"time"

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
// defaults in DefaultConfigMap.
var requiredGlobalScalars = []string{
	"log.format",
	"log.level",
	"log.color",
	"timeout",
	"default-input-format",
	"default-output-format",
}

// checkGlobalNulls returns an ErrInvalidConfigVal if any required global scalar
// key exists in ko but is explicitly null or an empty string. This is checked per-source
// before merging so that a clean validation error is surfaced instead of the cryptic
// type-mismatch error StrictMerge would otherwise produce.
func checkGlobalNulls(ko *koanf.Koanf) error {
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

// validateConfig performs semantic validation on a fully-merged (effective)
// koanf instance. It rejects explicitly-null required values and type-incorrect
// values for known keys. For boolean cluster keys such as enable-auth, string
// values are coerced for backward tolerance.
//
// It returns an ErrInvalidConfigVal describing the first problem encountered, or
// nil if the config is valid.
func validateConfig(ko *koanf.Koanf) error {
	if err := checkGlobalNulls(ko); err != nil {
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
// DefaultClusterConfigMap default.
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
