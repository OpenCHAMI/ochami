// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/pkg/config"
)

type runtimePayload struct {
	Name string `json:"name"`
}

func payloadCommand(t *testing.T, data string) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{}
	cmd.Flags().String("data", "", "")
	if err := cmd.Flags().Set("data", data); err != nil {
		t.Fatalf("set data flag: %v", err)
	}
	return cmd
}

func TestRuntimeHandlePayload_UsesRuntimeInput(t *testing.T) {
	t.Parallel()

	rt := NewTestRuntime(strings.NewReader(`{"name":"runtime"}`), io.Discard, io.Discard)
	var got runtimePayload
	if err := rt.HandlePayload(payloadCommand(t, "@-"), &got); err != nil {
		t.Fatalf("HandlePayload() error = %v", err)
	}
	if got.Name != "runtime" {
		t.Fatalf("HandlePayload() = %#v, want runtime input", got)
	}
}

func TestRuntimeHandlePayload_SliceUsesRuntimeInput(t *testing.T) {
	t.Parallel()

	rt := NewTestRuntime(strings.NewReader(`[{"name":"a"},{"name":"b"}]`), io.Discard, io.Discard)
	var got []runtimePayload
	if err := HandlePayloadSliceWithRuntime[runtimePayload](rt, payloadCommand(t, "@-"), &got); err != nil {
		t.Fatalf("HandlePayloadSliceWithRuntime() error = %v", err)
	}
	if len(got) != 2 || got[0].Name != "a" || got[1].Name != "b" {
		t.Fatalf("HandlePayloadSliceWithRuntime() = %#v", got)
	}
}

func TestRuntimeEnvironmentIsolation(t *testing.T) {
	t.Parallel()

	lookup := func(values map[string]string) EnvironmentFunc {
		return func(key string) (string, bool) {
			value, ok := values[key]
			return value, ok
		}
	}
	rtA := NewTestRuntime(nil, io.Discard, io.Discard).WithEnvironment(lookup(map[string]string{"CLUSTER_A_ACCESS_TOKEN": "token-a"}))
	rtB := NewTestRuntime(nil, io.Discard, io.Discard).WithEnvironment(lookup(map[string]string{"CLUSTER_B_ACCESS_TOKEN": "token-b"}))
	rtA.Config.DefaultCluster = "cluster-a"
	rtB.Config.DefaultCluster = "cluster-b"

	if err := rtA.SetTokenFromEnv(tokenTestCmd()); err != nil {
		t.Fatalf("runtime A SetTokenFromEnv() error = %v", err)
	}
	if err := rtB.SetTokenFromEnv(tokenTestCmd()); err != nil {
		t.Fatalf("runtime B SetTokenFromEnv() error = %v", err)
	}
	if rtA.Token != "token-a" || rtB.Token != "token-b" {
		t.Fatalf("tokens = %q, %q; want token-a, token-b", rtA.Token, rtB.Token)
	}
}

func TestRuntimeResolveUserConfigFileUsesRuntimeEnvironment(t *testing.T) {
	t.Parallel()

	rt := NewTestRuntime(nil, &bytes.Buffer{}, &bytes.Buffer{}).WithEnvironment(EnvironmentFunc(func(key string) (string, bool) {
		if key == "XDG_CONFIG_HOME" {
			return "/runtime/config", true
		}
		return "", false
	}))
	if err := rt.ResolveUserConfigFile(); err != nil {
		t.Fatalf("ResolveUserConfigFile() error = %v", err)
	}
	want := filepath.Join("/runtime/config", "ochami", "config.yaml")
	if rt.UserConfigFile != want {
		t.Fatalf("UserConfigFile = %q, want %q", rt.UserConfigFile, want)
	}
}

func TestRuntimeLoadMergedConfigUsesRuntimeEnvironment(t *testing.T) {
	t.Parallel()

	xdgHome := t.TempDir()
	configDir := filepath.Join(xdgHome, "ochami")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("create config directory: %v", err)
	}
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("log:\n  level: warning\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	rt := NewTestRuntime(nil, io.Discard, io.Discard).WithEnvironment(EnvironmentFunc(func(key string) (string, bool) {
		return xdgHome, key == "XDG_CONFIG_HOME"
	}))
	if err := rt.loadMergedConfig(); err != nil {
		t.Fatalf("loadMergedConfig() error = %v", err)
	}
	if rt.UserConfigFile != configPath {
		t.Fatalf("UserConfigFile = %q, want %q", rt.UserConfigFile, configPath)
	}
	if rt.Config.Log.Level != "warning" {
		t.Fatalf("Log.Level = %q, want warning", rt.Config.Log.Level)
	}
}

