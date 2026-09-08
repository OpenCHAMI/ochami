// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/pkg/client"
	"github.com/openchami/ochami/pkg/config"
	"github.com/openchami/ochami/pkg/format"
)

func newResolutionCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("cluster", "", "")
	cmd.Flags().String("cluster-uri", "", "")
	cmd.Flags().String("uri", "", "")
	cmd.Flags().String("api-version", "", "")
	cmd.Flags().Duration("timeout", 0, "")
	cmd.Flags().Bool("no-token", false, "")
	cmd.Flags().String("token", "", "")
	cmd.Flags().Bool("show-token", false, "")
	return cmd
}

func TestGetBaseURIExplicitClusterOverridesDefault(t *testing.T) {
	orig := ActiveConfig()
	t.Cleanup(func() { SetActiveConfig(orig) })
	SetActiveConfig(config.Config{
		DefaultCluster: "default",
		Clusters: []config.ConfigCluster{
			{Name: "default", Cluster: config.ConfigClusterConfig{BSS: config.ConfigClusterBSS{URI: "https://default.example/bss"}}},
			{Name: "chosen", Cluster: config.ConfigClusterConfig{BSS: config.ConfigClusterBSS{URI: "https://chosen.example/bss"}}},
		},
	})
	cmd := newResolutionCommand()
	if err := cmd.Flags().Set("cluster", "chosen"); err != nil {
		t.Fatal(err)
	}

	got, err := GetBaseURI(cmd, config.ServiceBSS)
	if err != nil {
		t.Fatalf("GetBaseURI: %v", err)
	}
	if got != "https://chosen.example/bss" {
		t.Errorf("GetBaseURI = %q, want explicit cluster URI", got)
	}
}

func TestGetAPIVersionPrecedenceAndErrors(t *testing.T) {
	orig := ActiveConfig()
	t.Cleanup(func() { SetActiveConfig(orig) })
	SetActiveConfig(config.Config{
		DefaultCluster: "default",
		Clusters: []config.ConfigCluster{
			{Name: "default", Cluster: config.ConfigClusterConfig{BootService: config.ConfigClusterBootService{APIVersion: "v1"}}},
			{Name: "chosen", Cluster: config.ConfigClusterConfig{BootService: config.ConfigClusterBootService{APIVersion: "v2"}}},
		},
	})
	cmd := newResolutionCommand()
	if err := cmd.Flags().Set("cluster", "chosen"); err != nil {
		t.Fatalf("set cluster flag: %v", err)
	}
	got, err := GetAPIVersion(cmd, config.ServiceBoot)
	if err != nil || got != "v2" {
		t.Fatalf("GetAPIVersion explicit cluster = %q, %v; want v2", got, err)
	}
	if err := cmd.Flags().Set("api-version", "v3"); err != nil {
		t.Fatalf("set api-version flag: %v", err)
	}
	got, err = GetAPIVersion(cmd, config.ServiceBoot)
	if err != nil || got != "v3" {
		t.Fatalf("GetAPIVersion flag = %q, %v; want v3", got, err)
	}
	if _, err := GetAPIVersion(cmd, config.ServiceBSS); err != nil {
		t.Fatalf("explicit api-version should be service-independent: %v", err)
	}
}

