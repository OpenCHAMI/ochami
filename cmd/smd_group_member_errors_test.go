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

// TestSMDGroupMemberDelete_Abort verifies answering "n" aborts without a request.
func TestSMDGroupMemberDelete_Abort(t *testing.T) {
	t.Parallel()

	var deletes int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deletes++
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithInputAndRuntime(t, "n\n",
		"smd", "group", "member", "delete", "--uri", srv.URL, "--token", "t",
		"compute", "x0c0s0b0n0")
	if res.exitCode != cli.CodeDeclined {
		t.Fatalf("result = (err %v, exit %d), want %d (%s)", res.err, res.exitCode, cli.CodeDeclined, cli.CodeName(cli.CodeDeclined))
	}
	if deletes != 0 {
		t.Errorf("DELETE count = %d, want 0", deletes)
	}
}

// TestSMDGroupMemberGet_HTTPError verifies a failing member get resolves to
// CodeHTTP.
func TestSMDGroupMemberGet_HTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "group", "member", "get", "--uri", srv.URL, "--token", "t",
		"compute")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestSMDGroupMemberSet_HTTPError verifies a failing member set resolves to
// CodeHTTP.
func TestSMDGroupMemberSet_HTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "group", "member", "set", "--uri", srv.URL, "--token", "t",
		"compute", "x0c0s0b0n0")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestSMDGroupMember_NetworkErrors verifies member verbs resolve a closed port to
// CodeNetwork.
func TestSMDGroupMember_NetworkErrors(t *testing.T) {
	t.Parallel()

	url := "http://127.0.0.1:1" // nothing listens on port 1, so connections are refused

	// add (per-item aggregation surfaces CodeNetwork)
	res := runOchamiWithRuntime(t, "smd", "group", "member", "add", "--uri", url, "--token", "t",
		"compute", "x0c0s0b0n0")
	if res.exitCode != cli.CodeNetwork {
		t.Errorf("member add network: err=%v exit=%d, want %d (%s)", res.err, res.exitCode, cli.CodeNetwork, cli.CodeName(cli.CodeNetwork))
	}
	// delete (per-item aggregation surfaces CodeNetwork)
	res = runOchamiWithRuntime(t, "smd", "group", "member", "delete", "--uri", url, "--token", "t",
		"--no-confirm", "compute", "x0c0s0b0n0")
	if res.exitCode != cli.CodeNetwork {
		t.Errorf("member delete network: err=%v exit=%d, want %d (%s)", res.err, res.exitCode, cli.CodeNetwork, cli.CodeName(cli.CodeNetwork))
	}
	// get (single request maps transport failure to CodeNetwork)
	res = runOchamiWithRuntime(t, "smd", "group", "member", "get", "--uri", url, "--token", "t", "compute")
	if res.err == nil || res.exitCode != cli.CodeNetwork {
		t.Errorf("member get network: err=%v exit=%d, want %d (%s)", res.err, res.exitCode, cli.CodeNetwork, cli.CodeName(cli.CodeNetwork))
	}
}
