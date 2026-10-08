// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// bss_errors_test.go covers the error arms of the "bss" command group; see
// bss_test.go for the success paths these mirror.

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// TestBSSBootParamsAdd_MissingSelectors verifies that add without -d and without
// any of --xname/--nid/--mac is a usage error.
func TestBSSBootParamsAdd_MissingSelectors(t *testing.T) {
	t.Parallel()

	res := runOchamiWithRuntime(t, "bss", "boot", "params", "add",
		"--ignore-config", "--uri", "http://127.0.0.1:0", "--token", "faketoken",
		"--kernel", "https://example.com/vmlinuz")

	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeUsage {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeUsage, cli.CodeName(cli.CodeUsage))
	}
}

// TestBSSServiceStatus_HTTPError verifies an unsuccessful HTTP response from the
// status endpoint resolves to CodeHTTP.
func TestBSSServiceStatus_HTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "bss", "service", "status", "--ignore-config", "--uri", srv.URL)

	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestBSSBootScriptGet_HTTPError verifies a failing boot-script GET resolves to
// CodeHTTP.
func TestBSSBootScriptGet_HTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "bss", "boot", "script", "get", "--ignore-config", "--uri", srv.URL,
		"--mac", "de:ad:be:ef:00:00")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestBSSHostsGet_HTTPError verifies a failing hosts GET resolves to CodeHTTP.
func TestBSSHostsGet_HTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "bss", "hosts", "get", "--ignore-config", "--uri", srv.URL, "--mac", "de:ad:be:ef:00:00")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestBSSHistoryGet_HTTPError verifies a failing history GET resolves to
// CodeHTTP.
func TestBSSHistoryGet_HTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "bss", "history", "--ignore-config", "--uri", srv.URL, "--endpoint", "x0c0s0b0")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestBSSBootParamsGet_AllHTTPError verifies an unsuccessful HTTP response resolves
// to CodeHTTP.
func TestBSSBootParamsGet_AllHTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "bss", "boot", "params", "get",
		"--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestBSSBootParamsGet_AllNetworkError verifies pointing at a closed port resolves
// to CodeNetwork.
func TestBSSBootParamsGet_AllNetworkError(t *testing.T) {
	t.Parallel()

	url := "http://127.0.0.1:1" // nothing listens on port 1, so connections are refused

	res := runOchamiWithRuntime(t, "bss", "boot", "params", "get",
		"--ignore-config", "--uri", url, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeNetwork {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeNetwork, cli.CodeName(cli.CodeNetwork))
	}
}

// TestBSSBootParamsAdd_MalformedPayload verifies malformed inline payload
// resolves to CodePayload.
func TestBSSBootParamsAdd_MalformedPayload(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "bss", "boot", "params", "add",
		"--ignore-config", "--uri", srv.URL, "--token", "t", "-d", `not json`)
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodePayload {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodePayload, cli.CodeName(cli.CodePayload))
	}
}

// TestBSSBootParamsUpdate_HTTPError verifies a failing PATCH resolves to
// CodeHTTP.
func TestBSSBootParamsUpdate_HTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad", http.StatusBadRequest)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "bss", "boot", "params", "update",
		"--ignore-config", "--uri", srv.URL, "--token", "t",
		"--mac", "de:ad:be:ef:00:00", "--kernel", "https://example.com/vmlinuz")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestBSSBootParamsDelete_MissingSelector verifies delete without -d and without
// a component selector is a usage error.
func TestBSSBootParamsDelete_MissingSelector(t *testing.T) {
	t.Parallel()

	res := runOchamiWithRuntime(t, "bss", "boot", "params", "delete",
		"--ignore-config", "--uri", "http://127.0.0.1:1", "--token", "t", "--no-confirm")
	if res.err == nil {
		t.Fatal("expected a usage error, got nil")
	}
	if res.exitCode != cli.CodeUsage {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeUsage, cli.CodeName(cli.CodeUsage))
	}
}

