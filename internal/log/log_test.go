// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package log

import (
	"bytes"
	"reflect"
	"testing"
)

func TestInit(t *testing.T) {
	type args struct {
		ll string
		lf string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "supported level and format",
			args: args{
				ll: "warning",
				lf: "basic",
			},
			wantErr: false,
		},
		{
			name: "unsupported level and supported format",
			args: args{
				ll: "unsupported",
				lf: "basic",
			},
			wantErr: true,
		},
		{
			name: "supported level and unsupported format",
			args: args{
				ll: "warning",
				lf: "unsupported",
			},
			wantErr: true,
		},
		{
			name: "unsupported level and unsupported format",
			args: args{
				ll: "unsupported",
				lf: "unsupported",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Init(tt.args.ll, tt.args.lf); (err != nil) != tt.wantErr {
				t.Errorf("Init() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewBasicLogger(t *testing.T) {
	type args struct {
		prefix  string
		verbose bool
	}
	tests := []struct {
		name    string
		args    args
		want    BasicLogger
		wantOut string
	}{
		{
			name: "empty prefix and verbose on",
			args: args{
				prefix:  "",
				verbose: true,
			},
			want: BasicLogger{
				EarlyVerbose: true,
				out:          &bytes.Buffer{},
				prefix:       "",
			},
			wantOut: "",
		},
		{
			name: "empty prefix and verbose off",
			args: args{
				prefix:  "",
				verbose: false,
			},
			want: BasicLogger{
				EarlyVerbose: false,
				out:          &bytes.Buffer{},
				prefix:       "",
			},
			wantOut: "",
		},
		{
			name: "non-empty prefix and verbose on",
			args: args{
				prefix:  "ochami",
				verbose: true,
			},
			want: BasicLogger{
				EarlyVerbose: true,
				out:          &bytes.Buffer{},
				prefix:       "ochami",
			},
			wantOut: "",
		},
		{
			name: "non-empty prefix and verbose off",
			args: args{
				prefix:  "ochami",
				verbose: false,
			},
			want: BasicLogger{
				EarlyVerbose: false,
				out:          &bytes.Buffer{},
				prefix:       "ochami",
			},
			wantOut: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := &bytes.Buffer{}
			if got := NewBasicLogger(out, tt.args.verbose, tt.args.prefix); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewBasicLogger() = %v, want %v", got, tt.want)
			}
			if gotOut := out.String(); gotOut != tt.wantOut {
				t.Errorf("NewBasicLogger() = %v, want %v", gotOut, tt.wantOut)
			}
		})
	}
}

func TestBasicLogger_BasicLog(t *testing.T) {
	type fields struct {
		prefix string
	}
	type args struct {
		arg     []interface{}
		verbose bool
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []byte
	}{
		{
			name: "verbose off, no prefix, no args",
			fields: fields{
				prefix: "",
			},
			args: args{
				arg:     []interface{}{},
				verbose: false,
			},
			want: nil,
		},
		{
			name: "verbose off, no prefix, with args",
			fields: fields{
				prefix: "",
			},
			args: args{
				arg:     []interface{}{"hello", 42},
				verbose: false,
			},
			want: nil,
		},
		{
			name: "verbose off, with prefix, no args",
			fields: fields{
				prefix: "ochami",
			},
			args: args{
				arg:     []interface{}{},
				verbose: false,
			},
			want: nil,
		},
		{
			name: "verbose off, with prefix, with args",
			fields: fields{
				prefix: "ochami",
			},
			args: args{
				arg:     []interface{}{"world", 99},
				verbose: false,
			},
			want: nil,
		},
		{
			name: "verbose on, no prefix, no args",
			fields: fields{
				prefix: "",
			},
			args: args{
				arg:     []interface{}{},
				verbose: true,
			},
			want: []byte("\n"),
		},
		{
			name: "verbose on, no prefix, with args",
			fields: fields{
				prefix: "",
			},
			args: args{
				arg:     []interface{}{"hello", 42},
				verbose: true,
			},
			want: []byte("hello 42\n"),
		},
		{
			name: "verbose on, with prefix, no args",
			fields: fields{
				prefix: "ochami",
			},
			args: args{
				arg:     []interface{}{},
				verbose: true,
			},
			want: []byte("ochami: \n"),
		},
		{
			name: "verbose on, with prefix, with args",
			fields: fields{
				prefix: "ochami",
			},
			args: args{
				arg:     []interface{}{"world", 99},
				verbose: true,
			},
			want: []byte("ochami: world 99\n"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			el := BasicLogger{
				EarlyVerbose: tt.args.verbose,
				out:          buf,
				prefix:       tt.fields.prefix,
			}
			el.BasicLog(tt.args.arg...)
			outBytes := buf.Bytes()
			if !reflect.DeepEqual(tt.want, outBytes) {
				t.Errorf("BasicLog() = %v, want %v", outBytes, tt.want)
			}
		})
	}
}

func TestBasicLogger_BasicLogf(t *testing.T) {
	type fields struct {
		prefix string
	}
	type args struct {
		fstr    string
		arg     []interface{}
		verbose bool
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []byte
	}{
		{
			name: "verbose off, no prefix, no format args",
			fields: fields{
				prefix: "",
			},
			args: args{
				fstr:    "msg",
				arg:     []interface{}{},
				verbose: false,
			},
			want: nil,
		},
		{
			name: "verbose off, no prefix, with format args",
			fields: fields{
				prefix: "",
			},
			args: args{
				fstr:    "val=%d",
				arg:     []interface{}{7},
				verbose: false,
			},
			want: nil,
		},
		{
			name: "verbose off, with prefix, no format args",
			fields: fields{
				prefix: "ochami",
			},
			args: args{
				fstr:    "hello",
				arg:     []interface{}{},
				verbose: false,
			},
			want: nil,
		},
		{
			name: "verbose off, with prefix, with format args",
			fields: fields{
				prefix: "ochami",
			},
			args: args{
				fstr:    "%s-%d",
				arg:     []interface{}{"x", 5},
				verbose: false,
			},
			want: nil,
		},
		{
			name: "verbose on, no prefix, no format args",
			fields: fields{
				prefix: "",
			},
			args: args{
				fstr:    "msg",
				arg:     []interface{}{},
				verbose: true,
			},
			want: []byte("msg\n"),
		},
		{
			name: "verbose on, no prefix, with format args",
			fields: fields{
				prefix: "",
			},
			args: args{
				fstr:    "val=%d",
				arg:     []interface{}{7},
				verbose: true,
			},
			want: []byte("val=7\n"),
		},
		{
			name: "verbose off, with prefix, no format args",
			fields: fields{
				prefix: "ochami",
			},
			args: args{
				fstr:    "hello",
				arg:     []interface{}{},
				verbose: true,
			},
			want: []byte("ochami: hello\n"),
		},
		{
			name: "verbose off, with prefix, with format args",
			fields: fields{
				prefix: "ochami",
			},
			args: args{
				fstr:    "%s-%d",
				arg:     []interface{}{"x", 5},
				verbose: true,
			},
			want: []byte("ochami: x-5\n"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			el := BasicLogger{
				EarlyVerbose: tt.args.verbose,
				out:          buf,
				prefix:       tt.fields.prefix,
			}
			el.BasicLogf(tt.args.fstr, tt.args.arg...)
			outBytes := buf.Bytes()
			if !reflect.DeepEqual(tt.want, outBytes) {
				t.Errorf("BasicLogf() = %v, want %v", outBytes, tt.want)
			}
		})
	}
}
