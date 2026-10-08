// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package client

import (
	"errors"
	"strings"
	"testing"

	"github.com/openchami/ochami/pkg/format"
)

type payloadItem struct {
	Name string `json:"name"`
}

type errorReader struct{ err error }

func (r errorReader) Read([]byte) (int, error) { return 0, r.err }

// TestReadPayloadWithReader_Success verifies that ReadPayloadWithReader reads
// "@-" from the given reader and parses an inline payload without reading it.
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

// TestReadPayloadSliceWithReader verifies that ReadPayloadSliceWithReader reads
// "@-" from the given reader.
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
