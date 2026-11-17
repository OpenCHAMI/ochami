// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cloudinit

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
)

// BundleFormat and BundleVersion identify a render bundle's manifest.
const (
	BundleFormat  = "ochami-cloud-init-render"
	BundleVersion = 1
)

// BundleMode is what a bundle renders.
type BundleMode string

const (
	// BundleTemplate renders one template.
	BundleTemplate BundleMode = "template"
	// BundleFull renders a node's user-data and vendor-data.
	BundleFull BundleMode = "full"
)

// Include statuses recorded in a bundle.
const (
	IncludeResolved   = "resolved"
	IncludeUnresolved = "unresolved"
	IncludeFailed     = "failed"
)

//go:embed helper/ochami_render.py
var helperScript []byte

// HelperScript returns the Python helper that renders a bundle with
// cloud-init's own code.
func HelperScript() []byte { return append([]byte(nil), helperScript...) }

// BundleInclude records an #include URL and what fetching it produced.
type BundleInclude struct {
	URL string `json:"url"`
	// File is the content's path in the bundle, for resolved includes.
	File    string `json:"file,omitempty"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	// Code is the HTTP status of a failed fetch, when known.
	Code    int    `json:"code,omitempty"`
	Content []byte `json:"-"`
}

// Bundle holds the inputs gathered for rendering, so they can be rendered
// later or elsewhere, for example with cloud-init on a node. It may contain
// sensitive metadata and extra variables, but never credentials.
type Bundle struct {
	Mode      BundleMode
	CreatedBy string
	Node      string
	Group     string
	// Template is the template rendered in template mode.
	Template []byte
	// InstanceData is instance-data.json, as built by InstanceDataJSON or
	// taken from a node.
	InstanceData []byte
	// UserData and VendorData are rendered in full mode.
	UserData   []byte
	VendorData []byte
	Includes   []BundleInclude
}

type bundleManifest struct {
	Format    string          `json:"format"`
	Version   int             `json:"version"`
	Mode      BundleMode      `json:"mode"`
	CreatedBy string          `json:"created_by,omitempty"`
	Node      string          `json:"node,omitempty"`
	Group     string          `json:"group,omitempty"`
	Files     bundleFiles     `json:"files"`
	Includes  []BundleInclude `json:"includes,omitempty"`
}

type bundleFiles struct {
	InstanceData string `json:"instance_data"`
	Template     string `json:"template,omitempty"`
	UserData     string `json:"user_data,omitempty"`
	VendorData   string `json:"vendor_data,omitempty"`
}

const bundleReadme = `# ochami cloud-init render bundle

This directory holds the inputs ochami gathered to render cloud-init data.
manifest.json describes them. It may contain sensitive node metadata and
extra variables, so share it with care. It never contains credentials.

Render it with cloud-init's own code, anywhere cloud-init and python3 are
installed, such as a booted node or its image:

    ./run.sh

Set PYTHON to choose the interpreter. Pass --cloud-init-src DIR to use a
cloud-init source tree. ochami renders it again with:

    ochami cloud-init render --bundle DIR [--engine internal|cloud-init] [--compare]

A template bundle can also be rendered with cloud-init's CLI:

    cloud-init devel render template -i instance-data.json
`

const bundleRunScript = `#!/bin/sh
# Render this bundle with cloud-init's own code. See README.
set -eu
here=$(cd "$(dirname "$0")" && pwd)
exec "${PYTHON:-python3}" -I "$here/ochami_render.py" --bundle "$here" --format text "$@"
`

var unsafeFileChars = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

// WriteBundle writes b to dir, which must not exist or be empty. The
// directory is created with mode 0700 and files with mode 0600, apart from
// run.sh.
func WriteBundle(dir string, b Bundle) error {
	if err := prepareBundleDir(dir); err != nil {
		return err
	}
	m := bundleManifest{
		Format: BundleFormat, Version: BundleVersion, Mode: b.Mode,
		CreatedBy: b.CreatedBy, Node: b.Node, Group: b.Group,
		Files: bundleFiles{InstanceData: "instance-data.json"},
	}
	files := map[string][]byte{"instance-data.json": b.InstanceData}
	switch b.Mode {
	case BundleTemplate:
		m.Files.Template = "template"
		files["template"] = b.Template
	case BundleFull:
		m.Files.UserData, m.Files.VendorData = "user-data", "vendor-data"
		files["user-data"], files["vendor-data"] = b.UserData, b.VendorData
		for i, inc := range b.Includes {
			if inc.Status == IncludeResolved {
				base := unsafeFileChars.ReplaceAllString(path.Base(inc.URL), "_")
				inc.File = fmt.Sprintf("parts/%03d-%s", i+1, base)
				files[inc.File] = inc.Content
			}
			m.Includes = append(m.Includes, inc)
		}
	default:
		return fmt.Errorf("unknown bundle mode %q", b.Mode)
	}
	manifest, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	files["manifest.json"] = append(manifest, '\n')
	files["ochami_render.py"] = helperScript
	files["README"] = []byte(bundleReadme)
	for name, data := range files {
		target := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return err
		}
		if err := os.WriteFile(target, data, 0o600); err != nil {
			return err
		}
	}
	return os.WriteFile(filepath.Join(dir, "run.sh"), []byte(bundleRunScript), 0o700) //nolint:gosec // the script must be executable
}

func prepareBundleDir(dir string) error {
	entries, err := os.ReadDir(dir)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return os.MkdirAll(dir, 0o700)
	case err != nil:
		return err
	case len(entries) > 0:
		return fmt.Errorf("bundle directory %s is not empty", dir)
	}
	return os.Chmod(dir, 0o700)
}

// LoadBundle reads a bundle written by WriteBundle.
func LoadBundle(dir string) (Bundle, error) {
	data, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return Bundle{}, err
	}
	var m bundleManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return Bundle{}, fmt.Errorf("invalid bundle manifest: %w", err)
	}
	if m.Format != BundleFormat || m.Version != BundleVersion {
		return Bundle{}, fmt.Errorf("unsupported bundle %q version %d, want %q version %d", m.Format, m.Version, BundleFormat, BundleVersion)
	}
	read := func(name string, required bool) ([]byte, error) {
		if name == "" {
			if required {
				return nil, fmt.Errorf("bundle manifest names no file where one is required")
			}
			return nil, nil
		}
		if !filepath.IsLocal(filepath.FromSlash(name)) {
			return nil, fmt.Errorf("bundle file %q is outside the bundle", name)
		}
		return os.ReadFile(filepath.Join(dir, filepath.FromSlash(name)))
	}
	b := Bundle{Mode: m.Mode, CreatedBy: m.CreatedBy, Node: m.Node, Group: m.Group}
	if b.InstanceData, err = read(m.Files.InstanceData, true); err != nil {
		return Bundle{}, err
	}
	switch m.Mode {
	case BundleTemplate:
		if b.Template, err = read(m.Files.Template, true); err != nil {
			return Bundle{}, err
		}
	case BundleFull:
		if b.UserData, err = read(m.Files.UserData, false); err != nil {
			return Bundle{}, err
		}
		if b.VendorData, err = read(m.Files.VendorData, false); err != nil {
			return Bundle{}, err
		}
		for _, inc := range m.Includes {
			if inc.Status == IncludeResolved {
				if inc.Content, err = read(inc.File, true); err != nil {
					return Bundle{}, err
				}
			}
			b.Includes = append(b.Includes, inc)
		}
	default:
		return Bundle{}, fmt.Errorf("unknown bundle mode %q", m.Mode)
	}
	return b, nil
}

// Fetcher returns a Fetcher that serves the bundle's recorded includes.
func (b Bundle) Fetcher() Fetcher {
	includes := map[string]BundleInclude{}
	for _, inc := range b.Includes {
		includes[inc.URL] = inc
	}
	return func(_ context.Context, url string) ([]byte, error) {
		inc, ok := includes[url]
		switch {
		case !ok:
			return nil, fmt.Errorf("%s is not in the bundle: %w", url, ErrIncludeUnresolved)
		case inc.Status == IncludeResolved:
			return inc.Content, nil
		case inc.Status == IncludeFailed:
			if inc.Message != "" {
				return nil, errors.New(inc.Message)
			}
			return nil, fmt.Errorf("HTTP %d", inc.Code)
		default:
			if inc.Message != "" {
				return nil, fmt.Errorf("%s (%s): %w", url, inc.Message, ErrIncludeUnresolved)
			}
			return nil, fmt.Errorf("%s: %w", url, ErrIncludeUnresolved)
		}
	}
}

// RenderFullRequest returns the FullRequest for a full-mode bundle.
func (b Bundle) RenderFullRequest(renderer PartRenderer) FullRequest {
	return FullRequest{
		UserData: b.UserData, VendorData: b.VendorData, InstanceData: b.InstanceData,
		Fetch: b.Fetcher(), Renderer: renderer,
	}
}

// HelperResult is the JSON the Python helper prints.
type HelperResult struct {
	CloudInitVersion string          `json:"cloud_init_version"`
	Output           *string         `json:"output,omitempty"`
	User             HelperSection   `json:"user"`
	Vendor           HelperSection   `json:"vendor"`
	Merged           json.RawMessage `json:"merged,omitempty"`
	Text             string          `json:"text"`
	Warnings         []string        `json:"warnings"`
}

// HelperSection is the helper's result for user-data or vendor-data.
type HelperSection struct {
	Parts []struct {
		Name   string     `json:"name"`
		Type   string     `json:"type"`
		Status PartStatus `json:"status"`
	} `json:"parts"`
	Failed   *string `json:"failed"`
	Disabled bool    `json:"disabled"`
}
