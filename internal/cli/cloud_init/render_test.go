// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cloud_init

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

func TestRender_HeaderModes(t *testing.T) {
	t.Parallel()

	one := []RenderItem{{Labels: "node=x0c0s0b0n0", Body: "one"}}
	multiple := []RenderItem{
		{Labels: "node=x0c0s0b0n0", Body: "one"},
		{Labels: "node=x0c0s0b0n1", Body: "two"},
	}
	tests := []struct {
		name  string
		mode  CIFlagHeaderWhen
		items []RenderItem
		want  string
	}{
		{name: "always zero", mode: CIFlagHeaderAlways},
		{name: "always one", mode: CIFlagHeaderAlways, items: one, want: "--- (1/1) node=x0c0s0b0n0\none\n"},
		{name: "always multiple", mode: CIFlagHeaderAlways, items: multiple, want: "--- (1/2) node=x0c0s0b0n0\none\n--- (2/2) node=x0c0s0b0n1\ntwo\n"},
		{name: "multiple zero", mode: CIFlagHeaderMultiple},
		{name: "multiple one", mode: CIFlagHeaderMultiple, items: one, want: "one\n"},
		{name: "multiple many", mode: CIFlagHeaderMultiple, items: multiple, want: "--- (1/2) node=x0c0s0b0n0\none\n--- (2/2) node=x0c0s0b0n1\ntwo\n"},
		{name: "never zero", mode: CIFlagHeaderNever},
		{name: "never one", mode: CIFlagHeaderNever, items: one, want: "one\n"},
		{name: "never multiple", mode: CIFlagHeaderNever, items: multiple, want: "one\ntwo\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var out bytes.Buffer
			if err := Render(&out, tt.mode, tt.items); err != nil {
				t.Fatalf("Render() error = %v", err)
			}
			if got := out.String(); got != tt.want {
				t.Errorf("Render() output = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRender_PreservesGroupAlwaysSpacing(t *testing.T) {
	t.Parallel()

	items := []RenderItem{{
		Labels:                     "group=compute",
		Body:                       "#cloud-config\nrole: worker\n",
		BlankLineAfterAlwaysHeader: true,
	}}

	var out bytes.Buffer
	if err := Render(&out, CIFlagHeaderAlways, items); err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	want := "--- (1/1) group=compute\n#cloud-config\nrole: worker\n\n\n"
	if got := out.String(); got != want {
		t.Errorf("Render() output = %q, want %q", got, want)
	}
}

type failingWriter struct {
	n   int
	err error
}

func (w failingWriter) Write(p []byte) (int, error) {
	return w.n, w.err
}

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
