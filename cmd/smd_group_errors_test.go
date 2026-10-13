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

// TestSMDGroupGet_HTTPError verifies an unsuccessful HTTP response resolves to
// CodeHTTP.
func TestSMDGroupGet_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "group", "get", "--uri", srv.URL, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestSMDGroupGet_NetworkError verifies pointing at a closed port resolves to
// CodeNetwork.
func TestSMDGroupGet_NetworkError(t *testing.T) {
	url := "http://127.0.0.1:1" // nothing listens on port 1, so connections are refused

	res := runOchamiWithRuntime(t, "smd", "group", "get", "--uri", url, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeNetwork {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeNetwork, cli.CodeName(cli.CodeNetwork))
	}
}

// TestSMDGroupAdd_HTTPError verifies a failing POST resolves to CodeHTTP.
func TestSMDGroupAdd_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad", http.StatusBadRequest)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "group", "add", "--uri", srv.URL, "--token", "t", "compute")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestSMDGroupUpdate_MissingFields verifies "update <label>" with no
// description/tag is a usage error.
func TestSMDGroupUpdate_MissingFields(t *testing.T) {
	res := runOchamiWithRuntime(t, "smd", "group", "update", "--uri", "http://127.0.0.1:1", "--token", "t",
		"compute")
	if res.err == nil {
		t.Fatal("expected a usage error, got nil")
	}
	if res.exitCode != cli.CodeUsage {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeUsage, cli.CodeName(cli.CodeUsage))
	}
}

// TestSMDGroupUpdate_HTTPError verifies a failing PATCH resolves to CodeHTTP.
func TestSMDGroupUpdate_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad", http.StatusBadRequest)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "group", "update", "--uri", srv.URL, "--token", "t",
		"--description", "updated", "compute")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestSMDGroupDelete_HTTPError verifies a failing DELETE resolves to CodeHTTP via
// the aggregate.
func TestSMDGroupDelete_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "group", "delete", "--uri", srv.URL, "--token", "t",
		"--no-confirm", "compute")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestSMDGroupMembership_HTTPError verifies an unsuccessful HTTP response
// resolves to CodeHTTP.
func TestSMDGroupMembership_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "group", "membership", "--uri", srv.URL, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestSMDGroupAdd_BadData verifies "add -d <malformed>" resolves to CodePayload.
func TestSMDGroupAdd_BadData(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "group", "add", "--uri", srv.URL, "--token", "t",
		"-d", `not json`)
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodePayload {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodePayload, cli.CodeName(cli.CodePayload))
	}
}

// TestSMDGroupAdd_NetworkError verifies a refused connection resolves to
// CodeNetwork for group add.
func TestSMDGroupAdd_NetworkError(t *testing.T) {
	url := "http://127.0.0.1:1" // nothing listens on port 1, so connections are refused

	res := runOchamiWithRuntime(t, "smd", "group", "add", "--uri", url, "--token", "t", "compute")
	if res.err == nil {
		t.Fatal("expected a network error, got nil")
	}
	if res.exitCode != cli.CodeNetwork {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeNetwork, cli.CodeName(cli.CodeNetwork))
	}
}
