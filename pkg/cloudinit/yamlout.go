// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cloudinit

import (
	"bytes"
	"encoding/base64"
	"math"
	"strconv"
	"strings"

	gj "github.com/jryberg/gojinja"
	"gopkg.in/yaml.v3"
)

// dumpYAML formats value as cloud-init's safeyaml.dumps does: block style,
// sorted keys, indent 4, and explicit document start and end markers. Strings
// that YAML 1.1 would read as another type are quoted, so cloud-init loads
// the output back as the same values.
func dumpYAML(value any) string {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(4)
	if err := enc.Encode(yamlNode(value)); err != nil {
		// Nodes built by yamlNode always encode.
		panic(err)
	}
	if err := enc.Close(); err != nil {
		panic(err)
	}
	return "---\n" + buf.String() + "...\n"
}

func yamlNode(value any) *yaml.Node {
	scalar := func(tag, text string) *yaml.Node {
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: tag, Value: text}
	}
	switch v := value.(type) {
	case nil:
		return scalar("!!null", "null")
	case bool:
		return scalar("!!bool", strconv.FormatBool(v))
	case int64:
		return scalar("!!int", strconv.FormatInt(v, 10))
	case float64:
		return scalar("!!float", pyYAMLFloat(v))
	case string:
		node := scalar("!!str", v)
		if resolvePlain(v) != tagStr {
			node.Style = yaml.DoubleQuotedStyle
		}
		return node
	case []byte:
		return scalar("!!binary", base64.StdEncoding.EncodeToString(v))
	case Timestamp:
		return scalar("!!timestamp", v.String())
	case []any:
		node := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		for _, item := range v {
			node.Content = append(node.Content, yamlNode(item))
		}
		return node
	case *gj.OrderedDict:
		node := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		for _, key := range sortedKeys(v) {
			child, _ := v.Get(key)
			node.Content = append(node.Content, yamlNode(key), yamlNode(child))
		}
		return node
	default:
		return scalar("!!str", pyKeyString(v))
	}
}

// pyYAMLFloat mirrors PyYAML's SafeRepresenter.represent_float.
func pyYAMLFloat(f float64) string {
	switch {
	case math.IsNaN(f):
		return ".nan"
	case math.IsInf(f, 1):
		return ".inf"
	case math.IsInf(f, -1):
		return "-.inf"
	}
	s := strings.ToLower(pyFloatRepr(f))
	if !strings.Contains(s, ".") && strings.Contains(s, "e") {
		s = strings.Replace(s, "e", ".0e", 1)
	}
	return s
}
