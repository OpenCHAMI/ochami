// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package config

import (
	"path/filepath"
	"testing"
)

func TestUserConfigPathWithEnv(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		env  map[string]string
		want string
	}{
		{
			name: "XDG config home takes precedence",
			env:  map[string]string{"XDG_CONFIG_HOME": "/xdg", "HOME": "/home/user"},
			want: filepath.Join("/xdg", "ochami", "config.yaml"),
		},
		{
			name: "empty XDG config home falls back to HOME",
			env:  map[string]string{"XDG_CONFIG_HOME": "", "HOME": "/home/user"},
			want: filepath.Join("/home/user", ".config", "ochami", "config.yaml"),
		},
		{
			name: "unset XDG config home falls back to HOME",
			env:  map[string]string{"HOME": "/other/home"},
			want: filepath.Join("/other/home", ".config", "ochami", "config.yaml"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			lookup := func(key string) (string, bool) {
				value, ok := tt.env[key]
				return value, ok
			}
			got, err := UserConfigPathWithEnv(lookup)
			if err != nil {
				t.Fatalf("UserConfigPathWithEnv() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("UserConfigPathWithEnv() = %q, want %q", got, tt.want)
			}
		})
	}
}
