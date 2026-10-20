// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package render

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// readExtraVars preserves source bytes so decoding can retain mapping order.
func readExtraVars(value string, stdin io.Reader) ([]byte, error) {
	if !strings.HasPrefix(value, "@") {
		return []byte(value), nil
	}
	path := strings.TrimPrefix(value, "@")
	return readRenderFile(path, stdin)
}

// readRenderFile reads a literal file path or invocation-owned stdin for "-".
func readRenderFile(path string, stdin io.Reader) ([]byte, error) {
	if path == "-" {
		if stdin == nil {
			return nil, fmt.Errorf("stdin reader is nil")
		}
		return io.ReadAll(stdin)
	}
	return os.ReadFile(path)
}
