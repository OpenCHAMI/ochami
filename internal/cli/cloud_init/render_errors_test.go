// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cloud_init

import (
	"errors"
	"io"
	"testing"
)

// TestRender_WriterFailures verifies that Render returns the writer's error and
// reports a short write as io.ErrShortWrite.
func TestRender_WriterFailures(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("write failed")
	item := []RenderItem{{Labels: "node=x0c0s0b0n0", Body: "body"}}

	if err := Render(failingWriter{err: wantErr}, CIFlagHeaderAlways, item); !errors.Is(err, wantErr) {
		t.Errorf("Render() error = %v, want %v", err, wantErr)
	}
	if err := Render(failingWriter{n: 1}, CIFlagHeaderAlways, item); !errors.Is(err, io.ErrShortWrite) {
		t.Errorf("Render() short-write error = %v, want %v", err, io.ErrShortWrite)
	}
}
