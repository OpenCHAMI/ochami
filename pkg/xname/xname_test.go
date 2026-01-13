// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package xname

import (
	"reflect"
	"testing"

	"github.com/openchami/schemas/schemas/csm"
)

func TestXNameComponentsToString(t *testing.T) {
	type args struct {
		x csm.XNameComponents
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "node xname",
			args: args{
				csm.XNameComponents{
					Cabinet:      1000,
					Chassis:      0,
					Slot:         0,
					BMCPosition:  0,
					NodePosition: 0,
					Type:         "n",
				},
			},
			want: "x1000c0s0b0n0",
		},
		{
			name: "bmc xname",
			args: args{
				csm.XNameComponents{
					Cabinet:     1000,
					Chassis:     0,
					Slot:        0,
					BMCPosition: 0,
					Type:        "b",
				},
			},
			want: "x1000c0s0b0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := XNameComponentsToString(tt.args.x); got != tt.want {
				t.Errorf("XNameComponentsToString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStringToXname(t *testing.T) {
	type args struct {
		xname string
	}
	tests := []struct {
		name string
		args args
		want csm.XNameComponents
	}{
		{
			name: "valid node xname",
			args: args{
				xname: "x1000c0s0b0n0",
			},
			want: csm.XNameComponents{
				Cabinet:      1000,
				Chassis:      0,
				Slot:         0,
				BMCPosition:  0,
				NodePosition: 0,
				Type:         "n",
			},
		},
		{
			name: "invalid node xname",
			args: args{
				xname: "x1000c0s0d0n0",
			},
			want: csm.XNameComponents{
				Cabinet:      1000,
				Chassis:      0,
				Slot:         0,
				BMCPosition:  0,
				NodePosition: 0,
				Type:         "",
			},
		},
		{
			name: "valid bmc xname",
			args: args{
				xname: "x1000c0s0b0",
			},
			want: csm.XNameComponents{
				Cabinet:     1000,
				Chassis:     0,
				Slot:        0,
				BMCPosition: 0,
				Type:        "b",
			},
		},
		{
			name: "invalid bmc xname",
			args: args{
				xname: "x1000c0s0b0d0",
			},
			want: csm.XNameComponents{
				Cabinet:      1000,
				Chassis:      0,
				Slot:         0,
				BMCPosition:  0,
				NodePosition: 0,
				Type:         "b",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StringToXname(tt.args.xname); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("StringToXname() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodeXnameToBMCXname(t *testing.T) {
	type args struct {
		xname string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "valid node xname to bmc xname",
			args: args{
				xname: "x1000c0s0b0n0",
			},
			want:    "x1000c0s0b0",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NodeXnameToBMCXname(tt.args.xname)
			if (err != nil) != tt.wantErr {
				t.Errorf("NodeXnameToBMCXname() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("NodeXnameToBMCXname() = %v, want %v", got, tt.want)
			}
		})
	}
}
