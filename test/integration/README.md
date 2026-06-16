<!--
SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC

SPDX-License-Identifier: MIT
-->

# Integration Tests

This directory contains integration tests for the `ochami` CLI tool.
Integration tests verify end-to-end behavior including CLI execution, config
file manipulation, URI routing, and request construction for service commands.

## Structure

```
test/integration/
├── harness/          # Shared test harness utilities
│   └── harness.go    # Helper functions for CLI execution, config files, fake servers
├── config/           # Config CLI command integration tests
│   └── config_test.go
├── uri/              # URI routing and precedence tests
│   └── uri_test.go
├── services/         # Current service integration tests (SMD, boot-service, metadata-service)
│   └── services_test.go
├── legacy/           # Legacy service integration tests (BSS, cloud-init)
│   └── legacy_test.go
```

## Running Tests

### All Integration Tests

```bash
make integration-test
```

### Specific Test Suites

```bash
make integration-test-config    # Config CLI tests
make integration-test-uri       # URI routing tests
make integration-test-services  # Current service request tests
make integration-test-legacy    # Legacy service request tests
```

### Run All Tests (Unit + Integration)

```bash
make test
```

### Skip Integration Tests in CI

Integration tests are tagged with `//go:build integration`. To run only unit
tests:

```bash
make unit-test
```

Or use Go directly:

```bash
go test ./...  # Runs only tests without build tags (unit tests)
```

## Test Categories

### Config Tests (`test/integration/config/`)

Tests for config file manipulation commands:

- `config set` - Setting configuration values
- `config show` - Displaying configuration values
- `config unset` - Removing configuration values
- `config cluster set/show/unset/delete` - Cluster-specific operations

These tests verify:

- File creation when config doesn't exist
- Proper YAML structure and formatting
- Rejection of cluster keys through general config commands
- Default cluster handling

### URI Tests (`test/integration/uri/`)

Tests for URI routing and precedence:

- Default cluster from config
- `--cluster` flag behavior
- `--cluster-uri` flag precedence
- Service-specific `--uri` flag (highest precedence)
- Default service paths
- Absolute vs relative service URIs

These tests use fake HTTP servers to verify that requests are sent to the
correct endpoints without requiring real services.

### Service Tests (`test/integration/services/`)

Tests for current OpenCHAMI service CLI commands using fake HTTP servers:

- SMD (State Management Database)
- boot-service
- metadata-service

These tests:

- Run the real `ochami` CLI binary
- Route commands to local fake HTTP servers
- Verify the outgoing HTTP method and path
- Return minimal valid mock responses for CLI formatting/unmarshalling
- Avoid Docker and external service dependencies so they run quickly in CI

### Legacy Tests (`test/integration/legacy/`)

Tests for legacy OpenCHAMI services:

- BSS (Boot Script Service)
- cloud-init

These services are still supported but being replaced by boot-service and
metadata-service. Tests remain CI-blocking and use fake HTTP servers to ensure
the CLI continues to construct the expected requests during transition.

## Test Harness

The `harness` package provides utilities for integration testing:

### CLI Execution

```go
// Run ochami CLI with args
result := harness.RunCLI(t, "smd", "components", "list")

// Run with config file
result := harness.RunCLIWithConfig(t, configYAML, "config", "show")
```

### Config File Helpers

```go
// Create temp config file
configPath := harness.TempConfigFile(t, configContent)

// Read config file
content := harness.ReadConfigFile(t, configPath)
```

### Fake HTTP Servers

```go
// Create fake server that records requests
server := harness.NewFakeHTTPServer(t, handlerFunc)
defer server.Close()

// Check recorded requests
for _, req := range server.Requests {
    // Verify request details
}
```

### Assertions

```go
harness.AssertContains(t, output, "expected string")
harness.AssertNotContains(t, output, "unwanted string")
harness.AssertEqual(t, got, want)
harness.AssertExitCode(t, result, 0)
harness.AssertLastRequest(t, server, http.MethodGet, "/expected/path")
```

## CI Integration

Integration tests are designed to run in CI:

- Tests have no external dependencies beyond Go
- Service and URI tests use local fake HTTP servers
- Tests clean up temporary files and servers properly

## Adding New Tests

### Config/URI Tests

1. Add test case to appropriate `*_test.go` file
2. Use harness helpers for CLI execution and assertions
3. No external dependencies needed

### Service Tests

1. Add a fake HTTP server in `services/` or `legacy/`
2. Configure the command under test to use the fake server URI
3. Return a minimal valid response body for the command
4. Assert the recorded request method and path

### Test Checklist

- [ ] Add `//go:build integration` tag
- [ ] Use harness helpers for common operations
- [ ] Clean up resources (temp files, servers, containers)
- [ ] Add descriptive test names
- [ ] Include both positive and negative test cases
- [ ] Verify test passes in CI environment
