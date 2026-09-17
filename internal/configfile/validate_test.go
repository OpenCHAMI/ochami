// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package configfile

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/openchami/ochami/pkg/config"
)

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
		var eicv config.ErrInvalidConfigVal
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
		var eicv config.ErrInvalidConfigVal
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
		var eicv config.ErrInvalidConfigVal
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
		var clusters []config.ConfigCluster
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
		var eicv config.ErrInvalidConfigVal
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

	t.Run("empty log.level rejected", func(t *testing.T) {
		path := writeCfg(t, "log:\n  level: \"\"\n")
		_, err := ReadConfigWithDefaults(path)
		if err == nil {
			t.Fatal("expected error for empty log.level, got nil")
		}
		var eicv config.ErrInvalidConfigVal
		if !errors.As(err, &eicv) {
			t.Fatalf("expected ErrInvalidConfigVal, got %T: %v", err, err)
		}
		if eicv.Key != "log.level" {
			t.Errorf("error key = %q, want log.level", eicv.Key)
		}
		if eicv.Value != "empty string" {
			t.Errorf("error value = %q, want empty string", eicv.Value)
		}
	})

	t.Run("empty log.format rejected", func(t *testing.T) {
		path := writeCfg(t, "log:\n  format: \"\"\n")
		_, err := ReadConfigWithDefaults(path)
		if err == nil {
			t.Fatal("expected error for empty log.format, got nil")
		}
		var eicv config.ErrInvalidConfigVal
		if !errors.As(err, &eicv) {
			t.Fatalf("expected ErrInvalidConfigVal, got %T: %v", err, err)
		}
		if eicv.Key != "log.format" {
			t.Errorf("error key = %q, want log.format", eicv.Key)
		}
	})
}
