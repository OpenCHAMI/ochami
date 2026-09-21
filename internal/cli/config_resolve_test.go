// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/pkg/config"
)

// newConfigEditCmd builds a three-level command tree (grandparent "config",
// parent "cluster", leaf "delete") with --system/--user/--config defined as
// persistent flags on the grandparent, mirroring cmd/config's real layout.
// This is deep enough to catch a ConfigFileToModify/ResolveShowEffective
// implementation that only checks a fixed number of cmd.Parent() calls
// instead of the whole inherited flag chain.
func newConfigEditCmd() (grandparent, leaf *cobra.Command) {
	grandparent = &cobra.Command{Use: "config"}
	grandparent.PersistentFlags().Bool("system", false, "")
	grandparent.PersistentFlags().Bool("user", true, "")
	grandparent.PersistentFlags().String("config", "", "")
	parent := &cobra.Command{Use: "cluster"}
	leaf = &cobra.Command{Use: "delete"}
	parent.AddCommand(leaf)
	grandparent.AddCommand(parent)
	// Cobra only merges inherited persistent flags into a command's Flags()
	// once the command tree has executed; Find forces that association so
	// leaf.Flag("system") resolves the same way it would at RunE time.
	if _, _, err := grandparent.Find([]string{"cluster", "delete"}); err != nil {
		panic(err)
	}
	return grandparent, leaf
}

// TestConfigFileToModify verifies that ConfigFileToModify chooses --config
// first, then --system at any nesting depth, and otherwise the user config
// file.
func TestConfigFileToModify(t *testing.T) {
	t.Run("explicit --config wins", func(t *testing.T) {
		_, leaf := newConfigEditCmd()
		rt := NewTestRuntime(nil, &bytes.Buffer{}, &bytes.Buffer{})
		rt.ConfigFile = "/explicit/path.yaml"
		rt.UserConfigFile = "/user/path.yaml"
		if got := rt.ConfigFileToModify(leaf); got != "/explicit/path.yaml" {
			t.Errorf("ConfigFileToModify() = %q, want explicit --config path", got)
		}
	})

	t.Run("--system overrides user default, at any nesting depth", func(t *testing.T) {
		grandparent, leaf := newConfigEditCmd()
		if err := grandparent.PersistentFlags().Set("system", "true"); err != nil {
			t.Fatalf("set --system: %v", err)
		}
		rt := NewTestRuntime(nil, &bytes.Buffer{}, &bytes.Buffer{})
		rt.UserConfigFile = "/user/path.yaml"
		if got := rt.ConfigFileToModify(leaf); got != config.SystemConfigFile {
			t.Errorf("ConfigFileToModify() = %q, want %q", got, config.SystemConfigFile)
		}
	})

	t.Run("falls back to user config file", func(t *testing.T) {
		_, leaf := newConfigEditCmd()
		rt := NewTestRuntime(nil, &bytes.Buffer{}, &bytes.Buffer{})
		rt.UserConfigFile = "/user/path.yaml"
		if got := rt.ConfigFileToModify(leaf); got != "/user/path.yaml" {
			t.Errorf("ConfigFileToModify() = %q, want user config file", got)
		}
	})
}

// TestResolveShowEffective verifies that ResolveShowEffective reads the
// runtime's user config file for --user and the named file for --config, and
// otherwise returns rt.Effective. The --system file is a fixed path, so it
// isn't tested here.
func TestResolveShowEffective(t *testing.T) {
	writeConfig := func(t *testing.T, content string) string {
		t.Helper()
		path := filepath.Join(t.TempDir(), "config.yaml")
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write config: %v", err)
		}
		return path
	}

	t.Run("--user reads rt.UserConfigFile", func(t *testing.T) {
		grandparent, leaf := newConfigEditCmd()
		if err := grandparent.PersistentFlags().Set("user", "true"); err != nil {
			t.Fatalf("set --user: %v", err)
		}
		rt := NewTestRuntime(nil, &bytes.Buffer{}, &bytes.Buffer{})
		rt.UserConfigFile = writeConfig(t, "timeout: 45s\n")

		eff, err := rt.ResolveShowEffective(leaf)
		if err != nil {
			t.Fatalf("ResolveShowEffective() error = %v", err)
		}
		if got, _ := eff.Get("timeout").(string); got != "45s" {
			t.Errorf("timeout = %q, want 45s", got)
		}
	})

	t.Run("--config reads the named file", func(t *testing.T) {
		grandparent, leaf := newConfigEditCmd()
		path := writeConfig(t, "timeout: 90s\n")
		if err := grandparent.PersistentFlags().Set("config", path); err != nil {
			t.Fatalf("set --config: %v", err)
		}
		rt := NewTestRuntime(nil, &bytes.Buffer{}, &bytes.Buffer{})

		eff, err := rt.ResolveShowEffective(leaf)
		if err != nil {
			t.Fatalf("ResolveShowEffective() error = %v", err)
		}
		if got, _ := eff.Get("timeout").(string); got != "90s" {
			t.Errorf("timeout = %q, want 90s", got)
		}
	})

	t.Run("no source flag returns rt.Effective", func(t *testing.T) {
		_, leaf := newConfigEditCmd()
		want, _, err := config.LoadDefaultsEffective()
		if err != nil {
			t.Fatalf("LoadDefaultsEffective(): %v", err)
		}
		rt := NewTestRuntime(nil, &bytes.Buffer{}, &bytes.Buffer{})
		rt.Effective = want

		got, err := rt.ResolveShowEffective(leaf)
		if err != nil {
			t.Fatalf("ResolveShowEffective() error = %v", err)
		}
		if got != want {
			t.Errorf("ResolveShowEffective() = %+v, want rt.Effective %+v", got, want)
		}
	})
}