// TestBSSBootParamsDelete_MissingConfig verifies delete with a component selector
// but no config selector is a usage error.
func TestBSSBootParamsDelete_MissingConfig(t *testing.T) {
	t.Parallel()

	res := runOchamiWithRuntime(t, "bss", "boot", "params", "delete",
		"--ignore-config", "--uri", "http://127.0.0.1:1", "--token", "t", "--no-confirm",
		"--mac", "de:ad:be:ef:00:00")
	if res.err == nil {
		t.Fatal("expected a usage error, got nil")
	}
	if res.exitCode != cli.CodeUsage {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeUsage, cli.CodeName(cli.CodeUsage))
	}
}

// TestBSSBootParamsDelete_HTTPError verifies a failing DELETE resolves to
// CodeHTTP.
func TestBSSBootParamsDelete_HTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "bss", "boot", "params", "delete",
		"--ignore-config", "--uri", srv.URL, "--token", "t", "--no-confirm",
		"--mac", "de:ad:be:ef:00:00", "--kernel", "https://example.com/vmlinuz")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestBSSBootParamsUpdate_MissingSelector verifies update without -d and without
// a component selector is a usage error.
func TestBSSBootParamsUpdate_MissingSelector(t *testing.T) {
	t.Parallel()

	res := runOchamiWithRuntime(t, "bss", "boot", "params", "update", "--ignore-config",
		"--uri", "http://127.0.0.1:1", "--token", "t")
	if res.err == nil || res.exitCode != cli.CodeUsage {
		t.Errorf("err=%v exit=%d, want %d (%s)", res.err, res.exitCode, cli.CodeUsage, cli.CodeName(cli.CodeUsage))
	}
}

// TestBSSBootParamsUpdate_MissingConfig verifies update with a component selector
// but no config selector is a usage error.
func TestBSSBootParamsUpdate_MissingConfig(t *testing.T) {
	t.Parallel()

	res := runOchamiWithRuntime(t, "bss", "boot", "params", "update", "--ignore-config",
		"--uri", "http://127.0.0.1:1", "--token", "t", "--mac", "de:ad:be:ef:00:00")
	if res.err == nil || res.exitCode != cli.CodeUsage {
		t.Errorf("err=%v exit=%d, want %d (%s)", res.err, res.exitCode, cli.CodeUsage, cli.CodeName(cli.CodeUsage))
	}
}

// TestBSSBootParamsSet_InvalidMac verifies an invalid MAC address is a usage
// error for "set".
func TestBSSBootParamsSet_InvalidMac(t *testing.T) {
	t.Parallel()

	res := runOchamiWithRuntime(t, "bss", "boot", "params", "set", "--ignore-config",
		"--uri", "http://127.0.0.1:1", "--token", "t", "--mac", "not-a-mac", "--kernel", "http://k")
	if res.err == nil || res.exitCode != cli.CodeUsage {
		t.Errorf("err=%v exit=%d, want %d (%s)", res.err, res.exitCode, cli.CodeUsage, cli.CodeName(cli.CodeUsage))
	}
}

// TestBSSBootParamsUpdate_NetworkError verifies a closed port surfaces
// CodeNetwork for update.
func TestBSSBootParamsUpdate_NetworkError(t *testing.T) {
	t.Parallel()

	url := "http://127.0.0.1:1" // nothing listens on port 1, so connections are refused

	res := runOchamiWithRuntime(t, "bss", "boot", "params", "update", "--ignore-config", "--uri", url, "--token", "t",
		"--mac", "de:ad:be:ef:00:00", "--kernel", "http://k")
	if res.err == nil || res.exitCode != cli.CodeNetwork {
		t.Errorf("err=%v exit=%d, want %d (%s)", res.err, res.exitCode, cli.CodeNetwork, cli.CodeName(cli.CodeNetwork))
	}
}

// TestBSSBootParamsAdd_InvalidMac verifies an invalid MAC is a usage error for
// "add".
func TestBSSBootParamsAdd_InvalidMac(t *testing.T) {
	t.Parallel()

	res := runOchamiWithRuntime(t, "bss", "boot", "params", "add", "--ignore-config",
		"--uri", "http://127.0.0.1:1", "--token", "t", "--mac", "not-a-mac", "--kernel", "http://k")
	if res.err == nil || res.exitCode != cli.CodeUsage {
		t.Errorf("err=%v exit=%d, want %d (%s)", res.err, res.exitCode, cli.CodeUsage, cli.CodeName(cli.CodeUsage))
	}
}

