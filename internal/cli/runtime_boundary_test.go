// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/pkg/config"
)

func TestRuntimeBuilderAndEnvironmentFallback(t *testing.T) {
	rt := NewRuntime().WithLogger(zerolog.Nop()).WithEnvironment(nil)
	const key = "OCHAMI_RUNTIME_BOUNDARY_TEST"
	t.Setenv(key, "present")
	if value, ok := rt.lookupEnv(key); !ok || value != "present" {
		t.Fatalf("lookupEnv(%q) = (%q, %v)", key, value, ok)
	}
}

// TestRuntimeFromCommandIsPureGetter verifies that RuntimeFromCommand does not
// validate or apply format flags itself; format-flag validation is
// PersistentPreRunE's responsibility (via ApplyFormatFlags, called once at
// the root, see TestApplyFormatFlags_InvalidFormat), so RuntimeFromCommand
// succeeds even when the command's format flags hold a value ApplyFormatFlags
// would reject.
func TestRuntimeFromCommandIsPureGetter(t *testing.T) {
	rt := NewTestRuntime(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("format-input", "json", "")
	cmd.Flags().String("format-output", "json", "")
	if err := cmd.Flags().Set("format-input", "toml"); err != nil {
		t.Fatal(err)
	}
	cmd.SetContext(ContextWithRuntime(context.Background(), rt))
	got, err := RuntimeFromCommand(cmd)
	if err != nil {
		t.Fatalf("RuntimeFromCommand() returned an error for an unvalidated format flag: %v", err)
	}
	if got != rt {
		t.Error("RuntimeFromCommand() did not return the runtime stored in the command's context")
	}
}

func TestSetToken_FromEnvRequiresClusterSelection(t *testing.T) {
	rt := NewTestRuntime(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	if err := rt.SetTokenFromEnv(&cobra.Command{Use: "test"}); err == nil {
		t.Fatal("SetTokenFromEnv() accepted absent cluster selection")
	}
}

func TestGetTimeout_FallsBackWhenFlagTypeIsWrong(t *testing.T) {
	rt := NewRuntime().WithConfig(config.Config{Timeout: 37 * time.Second})
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("timeout", "", "")
	if err := cmd.Flags().Set("timeout", "not-a-duration"); err != nil {
		t.Fatal(err)
	}
	if got := rt.GetTimeout(cmd); got != 37*time.Second {
		t.Fatalf("GetTimeout() = %s, want 37s", got)
	}
}

func TestGetBaseURI_IncludesClusterContextInError(t *testing.T) {
	rt := NewRuntime().WithConfig(config.Config{
		DefaultCluster: "demo",
		Clusters:       []config.ConfigCluster{{Name: "demo"}},
	})
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("cluster", "", "")
	cmd.Flags().String("cluster-uri", "", "")
	cmd.Flags().String("uri", "", "")
	_, err := rt.GetBaseURI(cmd, config.ServiceSMD)
	if err == nil || !strings.Contains(err.Error(), "cluster demo") {
		t.Fatalf("GetBaseURI() error = %v, want cluster context", err)
	}
}

func TestInitLoggingRejectsInvalidConfiguration(t *testing.T) {
	rt := NewTestRuntime(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	rt.Config.Log.Level = "not-a-level"
	rt.Config.Log.Format = "json"
	rt.Config.Log.Color = "never"
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("log-format", "", "")
	cmd.Flags().String("log-level", "", "")
	cmd.Flags().String("log-color", "", "")
	if err := rt.InitLogging(cmd); err == nil {
		t.Fatal("InitLogging() accepted invalid level")
	}
}

func TestInitConfigCreateRefusal(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	rt := NewTestRuntime(strings.NewReader("n\n"), &bytes.Buffer{}, &bytes.Buffer{}).WithConfigFile(path)
	cmd := &cobra.Command{Use: "test"}
	if err := rt.InitConfig(cmd, true); err == nil || !strings.Contains(err.Error(), "declined") {
		t.Fatalf("InitConfig() error = %v, want refusal", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("declined config path stat error = %v, want not-exist", err)
	}
}

func TestInitConfigAndLoggingClassifiesConfigFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.yaml")
	if err := os.WriteFile(path, []byte("clusters: [\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	rt := NewTestRuntime(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{}).WithConfigFile(path)
	cmd := &cobra.Command{Use: "test"}
	if err := rt.InitConfigAndLogging(cmd, false); err == nil || ExitCode(err) != CodeConfig {
		t.Fatalf("InitConfigAndLogging() error = %v, want config classification", err)
	}
}
