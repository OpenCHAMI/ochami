// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package client

import (
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/OpenCHAMI/ochami/pkg/format"
)

func TestNewHTTPHeaders(t *testing.T) {
	tests := []struct {
		name string
		want *HTTPHeaders
	}{
		{
			name: "empty headers",
			want: &HTTPHeaders{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewHTTPHeaders(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewHTTPHeaders() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHTTPHeaders_Add(t *testing.T) {
	type args struct {
		key   string
		value string
	}
	tests := []struct {
		name    string
		h       *HTTPHeaders
		args    args
		wantErr bool
	}{
		{
			name:    "nil receiver",
			h:       nil,
			args:    args{key: "K", value: "V"},
			wantErr: true,
		},
		{
			name:    "first add",
			h:       NewHTTPHeaders(),
			args:    args{key: "A", value: "first"},
			wantErr: false,
		},
		{
			name: "append value",
			h: func() *HTTPHeaders {
				h := NewHTTPHeaders()
				_ = h.Add("A", "first")
				return h
			}(),
			args:    args{key: "A", value: "second"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.h.Add(tt.args.key, tt.args.value); (err != nil) != tt.wantErr {
				t.Errorf("HTTPHeaders.Add() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHTTPHeaders_SetAuthorization(t *testing.T) {
	type args struct {
		token string
	}
	tests := []struct {
		name    string
		h       *HTTPHeaders
		args    args
		wantErr bool
	}{
		{
			name:    "nil receiver",
			h:       nil,
			args:    args{token: "tok"},
			wantErr: true,
		},
		{
			name:    "set auth header",
			h:       NewHTTPHeaders(),
			args:    args{token: "tok"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.h.SetAuthorization(tt.args.token); (err != nil) != tt.wantErr {
				t.Errorf("HTTPHeaders.SetAuthorization() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHTTPHeaders_SetContentType(t *testing.T) {
	type args struct {
		ct string
	}
	tests := []struct {
		name    string
		h       *HTTPHeaders
		args    args
		wantErr bool
	}{
		{
			name:    "nil receiver",
			h:       nil,
			args:    args{ct: "text/plain"},
			wantErr: true,
		},
		{
			name:    "set content type header",
			h:       NewHTTPHeaders(),
			args:    args{ct: "text/plain"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.h.SetContentType(tt.args.ct); (err != nil) != tt.wantErr {
				t.Errorf("HTTPHeaders.SetContentType() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewHTTPEnvelopeFromResponse(t *testing.T) {
	type args struct {
		res *http.Response
	}
	tests := []struct {
		name    string
		args    args
		want    HTTPEnvelope
		wantErr bool
	}{
		{
			name:    "nil response",
			args:    args{res: nil},
			want:    HTTPEnvelope{},
			wantErr: true,
		},
		{
			name: "successful envelope",
			args: args{
				res: &http.Response{
					Status:     "200 OK",
					StatusCode: 200,
					Proto:      "HTTP/1.1",
					Header:     http.Header{"Foo": {"bar"}},
					Body:       io.NopCloser(strings.NewReader("payload")),
				},
			},
			want: HTTPEnvelope{
				Status:     "200 OK",
				StatusCode: 200,
				Proto:      "HTTP/1.1",
				Headers:    &HTTPHeaders{"Foo": {"bar"}},
				Body:       HTTPBody("payload"),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewHTTPEnvelopeFromResponse(tt.args.res)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewHTTPEnvelopeFromResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewHTTPEnvelopeFromResponse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatBody(t *testing.T) {
	type args struct {
		body      HTTPBody
		outFormat format.DataFormat
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name:    "invalid json",
			args:    args{body: HTTPBody("not json"), outFormat: format.DataFormatJson},
			want:    nil,
			wantErr: true,
		},
		{
			name: "valid json",
			args: args{
				body:      HTTPBody(`{"foo":42}`),
				outFormat: format.DataFormatJson,
			},
			want:    []byte(`{"foo":42}`),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FormatBody(tt.args.body, tt.args.outFormat)
			if (err != nil) != tt.wantErr {
				t.Errorf("FormatBody() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FormatBody() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHTTPEnvelope_CheckResponse(t *testing.T) {
	type fields struct {
		Status     string
		StatusCode int
		Proto      string
		Headers    *HTTPHeaders
		Body       HTTPBody
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name:    "2xx success",
			fields:  fields{StatusCode: 204},
			wantErr: false,
		},
		{
			name: "4xx without body",
			fields: fields{
				StatusCode: 404,
				Proto:      "HTTP/1.1",
				Status:     "404 Not Found",
				Body:       HTTPBody(""),
			},
			wantErr: true,
		},
		{
			name: "5xx with body",
			fields: fields{
				StatusCode: 500,
				Proto:      "HTTP/1.1",
				Status:     "500 Internal Error",
				Body:       HTTPBody("oops"),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			he := HTTPEnvelope{
				Status:     tt.fields.Status,
				StatusCode: tt.fields.StatusCode,
				Proto:      tt.fields.Proto,
				Headers:    tt.fields.Headers,
				Body:       tt.fields.Body,
			}
			if err := he.CheckResponse(); (err != nil) != tt.wantErr {
				t.Errorf("HTTPEnvelope.CheckResponse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
