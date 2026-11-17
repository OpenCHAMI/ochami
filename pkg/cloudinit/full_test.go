// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cloudinit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type fullCase struct {
	Name       string `json:"name"`
	UserData   string `json:"user_data"`
	VendorData string `json:"vendor_data"`
	Includes   map[string]struct {
		Status  string `json:"status"`
		Content string `json:"content"`
		Code    int    `json:"code"`
	} `json:"includes"`
	InstanceData any `json:"instance_data"`
	Expected     any `json:"expected"`
}

func readFullCases(t *testing.T) []fullCase {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "full.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Cases []fullCase `json:"cases"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	return doc.Cases
}

// fullRequest builds the request for a case, fetching includes from it.
func fullRequest(t *testing.T, tc fullCase) FullRequest {
	t.Helper()
	instanceData, err := json.Marshal(tc.InstanceData)
	if err != nil {
		t.Fatal(err)
	}
	return FullRequest{
		UserData:     []byte(tc.UserData),
		VendorData:   []byte(tc.VendorData),
		InstanceData: instanceData,
		Fetch: func(_ context.Context, url string) ([]byte, error) {
			entry, ok := tc.Includes[url]
			switch {
			case !ok || entry.Status == "unresolved":
				return nil, fmt.Errorf("%s: %w", url, ErrIncludeUnresolved)
			case entry.Status == "failed":
				return nil, fmt.Errorf("HTTP %d", entry.Code)
			}
			return []byte(entry.Content), nil
		},
	}
}

// fullSummary mirrors full_summary() in scripts/cloud-init-reference.py.
func fullSummary(result FullResult) map[string]any {
	section := func(data, failed string) map[string]any {
		parts := []any{}
		for _, part := range result.Parts {
			if part.Data == data && part.Name != "" {
				parts = append(parts, []any{part.Name, string(part.Status)})
			}
		}
		return map[string]any{"failed": failed != "", "parts": parts}
	}
	vendor := section("vendor-data", result.VendorFailed)
	vendor["disabled"] = result.VendorDisabled
	return map[string]any{
		"user":   section("user-data", result.UserFailed),
		"vendor": vendor,
		"merged": result.Comparable(),
	}
}

// normalizeJSON round-trips value through JSON so it compares with decoded
// expectations.
func normalizeJSON(t *testing.T, value any) any {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var out any
	if err := json.Unmarshal(encoded, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

// TestRenderFull_Reference verifies full rendering against cloud-init's own
// processing, recorded in testdata/full.json.
func TestRenderFull_Reference(t *testing.T) {
	t.Parallel()
	for _, tc := range readFullCases(t) {
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			result, err := RenderFull(t.Context(), fullRequest(t, tc))
			if err != nil {
				t.Fatal(err)
			}
			got := normalizeJSON(t, fullSummary(result))
			if !reflect.DeepEqual(got, tc.Expected) {
				t.Fatalf("summary =\n%v\nwant\n%v\nparts: %+v", got, tc.Expected, result.Parts)
			}
		})
	}
}

// TestRenderFull_Reporting verifies the details ochami reports beyond what
// cloud-init's results show.
func TestRenderFull_Reporting(t *testing.T) {
	t.Parallel()
	cases := map[string]fullCase{}
	for _, tc := range readFullCases(t) {
		cases[tc.Name] = tc
	}
	run := func(name string) FullResult {
		t.Helper()
		result, err := RenderFull(t.Context(), fullRequest(t, cases[name]))
		if err != nil {
			t.Fatal(err)
		}
		return result
	}

	t.Run("sources", func(t *testing.T) {
		result := run("groups merged in order")
		var sources []string
		for _, part := range result.Parts {
			if part.Data == "vendor-data" {
				sources = append(sources, part.Source)
			}
		}
		want := []string{"http://ci.example/cloud-init/a.yaml", "http://ci.example/cloud-init/b.yaml"}
		if !reflect.DeepEqual(sources, want) {
			t.Fatalf("sources = %v, want %v", sources, want)
		}
		text := result.Text()
		for _, line := range []string{"#cloud-config", "# from 2 files", "# vendor-data part-001 (http://ci.example/cloud-init/a.yaml)", "hostname: b"} {
			if !strings.Contains(text, line+"\n") {
				t.Errorf("Text() missing %q:\n%s", line, text)
			}
		}
	})
	t.Run("unresolved and failed includes", func(t *testing.T) {
		result := run("include unresolved")
		if result.Parts[1].Status != PartUnresolved || result.Parts[1].Name != "" {
			t.Fatalf("part = %+v, want an unnamed unresolved include", result.Parts[1])
		}
		result = run("include fetch fails")
		last := result.Parts[len(result.Parts)-1]
		if last.Status != PartFailed || !strings.Contains(result.VendorFailed, "HTTP 404") {
			t.Fatalf("last part = %+v, vendor failure %q", last, result.VendorFailed)
		}
	})
	t.Run("headerless template warning", func(t *testing.T) {
		result := run("headerless template")
		part := result.Parts[len(result.Parts)-1]
		if len(part.Warnings) != 1 || !strings.Contains(part.Warnings[0], "## template: jinja") {
			t.Fatalf("warnings = %v", part.Warnings)
		}
	})
	t.Run("quoted YAML 1.1 strings", func(t *testing.T) {
		result, err := RenderFull(t.Context(), FullRequest{VendorData: []byte("#cloud-config\na: 'yes'\nb: '0644'\nc: 'null'\n")})
		if err != nil {
			t.Fatal(err)
		}
		text := result.Text()
		for _, line := range []string{`a: "yes"`, `b: "0644"`, `c: "null"`} {
			if !strings.Contains(text, line) {
				t.Errorf("Text() missing %s:\n%s", line, text)
			}
		}
		reparsed, err := parseYAML11([]byte(text))
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(typedDescription(reparsed), result.Comparable()) {
			t.Fatalf("Text() does not load back to the merged configuration:\n%s", text)
		}
	})
	t.Run("cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		if _, err := RenderFull(ctx, fullRequest(t, cases["nested include"])); !errors.Is(err, context.Canceled) {
			t.Fatalf("error = %v, want context.Canceled", err)
		}
	})
}
