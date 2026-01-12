// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package config

import (
	"fmt"
)

// ErrInvalidConfigVal represents an error that occurs when a value for a key in
// the configuration is invalid.
type ErrInvalidConfigVal struct {
	Key      string
	Value    string
	Expected string
	Line     int
}

func (eicv ErrInvalidConfigVal) Error() string {
	return fmt.Sprintf("line %d: invalid value for key %q: got %s but expected %s", eicv.Line, eicv.Key, eicv.Value, eicv.Expected)
}

// ErrUnknownCluster represents an error that occurs when a requested cluster
// name is not found in the config.
type ErrUnknownCluster struct {
	ClusterName string
}

func (euc ErrUnknownCluster) Error() string {
	return fmt.Sprintf("cluster %s not found", euc.ClusterName)
}

// ErrMissingURI represents an error that occurs when neither the cluster.uri
// nor the <service>.uri config values are set for a service. Service is the
// name of the service whose config value is being checked.
type ErrMissingURI struct {
	Service ServiceName
}

func (emu ErrMissingURI) Error() string {
	return fmt.Sprintf("base URI for %s not found (neither cluster.uri nor %s.uri specified)", emu.Service, emu.Service)
}

// ErrInvalidURI represents an error that occurs when the cluster URI is
// invalid, i.e. is not a valid absolute URI (proto://host[:port][/path]). Err
// contains the specific error representing the problem.
type ErrInvalidURI struct {
	Err error
}

func (eiu ErrInvalidURI) Error() string {
	return fmt.Sprintf("invalid URI: %v", eiu.Err)
}

// ErrInvalidServiceURI represents an error that occurs when a service's URI is
// invalid, i.e. is neither a valid absolute URI (proto://host[:port][/path])
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
