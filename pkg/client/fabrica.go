// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package client

import (
	"fmt"
	"regexp"
)

// fabricaHTTPErrorPattern matches the error message shapes produced by the
// HTTP layer that github.com/openchami/fabrica generates for its client
// packages (e.g. github.com/openchami/boot-service/pkg/client,
// github.com/openchami/metadata-service/pkg/client) when a response has an
// HTTP status code >= 400. Those clients expose no typed error or status
// code for these failures, only a plain fmt.Errorf-built message, so pattern
// matching on the message is the only way to distinguish an HTTP-level
// failure from a transport-level one (e.g. connection refused, timeout).
var fabricaHTTPErrorPattern = regexp.MustCompile(`^(?:PATCH )?(?:HTTP error \d+:|API error \(\d+\):)`)

// FabricaWrapHTTPError wraps err with UnsuccessfulHTTPError if err originated
// from a Fabrica-generated service client's response with an HTTP status code
// >= 400, so that callers can classify it the same way as every other service
// client, via errors.Is(err, client.UnsuccessfulHTTPError). If err is nil or
// does not match, it is returned unchanged.
func FabricaWrapHTTPError(err error) error {
	if err == nil || !fabricaHTTPErrorPattern.MatchString(err.Error()) {
		return err
	}
	return fmt.Errorf("%w: %w", UnsuccessfulHTTPError, err)
}
