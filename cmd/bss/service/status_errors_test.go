// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package service

import (
	"errors"
	"testing"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/client"
)

// TestServiceStatus_ClientConstructionError verifies client setup failures are
// returned unchanged.
func TestServiceStatus_ClientConstructionError(t *testing.T) {
	wantErr := errors.New("boom")
	cmd := newCmdServiceStatusWithClient(providerFor(nil, wantErr))
	cmd.SetArgs(nil)

	err := cmd.Execute()
	if !errors.Is(err, wantErr) {
		t.Fatalf("Execute() error = %v, want wrapped %v", err, wantErr)
	}
}

// TestServiceStatus_HTTPErrorMapping verifies service response failures map to
// the HTTP exit code.
func TestServiceStatus_HTTPErrorMapping(t *testing.T) {
	fake := &fakeBSSStatusClient{err: client.UnsuccessfulHTTPError}
	cmd := newCmdServiceStatusWithClient(providerFor(fake, nil))
	cmd.SetArgs(nil)

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute(): expected error, got nil")
	}
	if got := cli.ExitCode(err); got != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", got, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestServiceStatus_NetworkErrorMapping verifies transport failures map to the
// network exit code.
func TestServiceStatus_NetworkErrorMapping(t *testing.T) {
	fake := &fakeBSSStatusClient{err: errors.New("connection refused")}
	cmd := newCmdServiceStatusWithClient(providerFor(fake, nil))
	cmd.SetArgs(nil)

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute(): expected error, got nil")
	}
	if got := cli.ExitCode(err); got != cli.CodeNetwork {
		t.Errorf("exit code = %d, want %d (%s)", got, cli.CodeNetwork, cli.CodeName(cli.CodeNetwork))
	}
}
