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

// TestSMDIfaceGet_ByIPWithoutID verifies "--by-ip" without "--id" is a usage
// error.
func TestSMDIfaceGet_ByIPWithoutID(t *testing.T) {
	res := runOchami(t, "smd", "iface", "get", "--ignore-config", "--uri", "http://127.0.0.1:1",
		"--token", "t", "--by-ip")
	if res.err == nil {
		t.Fatal("expected a usage error, got nil")
	}
	if res.exitCode != cli.CodeUsage {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeUsage, cli.CodeName(cli.CodeUsage))
	}
}

// TestSMDIfaceGet_HTTPError verifies an unsuccessful HTTP response resolves to
// CodeHTTP.
func TestSMDIfaceGet_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "iface", "get", "--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestSMDIfaceGet_NetworkError verifies pointing at a closed port resolves to
// CodeNetwork.
func TestSMDIfaceGet_NetworkError(t *testing.T) {
	url := "http://127.0.0.1:1" // nothing listens on port 1, so connections are refused

	res := runOchami(t, "smd", "iface", "get", "--ignore-config", "--uri", url, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeNetwork {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeNetwork, cli.CodeName(cli.CodeNetwork))
	}
}

// TestSMDIfaceAdd_InvalidIP verifies an invalid IP in the net,ip pair is a usage
// error.
func TestSMDIfaceAdd_InvalidIP(t *testing.T) {
	res := runOchami(t, "smd", "iface", "add", "--ignore-config", "--uri", "http://127.0.0.1:1", "--token", "t",
		"x3000c1s7b55n0", "de:ca:fc:0f:fe:ee", "NMN,not-an-ip")
	if res.err == nil {
		t.Fatal("expected a usage error, got nil")
	}
	if res.exitCode != cli.CodeUsage {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeUsage, cli.CodeName(cli.CodeUsage))
	}
}

// TestSMDIfaceAdd_HTTPError verifies a failing POST resolves to CodeHTTP via the
// per-item aggregation.
func TestSMDIfaceAdd_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad", http.StatusBadRequest)
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "iface", "add", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"x3000c1s7b55n0", "de:ca:fc:0f:fe:ee", "NMN,172.16.0.55")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestSMDIfaceDelete_Abort verifies answering "n" aborts without a request.
func TestSMDIfaceDelete_Abort(t *testing.T) {
	var deletes int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deletes++
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithInput(t, "n\n",
		"smd", "iface", "delete", "--ignore-config", "--uri", srv.URL, "--token", "t", "decafc0ffeee")
	if res.exitCode != cli.CodeDeclined {
		t.Fatalf("result = (err %v, exit %d), want %d (%s)", res.err, res.exitCode, cli.CodeDeclined, cli.CodeName(cli.CodeDeclined))
	}
	if deletes != 0 {
		t.Errorf("DELETE count = %d, want 0", deletes)
	}
}

// TestSMDIfaceDelete_NoSelector verifies delete with neither -d, --all, nor args
// is a usage error.
func TestSMDIfaceDelete_NoSelector(t *testing.T) {
	res := runOchami(t, "smd", "iface", "delete", "--ignore-config", "--uri", "http://127.0.0.1:1",
		"--token", "t", "--no-confirm")
	if res.err == nil {
		t.Fatal("expected a usage error, got nil")
	}
	if res.exitCode != cli.CodeUsage {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeUsage, cli.CodeName(cli.CodeUsage))
	}
}

// TestSMDIfaceDelete_AllHTTPError verifies a failing "delete --all" resolves to
// CodeHTTP.
func TestSMDIfaceDelete_AllHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "iface", "delete", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"--no-confirm", "--all")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestSMDIfaceDelete_ByIDsHTTPError verifies a failing per-item DELETE resolves
// to CodeHTTP via the aggregate.
func TestSMDIfaceDelete_ByIDsHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "iface", "delete", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"--no-confirm", "decafc0ffeee")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}
