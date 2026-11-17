// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cloudinit

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// bundleForCase builds a full-mode bundle from a full.json case.
func bundleForCase(t *testing.T, tc fullCase) Bundle {
	t.Helper()
	instanceData, err := json.Marshal(tc.InstanceData)
	if err != nil {
		t.Fatal(err)
	}
	b := Bundle{Mode: BundleFull, CreatedBy: "test", Node: "node1", UserData: []byte(tc.UserData), VendorData: []byte(tc.VendorData), InstanceData: instanceData}
	urls := make([]string, 0, len(tc.Includes))
	for url := range tc.Includes {
		urls = append(urls, url)
	}
	sort.Strings(urls)
	for _, url := range urls {
		entry := tc.Includes[url]
		b.Includes = append(b.Includes, BundleInclude{URL: url, Status: entry.Status, Code: entry.Code, Content: []byte(entry.Content)})
	}
	return b
}

// TestBundle_FullRoundTrip verifies that rendering a written and reloaded
// bundle gives cloud-init's results for every full.json case.
func TestBundle_FullRoundTrip(t *testing.T) {
	t.Parallel()
	for _, tc := range readFullCases(t) {
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			dir := filepath.Join(t.TempDir(), "bundle")
			if err := WriteBundle(dir, bundleForCase(t, tc)); err != nil {
				t.Fatal(err)
			}
			loaded, err := LoadBundle(dir)
			if err != nil {
				t.Fatal(err)
			}
			result, err := RenderFull(t.Context(), loaded.RenderFullRequest(nil))
			if err != nil {
				t.Fatal(err)
			}
			if got := normalizeJSON(t, fullSummary(result)); !reflect.DeepEqual(got, tc.Expected) {
				t.Fatalf("summary = %v, want %v", got, tc.Expected)
			}
		})
	}
}

// TestBundle_Files verifies a template bundle's files and permissions.
func TestBundle_Files(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(t.TempDir(), "bundle")
	b := Bundle{Mode: BundleTemplate, Node: "node1", Template: []byte("## template: jinja\nx\n"), InstanceData: []byte(`{"ds":{}}`)}
	if err := WriteBundle(dir, b); err != nil {
		t.Fatal(err)
	}
	modes := map[string]os.FileMode{".": 0o700, "manifest.json": 0o600, "template": 0o600, "instance-data.json": 0o600, "ochami_render.py": 0o600, "README": 0o600, "run.sh": 0o700}
	for name, want := range modes {
		info, err := os.Stat(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != want {
			t.Errorf("%s mode = %o, want %o", name, got, want)
		}
	}
	script, err := os.ReadFile(filepath.Join(dir, "ochami_render.py"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(script, HelperScript()) {
		t.Error("bundle helper differs from the embedded helper")
	}
	loaded, err := LoadBundle(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(loaded, Bundle{Mode: BundleTemplate, Node: "node1", Template: b.Template, InstanceData: b.InstanceData}) {
		t.Fatalf("loaded = %+v", loaded)
	}
}

// TestBundle_Errors verifies that unsafe or unsupported bundles are refused.
func TestBundle_Errors(t *testing.T) {
	t.Parallel()
	t.Run("non-empty directory", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "x"), nil, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := WriteBundle(dir, Bundle{Mode: BundleTemplate}); err == nil || !strings.Contains(err.Error(), "not empty") {
			t.Fatalf("error = %v, want not empty", err)
		}
	})
	for name, manifest := range map[string]string{
		"outside path":    `{"format":"ochami-cloud-init-render","version":1,"mode":"template","files":{"instance_data":"../secret","template":"t"}}`,
		"absolute path":   `{"format":"ochami-cloud-init-render","version":1,"mode":"template","files":{"instance_data":"/etc/passwd","template":"t"}}`,
		"newer version":   `{"format":"ochami-cloud-init-render","version":2,"mode":"template","files":{"instance_data":"i"}}`,
		"other format":    `{"format":"other","version":1,"mode":"template","files":{"instance_data":"i"}}`,
		"unknown mode":    `{"format":"ochami-cloud-init-render","version":1,"mode":"other","files":{"instance_data":"i"}}`,
		"missing file":    `{"format":"ochami-cloud-init-render","version":1,"mode":"template","files":{"instance_data":"i"}}`,
		"invalid json":    `{`,
		"include outside": `{"format":"ochami-cloud-init-render","version":1,"mode":"full","files":{"instance_data":"i"},"includes":[{"url":"u","status":"resolved","file":"../../x"}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			for file, data := range map[string]string{"manifest.json": manifest, "i": "{}", "t": "x"} {
				if err := os.WriteFile(filepath.Join(dir, file), []byte(data), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := LoadBundle(dir); err == nil {
				t.Fatal("LoadBundle() error = nil")
			}
		})
	}
}
