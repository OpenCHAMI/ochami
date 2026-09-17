// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package config

import (
	"testing"

	kyaml "github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/knadh/koanf/v2"
)

func TestCoerceBool(t *testing.T) {
	tests := []struct {
		name   string
		in     any
		want   bool
		wantOK bool
	}{
		{"bool true", true, true, true},
		{"bool false", false, false, true},
		{"string true", "true", true, true},
		{"string True", "True", true, true},
		{"string false", "False", false, true},
		{"string 1", "1", true, true},
		{"string 0", "0", false, true},
		{"invalid string", "yesplease", false, false},
		{"int not supported", 1, false, false},
		{"nil", nil, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := coerceBool(tt.in)
			if ok != tt.wantOK || (ok && got != tt.want) {
				t.Fatalf("coerceBool(%v) = (%v, %v), want (%v, %v)", tt.in, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func loadKoanfYAML(t *testing.T, yaml string) *koanf.Koanf {
	t.Helper()
	ko := koanf.NewWithConf(koanfConf)
	if err := ko.Load(rawbytes.Provider([]byte(yaml)), kyaml.Parser()); err != nil {
		t.Fatalf("ko.Load: %v", err)
	}
	return ko
}

func TestValidateConfigErrors(t *testing.T) {
	// Invalid timeout duration.
	if err := ValidateConfig(loadKoanfYAML(t, "timeout: not-a-duration\n")); err == nil {
		t.Error("ValidateConfig(invalid timeout) = nil, want error")
	}

	// Invalid enable-auth value.
	yaml := `clusters:
- name: demo
  cluster:
    enable-auth: maybe
`
	if err := ValidateConfig(loadKoanfYAML(t, yaml)); err == nil {
		t.Error("ValidateConfig(invalid enable-auth) = nil, want error")
	}

	// Valid config passes.
	valid := `timeout: 30s
clusters:
- name: demo
  cluster:
    uri: https://demo.example.com
    enable-auth: true
`
	if err := ValidateConfig(loadKoanfYAML(t, valid)); err != nil {
		t.Errorf("ValidateConfig(valid) = %v, want nil", err)
	}
}
