// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cloud_init

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/openchami/ochami/pkg/cloudinit"
)

// bootstrap runs cloud-init's CLI with an optional source tree first on
// sys.path. Its first argument is the source tree, or an empty string.
const bootstrap = `import sys
src = sys.argv.pop(1)
if src:
    sys.path.insert(0, src)
from cloudinit.cmd.main import main
sys.exit(main())`

// probeScript prints the cloud-init version that an interpreter imports,
// with an optional source tree first on sys.path.
const probeScript = `import sys
src = sys.argv[1]
if src:
    sys.path.insert(0, src)
from cloudinit import version
print(version.version_string())`

// Delegate runs a real cloud-init to render templates and node data.
type Delegate struct {
	// Path is a cloud-init executable or a wrapper that accepts cloud-init's
	// arguments, such as a script that runs cloud-init in a container. It is
	// run with relative file arguments from the directory that holds them.
	Path string
	// Python is an interpreter that can import cloudinit. When empty it is
	// taken from Path's shebang if that names Python.
	Python string
	// Src is a cloud-init source tree to import, for checkouts and tests.
	Src string

	// Warn receives cloud-init's warnings.
	Warn func(string)

	python  string
	version string
}

// Configured reports whether any way to run cloud-init is set.
func (d *Delegate) Configured() bool {
	return d.Path != "" || d.Python != "" || d.Src != ""
}

// Exact reports whether full rendering runs entirely in cloud-init, which
// needs a Python interpreter that imports cloudinit. Otherwise templates are
// rendered by cloud-init and merged by ochami.
func (d *Delegate) Exact() bool { return d.python != "" }

// Version returns the cloud-init version found by Probe.
func (d *Delegate) Version() string { return d.version }

// Probe checks that cloud-init can run and records its version and whether
// exact full rendering is available.
func (d *Delegate) Probe(ctx context.Context) error {
	if !d.Configured() {
		return errors.New("no cloud-init is configured")
	}
	python := d.Python
	if python == "" && d.Src != "" {
		python = "python3"
	}
	if python == "" && d.Path != "" {
		python = shebangPython(d.Path)
	}
	if python != "" {
		version, err := d.run(ctx, "", nil, python, "-I", "-c", probeScript, d.Src)
		switch {
		case err == nil:
			d.python, d.version = python, strings.TrimSpace(string(version))
		case d.Python != "" || d.Src != "":
			return fmt.Errorf("%s cannot import cloud-init: %w", python, err)
		}
	}
	if d.Src != "" {
		if v := mesonVersion(d.Src); v != "" {
			d.version = v
		}
	}
	if d.python == "" {
		out, err := d.run(ctx, "", nil, d.Path, "--version")
		if err != nil {
			return fmt.Errorf("cannot run %s: %w", d.Path, err)
		}
		fields := strings.Fields(string(out))
		if len(fields) > 0 {
			d.version = fields[len(fields)-1]
		}
	}
	return nil
}

// shebangPython returns the interpreter named by path's shebang when it is
// Python, or an empty string.
func shebangPython(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	line, err := bufio.NewReader(f).ReadString('\n')
	if err != nil && line == "" {
		return ""
	}
	if !strings.HasPrefix(line, "#!") {
		return ""
	}
	fields := strings.Fields(strings.TrimPrefix(line, "#!"))
	if len(fields) == 0 {
		return ""
	}
	interpreter := fields[0]
	if filepath.Base(interpreter) == "env" {
		interpreter = ""
		for _, f := range fields[1:] {
			if !strings.HasPrefix(f, "-") {
				interpreter = f
				break
			}
		}
		if interpreter == "" {
			return ""
		}
		resolved, err := exec.LookPath(interpreter)
		if err != nil {
			return ""
		}
		interpreter = resolved
	}
	if !strings.Contains(filepath.Base(interpreter), "python") {
		return ""
	}
	return interpreter
}

var mesonVersionPattern = regexp.MustCompile(`(?m)^\s*version:\s*'([^']+)'`)

// mesonVersion returns the version a cloud-init source tree declares.
func mesonVersion(src string) string {
	data, err := os.ReadFile(filepath.Join(src, "meson.build"))
	if err != nil {
		return ""
	}
	if m := mesonVersionPattern.FindSubmatch(data); m != nil {
		return string(m[1])
	}
	return ""
}

// warningLine matches cloud-init's log lines, such as
// "2026-11-10 21:20:33,041 - render.py[WARNING]: message".
var warningLine = regexp.MustCompile(`\[(WARNING|ERROR)\]: (.*)$`)

// run runs a command from dir and returns its standard output. Warnings
// cloud-init logs go to Warn. A failure includes cloud-init's errors.
func (d *Delegate) run(ctx context.Context, dir string, stdin []byte, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	var errorsSeen []string
	for _, line := range strings.Split(stderr.String(), "\n") {
		m := warningLine.FindStringSubmatch(line)
		switch {
		case m != nil && m[1] == "ERROR":
			errorsSeen = append(errorsSeen, m[2])
		case m != nil && d.Warn != nil:
			d.Warn(m[2])
		case strings.HasPrefix(line, "WARNING: ") && d.Warn != nil:
			d.Warn(strings.TrimPrefix(line, "WARNING: "))
		case strings.HasPrefix(line, "error: "):
			errorsSeen = append(errorsSeen, strings.TrimPrefix(line, "error: "))
		}
	}
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		detail := strings.Join(errorsSeen, "; ")
		if detail == "" {
			detail = lastLines(stderr.String(), 5)
		}
		if detail != "" {
			return nil, fmt.Errorf("%w: %s", err, detail)
		}
		return nil, err
	}
	return stdout.Bytes(), nil
}

