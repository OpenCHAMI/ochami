// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package config

import (
	"fmt"
	"strings"
	"testing"
)

func TestErrInvalidConfigVal_Error(t *testing.T) {
	type fields struct {
		Key      string
		Value    string
		Expected string
		Line     int
	}
	tests := []struct {
		name   string
		fields fields
		want   fields
	}{
		{
			name: "expected fields contained in error",
			fields: fields{
				Key:      "enable-auth",
				Value:    "empty string",
				Expected: "true or false",
				Line:     1,
			},
			want: fields{
				Key:      "enable-auth",
				Value:    "empty string",
				Expected: "true or false",
				Line:     1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eicv := ErrInvalidConfigVal{
				Key:      tt.fields.Key,
				Value:    tt.fields.Value,
				Expected: tt.fields.Expected,
				Line:     tt.fields.Line,
			}
			got := eicv.Error()
			if !strings.Contains(got, tt.want.Key) {
				t.Errorf("ErrInvalidConfigVal.Error() does not contain expected Key %s, full error was %s", tt.want.Key, got)
			}
			if !strings.Contains(got, tt.want.Value) {
				t.Errorf("ErrInvalidConfigVal.Error() does not contain expected Value %s, full error was %s", tt.want.Value, got)
			}
			if !strings.Contains(got, tt.want.Expected) {
				t.Errorf("ErrInvalidConfigVal.Error() does not contain expected Expected %s, full error was %s", tt.want.Expected, got)
			}
			if !strings.Contains(got, fmt.Sprintf("%d", tt.want.Line)) {
				t.Errorf("ErrInvalidConfigVal.Error() does not contain expected Line %d, full error was %s", tt.want.Line, got)
			}
		})
	}
}

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
