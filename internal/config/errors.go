// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package config

import (
	"fmt"
)

// ErrMissingURI represents an error that occurs when neither the api-uri nor
// the <service>-uri config values are set for a service. Service is the name of
// the service whose config value is being checked.
type ErrMissingURI struct {
	Service ServiceName
}

func (emu ErrMissingURI) Error() string {
	return fmt.Sprintf("base URI for %s not found (neither api-uri nor %s-uri specified)", emu.Service, emu.Service)
}

// ErrInvalidAPIURI represents an error that occurs when the api-uri is invalid,
// i.e. is not a valid absolute URI (proto://host[:port][/path]). Err contains
// the specific error representing the problem.
type ErrInvalidAPIURI struct {
	Err error
}

func (eiu ErrInvalidAPIURI) Error() string {
	return fmt.Sprintf("invalid API URI: %v", eiu.Err)
}

// ErrInvalidServiceURI represents an error that occurs when the <service>-uri
// is invalid, i.e. is neither a valid absolute URI (proto://host[:port][/path])
// nor a valid relative path (/path).
type ErrInvalidServiceURI struct {
	Err     error
	Service ServiceName
}

func (eisu ErrInvalidServiceURI) Error() string {
	return fmt.Sprintf("invalid service URI for %s: %v", eisu.Service, eisu.Err)
}

// ErrUnknownService represents an error that occurs when the service name
// presented is unknown.
type ErrUnknownService struct {
	Service string
}

func (eus ErrUnknownService) Error() string {
	return fmt.Sprintf("unknown service: %s", eus.Service)
}
