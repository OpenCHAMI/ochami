// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
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

// writeCfg writes content to a temp config file and returns its path.
func writeCfg(t *testing.T, content string) string {
	t.Helper()
	tmp := t.TempDir()
	path := filepath.Join(tmp, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	return path
}

func TestReadConfigWithDefaults_Validation(t *testing.T) {
	t.Run("null timeout rejected", func(t *testing.T) {
		path := writeCfg(t, "timeout:\n")
		_, err := ReadConfigWithDefaults(path)
		if err == nil {
			t.Fatal("expected error for null timeout, got nil")
		}
		var eicv ErrInvalidConfigVal
		if !errors.As(err, &eicv) {
			t.Fatalf("expected ErrInvalidConfigVal, got %T: %v", err, err)
		}
		if eicv.Key != "timeout" {
			t.Errorf("error key = %q, want timeout", eicv.Key)
		}
	})

	t.Run("invalid timeout duration rejected", func(t *testing.T) {
		path := writeCfg(t, "timeout: notaduration\n")
		_, err := ReadConfigWithDefaults(path)
		if err == nil {
			t.Fatal("expected error for invalid timeout, got nil")
		}
		var eicv ErrInvalidConfigVal
		if !errors.As(err, &eicv) {
			t.Fatalf("expected ErrInvalidConfigVal, got %T: %v", err, err)
		}
	})

	t.Run("null enable-auth rejected", func(t *testing.T) {
		path := writeCfg(t, "clusters:\n  - name: foo\n    cluster:\n      uri: https://foo\n      enable-auth:\n")
		_, err := ReadConfigWithDefaults(path)
		if err == nil {
			t.Fatal("expected error for null enable-auth, got nil")
		}
		var eicv ErrInvalidConfigVal
		if !errors.As(err, &eicv) {
			t.Fatalf("expected ErrInvalidConfigVal, got %T: %v", err, err)
		}
	})

	t.Run("string enable-auth coerced", func(t *testing.T) {
		path := writeCfg(t, "clusters:\n  - name: foo\n    cluster:\n      uri: https://foo\n      enable-auth: \"false\"\n")
		ko, err := ReadConfigWithDefaults(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var clusters []ConfigCluster
		if err := ko.Unmarshal("clusters", &clusters); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if len(clusters) != 1 {
			t.Fatalf("got %d clusters, want 1", len(clusters))
		}
		if clusters[0].Cluster.EnableAuth {
			t.Errorf("EnableAuth = true, want false (coerced from string \"false\")")
		}
	})

	t.Run("invalid enable-auth string rejected", func(t *testing.T) {
		path := writeCfg(t, "clusters:\n  - name: foo\n    cluster:\n      uri: https://foo\n      enable-auth: maybe\n")
		_, err := ReadConfigWithDefaults(path)
		if err == nil {
			t.Fatal("expected error for invalid enable-auth string, got nil")
		}
		var eicv ErrInvalidConfigVal
		if !errors.As(err, &eicv) {
			t.Fatalf("expected ErrInvalidConfigVal, got %T: %v", err, err)
		}
	})

	t.Run("valid config accepted", func(t *testing.T) {
		path := writeCfg(t, "timeout: 45s\nclusters:\n  - name: foo\n    cluster:\n      uri: https://foo\n      enable-auth: false\n")
		ko, err := ReadConfigWithDefaults(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := ko.String("timeout"); got != "45s" {
			t.Errorf("timeout = %q, want 45s", got)
		}
	})
}
