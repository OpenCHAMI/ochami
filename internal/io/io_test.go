// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package io

import (
	"io"
	"reflect"
	"strings"
	"testing"
)

func TestIOReader_readIn(t *testing.T) {
	type fields struct {
		in io.Reader
	}
	tests := []struct {
		name    string
		fields  fields
		want    []byte
		wantErr bool
	}{
		{
			name: "read in text",
			fields: fields{
				in: strings.NewReader("test"),
			},
			want:    []byte("test\n"),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ior := ioReader{
				in: tt.fields.in,
			}
			got, err := ior.readIn()
			if (err != nil) != tt.wantErr {
				t.Errorf("ioReader.readIn() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ioReader.readIn() = %v, want %v", got, tt.want)
			}
		})
	}
}