// TestBSSBootParamsAdd_NetworkError verifies a closed port surfaces CodeNetwork
// for "add".
func TestBSSBootParamsAdd_NetworkError(t *testing.T) {
	t.Parallel()

	url := "http://127.0.0.1:1" // nothing listens on port 1, so connections are refused

	res := runOchamiWithRuntime(t, "bss", "boot", "params", "add", "--ignore-config", "--uri", url, "--token", "t",
		"--mac", "de:ad:be:ef:00:00", "--kernel", "http://k")
	if res.err == nil || res.exitCode != cli.CodeNetwork {
		t.Errorf("err=%v exit=%d, want %d (%s)", res.err, res.exitCode, cli.CodeNetwork, cli.CodeName(cli.CodeNetwork))
	}
}

// TestBSSBootParamsSet_NetworkError verifies a closed port surfaces CodeNetwork
// for "set".
func TestBSSBootParamsSet_NetworkError(t *testing.T) {
	t.Parallel()

	url := "http://127.0.0.1:1" // nothing listens on port 1, so connections are refused

	res := runOchamiWithRuntime(t, "bss", "boot", "params", "set", "--ignore-config", "--uri", url, "--token", "t",
		"--mac", "de:ad:be:ef:00:00", "--kernel", "http://k")
	if res.err == nil || res.exitCode != cli.CodeNetwork {
		t.Errorf("err=%v exit=%d, want %d (%s)", res.err, res.exitCode, cli.CodeNetwork, cli.CodeName(cli.CodeNetwork))
	}
}

// TestBSSBootParamsDelete_NetworkError verifies a closed port surfaces
// CodeNetwork for "delete".
func TestBSSBootParamsDelete_NetworkError(t *testing.T) {
	t.Parallel()

	url := "http://127.0.0.1:1" // nothing listens on port 1, so connections are refused

	res := runOchamiWithRuntime(t, "bss", "boot", "params", "delete", "--ignore-config", "--uri", url, "--token", "t",
		"--no-confirm", "--mac", "de:ad:be:ef:00:00", "--kernel", "http://k")
	if res.err == nil || res.exitCode != cli.CodeNetwork {
		t.Errorf("err=%v exit=%d, want %d (%s)", res.err, res.exitCode, cli.CodeNetwork, cli.CodeName(cli.CodeNetwork))
	}
}

// TestBSSBootParamsGet_MalformedResponse verifies that "bss boot params get
// --xname" fails with CodePayload when the response can't be decoded.
func TestBSSBootParamsGet_MalformedResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"BootParameters":[{"ID":"`)); err != nil {
			t.Errorf("writing response: %v", err)
		}
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "--cluster-uri", srv.URL, "--token", "t",
		"bss", "boot", "params", "get", "--xname", "x0c0s1b0n0")

	if res.err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}
	if res.exitCode != cli.CodePayload {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodePayload, cli.CodeName(cli.CodePayload))
	}
}

// TestBSSBootParamsGetByXname_HTTPError verifies that "bss boot params get
// --xname" fails with CodeHTTP for an HTTP 503 response.
func TestBSSBootParamsGetByXname_HTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "--cluster-uri", srv.URL, "--token", "t",
		"bss", "boot", "params", "get", "--xname", "x0c0s1b0n0")

	if res.err == nil {
		t.Fatal("expected error for HTTP 503, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestBSSBootParamsGetByXname_NetworkError verifies that "bss boot params get
// --xname" fails with CodeNetwork when BSS can't be reached.
func TestBSSBootParamsGetByXname_NetworkError(t *testing.T) {
	t.Parallel()

	// Nothing listens on port 1, so the connection is refused.
	res := runOchamiWithRuntime(t, "--ignore-config", "--cluster-uri", "http://127.0.0.1:1", "--token", "t",
		"bss", "boot", "params", "get", "--xname", "x0c0s1b0n0")

	if res.err == nil {
		t.Fatal("expected network error, got nil")
	}
	if res.exitCode != cli.CodeNetwork {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeNetwork, cli.CodeName(cli.CodeNetwork))
	}
}
