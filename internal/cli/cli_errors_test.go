// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cli

import (
	"testing"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/config"
)

// TestSetToken_NoTokenNoCluster verifies that SetToken returns a CodeAuth
// error when neither --token nor --cluster/default-cluster is available to
// resolve a token from.
func TestSetToken_NoTokenNoCluster(t *testing.T) {
	// Save original token/default-cluster and restore after test
	originalToken := Token
	originalDefaultCluster := config.GlobalConfig.DefaultCluster
	defer func() {
		Token = originalToken
		config.GlobalConfig.DefaultCluster = originalDefaultCluster
	}()
	config.GlobalConfig.DefaultCluster = ""

	cmd := &cobra.Command{}
	cmd.Flags().String("token", "", "token flag")
	cmd.Flags().String("cluster", "", "cluster flag")

	err := SetToken(cmd)
	if err == nil {
		t.Fatal("SetToken should have returned an error when no token or cluster is available")
	}
	if ExitCode(err) != CodeAuth {
		t.Errorf("ExitCode = %d, want %d (%s)", ExitCode(err), CodeAuth, CodeName(CodeAuth))
	}
}
