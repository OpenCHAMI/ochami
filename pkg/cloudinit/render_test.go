// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cloudinit_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/openchami/ochami/pkg/cloudinit"
	"github.com/openchami/ochami/pkg/format"
)

type referenceFixture struct {
	Name      string `json:"name"`
	Template  string `json:"template"`
	Metadata  string `json:"metadata"`
	ExtraVars string `json:"extra_vars"`
	Expected  string `json:"expected"`
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// TestRender_ReferenceCorpus verifies exact Python output and the mount and
// command lists for the supported cloud-init fixtures with JSON and YAML inputs.
func TestRender_ReferenceCorpus(t *testing.T) {
	t.Parallel()
	var manifest struct {
		Fixtures []referenceFixture `json:"fixtures"`
	}
	if err := json.Unmarshal(readFixture(t, "reference.json"), &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Fixtures) != 5 {
		t.Fatalf("fixture count = %d, want 5", len(manifest.Fixtures))
	}
	for _, fixture := range manifest.Fixtures {
		for _, inputFormat := range []format.DataFormat{format.DataFormatJson, format.DataFormatYaml} {
			t.Run(fixture.Name+"/"+string(inputFormat), func(t *testing.T) {
				t.Parallel()
				metadata := readFixture(t, fixture.Metadata)
				if inputFormat == format.DataFormatYaml {
					var node yaml.Node
					if err := yaml.Unmarshal(metadata, &node); err != nil {
						t.Fatal(err)
					}
					var err error
					metadata, err = yaml.Marshal(&node)
					if err != nil {
						t.Fatal(err)
					}
				}
				out, err := cloudinit.Render(t.Context(), cloudinit.RenderRequest{
					Template:  readFixture(t, fixture.Template),
					Metadata:  cloudinit.Document{Data: metadata, Format: inputFormat},
					ExtraVars: cloudinit.Document{Data: readFixture(t, fixture.ExtraVars)},
				})
				if err != nil {
					t.Fatal(err)
				}
				if want := string(readFixture(t, fixture.Expected)); out != want {
					t.Fatalf("rendered output = %q, want %q", out, want)
				}
				var config struct {
					Files    []any `yaml:"write_files"`
					Commands []any `yaml:"runcmd"`
				}
				if err := yaml.Unmarshal([]byte(out), &config); err != nil {
					t.Fatal(err)
				}
				if len(config.Files) != 3 || len(config.Commands) != 7 {
					t.Fatalf("files, commands = %d, %d, want 3, 7", len(config.Files), len(config.Commands))
				}
			})
		}
	}
}
