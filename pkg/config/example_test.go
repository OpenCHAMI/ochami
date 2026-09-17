// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package config_test

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/openchami/ochami/pkg/config"
)

// ExampleLoad demonstrates loading configuration from an explicit set of
// sources and resolving a service base URI for a cluster.
func ExampleLoad() {
	dir, _ := os.MkdirTemp("", "ochami-config-example-")
	defer os.RemoveAll(dir)

	path := filepath.Join(dir, "config.yaml")
	_ = os.WriteFile(path, []byte(`default-cluster: foobar
clusters:
  - name: foobar
    cluster:
      uri: https://foobar.openchami.cluster
`), 0o644)

	cfg, err := config.Load([]config.Source{{Name: "example", Path: path}})
	if err != nil {
		fmt.Println("load error:", err)
		return
	}

	cluster, err := cfg.GetCluster(cfg.DefaultCluster)
	if err != nil {
		fmt.Println("cluster error:", err)
		return
	}

	uri, err := cluster.Cluster.GetServiceBaseURI(config.ServiceSMD)
	if err != nil {
		fmt.Println("uri error:", err)
		return
	}

	fmt.Println(uri)
	// Output: https://foobar.openchami.cluster/hsm/v2
}

// ExampleConfigClusterConfig_GetServiceBaseURI shows resolving a service URI
// directly from a cluster configuration, including a relative service override.
func ExampleConfigClusterConfig_GetServiceBaseURI() {
	ccc := config.ConfigClusterConfig{
		URI: "https://cluster.example.com",
		BSS: config.ConfigClusterBSS{URI: "/custom-bss"},
	}

	uri, err := ccc.GetServiceBaseURI(config.ServiceBSS)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println(uri)
	// Output: https://cluster.example.com/custom-bss
}
