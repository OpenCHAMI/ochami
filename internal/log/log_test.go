// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package log

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"
)

// TestInit_Table verifies that Init accepts a supported log level, format, and
// color setting and rejects an unsupported one.
func TestInit_Table(t *testing.T) {
	type args struct {
		ll string
		lf string
		lc string
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
				lc: "auto",
			},
			wantErr: false,
		},
		{
			name: "unsupported level and supported format",
			args: args{
				ll: "unsupported",
				lf: "basic",
				lc: "auto",
			},
			wantErr: true,
		},
		{
			name: "supported level and unsupported format",
			args: args{
				ll: "warning",
				lf: "unsupported",
				lc: "auto",
			},
			wantErr: true,
		},
		{
			name: "unsupported level and unsupported format",
			args: args{
				ll: "unsupported",
				lf: "unsupported",
				lc: "auto",
			},
			wantErr: true,
		},
		{
			name: "supported level and format, unsupported color",
			args: args{
				ll: "warning",
				lf: "basic",
				lc: "unsupported",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Init(tt.args.ll, tt.args.lf, tt.args.lc); (err != nil) != tt.wantErr {
				t.Errorf("Init() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestNewBasicLogger verifies that NewBasicLogger returns a logger with the
// given prefix and verbosity and writes nothing when it is created.
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

// TestBasicLogger_BasicLog verifies that BasicLog writes its arguments, with
// the logger's prefix, only when verbose output is on.
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

// TestBasicLogger_BasicLogf verifies that BasicLogf writes its formatted
// message, with the logger's prefix, only when verbose output is on.
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

// TestNewDefault verifies the pre-initialization logger writes plain,
// program-prefixed lines at warning level and above.
func TestNewDefault(t *testing.T) {
	var buf bytes.Buffer
	logger := NewDefault(&buf)
	logger.Info().Msg("hidden")
	logger.Error().Err(errors.New("boom")).Msg("failed to execute command")

	got := buf.String()
	if strings.Contains(got, "hidden") {
		t.Errorf("output %q contains info-level message, want warning and above only", got)
	}
	if !strings.HasPrefix(got, "ochami: failed to execute command") || !strings.Contains(got, "boom") {
		t.Errorf("output = %q, want plain \"ochami: failed to execute command ... boom\" line", got)
	}
	if strings.Contains(got, "{") {
		t.Errorf("output = %q, want non-JSON output", got)
	}
}
