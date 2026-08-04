// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package boot_service

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestBootConfigSpecUnmarshalsFlatYAML(t *testing.T) {
	data := []byte(`
- name: compute-debug-rocky9
  kernel: http://172.16.0.254:7070/boot-images/compute/debug/vmlinuz
  initrd: http://172.16.0.254:7070/boot-images/compute/debug/initramfs.img
  params: ip=dhcp console=ttyS0,115200
  macs:
    - 52:54:00:be:ef:01
    - 52:54:00:be:ef:02
`)

	var got []BootConfigSpec
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatalf("failed to unmarshal boot configuration YAML: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d boot configurations, want 1", len(got))
	}
	if got[0].Name != "compute-debug-rocky9" {
		t.Errorf("name = %q, want compute-debug-rocky9", got[0].Name)
	}
	if got[0].Kernel != "http://172.16.0.254:7070/boot-images/compute/debug/vmlinuz" {
		t.Errorf("kernel = %q, want flat YAML kernel value", got[0].Kernel)
	}
	if got[0].Initrd != "http://172.16.0.254:7070/boot-images/compute/debug/initramfs.img" {
		t.Errorf("initrd = %q, want flat YAML initrd value", got[0].Initrd)
	}
	if got[0].Params != "ip=dhcp console=ttyS0,115200" {
		t.Errorf("params = %q, want flat YAML params value", got[0].Params)
	}
	if len(got[0].MACs) != 2 || got[0].MACs[0] != "52:54:00:be:ef:01" {
		t.Errorf("macs = %#v, want flat YAML MAC values", got[0].MACs)
	}
}

func TestNodeSpecUnmarshalsFlatYAML(t *testing.T) {
	var got NodeSpec
	if err := yaml.Unmarshal([]byte("name: node-1\nxname: x1000c0s0b0n0\nnid: 42\n"), &got); err != nil {
		t.Fatalf("failed to unmarshal node YAML: %v", err)
	}
	if got.Name != "node-1" || got.XName != "x1000c0s0b0n0" || got.NID != 42 {
		t.Errorf("node = %+v, want flat YAML name, xname, and nid values", got)
	}
}

func TestBMCSpecUnmarshalsFlatYAML(t *testing.T) {
	var got BMCSpec
	if err := yaml.Unmarshal([]byte("name: bmc-1\nxname: x1000c0s0b0\ndescription: test BMC\n"), &got); err != nil {
		t.Fatalf("failed to unmarshal BMC YAML: %v", err)
	}
	if got.Name != "bmc-1" || got.XName != "x1000c0s0b0" || got.Description != "test BMC" {
		t.Errorf("BMC = %+v, want flat YAML name, xname, and description values", got)
	}
}
