// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cloud_init

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/openchami/ochami/pkg/cloudinit"
)

// fakeCloudInit is a cloud-init stand-in. devel render checks that it runs
// from the directory holding its relative arguments, logs a warning, and
// prints the template with its header removed and NAME replaced by the
// instance data. A template containing FAIL makes it fail as cloud-init does.
const fakeCloudInit = `#!/bin/sh
if [ "$1" = "--version" ]; then
	echo "/usr/bin/cloud-init 24.4"
	exit 0
fi
[ "$1 $2 $3 $4 $5" = "devel render template -i instance-data.json" ] || { echo "unexpected arguments: $*" >&2; exit 2; }
[ -f template ] && [ -f instance-data.json ] || { echo "not run from the bundle directory" >&2; exit 2; }
if grep -q FAIL template; then
	echo "2026-11-10 21:20:33,725 - render.py[ERROR]: Failed to render templated user-data file 'template'. syntax error" >&2
	exit 1
fi
echo "2026-11-10 21:20:33,041 - jinja_template.py[WARNING]: Could not render jinja template variables in file 'template': 'nope'" >&2
name=$(sed -n 's/.*"local_hostname": "\([^"]*\)".*/\1/p' instance-data.json)
tail -n +2 template | sed "s/NAME/$name/"
`

func writeFake(t *testing.T, script string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "cloud-init")
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil { //nolint:gosec // the fake must be executable
		t.Fatal(err)
	}
	return path
}

// TestDelegate_TemplateWithExecutable verifies template rendering through a
// cloud-init executable: relative arguments, forwarded warnings, headerless
// templates returned unchanged, and cloud-init's errors.
func TestDelegate_TemplateWithExecutable(t *testing.T) {
	t.Parallel()
	var warnings []string
	d := &Delegate{Path: writeFake(t, fakeCloudInit), Warn: func(w string) { warnings = append(warnings, w) }}
	if err := d.Probe(t.Context()); err != nil {
		t.Fatal(err)
	}
	if d.Exact() || d.Version() != "24.4" {
		t.Fatalf("exact = %v, version = %q, want false, 24.4", d.Exact(), d.Version())
	}
	instanceData := []byte(`{"v1": {"local_hostname": "node1"}}`)

	out, err := d.RenderTemplate(t.Context(), []byte("## template: jinja\nhost: NAME\n"), instanceData)
	if err != nil {
		t.Fatal(err)
	}
	if out != "host: node1\n" {
		t.Fatalf("output = %q", out)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "'nope'") {
		t.Fatalf("warnings = %q", warnings)
	}

	out, err = d.RenderTemplate(t.Context(), []byte("#cloud-config\nhost: NAME\n"), instanceData)
	if err != nil || out != "#cloud-config\nhost: NAME\n" {
		t.Fatalf("headerless output = %q, error %v, want the template unchanged", out, err)
	}

	_, err = d.RenderTemplate(t.Context(), []byte("## template: jinja\nFAIL\n"), instanceData)
	if err == nil || !strings.Contains(err.Error(), "syntax error") {
		t.Fatalf("error = %v, want cloud-init's error", err)
	}
}

// TestDelegate_FullWithExecutable verifies that without a Python interpreter,
// cloud-init renders each template and ochami merges the parts.
func TestDelegate_FullWithExecutable(t *testing.T) {
	t.Parallel()
	d := &Delegate{Path: writeFake(t, fakeCloudInit)}
	if err := d.Probe(t.Context()); err != nil {
		t.Fatal(err)
	}
	b := cloudinit.Bundle{
		Mode:         cloudinit.BundleFull,
		UserData:     []byte("#cloud-config\n"),
		VendorData:   []byte("#include\nhttp://x/a.yaml\nhttp://x/b.yaml\n"),
		InstanceData: []byte(`{"v1": {"local_hostname": "node1"}}`),
		Includes: []cloudinit.BundleInclude{
			{URL: "http://x/a.yaml", Status: cloudinit.IncludeResolved, Content: []byte("## template: jinja\n#cloud-config\nhostname: NAME\n")},
			{URL: "http://x/b.yaml", Status: cloudinit.IncludeResolved, Content: []byte("## template: jinja\nFAIL\n")},
		},
	}
	out, err := d.RenderFull(t.Context(), b)
	if err != nil {
		t.Fatal(err)
	}
	if !out.MergedByOchami || !strings.Contains(out.Text, "hostname: node1\n") {
		t.Fatalf("output = %+v", out)
	}
	statuses := map[string]cloudinit.PartStatus{}
	for _, p := range out.Parts {
		statuses[p.Source] = p.Status
	}
	if statuses["http://x/a.yaml"] != cloudinit.PartMerged || statuses["http://x/b.yaml"] != cloudinit.PartDropped {
		t.Fatalf("statuses = %v", statuses)
	}
}

// TestDelegate_ProbeErrors verifies that Probe fails when cloud-init is not
// configured or cannot run.
func TestDelegate_ProbeErrors(t *testing.T) {
	t.Parallel()
	if err := (&Delegate{}).Probe(t.Context()); err == nil {
		t.Error("Probe() without cloud-init succeeded")
	}
	missing := filepath.Join(t.TempDir(), "missing")
	if err := (&Delegate{Path: missing}).Probe(t.Context()); err == nil {
		t.Error("Probe() with a missing executable succeeded")
	}
	if err := (&Delegate{Python: missing}).Probe(t.Context()); err == nil {
		t.Error("Probe() with a missing interpreter succeeded")
	}
}

// TestShebangPython verifies that a Python interpreter is found only in a
// Python shebang.
func TestShebangPython(t *testing.T) {
	t.Parallel()
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 is not installed")
	}
	tests := map[string]string{
		"#!" + python + "\n":                python,
		"#!/usr/bin/env python3\n":          python,
		"#!/usr/bin/env -S python3 -I\n":    python,
		"#!/bin/sh\nexec cloud-init \"$@\"": "",
		"no shebang\n":                      "",
	}
	for content, want := range tests {
		path := writeFake(t, content)
		if got := shebangPython(path); got != want {
			t.Errorf("shebangPython(%q) = %q, want %q", content, got, want)
		}
	}
}

// TestUnifiedDiff verifies unified diff hunks and an empty diff for equal
// documents.
func TestUnifiedDiff(t *testing.T) {
	t.Parallel()
	if got := UnifiedDiff("a", "b", "x\ny\n", "x\ny\n"); got != "" {
		t.Fatalf("equal diff = %q", got)
	}
	a := "1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n11\n12\n"
	b := "1\n2\n3\n4\nfive\n6\n7\n8\n9\n10\n11\n12\n13\n"
	want := strings.Join([]string{
		"--- internal",
		"+++ cloud-init",
		"@@ -2,7 +2,7 @@",
		" 2",
		" 3",
		" 4",
		"-5",
		"+five",
		" 6",
		" 7",
		" 8",
		"@@ -10,3 +10,4 @@",
		" 10",
		" 11",
		" 12",
		"+13",
		"",
	}, "\n")
	if got := UnifiedDiff("internal", "cloud-init", a, b); got != want {
		t.Fatalf("diff =\n%s\nwant\n%s", got, want)
	}
}
