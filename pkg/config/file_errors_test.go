// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package config_test

import (
	"bytes"
	"errors"
	"os"
	"reflect"
	"testing"

	"github.com/openchami/ochami/pkg/config"
)

// duplicateClusterFixture defines cluster "demo" twice, as a manual edit
// might, alongside a uniquely named cluster.
const duplicateClusterFixture = `default-cluster: demo
clusters:
  - name: demo
    cluster:
      uri: https://first.example
      bss:
        uri: /first
  - name: other
    cluster:
      uri: https://other.example
  - name: demo
    cluster:
      uri: https://second.example
      bss:
        uri: /second
`

// TestFile_DuplicateClusterErrors verifies that every File method that would
// have to pick one definition of a cluster defined more than once returns
// ErrDuplicateCluster naming the file and the positions of the cluster's
// entries, and leaves the file unchanged, while renaming another cluster to
// the duplicated name still returns ErrClusterExists.
func TestFile_DuplicateClusterErrors(t *testing.T) {
	tests := []struct {
		name string
		op   func(f *config.File) error
	}{
		{"Cluster", func(f *config.File) error {
			_, err := f.Cluster("demo")
			return err
		}},
		{"SetClusterKey", func(f *config.File) error { return f.SetClusterKey("demo", "cluster.uri", "https://new.example") }},
		{"UnsetClusterKey", func(f *config.File) error { return f.UnsetClusterKey("demo", "cluster.bss.uri") }},
		{"RenameCluster", func(f *config.File) error { return f.RenameCluster("demo", "prod") }},
		{"SetDefaultCluster", func(f *config.File) error { return f.SetDefaultCluster("demo") }},
		{"DeleteCluster", func(f *config.File) error { return f.DeleteCluster("demo") }},
		{"Update", func(f *config.File) error {
			return f.Update(func(f *config.File) error {
				if err := f.SetKey("log.level", "debug"); err != nil {
					return err
				}
				return f.SetClusterKey("demo", "cluster.uri", "https://new.example")
			})
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeFileFixture(t, duplicateClusterFixture)
			f, err := config.OpenFile(path)
			if err != nil {
				t.Fatalf("OpenFile(): %v", err)
			}

			err = tt.op(f)
			var dup config.ErrDuplicateCluster
			if !errors.As(err, &dup) {
				t.Fatalf("%s error = %v, want ErrDuplicateCluster", tt.name, err)
			}
			if dup.Name != "demo" || dup.Path != path || !reflect.DeepEqual(dup.Indices, []int{0, 2}) {
				t.Errorf("ErrDuplicateCluster = %+v, want Name demo, Path %s, Indices [0 2]", dup, path)
			}

			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read back failed: %v", err)
			}
			if !bytes.Equal(got, []byte(duplicateClusterFixture)) {
				t.Errorf("config file changed to:\n%s", got)
			}
		})
	}

	t.Run("rename to a duplicated name returns ErrClusterExists", func(t *testing.T) {
		path := writeFileFixture(t, duplicateClusterFixture)
		f, err := config.OpenFile(path)
		if err != nil {
			t.Fatalf("OpenFile(): %v", err)
		}
		if err := f.RenameCluster("other", "demo"); !errors.As(err, &config.ErrClusterExists{}) {
			t.Errorf("RenameCluster(other, demo) error = %v, want ErrClusterExists", err)
		}
	})
}
