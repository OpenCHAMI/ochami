// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package configfile

import (
	"testing"
)

// TestReadConfig_MissingClusterName verifies a cluster entry without a name is
// rejected.
func TestReadConfig_MissingClusterName(t *testing.T) {
	path := writeTemp(t, `clusters:
- cluster:
    uri: https://example.com
`)
	if _, err := ReadConfigWithDefaults(path); err == nil {
		t.Error("ReadConfigWithDefaults with unnamed cluster = nil, want error")
	}
}

// TestReadConfig_NonMapClusterBlock verifies a cluster entry whose "cluster"
// block is not a map is rejected.
func TestReadConfig_NonMapClusterBlock(t *testing.T) {
	path := writeTemp(t, `clusters:
- name: demo
  cluster: "not-a-map"
`)
	if _, err := ReadConfigWithDefaults(path); err == nil {
		t.Error("ReadConfigWithDefaults with non-map cluster block = nil, want error")
	}
}
