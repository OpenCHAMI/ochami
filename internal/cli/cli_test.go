// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cli

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIOStream_AskToCreate(t *testing.T) {
	t.Run("empty path", func(t *testing.T) {
		t.Parallel()
		inBuf := &bytes.Buffer{}
		outBuf := &bytes.Buffer{}
		errBuf := &bytes.Buffer{}
		ios := newIOStream(inBuf, outBuf, errBuf)

		got, err := ios.AskToCreate("")
		if got != false {
			t.Errorf("AskToCreate(\"\") = %v, want false", got)
		}
		if err == nil || !strings.Contains(err.Error(), "path cannot be empty") {
			t.Errorf("AskToCreate(\"\") error = %v, want non-nil containing “path cannot be empty”", err)
		}
		if outBuf.Len() != 0 {
			t.Errorf("stdout = %q, want empty", outBuf.String())
		}
		if errBuf.Len() != 0 {
			t.Errorf("stderr = %q, want empty", errBuf.String())
		}
	})

	t.Run("existing file", func(t *testing.T) {
		t.Parallel()
		tmp := t.TempDir()
		f := filepath.Join(tmp, "exists")
		if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
			t.Fatalf("setup write: %v", err)
		}

		inBuf := &bytes.Buffer{}
		outBuf := &bytes.Buffer{}
		errBuf := &bytes.Buffer{}
		ios := newIOStream(inBuf, outBuf, errBuf)

		got, err := ios.AskToCreate(f)
		if got != false {
			t.Errorf("AskToCreate(%q) = %v, want false", f, got)
		}
		if !errors.Is(err, FileExistsError) {
			t.Errorf("AskToCreate(%q) error = %v, want FileExistsError", f, err)
		}
		if outBuf.Len() != 0 {
			t.Errorf("stdout = %q, want empty", outBuf.String())
		}
		if errBuf.Len() != 0 {
			t.Errorf("stderr = %q, want empty", errBuf.String())
		}
	})

	t.Run("nonexistent file, user declines", func(t *testing.T) {
		t.Parallel()
		tmp := t.TempDir()
		path := filepath.Join(tmp, "noexist")

		inBuf := bytes.NewBufferString("n\n")
		outBuf := &bytes.Buffer{}
		errBuf := &bytes.Buffer{}
		ios := newIOStream(inBuf, outBuf, errBuf)

		got, err := ios.AskToCreate(path)
		if got != false {
			t.Errorf("AskToCreate(%q) decline = %v, want false", path, got)
		}
		if err != nil {
			t.Errorf("AskToCreate(%q) decline error = %v, want nil", path, err)
		}
		wantPrompt := fmt.Sprintf("%s does not exist. Create it? [yn]:", path)
		if errBuf.String() != wantPrompt {
			t.Errorf("stderr = %q, want %q", errBuf.String(), wantPrompt)
		}
		if outBuf.Len() != 0 {
			t.Errorf("stdout = %q, want empty", outBuf.String())
		}
	})

	t.Run("nonexistent file, user accepts", func(t *testing.T) {
		t.Parallel()
		tmp := t.TempDir()
		path := filepath.Join(tmp, "noexist2")

		inBuf := bytes.NewBufferString("y\n")
		outBuf := &bytes.Buffer{}
		errBuf := &bytes.Buffer{}
		ios := newIOStream(inBuf, outBuf, errBuf)

		got, err := ios.AskToCreate(path)
		if got != true {
			t.Errorf("AskToCreate(%q) accept = %v, want true", path, got)
		}
		if err != nil {
			t.Errorf("AskToCreate(%q) accept error = %v, want nil", path, err)
		}
		wantPrompt := fmt.Sprintf("%s does not exist. Create it? [yn]:", path)
		if errBuf.String() != wantPrompt {
			t.Errorf("stderr = %q, want %q", errBuf.String(), wantPrompt)
		}
		if outBuf.Len() != 0 {
			t.Errorf("stdout = %q, want empty", outBuf.String())
		}
	})
}

func TestIOStream_LoopYesNo(t *testing.T) {
	cases := []struct {
		name      string
		input     string
		want      bool
		wantCount int
	}{
		{
			name:      "yes first try",
			input:     "y\n",
			want:      true,
			wantCount: 1,
		},
		{
			name:      "no first try",
			input:     "n\n",
			want:      false,
			wantCount: 1,
		},
		{
			name:      "invalid then no",
			input:     "maybe\nn\n",
			want:      false,
			wantCount: 2,
		},
	}

	for _, tt := range cases {
		// Create per-iteration copy of test tt so that running
		// tests in parallel does not reuse the same test for
		// each run.
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			inBuf := bytes.NewBufferString(tc.input)
			errBuf := &bytes.Buffer{}
			ios := newIOStream(inBuf, io.Discard, errBuf)

			got, err := ios.LoopYesNo("Proceed?")
			if err != nil {
				t.Fatalf("LoopYesNo() error = %v, want nil", err)
			}
			if got != tc.want {
				t.Errorf("LoopYesNo() = %v, want %v", got, tc.want)
			}

			prompt := "Proceed? [yn]:"
			if count := strings.Count(errBuf.String(), prompt); count != tc.wantCount {
				t.Errorf("prompt count = %d, want %d", count, tc.wantCount)
			}
		})
	}
}

func Test_CreateIfNotExists(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "empty path",
			args: args{
				path: "",
			},
			wantErr: true,
		},
		{
			name: "create new file",
			args: args{
				path: "/tmp/newfile",
			},
			wantErr: false,
		},
		{
			name: "already exists",
			args: args{
				path: "/tmp/newfile",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CreateIfNotExists(tt.args.path); (err != nil) != tt.wantErr {
				t.Errorf("CreateIfNotExists() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
