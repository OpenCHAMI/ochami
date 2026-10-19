// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package config

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// TestErrInvalidConfigVal_Error verifies ErrInvalidConfigVal's message,
// including the line number only when it is known.
func TestErrInvalidConfigVal_Error(t *testing.T) {
	tests := []struct {
		name string
		err  ErrInvalidConfigVal
		want string
	}{
		{
			name: "known line is included",
			err: ErrInvalidConfigVal{
				Key:      "enable-auth",
				Value:    "empty string",
				Expected: "true or false",
				Line:     1,
			},
			want: `line 1: invalid value for key "enable-auth": got empty string but expected true or false`,
		},
		{
			name: "unknown line is omitted",
			err: ErrInvalidConfigVal{
				Key:      "enable-auth",
				Value:    "null",
				Expected: "boolean",
			},
			want: `invalid value for key "enable-auth": got null but expected boolean`,
		},
		{
			name: "negative line is omitted",
			err: ErrInvalidConfigVal{
				Key:      "timeout",
				Value:    "invalid",
				Expected: "duration",
				Line:     -1,
			},
			want: `invalid value for key "timeout": got invalid but expected duration`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("ErrInvalidConfigVal.Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestErrUnknownCluster_Error verifies that ErrUnknownCluster's message names
// the cluster.
func TestErrUnknownCluster_Error(t *testing.T) {
	type fields struct {
		ClusterName string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "cluster name contained in error",
			fields: fields{
				ClusterName: "test_cluster",
			},
			want: "test_cluster",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			euc := ErrUnknownCluster{
				ClusterName: tt.fields.ClusterName,
			}
			if got := euc.Error(); !strings.Contains(got, tt.want) {
				t.Errorf("ErrUnknownCluster.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestErrMissingURI_Error verifies that ErrMissingURI's message names the
// service and the cluster.uri and <service>.uri keys either of which would
// provide it.
func TestErrMissingURI_Error(t *testing.T) {
	type fields struct {
		Service ServiceName
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "valid error",
			fields: fields{
				Service: ServiceBSS,
			},
			want: fmt.Sprintf("base URI for %s not found (neither cluster.uri nor %s.uri specified)", ServiceBSS, ServiceBSS),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			emu := ErrMissingURI{
				Service: tt.fields.Service,
			}
			if got := emu.Error(); got != tt.want {
				t.Errorf("ErrMissingURI.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestErrInvalidURI_Error verifies that ErrInvalidURI's message includes the
// underlying error.
func TestErrInvalidURI_Error(t *testing.T) {
	type fields struct {
		Err error
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "valid contained error",
			fields: fields{
				Err: fmt.Errorf("unknown URI format (must be \"proto://host[:port][/path]\")"),
			},
			want: fmt.Sprintf("invalid URI: %v", fmt.Sprintf("unknown URI format (must be \"proto://host[:port][/path]\")")),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eiu := ErrInvalidURI{
				Err: tt.fields.Err,
			}
			if got := eiu.Error(); got != tt.want {
				t.Errorf("ErrInvalidURI.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestErrInvalidServiceURI_Error verifies that ErrInvalidServiceURI's message
// names the service and includes the underlying error.
func TestErrInvalidServiceURI_Error(t *testing.T) {
	type fields struct {
		Err     error
		Service ServiceName
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "valid contained error and service",
			fields: fields{
				Err:     fmt.Errorf("unknown URI format (must be \"proto://host[:port][/path]\")"),
				Service: ServiceBSS,
			},
			want: fmt.Sprintf("invalid service URI for %s: %v", ServiceBSS, fmt.Sprintf("unknown URI format (must be \"proto://host[:port][/path]\")")),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eisu := ErrInvalidServiceURI{
				Err:     tt.fields.Err,
				Service: tt.fields.Service,
			}
			if got := eisu.Error(); got != tt.want {
				t.Errorf("ErrInvalidServiceURI.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestErrUnknownService_Error verifies that ErrUnknownService's message names
// the service.
func TestErrUnknownService_Error(t *testing.T) {
	type fields struct {
		Service string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "unknown service",
			fields: fields{
				Service: "unk_svc",
			},
			want: "unknown service: unk_svc",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eus := ErrUnknownService{
				Service: tt.fields.Service,
			}
			if got := eus.Error(); got != tt.want {
				t.Errorf("ErrUnknownService.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestErrDuplicateCluster_Error verifies that ErrDuplicateCluster's message
// names the cluster, how many times it is defined, where (the file path, or
// else the source name), and the positions of its entries, and that it is a
// config Error.
func TestErrDuplicateCluster_Error(t *testing.T) {
	tests := []struct {
		name string
		err  ErrDuplicateCluster
		want string
	}{
		{
			name: "path",
			err:  ErrDuplicateCluster{Source: "user", Path: "/home/u/.config/ochami/config.yaml", Name: "demo", Indices: []int{0, 2}},
			want: `cluster "demo" is defined 2 times in /home/u/.config/ochami/config.yaml (clusters[0], clusters[2]); edit the file to remove or merge the duplicates`,
		},
		{
			name: "source without path",
			err:  ErrDuplicateCluster{Source: "inline", Name: "demo", Indices: []int{1, 3, 4}},
			want: `cluster "demo" is defined 3 times in config 'inline' (clusters[1], clusters[3], clusters[4]); edit the file to remove or merge the duplicates`,
		},
		{
			name: "no location",
			err:  ErrDuplicateCluster{Name: "demo", Indices: []int{0, 1}},
			want: `cluster "demo" is defined 2 times (clusters[0], clusters[1]); edit the file to remove or merge the duplicates`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("ErrDuplicateCluster.Error() = %q, want %q", got, tt.want)
			}
			var cfgErr Error
			if !errors.As(error(tt.err), &cfgErr) {
				t.Error("ErrDuplicateCluster does not implement Error")
			}
		})
	}
}
