// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cloudinit

import (
	"encoding/base64"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"

	gj "github.com/jryberg/gojinja"
)

// taggedValue mirrors tagged() in scripts/cloud-init-reference.py: a JSON
// description of a decoded value that keeps its Python type.
func taggedValue(t *testing.T, value any) any {
	t.Helper()
	switch v := value.(type) {
	case nil:
		return map[string]any{"t": "null"}
	case bool:
		return map[string]any{"t": "bool", "v": v}
	case int64:
		return map[string]any{"t": "int", "v": strconv.FormatInt(v, 10)}
	case float64:
		if math.IsNaN(v) {
			return map[string]any{"t": "float", "v": "nan"}
		}
		return map[string]any{"t": "float", "v": pyFloatRepr(v)}
	case string:
		return map[string]any{"t": "str", "v": v}
	case []byte:
		return map[string]any{"t": "bytes", "v": base64.StdEncoding.EncodeToString(v)}
	case Timestamp:
		if v.DateOnly {
			return map[string]any{"t": "date", "v": v.String()}
		}
		return map[string]any{"t": "datetime", "v": v.String()}
	case []any:
		items := make([]any, len(v))
		for i, item := range v {
			items[i] = taggedValue(t, item)
		}
		return map[string]any{"t": "list", "v": items}
	case *gj.OrderedDict:
		items := []any{}
		for _, key := range sortedKeys(v) {
			child, _ := v.Get(key)
			items = append(items, []any{key, taggedValue(t, child)})
		}
		return map[string]any{"t": "dict", "v": items}
	default:
		t.Fatalf("unexpected decoded type %T", value)
		return nil
	}
}

// TestParseYAML11_Reference verifies YAML decoding against PyYAML's safe_load
// results recorded in testdata/yaml11.json.
func TestParseYAML11_Reference(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile(filepath.Join("testdata", "yaml11.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Cases []struct {
			Name     string `json:"name"`
			YAML     string `json:"yaml"`
			Expected any    `json:"expected"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	for _, tc := range doc.Cases {
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			value, err := parseYAML11([]byte(tc.YAML))
			var got any
			if err != nil {
				got = map[string]any{"error": true}
			} else {
				// Normalize through JSON so types match the decoded file.
				encoded, err := json.Marshal(taggedValue(t, value))
				if err != nil {
					t.Fatal(err)
				}
				if err := json.Unmarshal(encoded, &got); err != nil {
					t.Fatal(err)
				}
			}
			if !reflect.DeepEqual(got, tc.Expected) {
				t.Fatalf("parseYAML11(%q) = %v (error %v), want %v", tc.YAML, got, err, tc.Expected)
			}
		})
	}
}

// TestParseYAML11_Limits verifies the limits ochami adds to PyYAML's rules.
func TestParseYAML11_Limits(t *testing.T) {
	t.Parallel()
	for name, doc := range map[string]string{
		"integer beyond int64": "x: 9223372036854775808\n",
		"recursive alias":      "x: &x [*x]\n",
		"alias expansion":      "a: &a [x,x,x,x,x,x,x,x,x,x]\nb: &b [*a,*a,*a,*a,*a,*a,*a,*a,*a,*a]\nc: &c [*b,*b,*b,*b,*b,*b,*b,*b,*b,*b]\nd: &d [*c,*c,*c,*c,*c,*c,*c,*c,*c,*c]\ne: &e [*d,*d,*d,*d,*d,*d,*d,*d,*d,*d]\nf: &f [*e,*e,*e,*e,*e,*e,*e,*e,*e,*e]\ng: [*f,*f,*f,*f,*f,*f,*f,*f,*f,*f]\n",
	} {
		if _, err := parseYAML11([]byte(doc)); err == nil {
			t.Errorf("%s: parseYAML11() error = nil, want error", name)
		}
	}
}