func lastLines(s string, n int) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.TrimSpace(strings.Join(lines, " | "))
}

// cliCommand returns how to run cloud-init's CLI.
func (d *Delegate) cliCommand() (string, []string) {
	if d.Path != "" && d.Src == "" {
		return d.Path, nil
	}
	return d.python, []string{"-I", "-c", bootstrap, d.Src}
}

// RenderTemplate renders a template with cloud-init devel render. A template
// without the jinja header is returned unchanged, as a node uses it.
func (d *Delegate) RenderTemplate(ctx context.Context, template, instanceData []byte) (string, error) {
	if !cloudinit.HasJinjaHeader(template) {
		return string(template), nil
	}
	dir, err := os.MkdirTemp("", "ochami-render-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(dir)
	if err := os.WriteFile(filepath.Join(dir, "template"), template, 0o600); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(dir, "instance-data.json"), instanceData, 0o600); err != nil {
		return "", err
	}
	name, args := d.cliCommand()
	out, err := d.run(ctx, dir, nil, name, append(args, "devel", "render", "template", "-i", "instance-data.json")...)
	if err != nil {
		return "", fmt.Errorf("cloud-init devel render failed: %w", err)
	}
	return string(out), nil
}

// RenderPart implements cloudinit.PartRenderer with cloud-init devel render.
func (d *Delegate) RenderPart(ctx context.Context, _ cloudinit.Part, payload string, instanceData []byte) (string, error) {
	return d.RenderTemplate(ctx, []byte(payload), instanceData)
}

// FullOutput is the result of a full render by cloud-init.
type FullOutput struct {
	// Text is the merged cloud-config to print.
	Text string
	// Comparable describes the merged configuration as
	// cloudinit.FullResult.Comparable does.
	Comparable any
	// Parts lists each named part's status.
	Parts []cloudinit.Part
	// UserFailed and VendorFailed explain aborted processing.
	UserFailed, VendorFailed string
	// VendorDisabled is set when user-data disables vendor-data.
	VendorDisabled bool
	// MergedByOchami is set when cloud-init rendered the templates but ochami
	// merged the parts, because no Python interpreter that imports cloud-init
	// was found.
	MergedByOchami bool
}

// RenderFull renders a full-mode bundle with cloud-init. With a Python
// interpreter that imports cloud-init, the embedded helper processes it
// entirely in cloud-init. Otherwise each template is rendered with devel
// render and ochami merges the parts.
func (d *Delegate) RenderFull(ctx context.Context, b cloudinit.Bundle) (FullOutput, error) {
	if !d.Exact() {
		result, err := cloudinit.RenderFull(ctx, b.RenderFullRequest(d))
		if err != nil {
			return FullOutput{}, err
		}
		return FullOutput{
			Text: result.Text(), Comparable: result.Comparable(), Parts: result.Parts,
			UserFailed: result.UserFailed, VendorFailed: result.VendorFailed,
			VendorDisabled: result.VendorDisabled, MergedByOchami: true,
		}, nil
	}
	parent, err := os.MkdirTemp("", "ochami-render-")
	if err != nil {
		return FullOutput{}, err
	}
	defer os.RemoveAll(parent)
	dir := filepath.Join(parent, "bundle")
	if err := cloudinit.WriteBundle(dir, b); err != nil {
		return FullOutput{}, err
	}
	args := []string{"-I", filepath.Join(dir, "ochami_render.py"), "--bundle", dir}
	if d.Src != "" {
		args = append(args, "--cloud-init-src", d.Src)
	}
	out, err := d.run(ctx, dir, nil, d.python, args...)
	if err != nil {
		return FullOutput{}, fmt.Errorf("cloud-init render helper failed: %w", err)
	}
	var result cloudinit.HelperResult
	if err := json.Unmarshal(out, &result); err != nil {
		return FullOutput{}, fmt.Errorf("cloud-init render helper printed invalid output: %w", err)
	}
	if d.Warn != nil {
		for _, w := range result.Warnings {
			d.Warn(w)
		}
	}
	var comparable any
	if err := json.Unmarshal(result.Merged, &comparable); err != nil {
		return FullOutput{}, fmt.Errorf("cloud-init render helper printed invalid output: %w", err)
	}
	output := FullOutput{Text: result.Text, Comparable: comparable, VendorDisabled: result.Vendor.Disabled}
	for _, section := range []struct {
		data string
		s    cloudinit.HelperSection
	}{{"user-data", result.User}, {"vendor-data", result.Vendor}} {
		for _, p := range section.s.Parts {
			output.Parts = append(output.Parts, cloudinit.Part{Data: section.data, Name: p.Name, Source: section.data, Type: p.Type, Status: p.Status})
		}
	}
	if result.User.Failed != nil {
		output.UserFailed = *result.User.Failed
	}
	if result.Vendor.Failed != nil {
		output.VendorFailed = *result.Vendor.Failed
	}
	return output, nil
}
