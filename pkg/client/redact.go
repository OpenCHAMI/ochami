// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package client

import (
	"net/http"
	"strings"
)

// tokenPrefixLen is the number of leading characters of a token to keep when
// truncating it for logs.
const tokenPrefixLen = 6

// RedactToken returns token unchanged if show is true. Otherwise, it returns a
// truncated form of the token that still indicates a token exists: the first
// tokenPrefixLen characters followed by an ellipsis (e.g. "eyJhbG..."). An empty
// token returns an empty string. Tokens that are tokenPrefixLen characters or
// shorter are fully masked (returned as just "...") to avoid revealing the
// entire value.
func RedactToken(token string, show bool) string {
	if show || token == "" {
		return token
	}
	if len(token) <= tokenPrefixLen {
		return "..."
	}
	return token[:tokenPrefixLen] + "..."
}

// redactAuthHeaderValues returns a copy of the given Authorization header values
// with any bearer token redacted via RedactToken when show is false. Values of
// the form "Bearer <token>" have their token portion truncated; other values
// are truncated wholesale. When show is true, the values are returned
// unchanged.
func redactAuthHeaderValues(vals []string, show bool) []string {
	if show {
		return vals
	}
	out := make([]string, len(vals))
	for i, v := range vals {
		const bearerPrefix = "Bearer "
		if strings.HasPrefix(v, bearerPrefix) {
			out[i] = bearerPrefix + RedactToken(strings.TrimPrefix(v, bearerPrefix), show)
		} else {
			out[i] = RedactToken(v, show)
		}
	}
	return out
}

// isAuthorizationHeader reports whether the given header key is the
// Authorization header, using canonical header comparison.
func isAuthorizationHeader(key string) bool {
	return http.CanonicalHeaderKey(key) == "Authorization"
}