func TestBooleanFlagsUseTheirValue(t *testing.T) {
	t.Run("ignore-config false", func(t *testing.T) {
		orig := ConfigFile
		t.Cleanup(func() { ConfigFile = orig })
		ConfigFile = t.TempDir() + "/missing.yaml"
		cmd := &cobra.Command{Use: "test"}
		cmd.Flags().Bool("ignore-config", false, "")
		if err := cmd.Flags().Set("ignore-config", "false"); err != nil {
			t.Fatalf("set ignore-config flag: %v", err)
		}
		if err := InitConfig(cmd, false); err == nil {
			t.Fatal("InitConfig unexpectedly ignored a false --ignore-config flag")
		}
	})

	t.Run("no-token false", func(t *testing.T) {
		origCfg, origToken := ActiveConfig(), Token
		t.Cleanup(func() { SetActiveConfig(origCfg); Token = origToken })
		SetActiveConfig(config.Config{
			DefaultCluster: "auth-cluster",
			Clusters:       []config.ConfigCluster{{Name: "auth-cluster", Cluster: config.ConfigClusterConfig{EnableAuth: true}}},
		})
		Token = ""
		_ = os.Unsetenv("AUTH_CLUSTER_ACCESS_TOKEN")
		cmd := newResolutionCommand()
		if err := cmd.Flags().Set("no-token", "false"); err != nil {
			t.Fatalf("set no-token flag: %v", err)
		}
		if err := HandleToken(cmd); err == nil || ExitCode(err) != CodeAuth {
			t.Fatalf("HandleToken error = %v, want CodeAuth", err)
		}
	})
}

func TestPayloadReaderHelpers(t *testing.T) {
	origFormat := FormatInput
	t.Cleanup(func() { FormatInput = origFormat })
	FormatInput = format.DataFormatJson

	var one map[string]interface{}
	restore := SetIOStream(strings.NewReader(`{"name":"node"}`), &bytes.Buffer{}, &bytes.Buffer{})
	if err := HandlePayloadStdin(&cobra.Command{}, &one); err != nil {
		restore()
		t.Fatalf("HandlePayloadStdin: %v", err)
	}
	restore()
	if one["name"] != "node" {
		t.Errorf("payload = %#v", one)
	}

	var many []map[string]interface{}
	restore = SetIOStream(strings.NewReader(`{"name":"node"}`), &bytes.Buffer{}, &bytes.Buffer{})
	if err := HandlePayloadStdinSlice(&cobra.Command{}, &many); err != nil {
		restore()
		t.Fatalf("HandlePayloadStdinSlice: %v", err)
	}
	restore()
	if len(many) != 1 {
		t.Errorf("slice length = %d, want 1", len(many))
	}

	restore = SetIOStream(strings.NewReader(`{`), &bytes.Buffer{}, &bytes.Buffer{})
	if err := HandlePayloadStdin(&cobra.Command{}, &one); err == nil || ExitCode(err) != CodePayload {
		restore()
		t.Fatalf("invalid payload error = %v, want CodePayload", err)
	}
	restore()
}

func TestGetTimeoutAndCompletions(t *testing.T) {
	orig := ActiveConfig()
	t.Cleanup(func() { SetActiveConfig(orig) })
	SetActiveConfig(config.Config{Timeout: 9 * time.Second})
	cmd := newResolutionCommand()
	if got := GetTimeout(cmd); got != 9*time.Second {
		t.Errorf("GetTimeout config = %v", got)
	}
	if err := cmd.Flags().Set("timeout", "2s"); err != nil {
		t.Fatalf("set timeout flag: %v", err)
	}
	if got := GetTimeout(cmd); got != 2*time.Second {
		t.Errorf("GetTimeout flag = %v", got)
	}

	for name, fn := range map[string]func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective){
		"format":    CompletionFormatData,
		"discovery": CompletionDiscoveryVersion,
		"patch":     CompletionPatchMethod,
	} {
		values, directive := fn(cmd, nil, "")
		if len(values) == 0 || directive != cobra.ShellCompDirectiveDefault {
			t.Errorf("%s completion = %v, %v", name, values, directive)
		}
	}
}

func TestPatchMethodValue(t *testing.T) {
	var method client.PatchMethod
	for _, value := range []string{"rfc6902", "rfc7386", "keyval"} {
		if err := method.Set(value); err != nil {
			t.Errorf("Set(%q): %v", value, err)
		}
	}
	if err := method.Set("invalid"); err == nil {
		t.Error("Set(invalid) returned nil")
	}
	if method.Type() != "PatchMethod" {
		t.Errorf("Type = %q", method.Type())
	}
}
