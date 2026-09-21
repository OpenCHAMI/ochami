// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// TestSMDRFEGet_HTTPError verifies an unsuccessful HTTP response resolves to
// CodeHTTP.
func TestSMDRFEGet_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "rfe", "get", "--uri", srv.URL, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestSMDRFEGet_NetworkError verifies pointing at a closed port resolves to
// CodeNetwork.
func TestSMDRFEGet_NetworkError(t *testing.T) {
	url := "http://127.0.0.1:1" // nothing listens on port 1, so connections are refused

	res := runOchamiWithRuntime(t, "smd", "rfe", "get", "--uri", url, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeNetwork {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeNetwork, cli.CodeName(cli.CodeNetwork))
	}
}

// TestSMDRFEAdd_WrongArgs verifies that fewer than 4 args without -d is a usage
// error.
func TestSMDRFEAdd_WrongArgs(t *testing.T) {
	res := runOchamiWithRuntime(t, "smd", "rfe", "add", "--uri", "http://127.0.0.1:1", "--token", "t",
		"x3000c1s7b56", "bmc-node56")
	if res.err == nil {
		t.Fatal("expected a usage error, got nil")
	}
	if res.exitCode != cli.CodeUsage {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeUsage, cli.CodeName(cli.CodeUsage))
	}
}

// TestSMDRFEAdd_HTTPError verifies a failing POST resolves to CodeHTTP.
func TestSMDRFEAdd_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad", http.StatusBadRequest)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "rfe", "add", "--uri", srv.URL, "--token", "t",
		"x3000c1s7b56", "bmc-node56", "172.16.0.156", "de:ca:fc:0f:fe:ee")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestSMDRFEDelete_Abort verifies answering "n" aborts without a request.
func TestSMDRFEDelete_Abort(t *testing.T) {
	var deletes int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deletes++
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithInputAndRuntime(t, "n\n",
		"--ignore-config", "smd", "rfe", "delete", "--uri", srv.URL, "--token", "t", "x3000c1s7b56")
	if res.exitCode != cli.CodeDeclined {
		t.Fatalf("result = (err %v, exit %d), want %d (%s)", res.err, res.exitCode, cli.CodeDeclined, cli.CodeName(cli.CodeDeclined))
	}
	if deletes != 0 {
		t.Errorf("DELETE count = %d, want 0", deletes)
	}
}

// TestSMDRFEDelete_NoSelector verifies delete with neither -d, --all, nor args is
// a usage error.
func TestSMDRFEDelete_NoSelector(t *testing.T) {
	res := runOchamiWithRuntime(t, "smd", "rfe", "delete", "--uri", "http://127.0.0.1:1",
		"--token", "t", "--no-confirm")
	if res.err == nil {
		t.Fatal("expected a usage error, got nil")
	}
	if res.exitCode != cli.CodeUsage {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeUsage, cli.CodeName(cli.CodeUsage))
	}
}

// TestSMDRFEDelete_ByXnamesHTTPError verifies a failing per-item DELETE resolves
// to CodeHTTP via the aggregate.
func TestSMDRFEDelete_ByXnamesHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "rfe", "delete", "--uri", srv.URL, "--token", "t",
		"--no-confirm", "x3000c1s7b56")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}
