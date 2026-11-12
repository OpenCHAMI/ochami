// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cloudinit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	gj "github.com/jryberg/gojinja"
)

// parseMapping parses a YAML document into a mapping, with an empty document
// as nil.
func parseMapping(t *testing.T, doc string) *gj.OrderedDict {
	t.Helper()
	value, err := parseYAML11([]byte(doc))
	if err != nil {
		t.Fatal(err)
	}
	if value == nil {
		return nil
	}
	mapping, ok := value.(*gj.OrderedDict)
	if !ok {
		t.Fatalf("document %q is %T, not a mapping", doc, value)
	}
	return mapping
}

// TestMerge_Reference verifies the merger port against cloud-init's results
// recorded in testdata/merge.json.
func TestMerge_Reference(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile(filepath.Join("testdata", "merge.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Cases []struct {
			Name     string   `json:"name"`
			Base     string   `json:"base"`
			Payload  string   `json:"payload"`
			Header   string   `json:"header"`
			Many     []string `json:"many"`
			Expected any      `json:"expected"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	for _, tc := range doc.Cases {
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			var result *gj.OrderedDict
			var err error
			if tc.Many != nil {
				var sources []*gj.OrderedDict
				for _, d := range tc.Many {
					sources = append(sources, parseMapping(t, d))
				}
				result, err = mergeMany(sources)
			} else {
				base := parseMapping(t, tc.Base)
				if base == nil {
					base = gj.NewOrderedDict()
				}
				result, err = mergePart(base, parseMapping(t, tc.Payload), tc.Header)
			}
			var got any
			if err != nil {
				got = map[string]any{"error": true}
			} else {
				encoded, err := json.Marshal(taggedValue(t, result))
				if err != nil {
					t.Fatal(err)
				}
				if err := json.Unmarshal(encoded, &got); err != nil {
					t.Fatal(err)
				}
			}
			if !reflect.DeepEqual(got, tc.Expected) {
				t.Fatalf("merge = %v (error %v), want %v", got, err, tc.Expected)
			}
		})
	}
}

// TestMerge_DoesNotModifyInputs verifies that merging leaves the merged
// configuration's inputs unchanged apart from the removed merger keys.
func TestMerge_DoesNotModifyInputs(t *testing.T) {
	t.Parallel()
	base := parseMapping(t, "d: {x: [1]}\n")
	payload := parseMapping(t, "merge_how: dict(no_replace,recurse_list)+list(append)\nd: {x: [2]}\n")
	if _, err := mergePart(base, payload, ""); err != nil {
		t.Fatal(err)
	}
	if got := pyJSONDumps(base); got != pyJSONDumps(parseMapping(t, "d: {x: [1]}\n")) {
		t.Fatalf("base changed to %s", got)
	}
	if got := pyJSONDumps(payload); got != pyJSONDumps(parseMapping(t, "d: {x: [2]}\n")) {
		t.Fatalf("payload = %s, want merger keys removed only", got)
	}
}
