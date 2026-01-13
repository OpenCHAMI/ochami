// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package io

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

// ReadStdin reads all of standard input and returns the bytes. If an error
// occurs during scanning, it is returned.
func ReadStdin() ([]byte, error) {
	ior := newIOReader(os.Stdin)
	return ior.readIn()
}

// ioReader stores an io.Reader to be read from using readIn. It's purpose is to
// abstract reading from any io.Reader, specifically for unit testing.
type ioReader struct {
	in io.Reader
}

// newIOReader creates a new ioReader containing the io.Reader to be read from.
func newIOReader(in io.Reader) ioReader {
	return ioReader{
		in: in,
	}
}

// readIn reads the contents of the ioReader's io.Reader and returns the bytes.
// Line buffering is assumed and bufio.Scanner.Scan() is used iteratively to
// scan lines, appending a newline character at the end of the returned bytes.
// If an error occurs during scanning, it is returned.
//
// This function is meant to allow the invoker to read from anything
// implementing io.Reader (e.g. os.Stdin).
func (ior ioReader) readIn() ([]byte, error) {
	var b []byte
	input := bufio.NewScanner(ior.in)
	for input.Scan() {
		b = append(b, input.Bytes()...)
		b = append(b, byte('\n'))
	}
	if err := input.Err(); err != nil {
		return b, fmt.Errorf("failed to read stdin: %w", err)
	}
	return b, nil
}
