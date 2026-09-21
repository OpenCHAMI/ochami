// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package client

import (
	"errors"
	"io"
	"testing"

	"github.com/openchami/ochami/pkg/format"
)

// TestReadPayloadWithReader_Errors verifies that the reader-based payload
// helpers return the reader's error and reject "@-" with no reader.
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
