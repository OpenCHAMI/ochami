// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package format

import (
	"fmt"
	"reflect"
	"testing"
)

func TestDataFormat_String(t *testing.T) {
	tests := []struct {
		name string
		df   DataFormat
		want string
	}{
		{name: "json", df: DataFormatJson, want: "json"},
		{name: "json-pretty", df: DataFormatJsonPretty, want: "json-pretty"},
		{name: "yaml", df: DataFormatYaml, want: "yaml"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.df.String(); got != tt.want {
				t.Errorf("DataFormat.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDataFormat_Set(t *testing.T) {
	type args struct {
		v string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "json", args: args{v: "json"}, wantErr: false},
		{name: "json-pretty", args: args{v: "json-pretty"}, wantErr: false},
		{name: "yaml", args: args{v: "yaml"}, wantErr: false},
		{name: "unsupported", args: args{v: "unsupported"}, wantErr: true},
	}
	for _, tt := range tests {
		df := DataFormatJson // Start with dummy value for instantiation
		t.Run(tt.name, func(t *testing.T) {
			if err := df.Set(tt.args.v); (err != nil) != tt.wantErr {
				t.Errorf("DataFormat.Set() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDataFormat_Type(t *testing.T) {
	tests := []struct {
		name string
		df   DataFormat
		want string
	}{
		{name: "json", df: DataFormatJson, want: "DataFormat"},
		{name: "json-pretty", df: DataFormatJsonPretty, want: "DataFormat"},
		{name: "yaml", df: DataFormatYaml, want: "DataFormat"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.df.Type(); got != tt.want {
				t.Errorf("DataFormat.Type() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMarshalData(t *testing.T) {
	type args struct {
		data      interface{}
		outFormat DataFormat
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "valid json",
			args: args{
				data: struct {
					Key string `json:"key" yaml:"key"`
					Arr []int  `json:"arr" yaml:"arr"`
				}{
					Key: "value",
					Arr: []int{1, 2},
				},
				outFormat: DataFormatJson,
			},
			want:    []byte(`{"key":"value","arr":[1,2]}`),
			wantErr: false,
		},
		{
			name: "valid pretty json",
			args: args{
				data: struct {
					Key string `json:"key" yaml:"key"`
					Arr []int  `json:"arr" yaml:"arr"`
				}{
					Key: "value",
					Arr: []int{1, 2},
				},
				outFormat: DataFormatJsonPretty,
			},
			want: []byte(`{
  "key": "value",
  "arr": [
    1,
    2
  ]
}`),
			wantErr: false,
		},
		{
			name: "valid yaml",
			args: args{
				data: struct {
					Key string `json:"key" yaml:"key"`
					Arr []int  `json:"arr" yaml:"arr"`
				}{
					Key: "value",
					Arr: []int{1, 2},
				},
				outFormat: DataFormatYaml,
			},
			want: []byte(`key: value
arr:
    - 1
    - 2
`),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MarshalData(tt.args.data, tt.args.outFormat)
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalData() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MarshalData() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUnmarshalData(t *testing.T) {
	type args struct {
		data     []byte
		inFormat DataFormat
	}
	tests := []struct {
		name    string
		want    map[string]interface{}
		args    args
		wantErr bool
	}{
		{
			name: "valid json",
			want: map[string]interface{}{
				"key": "value",
				"arr": []int{1, 2},
			},
			args: args{
				data:     []byte(`{"key":"value","arr":[1,2]}`),
				inFormat: DataFormatJson,
			},
			wantErr: false,
		},
		{
			name: "valid pretty json",
			want: map[string]interface{}{
				"key": "value",
				"arr": []int{1, 2},
			},
			args: args{
				data: []byte(`{
  "key": "value",
  "arr": [
    1,
    2
  ]
}`),
				inFormat: DataFormatJsonPretty,
			},
			wantErr: false,
		},
		{
			name: "valid yaml",
			want: map[string]interface{}{
				"key": "value",
				"arr": []int{1, 2},
			},
			args: args{
				data: []byte(`key: value
arr:
    - 1
    - 2
`),
				inFormat: DataFormatYaml,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got map[string]interface{}
			if err := UnmarshalData(tt.args.data, &got, tt.args.inFormat); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalData() error = %v, wantErr %v", err, tt.wantErr)
			}
			// For some reason, reflect.DeepEqual returns false even when the
			// maps are equal, so we compare the string representations instead.
			if fmt.Sprint(got) != fmt.Sprint(tt.want) {
				t.Errorf("MarshalData() = %v, want %v", got, tt.want)
			}
		})
	}
}
