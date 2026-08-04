// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package metadata_service

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestClusterDefaultsSpecUnmarshalsFlatYAML(t *testing.T) {
	var got ClusterDefaultsSpec
	data := []byte("name: defaults\nbase_url: https://example.com/cloud-init\ncluster_name: demo\n")
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatalf("failed to unmarshal cluster defaults YAML: %v", err)
	}
	if got.Name != "defaults" || got.BaseURL != "https://example.com/cloud-init" || got.ClusterName != "demo" {
		t.Errorf("cluster defaults = %+v, want flat YAML name, base URL, and cluster name values", got)
	}
}

func TestGroupSpecUnmarshalsFlatYAML(t *testing.T) {
	var got GroupSpec
	if err := yaml.Unmarshal([]byte("name: computes\ntemplate: compute.yaml\nosVersion: rocky9\n"), &got); err != nil {
		t.Fatalf("failed to unmarshal group YAML: %v", err)
	}
	if got.Name != "computes" || got.Template != "compute.yaml" || got.OSVersion != "rocky9" {
		t.Errorf("group = %+v, want flat YAML name, template, and OS version values", got)
	}
}

func TestInstanceInfoSpecUnmarshalsFlatYAML(t *testing.T) {
	var got InstanceInfoSpec
	data := []byte("name: node-1\ninstance_id: x1000c0s0b0n0\nlocal_hostname: node-1\n")
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatalf("failed to unmarshal instance info YAML: %v", err)
	}
	if got.Name != "node-1" || got.InstanceID != "x1000c0s0b0n0" || got.LocalHostname != "node-1" {
		t.Errorf("instance info = %+v, want flat YAML name, instance ID, and hostname values", got)
	}
}

func TestWireGuardPeerSpecUnmarshalsFlatYAML(t *testing.T) {
	var got WireGuardPeerSpec
	data := []byte("name: peer-1\npublic_key: test-key\nallowed_ip: 10.0.0.1/32\n")
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatalf("failed to unmarshal WireGuard peer YAML: %v", err)
	}
	if got.Name != "peer-1" || got.PublicKey != "test-key" || got.AllowedIP != "10.0.0.1/32" {
		t.Errorf("WireGuard peer = %+v, want flat YAML name, public key, and allowed IP values", got)
	}
}
