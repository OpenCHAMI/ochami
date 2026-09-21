// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package configfile

import (
	"testing"
)

// TestReadConfigWithDefaults_MissingClusterName verifies a cluster entry
// without a name is rejected.
func TestReadConfigWithDefaults_MissingClusterName(t *testing.T) {
	path := writeTemp(t, `clusters:
- cluster:
    uri: https://example.com
`)
	if _, err := ReadConfigWithDefaults(path); err == nil {
		t.Error("ReadConfigWithDefaults with unnamed cluster = nil, want error")
	}
}

// TestReadConfigWithDefaults_NonMapClusterBlock verifies a cluster entry whose
// "cluster" block is not a map is rejected.
func TestReadConfigWithDefaults_NonMapClusterBlock(t *testing.T) {
	path := writeTemp(t, `clusters:
- name: demo
  cluster: "not-a-map"
`)
	if _, err := ReadConfigWithDefaults(path); err == nil {
		t.Error("ReadConfigWithDefaults with non-map cluster block = nil, want error")
	}
}
