// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package configfile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/openchami/ochami/pkg/config"
)

func writeBoundaryConfig(t *testing.T, data string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestReadConfigWithDefaults_RejectsInvalidSources(t *testing.T) {
	tests := []struct {
		name string
		data string
		want string
	}{
		{name: "malformed YAML", data: "clusters: [\n", want: "yaml"},
		{name: "null global", data: "timeout:\n", want: "non-null"},
		{name: "global type conflict", data: "log: scalar\n", want: "incorrect types"},
		{name: "missing cluster name", data: "clusters:\n  - cluster: {}\n", want: "missing a name"},
		{name: "non-map cluster", data: "clusters:\n  - name: demo\n    cluster: value\n", want: "not a map"},
		{name: "invalid cluster boolean", data: "clusters:\n  - name: demo\n    cluster:\n      enable-auth: maybe\n", want: "boolean"},
		{name: "invalid timeout", data: "timeout: -1s\n", want: "positive duration"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ReadConfigWithDefaults(writeBoundaryConfig(t, tt.data))
			if err == nil || !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.want)) {
				t.Fatalf("ReadConfigWithDefaults() error = %v, want text %q", err, tt.want)
			}
		})
	}
}

func TestReadConfigWithDefaults_AcceptsNullCluster(t *testing.T) {
	ko, err := ReadConfigWithDefaults(writeBoundaryConfig(t, "clusters:\n  - name: demo\n    cluster:\n"))
	if err != nil {
		t.Fatalf("ReadConfigWithDefaults() error = %v", err)
	}
	var clusters []config.ConfigCluster
	if err := ko.Unmarshal("clusters", &clusters); err != nil {
		t.Fatal(err)
	}
	if len(clusters) != 1 || !clusters[0].Cluster.EnableAuth {
		t.Fatalf("clusters = %#v, want defaulted demo cluster", clusters)
	}
}
