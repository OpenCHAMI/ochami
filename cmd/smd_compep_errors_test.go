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

// TestSMDCompepGet_AllHTTPError verifies an unsuccessful HTTP response on get-all
// resolves to CodeHTTP.
func TestSMDCompepGet_AllHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "--ignore-config", "compep", "get", "--uri", srv.URL, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestSMDCompepGet_ByXnameHTTPError verifies a failing per-xname get resolves to
// CodeHTTP via the aggregate.
func TestSMDCompepGet_ByXnameHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "--ignore-config", "compep", "get", "--uri", srv.URL, "--token", "t",
		"x3000c1s7b56n0")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestSMDCompepGet_NetworkError verifies a closed port resolves to CodeNetwork.
func TestSMDCompepGet_NetworkError(t *testing.T) {
	url := "http://127.0.0.1:1" // nothing listens on port 1, so connections are refused

	res := runOchamiWithRuntime(t, "smd", "--ignore-config", "compep", "get", "--uri", url, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeNetwork {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeNetwork, cli.CodeName(cli.CodeNetwork))
	}
}

// TestSMDCompepDelete_Abort verifies answering "n" aborts without a request.
func TestSMDCompepDelete_Abort(t *testing.T) {
	var deletes int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deletes++
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithInputAndRuntime(t, "n\n",
		"smd", "--ignore-config", "compep", "delete", "--uri", srv.URL, "--token", "t", "x3000c1s7b56n0")
	if res.exitCode != cli.CodeDeclined {
		t.Fatalf("result = (err %v, exit %d), want %d (%s)", res.err, res.exitCode, cli.CodeDeclined, cli.CodeName(cli.CodeDeclined))
	}
	if deletes != 0 {
		t.Errorf("DELETE count = %d, want 0", deletes)
	}
}

// TestSMDCompepDelete_NoSelector verifies delete with neither -d, --all, nor args
// is a usage error.
func TestSMDCompepDelete_NoSelector(t *testing.T) {
	res := runOchamiWithRuntime(t, "smd", "--ignore-config", "compep", "delete", "--uri", "http://127.0.0.1:1",
		"--token", "t", "--no-confirm")
	if res.err == nil {
		t.Fatal("expected a usage error, got nil")
	}
	if res.exitCode != cli.CodeUsage {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeUsage, cli.CodeName(cli.CodeUsage))
	}
}

// TestSMDCompepDelete_AllHTTPError verifies a failing "delete --all" resolves to
// CodeHTTP.
func TestSMDCompepDelete_AllHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "--ignore-config", "compep", "delete", "--uri", srv.URL, "--token", "t",
		"--no-confirm", "--all")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}