func TestHandleTokenFlagPrecedesRuntimeEnvironment(t *testing.T) {
	t.Parallel()

	now := time.Now()
	flagToken, err := generateTestToken(now.Add(time.Hour), now.Add(-time.Hour), now.Add(-time.Hour))
	if err != nil {
		t.Fatalf("generate flag token: %v", err)
	}
	envToken, err := generateTestToken(now.Add(2*time.Hour), now.Add(-time.Hour), now.Add(-time.Hour))
	if err != nil {
		t.Fatalf("generate environment token: %v", err)
	}
	rt := NewTestRuntime(nil, io.Discard, io.Discard).WithEnvironment(EnvironmentFunc(func(key string) (string, bool) {
		return envToken, key == "MY_CLUSTER_ACCESS_TOKEN"
	}))
	rt.Config = config.Config{
		DefaultCluster: "my-cluster",
		Clusters: []config.ConfigCluster{{
			Name:    "my-cluster",
			Cluster: config.ConfigClusterConfig{EnableAuth: true},
		}},
	}
	cmd := tokenTestCmd()
	if err := cmd.Flags().Set("token", flagToken); err != nil {
		t.Fatalf("set token flag: %v", err)
	}
	if err := rt.HandleToken(cmd); err != nil {
		t.Fatalf("HandleToken() error = %v", err)
	}
	if rt.Token != flagToken {
		t.Fatal("HandleToken() did not preserve explicit token precedence")
	}
}

func TestTokenHelpersAllowStandaloneCommands(t *testing.T) {
	t.Parallel()

	rt := NewTestRuntime(nil, io.Discard, io.Discard)
	cmd := &cobra.Command{}
	if err := rt.SetTokenFromFlag(cmd); err != nil {
		t.Fatalf("SetTokenFromFlag() error = %v", err)
	}
	if err := rt.HandleToken(cmd); err != nil {
		t.Fatalf("HandleToken() error = %v", err)
	}
}

// TestInitConfigIgnoreConfigResolvesUserConfigFile verifies that --ignore-config
// still resolves rt.UserConfigFile (without reading it), so commands that
// report or target the user config path (e.g. "config show --user") get a
// usable path instead of an empty string.
func TestInitConfigIgnoreConfigResolvesUserConfigFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))

	rt := NewTestRuntime(nil, &bytes.Buffer{}, &bytes.Buffer{}).WithEnvironment(EnvironmentFunc(os.LookupEnv))
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().Bool("ignore-config", false, "")
	if err := cmd.Flags().Set("ignore-config", "true"); err != nil {
		t.Fatalf("set ignore-config flag: %v", err)
	}

	if err := rt.InitConfig(cmd, false); err != nil {
		t.Fatalf("InitConfig() error = %v", err)
	}
	if rt.UserConfigFile == "" {
		t.Error("InitConfig() with --ignore-config did not resolve the user config path")
	}
}

// TestConfigLoadOptsTracesOnlyWhenVerbose verifies that rt.configLoadOpts
// attaches a working trace logger when --verbose is on (EarlyVerbose), and
// omits it entirely otherwise, so pkg/config's key-enumeration trace path is
// only paid for when it will actually be observed.
func TestConfigLoadOptsTracesOnlyWhenVerbose(t *testing.T) {
	t.Run("verbose", func(t *testing.T) {
		var stderr bytes.Buffer
		rt := NewTestRuntime(nil, io.Discard, &stderr)
		rt.EarlyVerbose = true
		cmd := &cobra.Command{Use: "test"}
		cmd.Flags().Bool("ignore-config", false, "")
		if err := cmd.Flags().Set("ignore-config", "true"); err != nil {
			t.Fatalf("set ignore-config flag: %v", err)
		}
		if err := rt.InitConfig(cmd, false); err != nil {
			t.Fatalf("InitConfig() error = %v", err)
		}
		if !strings.Contains(stderr.String(), "final config") {
			t.Errorf("expected verbose trace output, got: %q", stderr.String())
		}
	})

	t.Run("not verbose", func(t *testing.T) {
		rt := NewTestRuntime(nil, io.Discard, io.Discard)
		if opts := rt.configLoadOpts(); opts != nil {
			t.Errorf("configLoadOpts() = %v, want nil when not verbose", opts)
		}
	})
}
