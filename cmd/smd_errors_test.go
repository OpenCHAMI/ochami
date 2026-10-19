// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// smd_errors_test.go covers failure paths for the "smd" command group; see
// smd_test.go for the success paths these mirror.

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/config"
)

// TestSMDDeprecatedStatus_HTTPError verifies that an unsuccessful HTTP response from the
// SMD service status endpoint resolves to CodeHTTP.
func TestSMDDeprecatedStatus_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "service", "status", "--ignore-config", "--uri", srv.URL)
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestSMDServiceStatus_DuplicateCluster verifies that "smd service status"
// fails with a config error, without contacting SMD, when the cluster
// selected through default-cluster or --cluster is defined more than once in
// the config file, even when --uri is also passed.
func TestSMDServiceStatus_DuplicateCluster(t *testing.T) {
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
	}))
	defer srv.Close()
	cfg := writeTempConfig(t, duplicateClusterConfig)

	for _, args := range [][]string{
		{"--config", cfg, "smd", "service", "status"},
		{"--config", cfg, "smd", "service", "status", "--cluster", "demo"},
		{"--config", cfg, "smd", "service", "status", "--uri", srv.URL},
	} {
		res := runOchami(t, args...)
		if !errors.As(res.err, &config.ErrDuplicateCluster{}) {
			t.Errorf("%v: error = %v, want ErrDuplicateCluster", args, res.err)
		}
		if res.exitCode != cli.CodeConfig {
			t.Errorf("%v: exit code = %d, want %d (%s)", args, res.exitCode, cli.CodeConfig, cli.CodeName(cli.CodeConfig))
		}
	}
	if n := requests.Load(); n != 0 {
		t.Errorf("SMD received %d requests, want 0", n)
	}
}
