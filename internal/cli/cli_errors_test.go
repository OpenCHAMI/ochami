// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/pkg/config"
)

type zeroWriter struct{}

func (zeroWriter) Write([]byte) (int, error) { return 0, nil }

type oversizedWriter struct{}

func (oversizedWriter) Write(p []byte) (int, error) { return len(p) + 1, nil }

type errorWriter struct{ err error }

func (w errorWriter) Write([]byte) (int, error) { return 0, w.err }

type errorReader struct{ err error }

func (r errorReader) Read([]byte) (int, error) { return 0, r.err }

// TestIOStreams_Errors verifies that LoopYesNo returns prompt-write and
// input-read failures, and that WriteOutput reports a writer that writes too
// few or too many bytes as io.ErrShortWrite.
func TestIOStreams_Errors(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("stream failure")
	t.Run("prompt writer", func(t *testing.T) {
		ios := NewIOStreams(strings.NewReader("y\n"), io.Discard, errorWriter{sentinel})
		if _, err := ios.LoopYesNo("Proceed?"); !errors.Is(err, sentinel) {
			t.Fatalf("LoopYesNo() error = %v, want stream failure", err)
		}
	})
	t.Run("input reader", func(t *testing.T) {
		ios := NewIOStreams(errorReader{sentinel}, io.Discard, io.Discard)
		if _, err := ios.LoopYesNo("Proceed?"); !errors.Is(err, sentinel) {
			t.Fatalf("LoopYesNo() error = %v, want stream failure", err)
		}
	})
	t.Run("zero write", func(t *testing.T) {
		if err := WriteOutput(zeroWriter{}, []byte("data")); !errors.Is(err, io.ErrShortWrite) {
			t.Fatalf("WriteOutput() error = %v, want io.ErrShortWrite", err)
		}
	})
	t.Run("oversized write", func(t *testing.T) {
		if err := WriteOutput(oversizedWriter{}, []byte("data")); !errors.Is(err, io.ErrShortWrite) {
			t.Fatalf("WriteOutput() error = %v, want io.ErrShortWrite", err)
		}
	})
}

// TestSetTokenFromEnv_NoCluster verifies that SetTokenFromEnv returns a
// CodeAuth error when neither --token nor --cluster/default-cluster is
// available to resolve a token from.
func TestSetTokenFromEnv_NoCluster(t *testing.T) {
	rt := NewTestRuntime(strings.NewReader(""), io.Discard, io.Discard)
	cmd := &cobra.Command{}
	cmd.Flags().String("cluster", "", "cluster flag")

	err := rt.SetTokenFromEnv(cmd)
	if err == nil {
		t.Fatal("SetTokenFromEnv should have returned an error when no token or cluster is available")
	}
	if ExitCode(err) != CodeAuth {
		t.Errorf("ExitCode = %d, want %d (%s)", ExitCode(err), CodeAuth, CodeName(CodeAuth))
	}
}

// TestHandleToken_UnknownCluster verifies direct callers cannot silently skip
// token handling for a cluster that does not exist.
func TestHandleToken_UnknownCluster(t *testing.T) {
	rt := NewTestRuntime(nil, &bytes.Buffer{}, &bytes.Buffer{})
	rt.Config = config.Config{DefaultCluster: "missing"}
	cmd := tokenTestCmd()
	cmd.SetContext(ContextWithRuntime(context.Background(), rt))
	err := rt.HandleToken(cmd)
	if err == nil {
		t.Fatal("HandleToken(): expected error for unknown cluster, got nil")
	}
	if got := ExitCode(err); got != CodeConfig {
		t.Errorf("exit code = %d, want %d (%s)", got, CodeConfig, CodeName(CodeConfig))
	}
}

// TestHandleToken_AuthEnabledMissingToken verifies that an auth-enabled cluster
// requires a token and errors when none is available.
func TestHandleToken_AuthEnabledMissingToken(t *testing.T) {
	rt := NewTestRuntime(nil, &bytes.Buffer{}, &bytes.Buffer{})
	rt.Config = config.Config{
		DefaultCluster: "secure",
		Clusters: []config.Cluster{
			{Name: "secure", Cluster: config.ClusterConfig{EnableAuth: true}},
		},
	}
	rt.Token = ""
	rt.WithEnvironment(EnvironmentFunc(func(string) (string, bool) { return "", false }))
	cmd := tokenTestCmd()
	cmd.SetContext(ContextWithRuntime(context.Background(), rt))

	err := rt.HandleToken(cmd)
	if err == nil {
		t.Fatal("HandleToken(): expected error for missing token, got nil")
	}
	if got := ExitCode(err); got != CodeAuth {
		t.Errorf("exit code = %d, want %d (%s)", got, CodeAuth, CodeName(CodeAuth))
	}
}

// TestSetTokenFromEnv_MissingEnvVar verifies SetTokenFromEnv errors (CodeAuth) when
// neither --token nor the cluster env var is set.
func TestSetTokenFromEnv_MissingEnvVar(t *testing.T) {
	rt := NewTestRuntime(nil, &bytes.Buffer{}, &bytes.Buffer{})
	rt.Config = config.Config{DefaultCluster: "nope"}
	rt.Token = ""
	rt.WithEnvironment(EnvironmentFunc(func(string) (string, bool) { return "", false }))
	cmd := tokenTestCmd()
	cmd.SetContext(ContextWithRuntime(context.Background(), rt))

	err := rt.SetTokenFromEnv(cmd)
	if err == nil {
		t.Fatal("SetToken(): expected error, got nil")
	}
	if got := ExitCode(err); got != CodeAuth {
		t.Errorf("exit code = %d, want %d (%s)", got, CodeAuth, CodeName(CodeAuth))
	}
	var ce *CodedError
	if !errors.As(err, &ce) {
		t.Errorf("error %v is not a *CodedError", err)
	}
}
