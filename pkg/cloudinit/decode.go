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
	value, err := parseYAML11(data)
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

// maxYAMLNodes bounds the nodes visited while expanding aliases, which
// protects against documents that expand exponentially.
const maxYAMLNodes = 1 << 20

// parseYAML11 parses a single YAML document as PyYAML's safe_load does:
// YAML 1.1 scalar resolution, duplicate keys keeping the last value, and merge
// keys. Mappings become *gj.OrderedDict with string keys. An empty document is
// nil.
func parseYAML11(data []byte) (any, error) {
	if err := checkTabs(data); err != nil {
		return nil, err
	}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	var root yaml.Node
	if err := dec.Decode(&root); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, nil
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
	d := &yamlDecoder{active: map[*yaml.Node]bool{}}
	return d.node(&root)
}

type yamlDecoder struct {
	visited int
	active  map[*yaml.Node]bool
}

// explicitTag returns a scalar's explicit tag, or "" when its tag was
// resolved implicitly.
func explicitTag(node *yaml.Node) string {
	if node.Style&yaml.TaggedStyle != 0 {
		return node.Tag
	}
	return ""
}

func (d *yamlDecoder) node(node *yaml.Node) (any, error) {
	if d.visited++; d.visited > maxYAMLNodes {
		return nil, fmt.Errorf("YAML document expands to more than %d nodes", maxYAMLNodes)
	}
	switch node.Kind {
	case yaml.DocumentNode:
		if len(node.Content) == 0 {
			return nil, nil
		}
		return d.node(node.Content[0])
	case yaml.AliasNode:
		if d.active[node.Alias] {
			return nil, fmt.Errorf("recursive YAML alias (line %d)", node.Line)
		}
		d.active[node.Alias] = true
		defer delete(d.active, node.Alias)
		return d.node(node.Alias)
	case yaml.MappingNode:
		if tag := explicitTag(node); tag != "" && tag != "!!map" {
			return nil, fmt.Errorf("unsupported YAML tag %q", tag)
		}
		return d.mapping(node)
	case yaml.SequenceNode:
		if tag := explicitTag(node); tag != "" && tag != "!!seq" {
			return nil, fmt.Errorf("unsupported YAML tag %q", tag)
		}
		items := make([]any, 0, len(node.Content))
		for _, child := range node.Content {
			value, err := d.node(child)
			if err != nil {
				return nil, err
			}
			items = append(items, value)
		}
		return items, nil
	case yaml.ScalarNode:
		tag := explicitTag(node)
		switch {
		case tag != "":
		case node.Style == 0:
			tag = resolvePlain(node.Value)
		default:
			tag = tagStr
		}
		value, err := constructScalar(tag, node.Value)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", node.Line, err)
		}
		return value, nil
	default:
		return nil, fmt.Errorf("unsupported YAML node kind %d", node.Kind)
	}
}

// isMergeKey reports whether a mapping key is the YAML merge key.
func isMergeKey(key *yaml.Node) bool {
	if key.Kind != yaml.ScalarNode {
		return false
	}
	if tag := explicitTag(key); tag != "" {
		return tag == tagMerge
	}
	return key.Style == 0 && resolvePlain(key.Value) == tagMerge
}

func (d *yamlDecoder) mapping(node *yaml.Node) (*gj.OrderedDict, error) {
	out := gj.NewOrderedDict()
	for idx := 0; idx+1 < len(node.Content); idx += 2 {
		key, value := node.Content[idx], node.Content[idx+1]
		if isMergeKey(key) {
			if err := d.merge(out, value); err != nil {
				return nil, err
			}
			continue
		}
		if key.Kind != yaml.ScalarNode && key.Kind != yaml.AliasNode {
			return nil, fmt.Errorf("YAML mapping keys must be scalars (line %d)", key.Line)
		}
		k, err := d.node(key)
		if err != nil {
			return nil, err
		}
		if _, isMapping := k.(*gj.OrderedDict); isMapping {
			return nil, fmt.Errorf("YAML mapping keys must be scalars (line %d)", key.Line)
		}
		if _, isList := k.([]any); isList {
			return nil, fmt.Errorf("YAML mapping keys must be scalars (line %d)", key.Line)
		}
		decoded, err := d.node(value)
		if err != nil {
			return nil, err
		}
		out.Set(pyKeyString(k), decoded)
	}
	return out, nil
}

// merge applies a merge key value. As in PyYAML, explicit keys override
// merged keys, and earlier merge sources win over later ones.
func (d *yamlDecoder) merge(out *gj.OrderedDict, node *yaml.Node) error {
	var sources []*gj.OrderedDict
	resolved, err := d.node(node)
	if err != nil {
		return err
	}
	switch v := resolved.(type) {
	case *gj.OrderedDict:
		sources = append(sources, v)
	case []any:
		for _, item := range v {
			mapping, ok := item.(*gj.OrderedDict)
			if !ok {
				return fmt.Errorf("YAML merge value must be a mapping or sequence of mappings")
			}
			sources = append(sources, mapping)
		}
	default:
		return fmt.Errorf("YAML merge value must be a mapping or sequence of mappings")
	}
	for _, source := range sources {
		for _, key := range source.Keys() {
			if _, exists := out.Get(key); !exists {
				value, _ := source.Get(key)
				out.Set(key, value)
			}
		}
	}
	return nil
}
