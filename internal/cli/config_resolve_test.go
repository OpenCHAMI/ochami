// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/knadh/koanf/v2"
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/pkg/config"
)

// newConfigEditCmd builds a three-level command tree (grandparent "config",
// parent "cluster", leaf "delete") with --system/--user/--config defined as
// persistent flags on the grandparent, mirroring cmd/config's real layout.
// This is deep enough to catch a ConfigFileToModify/ResolveShowKoanf
// implementation that (like the code this replaced) only checks a fixed
// number of cmd.Parent() calls instead of the whole inherited flag chain.
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

func TestConfigFileToModify(t *testing.T) {
	origConfigFile := ConfigFile
	origUserConfigFile := UserConfigFile
	t.Cleanup(func() {
		ConfigFile = origConfigFile
		UserConfigFile = origUserConfigFile
	})

	t.Run("explicit --config wins", func(t *testing.T) {
		_, leaf := newConfigEditCmd()
		ConfigFile = "/explicit/path.yaml"
		UserConfigFile = "/user/path.yaml"
		if got := ConfigFileToModify(leaf); got != "/explicit/path.yaml" {
			t.Errorf("ConfigFileToModify() = %q, want explicit --config path", got)
		}
	})

	t.Run("--system overrides user default, at any nesting depth", func(t *testing.T) {
		grandparent, leaf := newConfigEditCmd()
		if err := grandparent.PersistentFlags().Set("system", "true"); err != nil {
			t.Fatalf("set --system: %v", err)
		}
		ConfigFile = ""
		UserConfigFile = "/user/path.yaml"
		if got := ConfigFileToModify(leaf); got != config.SystemConfigFile {
			t.Errorf("ConfigFileToModify() = %q, want %q", got, config.SystemConfigFile)
		}
	})

	t.Run("falls back to user config file", func(t *testing.T) {
		_, leaf := newConfigEditCmd()
		ConfigFile = ""
		UserConfigFile = "/user/path.yaml"
		if got := ConfigFileToModify(leaf); got != "/user/path.yaml" {
			t.Errorf("ConfigFileToModify() = %q, want user config file", got)
		}
	})
}

// TestResolveShowKoanf covers the --user, --config, and default
// (ActiveKoanf()) branches with real temp files. The --system branch reads
// the fixed, unoverridable config.SystemConfigFile path, so it isn't
// exercised here; configfile.ReadConfigWithDefaults (which every branch
// delegates to) has its own direct coverage in internal/configfile.
func TestResolveShowKoanf(t *testing.T) {
	origUserConfigFile := UserConfigFile
	origConfig := ActiveConfig()
	origKoanf := activeKoanf
	t.Cleanup(func() {
		UserConfigFile = origUserConfigFile
		SetActiveConfig(origConfig)
		activeKoanf = origKoanf
	})

	writeConfig := func(t *testing.T, content string) string {
		t.Helper()
		path := filepath.Join(t.TempDir(), "config.yaml")
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write config: %v", err)
		}
		return path
	}

	t.Run("--user reads UserConfigFile", func(t *testing.T) {
		grandparent, leaf := newConfigEditCmd()
		if err := grandparent.PersistentFlags().Set("user", "true"); err != nil {
			t.Fatalf("set --user: %v", err)
		}
		UserConfigFile = writeConfig(t, "timeout: 45s\n")

		ko, err := ResolveShowKoanf(leaf)
		if err != nil {
			t.Fatalf("ResolveShowKoanf() error = %v", err)
		}
		if ko.String("timeout") != "45s" {
			t.Errorf("timeout = %q, want 45s", ko.String("timeout"))
		}
	})

	t.Run("--config reads the named file", func(t *testing.T) {
		grandparent, leaf := newConfigEditCmd()
		path := writeConfig(t, "timeout: 90s\n")
		if err := grandparent.PersistentFlags().Set("config", path); err != nil {
			t.Fatalf("set --config: %v", err)
		}

		ko, err := ResolveShowKoanf(leaf)
		if err != nil {
			t.Fatalf("ResolveShowKoanf() error = %v", err)
		}
		if ko.String("timeout") != "90s" {
			t.Errorf("timeout = %q, want 90s", ko.String("timeout"))
		}
	})

	t.Run("no source flag returns ActiveKoanf", func(t *testing.T) {
		_, leaf := newConfigEditCmd()
		want := koanf.New(".")
		activeKoanf = want

		got, err := ResolveShowKoanf(leaf)
		if err != nil {
			t.Fatalf("ResolveShowKoanf() error = %v", err)
		}
		if got != want {
			t.Errorf("ResolveShowKoanf() = %p, want ActiveKoanf() %p", got, want)
		}
	})
}
