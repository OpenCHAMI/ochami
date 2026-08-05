// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package client

import (
	"net/http"
	"strings"
)

// ShowToken controls whether access tokens are shown in full in debug logs. It
// defaults to false, meaning tokens are truncated (see RedactToken). The CLI
// layer sets this from the --show-token flag before any request logging occurs.
var ShowToken bool

// tokenPrefixLen is the number of leading characters of a token to keep when
// truncating it for logs.
const tokenPrefixLen = 6

// RedactToken returns token unchanged if ShowToken is true. Otherwise, it
// returns a truncated form of the token that still indicates a token exists: the
// first tokenPrefixLen characters followed by an ellipsis (e.g. "eyJhbG..."). An
// empty token returns an empty string. Tokens that are tokenPrefixLen
// characters or shorter are fully masked (returned as just "...") to avoid
// revealing the entire value.
func RedactToken(token string) string {
	if ShowToken || token == "" {
		return token
	}
	if len(token) <= tokenPrefixLen {
		return "..."
	}
	return token[:tokenPrefixLen] + "..."
}

// redactAuthHeaderValues returns a copy of the given Authorization header values
// with any bearer token redacted via RedactToken when ShowToken is false. Values
// of the form "Bearer <token>" have their token portion truncated; other values
// are truncated wholesale. When ShowToken is true, the values are returned
// unchanged.
func redactAuthHeaderValues(vals []string) []string {
	if ShowToken {
		return vals
	}
	out := make([]string, len(vals))
	for i, v := range vals {
		const bearerPrefix = "Bearer "
		if strings.HasPrefix(v, bearerPrefix) {
			out[i] = bearerPrefix + RedactToken(strings.TrimPrefix(v, bearerPrefix))
		} else {
			out[i] = RedactToken(v)
		}
	}
	return out
}

// isAuthorizationHeader reports whether the given header key is the
// Authorization header, using canonical header comparison.
func isAuthorizationHeader(key string) bool {
	return http.CanonicalHeaderKey(key) == "Authorization"
}
