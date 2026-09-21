// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package client

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/openchami/ochami/pkg/format"
)

type payloadItem struct {
	Name string `json:"name"`
}

type errorReader struct{ err error }

func (r errorReader) Read([]byte) (int, error) { return 0, r.err }

func TestReadPayloadWithReader_Success(t *testing.T) {
	t.Parallel()

	var got payloadItem
	if err := ReadPayloadWithReader("@-", strings.NewReader(`{"name":"stdin"}`), format.DataFormatJson, &got); err != nil {
		t.Fatalf("ReadPayloadWithReader() error = %v", err)
	}
	if got.Name != "stdin" {
		t.Fatalf("ReadPayloadWithReader() = %#v, want stdin payload", got)
	}

	got = payloadItem{}
	if err := ReadPayloadWithReader(`{"name":"inline"}`, errorReader{err: errors.New("must not read")}, format.DataFormatJson, &got); err != nil {
		t.Fatalf("inline ReadPayloadWithReader() error = %v", err)
	}
	if got.Name != "inline" {
		t.Fatalf("inline ReadPayloadWithReader() = %#v, want inline payload", got)
	}
}

func TestReadPayloadSliceWithReader(t *testing.T) {
	t.Parallel()

	var got []payloadItem
	if err := ReadPayloadSliceWithReader[payloadItem]("@-", strings.NewReader(`[{"name":"a"},{"name":"b"}]`), format.DataFormatJson, &got); err != nil {
		t.Fatalf("ReadPayloadSliceWithReader() error = %v", err)
	}
	if len(got) != 2 || got[0].Name != "a" || got[1].Name != "b" {
		t.Fatalf("ReadPayloadSliceWithReader() = %#v", got)
	}
}

func TestReadPayloadWithReader_Errors(t *testing.T) {
	t.Parallel()

	want := errors.New("read failed")
	var scalar payloadItem
	if err := ReadPayloadWithReader("@-", errorReader{err: want}, format.DataFormatJson, &scalar); !errors.Is(err, want) {
		t.Fatalf("ReadPayloadWithReader() error = %v, want wrapped %v", err, want)
	}
	if err := ReadPayloadWithReader("@-", nil, format.DataFormatJson, &scalar); err == nil {
		t.Fatal("ReadPayloadWithReader() with nil stdin returned nil error")
	}

	var slice []payloadItem
	if err := ReadPayloadSliceWithReader[payloadItem]("@-", errorReader{err: io.ErrUnexpectedEOF}, format.DataFormatJson, &slice); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("ReadPayloadSliceWithReader() error = %v, want wrapped %v", err, io.ErrUnexpectedEOF)
	}
}
