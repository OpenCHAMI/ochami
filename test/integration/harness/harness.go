// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

//go:build integration

package harness

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// CLIResult holds the result of a CLI command execution
type CLIResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Err      error
}

// RunCLI executes the ochami CLI with the given arguments
func RunCLI(t *testing.T, args ...string) CLIResult {
	t.Helper()
	return RunCLIWithInput(t, "", args...)
}

// RunCLIWithInput executes the ochami CLI with stdin content and arguments.
func RunCLIWithInput(t *testing.T, stdin string, args ...string) CLIResult {
	t.Helper()

	// Build the ochami binary if not already built
	ochamiBin := filepath.Join(t.TempDir(), "ochami")
	buildCmd := exec.Command("go", "build", "-o", ochamiBin, repoRoot(t))
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("failed to build ochami: %v", err)
	}

	// Execute the command
	cmd := exec.Command(ochamiBin, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdin = strings.NewReader(stdin)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Logf("command execution error: %v", err)
		}
	}

	return CLIResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
		Err:      err,
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to determine harness source path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../../.."))
}

// RunCLIWithConfig executes ochami CLI with a temporary config file
func RunCLIWithConfig(t *testing.T, configContent string, args ...string) CLIResult {
	t.Helper()

	// Create temporary config file
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Prepend config flag to args
	allArgs := append([]string{"--config", configPath}, args...)
	return RunCLI(t, allArgs...)
}

// TempConfigFile creates a temporary config file with the given content
func TempConfigFile(t *testing.T, content string) string {
	t.Helper()

	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp config file: %v", err)
	}
	return configPath
}

// ReadConfigFile reads and returns the content of a config file
func ReadConfigFile(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read config file %s: %v", path, err)
	}
	return string(content)
}

// FakeHTTPServer creates a test HTTP server with custom handlers
type FakeHTTPServer struct {
	*httptest.Server
	Requests []*http.Request
}

// NewFakeHTTPServer creates a new fake HTTP server that records requests
func NewFakeHTTPServer(t *testing.T, handler http.HandlerFunc) *FakeHTTPServer {
	t.Helper()

	fs := &FakeHTTPServer{
		Requests: make([]*http.Request, 0),
	}

	// Wrap handler to record requests
	recordingHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Clone the request for recording
		body, _ := io.ReadAll(r.Body)
		r.Body = io.NopCloser(bytes.NewReader(body))

		clonedReq := r.Clone(context.Background())
		clonedReq.Body = io.NopCloser(bytes.NewReader(body))
		fs.Requests = append(fs.Requests, clonedReq)

		// Restore body for handler
		r.Body = io.NopCloser(bytes.NewReader(body))
		handler(w, r)
	})

	fs.Server = httptest.NewServer(recordingHandler)
	return fs
}

// WaitForReady waits for a condition to become true within a timeout
func WaitForReady(t *testing.T, timeout time.Duration, checkFn func() bool) bool {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if checkFn() {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

// AssertContains checks if a string contains a substring
func AssertContains(t *testing.T, got, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Errorf("output does not contain expected string:\ngot:  %q\nwant substring: %q", got, want)
	}
}

// AssertNotContains checks if a string does not contain a substring
func AssertNotContains(t *testing.T, got, unwanted string) {
	t.Helper()
	if strings.Contains(got, unwanted) {
		t.Errorf("output contains unwanted string:\ngot:  %q\nunwanted substring: %q", got, unwanted)
	}
}

// AssertEqual checks if two values are equal
func AssertEqual(t *testing.T, got, want interface{}) {
	t.Helper()
	if got != want {
		t.Errorf("values not equal:\ngot:  %v\nwant: %v", got, want)
	}
}

// AssertExitCode checks if the exit code matches expected
func AssertExitCode(t *testing.T, result CLIResult, wantCode int) {
	t.Helper()
	if result.ExitCode != wantCode {
		t.Fatalf("exit code = %d, want %d\nstdout: %s\nstderr: %s",
			result.ExitCode, wantCode, result.Stdout, result.Stderr)
	}
}

// ServiceResponse represents a test service response
type ServiceResponse struct {
	StatusCode int
	Body       string
	Headers    map[string]string
}

// NewServiceHandler creates an HTTP handler that returns predefined responses
func NewServiceHandler(responses map[string]ServiceResponse) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := fmt.Sprintf("%s %s", r.Method, r.URL.Path)
		if resp, ok := responses[key]; ok {
			for k, v := range resp.Headers {
				w.Header().Set(k, v)
			}
			w.WriteHeader(resp.StatusCode)
			fmt.Fprint(w, resp.Body)
			return
		}
		// Default 404
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, `{"error": "not found: %s"}`, key)
	}
}

// ClusterConfig returns a minimal ochami config with service-specific absolute
// URIs. Values should include service base paths where the service expects one
// (for example BSS uses /boot/v1 and SMD uses /hsm/v2).
func ClusterConfig(serviceYAML string) string {
	return fmt.Sprintf(`log:
  level: warning
  format: rfc3339
default-cluster: integration
clusters:
  - name: integration
    cluster:
      enable-auth: false
%s
`, indentYAML(serviceYAML, 6))
}

func indentYAML(s string, spaces int) string {
	if strings.TrimSpace(s) == "" {
		return ""
	}
	indent := strings.Repeat(" ", spaces)
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) != "" {
			lines[i] = indent + line
		}
	}
	return strings.Join(lines, "\n")
}

// AssertRequestCount checks how many requests a fake server recorded.
func AssertRequestCount(t *testing.T, server *FakeHTTPServer, want int) {
	t.Helper()
	if len(server.Requests) != want {
		t.Fatalf("recorded requests = %d, want %d", len(server.Requests), want)
	}
}

// AssertRequest checks the method and path of a recorded HTTP request.
func AssertRequest(t *testing.T, req *http.Request, method, path string) {
	t.Helper()
	if req.Method != method {
		t.Errorf("request method = %q, want %q", req.Method, method)
	}
	if req.URL.Path != path {
		t.Errorf("request path = %q, want %q", req.URL.Path, path)
	}
}

// AssertHeader checks the value of a recorded HTTP request header.
func AssertHeader(t *testing.T, req *http.Request, key, want string) {
	t.Helper()
	if got := req.Header.Get(key); got != want {
		t.Errorf("request header %q = %q, want %q", key, got, want)
	}
}

// AssertLastRequest checks the method and path of the most recent request.
func AssertLastRequest(t *testing.T, server *FakeHTTPServer, method, path string) {
	t.Helper()
	if len(server.Requests) == 0 {
		t.Fatal("no requests recorded")
	}
	AssertRequest(t, server.Requests[len(server.Requests)-1], method, path)
}

// AssertLastRequestHeader checks a header on the most recent request.
func AssertLastRequestHeader(t *testing.T, server *FakeHTTPServer, key, want string) {
	t.Helper()
	if len(server.Requests) == 0 {
		t.Fatal("no requests recorded")
	}
	AssertHeader(t, server.Requests[len(server.Requests)-1], key, want)
}
