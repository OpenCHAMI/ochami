// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

// Package conventions holds no code. Its tests parse the module's Go
// sources and the ochami(1) man page and fail when they break a convention
// from CONTRIBUTING.md or a documented API contract: test names, files, and
// doc comments; exit code assertions and messages; the man page's exit code
// table; and the pkg/config API's independence from koanf.
//
// The checks run with the rest of the tests (make test) or alone:
//
//	go test ./internal/conventions
package conventions
