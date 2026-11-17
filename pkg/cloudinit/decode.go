// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cloudinit

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	gj "github.com/jryberg/gojinja"
	"gopkg.in/yaml.v3"

	"github.com/openchami/ochami/pkg/format"
)

func decodeDocument(doc Document) (*gj.OrderedDict, error) {
	switch doc.Format {
	case "", format.DataFormatJson, format.DataFormatJsonPretty:
		return decodeJSON(doc.Data)
	case format.DataFormatYaml:
		return decodeYAML(doc.Data)
	default:
		return nil, fmt.Errorf("unsupported document format %q", doc.Format)
	}
}

func decodeJSON(data []byte) (*gj.OrderedDict, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		return gj.NewOrderedDict(), nil
	}
	// JSONVars accepts trailing input and converts oversized integers to floats.
	// Validate the complete document and integer range before using its ordered decoder.
	if !json.Valid(data) {
		return nil, fmt.Errorf("invalid JSON document")
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		if number, ok := tok.(json.Number); ok && !strings.ContainsAny(number.String(), ".eE") {
			if _, err := strconv.ParseInt(number.String(), 10, 64); err != nil {
				return nil, fmt.Errorf("integer outside signed 64-bit range: %w", err)
			}
		}
	}
	return gj.JSONVars(data)
}

func decodeYAML(data []byte) (*gj.OrderedDict, error) {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	var root yaml.Node
	if err := dec.Decode(&root); err != nil {
		if errors.Is(err, io.EOF) {
			return gj.NewOrderedDict(), nil
		}
		return nil, err
	}
	var next yaml.Node
	if err := dec.Decode(&next); !errors.Is(err, io.EOF) {
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("expected a single YAML document")
	}
	// yaml.v3 validates duplicate keys and limits alias expansion before the
	// ordered walk. This temporary value is never used as rendering input.
	var validated any
	if err := root.Decode(&validated); err != nil {
		return nil, err
	}
	value, err := decodeYAMLNode(&root)
	if err != nil {
		return nil, err
	}
	if value == nil {
		return gj.NewOrderedDict(), nil
	}
	mapping, ok := value.(*gj.OrderedDict)
	if !ok {
		return nil, fmt.Errorf("expected a YAML mapping, got %T", value)
	}
	return mapping, nil
}

func decodeYAMLNode(node *yaml.Node) (any, error) {
	switch node.Kind {
	case yaml.DocumentNode:
		return decodeYAMLNode(node.Content[0])
	case yaml.AliasNode:
		return decodeYAMLNode(node.Alias)
	case yaml.MappingNode:
		if node.Tag != "!!map" {
			return nil, fmt.Errorf("unsupported YAML tag %q", node.Tag)
		}
		return decodeYAMLMapping(node)
	case yaml.SequenceNode:
		if node.Tag != "!!seq" {
			return nil, fmt.Errorf("unsupported YAML tag %q", node.Tag)
		}
		items := make([]any, 0, len(node.Content))
		for _, child := range node.Content {
			value, err := decodeYAMLNode(child)
			if err != nil {
				return nil, err
			}
			items = append(items, value)
		}
		return items, nil
	case yaml.ScalarNode:
		// yaml.v3 resolves decimal integers beyond uint64 as floats. Reject
		// that implicit conversion rather than silently losing precision.
		if node.Style == 0 {
			literal := strings.ReplaceAll(node.Value, "_", "")
			digits := strings.TrimLeft(literal, "+-")
			if digits != "" && strings.Trim(digits, "0123456789") == "" {
				if _, err := strconv.ParseInt(literal, 10, 64); err != nil {
					return nil, fmt.Errorf("integer outside signed 64-bit range: %w", err)
				}
			}
		}
		switch node.Tag {
		case "!!int":
			var value int64
			if err := node.Decode(&value); err != nil {
				return nil, fmt.Errorf("invalid signed 64-bit integer: %w", err)
			}
			return value, nil
		case "!!str", "!!bool", "!!null", "!!float", "!!timestamp", "!!binary":
			var value any
			if err := node.Decode(&value); err != nil {
				return nil, err
			}
			return value, nil
		default:
			return nil, fmt.Errorf("unsupported YAML tag %q", node.Tag)
		}
	default:
		return nil, fmt.Errorf("unsupported YAML node kind %d", node.Kind)
	}
}

func decodeYAMLMapping(node *yaml.Node) (*gj.OrderedDict, error) {
	out := gj.NewOrderedDict()
	for idx := 0; idx < len(node.Content); idx += 2 {
		key, value := node.Content[idx], node.Content[idx+1]
		if key.Tag == "!!merge" {
			if err := mergeYAML(out, value); err != nil {
				return nil, err
			}
			continue
		}
		if key.Kind != yaml.ScalarNode || key.Tag != "!!str" {
			return nil, fmt.Errorf("YAML mapping keys must be strings (line %d)", key.Line)
		}
		decoded, err := decodeYAMLNode(value)
		if err != nil {
			return nil, err
		}
		out.Set(key.Value, decoded)
	}
	return out, nil
}

func mergeYAML(out *gj.OrderedDict, node *yaml.Node) error {
	if node.Kind == yaml.AliasNode {
		return mergeYAML(out, node.Alias)
	}
	if node.Kind == yaml.SequenceNode {
		for _, child := range node.Content {
			if err := mergeYAML(out, child); err != nil {
				return err
			}
		}
		return nil
	}
	if node.Kind != yaml.MappingNode || node.Tag != "!!map" {
		return fmt.Errorf("YAML merge value must be a mapping or sequence of mappings")
	}
	merged, err := decodeYAMLMapping(node)
	if err != nil {
		return err
	}
	for _, key := range merged.Keys() {
		if _, exists := out.Get(key); !exists {
			value, _ := merged.Get(key)
			out.Set(key, value)
		}
	}
	return nil
}
