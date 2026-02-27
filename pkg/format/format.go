// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package format

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// DataType represents the supported data formats
type DataFormat string

const (
	DataFormatJson       DataFormat = "json"
	DataFormatJsonPretty DataFormat = "json-pretty"
	DataFormatYaml       DataFormat = "yaml"
)

var (
	DataFormatHelp = map[string]string{
		string(DataFormatJson):       "One-line JSON format",
		string(DataFormatJsonPretty): "Unindented JSON format",
		string(DataFormatYaml):       "YAML format",
	}
)

func (df DataFormat) String() string {
	return string(df)
}

func (df *DataFormat) Set(v string) error {
	switch DataFormat(v) {
	case DataFormatJson,
		DataFormatJsonPretty,
		DataFormatYaml:
		*df = DataFormat(v)
		return nil
	default:
		return fmt.Errorf("must be one of %v", []DataFormat{
			DataFormatJson,
			DataFormatJsonPretty,
			DataFormatYaml,
		})
	}
}

func (df DataFormat) Type() string {
	return "DataFormat"
}

// MarshalData marshals arbitrary data into a byte slice formatted as outFormat.
// If a marshalling error occurs or outFormat is unknown, an error is returned.
//
// Supported values are: json, json-pretty, yaml
func MarshalData(data interface{}, outFormat DataFormat) ([]byte, error) {
	switch outFormat {
	case DataFormatJson:
		if bytes, err := json.Marshal(data); err != nil {
			return nil, fmt.Errorf("failed to marshal data into JSON: %w", err)
		} else {
			return bytes, nil
		}
	case DataFormatJsonPretty:
		if bytes, err := json.MarshalIndent(data, "", "  "); err != nil {
			return nil, fmt.Errorf("failed to marshal data into pretty JSON: %w", err)
		} else {
			return bytes, nil
		}
	case DataFormatYaml:
		if bytes, err := yaml.Marshal(data); err != nil {
			return nil, fmt.Errorf("failed to marshal data into YAML: %w", err)
		} else {
			return bytes, nil
		}
	default:
		return nil, fmt.Errorf("unknown data format: %s", outFormat)
	}
}

// UnmarshalData unmarshals a byte slice formatted as inFormat into an interface
// v. If an unmarshalling error occurs or inFormat is unknown, an error is
// returned.
//
// Supported values are: json, json-pretty, yaml
func UnmarshalData(data []byte, v interface{}, inFormat DataFormat) error {
	switch inFormat {
	case DataFormatJson, DataFormatJsonPretty:
		if err := json.Unmarshal(data, v); err != nil {
			return fmt.Errorf("failed to unmarshal data into JSON: %w", err)
		}
	case DataFormatYaml:
		if err := yaml.Unmarshal(data, v); err != nil {
			return fmt.Errorf("failed to unmarshal data into YAML: %w", err)
		}
	default:
		return fmt.Errorf("unknown data format: %s", inFormat)
	}

	return nil
}

// UnmarshalDataSlice unmarshals data formatted as inFormat into v, which is a
// slice of type T. If data is a single T (not a []T), then it is placed in a
// []T such that it is the only element. If data is already a []T, then it is
// unmarshalled into v.
func UnmarshalDataSlice[T any](data []byte, v *[]T, inFormat DataFormat) error {
	switch inFormat {
	case DataFormatJson, DataFormatJsonPretty:
		// JSON
		return unmarshalDataSliceJSON[T](data, v)
	case DataFormatYaml:
		// YAML
		return unmarshalDataSliceYAML[T](data, v)
	default:
		return fmt.Errorf("unknown data format: %s", inFormat)
	}
}

func unmarshalDataSliceJSON[T any](data []byte, v *[]T) error {
	if v == nil {
		return fmt.Errorf("cannot unmarshal JSON into nil slice")
	}

	switch firstNonSpaceByte(data) {
	case '{':
		var one T
		if err := json.Unmarshal(data, &one); err != nil {
			return fmt.Errorf("failed to unmarshal single JSON object: %w", err)
		}
		*v = []T{one}
		return nil
	case '[':
		var many []T
		if err := json.Unmarshal(data, &many); err != nil {
			return fmt.Errorf("failed to unmarshal JSON array: %w", err)
		}
		*v = many
		return nil
	default:
		return fmt.Errorf("failed to unmarshal JSON: expected object or array")
	}
}

func unmarshalDataSliceYAML[T any](data []byte, v *[]T) error {
	// YAML can represent sequences in two common styles:
	//
	//   - block sequence (starts with '-' after whitespace/comments
	//   - flow sequence (starts with '[')
	//
	// Mappings can also start with '{' (flow mapping) or a key (block mapping).

	if v == nil {
		return fmt.Errorf("cannot unmarshal YAML into nil slice")
	}

	// Parse the YAML into a node so it can be inspected.
	var n yaml.Node
	if err := yaml.Unmarshal(data, &n); err != nil {
		return fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	// Inspect what kind of YAML node the root node is.
	root := &n
	if n.Kind == yaml.DocumentNode && len(n.Content) > 0 {
		// If the first YAML node is a document, descend to the first node
		// within the document and inspect that instead.
		root = n.Content[0]
	}

	switch root.Kind {
	case yaml.MappingNode:
		// YAML node is a dictionary
		var one T
		if err := yaml.Unmarshal(data, &one); err != nil {
			return fmt.Errorf("failed to unmarshal single YAML mapping: %w", err)
		}
		*v = []T{one}
		return nil
	case yaml.SequenceNode:
		// YAML node is an array
		var many []T
		if err := yaml.Unmarshal(data, &many); err != nil {
			return fmt.Errorf("failed to unmarshal YAML sequence: %w", err)
		}
		*v = many
		return nil
	default:
		return fmt.Errorf("failed to unmarshal YAML: expected mapping or sequence, got %v", root.Kind)
	}
}

// SetNestedField sets a field in a nested map using dot notation
// Example: SetNestedField(map, "status.health", "OK") sets map["status"]["health"] = "OK"
func SetNestedField(target map[string]interface{}, path string, value interface{}) {
	if target == nil {
		return
	}

	// Split + drop empty segments to avoid "" keys and slice issues.
	raw := strings.Split(path, ".")
	parts := make([]string, 0, len(raw))
	for _, p := range raw {
		if p != "" {
			parts = append(parts, p)
		}
	}
	if len(parts) == 0 {
		return
	}

	current := target

	// Navigate to the parent of the target field
	for _, part := range parts[:len(parts)-1] {
		next, ok := current[part].(map[string]interface{})
		if !ok || next == nil {
			next = make(map[string]interface{})
			current[part] = next
		}
		current = next
	}

	// Set the final field
	finalField := parts[len(parts)-1]
	if value == nil {
		// For unset operations, we use JSON Merge Patch null semantics
		current[finalField] = nil
		return
	}

	if stringValue, ok := value.(string); ok {
		// Try to parse as JSON first, then as string
		var jsonValue interface{}
		if err := json.Unmarshal([]byte(stringValue), &jsonValue); err == nil {
			current[finalField] = jsonValue
		} else {
			current[finalField] = stringValue
		}
		return
	}

	current[finalField] = value
}

func firstNonSpaceByte(b []byte) byte {
	b = bytes.TrimSpace(b)
	if len(b) == 0 {
		return 0
	}
	return b[0]
}
