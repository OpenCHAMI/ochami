// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

//go:build cloudinit_integration

package cloud_init

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/openchami/ochami/pkg/cloudinit"
)

// These tests run a real cloud-init. Set OCHAMI_TEST_CLOUD_INIT_SRC to a
// cloud-init source tree and OCHAMI_TEST_CLOUD_INIT_PYTHON to an interpreter
// with its dependencies. Results must match the recorded cloud-init results
// exactly for the reference release in testdata/reference.json. Other
// releases must run, and their differences are logged.

const testdata = "../../../pkg/cloudinit/testdata"

func readTestdata(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(testdata, name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// integrationDelegate returns a delegate for the configured cloud-init and
// whether it is the reference release.
func integrationDelegate(t *testing.T) (*Delegate, bool) {
	t.Helper()
	src := os.Getenv("OCHAMI_TEST_CLOUD_INIT_SRC")
	if src == "" {
		t.Skip("OCHAMI_TEST_CLOUD_INIT_SRC is not set")
	}
	python := os.Getenv("OCHAMI_TEST_CLOUD_INIT_PYTHON")
	if python == "" {
		python = "python3"
	}
	d := &Delegate{Python: python, Src: src, Warn: func(w string) { t.Logf("cloud-init: %s", w) }}
	if err := d.Probe(t.Context()); err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		CloudInit string `json:"cloud_init"`
	}
	if err := json.Unmarshal(readTestdata(t, "reference.json"), &manifest); err != nil {
		t.Fatal(err)
	}
	t.Logf("cloud-init %s, reference %s", d.Version(), manifest.CloudInit)
	return d, d.Version() == manifest.CloudInit
}

// check fails on a difference for the reference release and logs it for
// others.
func check(t *testing.T, reference bool, what string, got, want any) {
	t.Helper()
	if reflect.DeepEqual(got, want) {
		return
	}
	if reference {
		t.Errorf("%s = %v, want %v", what, got, want)
		return
	}
	t.Logf("%s differs from the reference release: %v, reference %v", what, got, want)
}

// TestCloudInitIntegration_Templates verifies that cloud-init devel render
// matches the internal engine on the reference fixtures.
func TestCloudInitIntegration_Templates(t *testing.T) {
	d, reference := integrationDelegate(t)
	var manifest struct {
		Fixtures []struct {
			Name, Template string
			InstanceData   string `json:"instance_data"`
		} `json:"fixtures"`
	}
	if err := json.Unmarshal(readTestdata(t, "reference.json"), &manifest); err != nil {
		t.Fatal(err)
	}
	for _, f := range manifest.Fixtures {
		t.Run(f.Name, func(t *testing.T) {
			template, instanceData := readTestdata(t, f.Template), readTestdata(t, f.InstanceData)
			internal, err := cloudinit.Render(t.Context(), cloudinit.RenderRequest{Template: template, InstanceData: cloudinit.Document{Data: instanceData}})
			if err != nil {
				t.Fatal(err)
			}
			delegated, err := d.RenderTemplate(t.Context(), template, instanceData)
			if err != nil {
				if reference {
					t.Fatal(err)
				}
				// Older releases may reject what the internal engine renders,
				// such as filters newer Jinja2 releases added.
				t.Logf("cloud-init %s cannot render this template, which a node would skip: %v", d.Version(), err)
				return
			}
			if internal != delegated {
				diff := UnifiedDiff("internal", "cloud-init "+d.Version(), internal, delegated)
				if reference {
					t.Errorf("rendering differs:\n%s", diff)
				} else {
					t.Logf("rendering differs from the internal engine:\n%s", diff)
				}
			}
		})
	}
}

type integrationCase struct {
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

func integrationCases(t *testing.T) []integrationCase {
	t.Helper()
	var doc struct {
		Cases []integrationCase `json:"cases"`
	}
	if err := json.Unmarshal(readTestdata(t, "full.json"), &doc); err != nil {
		t.Fatal(err)
	}
	return doc.Cases
}

func (tc integrationCase) bundle(t *testing.T) cloudinit.Bundle {
	t.Helper()
	instanceData, err := json.Marshal(tc.InstanceData)
	if err != nil {
		t.Fatal(err)
	}
	b := cloudinit.Bundle{Mode: cloudinit.BundleFull, UserData: []byte(tc.UserData), VendorData: []byte(tc.VendorData), InstanceData: instanceData}
	urls := make([]string, 0, len(tc.Includes))
	for url := range tc.Includes {
		urls = append(urls, url)
	}
	sort.Strings(urls)
	for _, url := range urls {
		e := tc.Includes[url]
		b.Includes = append(b.Includes, cloudinit.BundleInclude{URL: url, Status: e.Status, Code: e.Code, Content: []byte(e.Content)})
	}
	return b
}

// summary mirrors full_summary() in scripts/cloud-init-reference.py.
func summary(t *testing.T, out FullOutput) any {
	t.Helper()
	section := func(data, failed string) map[string]any {
		parts := []any{}
		for _, p := range out.Parts {
			if p.Data == data && p.Name != "" {
				parts = append(parts, []any{p.Name, string(p.Status)})
			}
		}
		return map[string]any{"failed": failed != "", "parts": parts}
	}
	vendor := section("vendor-data", out.VendorFailed)
	vendor["disabled"] = out.VendorDisabled
	value := map[string]any{"user": section("user-data", out.UserFailed), "vendor": vendor, "merged": out.Comparable}
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var normalized any
	if err := json.Unmarshal(encoded, &normalized); err != nil {
		t.Fatal(err)
	}
	return normalized
}

// TestCloudInitIntegration_Full verifies the embedded helper's full render
// against the recorded cloud-init results.
func TestCloudInitIntegration_Full(t *testing.T) {
	d, reference := integrationDelegate(t)
	if !d.Exact() {
		t.Fatal("the configured interpreter cannot import cloud-init")
	}
	for _, tc := range integrationCases(t) {
		t.Run(tc.Name, func(t *testing.T) {
			out, err := d.RenderFull(t.Context(), tc.bundle(t))
			if err != nil {
				t.Fatal(err)
			}
			check(t, reference, "summary", summary(t, out), tc.Expected)
		})
	}
}

// TestCloudInitIntegration_PerPart verifies full rendering through a
// cloud-init wrapper without a Python interpreter, where cloud-init renders
// each template and ochami merges the parts.
func TestCloudInitIntegration_PerPart(t *testing.T) {
	exact, reference := integrationDelegate(t)
	dir := t.TempDir()
	boot := filepath.Join(dir, "boot.py")
	if err := os.WriteFile(boot, []byte(bootstrap), 0o600); err != nil {
		t.Fatal(err)
	}
	wrapper := filepath.Join(dir, "cloud-init")
	script := "#!/bin/sh\nexec '" + exact.Python + "' -I '" + boot + "' '" + exact.Src + "' \"$@\"\n"
	if err := os.WriteFile(wrapper, []byte(script), 0o700); err != nil { //nolint:gosec // the wrapper must be executable
		t.Fatal(err)
	}
	d := &Delegate{Path: wrapper, Warn: func(w string) { t.Logf("cloud-init: %s", w) }}
	if err := d.Probe(t.Context()); err != nil {
		t.Fatal(err)
	}
	if d.Exact() {
		t.Fatal("a shell wrapper should not enable exact rendering")
	}
	for _, tc := range integrationCases(t) {
		t.Run(tc.Name, func(t *testing.T) {
			out, err := d.RenderFull(t.Context(), tc.bundle(t))
			if err != nil {
				t.Fatal(err)
			}
			check(t, reference, "summary", summary(t, out), tc.Expected)
		})
	}
}
