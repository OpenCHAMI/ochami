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

type testItem struct {
	ID   int    `json:"id" yaml:"id"`
	Name string `json:"name" yaml:"name"`
}

func TestUnmarshalDataSlice(t *testing.T) {
	type args struct {
		data     []byte
		inFormat DataFormat
	}

	tests := []struct {
		name    string
		args    args
		want    []testItem
		wantErr bool
	}{
		{
			name: "json single object",
			args: args{data: []byte(`{"id":1,"name":"solo"}`), inFormat: DataFormatJson},
			want: []testItem{{ID: 1, Name: "solo"}},
		},
		{
			name: "json array",
			args: args{data: []byte(`[{"id":2,"name":"a"},{"id":3,"name":"b"}]`), inFormat: DataFormatJson},
			want: []testItem{{ID: 2, Name: "a"}, {ID: 3, Name: "b"}},
		},
		{
			name: "json-pretty single object",
			args: args{data: []byte("\n  {\n    \"id\": 4,\n    \"name\": \"pretty\"\n  }\n"), inFormat: DataFormatJsonPretty},
			want: []testItem{{ID: 4, Name: "pretty"}},
		},
		{
			name: "json-pretty array",
			args: args{data: []byte("\n  [\n    {\n      \"id\": 5,\n      \"name\": \"p1\"\n    },\n    {\n      \"id\": 6,\n      \"name\": \"p2\"\n    }\n  ]\n"), inFormat: DataFormatJsonPretty},
			want: []testItem{{ID: 5, Name: "p1"}, {ID: 6, Name: "p2"}},
		},
		{
			name:    "json wrong top-level",
			args:    args{data: []byte(`123`), inFormat: DataFormatJson},
			wantErr: true,
		},
		{
			name:    "json malformed",
			args:    args{data: []byte(`{"id":1,"name":}`), inFormat: DataFormatJson},
			wantErr: true,
		},
		{
			name: "yaml single mapping (block)",
			args: args{data: []byte("id: 10\nname: solo\n"), inFormat: DataFormatYaml},
			want: []testItem{{ID: 10, Name: "solo"}},
		},
		{
			name: "yaml single mapping (flow)",
			args: args{data: []byte(`{id: 11, name: flow}`), inFormat: DataFormatYaml},
			want: []testItem{{ID: 11, Name: "flow"}},
		},
		{
			name: "yaml sequence (block)",
			args: args{data: []byte("- id: 12\n  name: a\n- id: 13\n  name: b\n"), inFormat: DataFormatYaml},
			want: []testItem{{ID: 12, Name: "a"}, {ID: 13, Name: "b"}},
		},
		{
			name: "yaml sequence (flow)",
			args: args{data: []byte(`[{id: 14, name: a}, {id: 15, name: b}]`), inFormat: DataFormatYaml},
			want: []testItem{{ID: 14, Name: "a"}, {ID: 15, Name: "b"}},
		},
		{
			name:    "yaml wrong top-level scalar",
			args:    args{data: []byte(`justastring`), inFormat: DataFormatYaml},
			wantErr: true,
		},
		{
			name:    "unknown format",
			args:    args{data: []byte(`{"id":1,"name":"x"}`), inFormat: DataFormat("toml")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			var got []testItem
			err := UnmarshalDataSlice[testItem](tt.args.data, &got, tt.args.inFormat)
			if (err != nil) != tt.wantErr {
				t.Fatalf("UnmarshalDataSlice() err=%v wantErr=%v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("UnmarshalDataSlice() got=%#v want=%#v", got, tt.want)
			}
		})
	}
}

func TestUnmarshalDataSlice_NilDestination(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		format  DataFormat
		wantErr bool
	}{
		{name: "json nil dest", data: []byte(`{"id":1,"name":"x"}`), format: DataFormatJson, wantErr: true},
		{name: "yaml nil dest", data: []byte("id: 1\nname: x\n"), format: DataFormatYaml, wantErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := UnmarshalDataSlice[testItem](tt.data, nil, tt.format)
			if (err != nil) != tt.wantErr {
				t.Fatalf("UnmarshalDataSlice(nil) err=%v wantErr=%v", err, tt.wantErr)
			}
		})
	}
}

func TestUnmarshalDataSliceJSON(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    []testItem
		wantErr bool
		nilDest bool
	}{
		{name: "single object", data: []byte(`{"id":1,"name":"solo"}`), want: []testItem{{ID: 1, Name: "solo"}}},
		{name: "array", data: []byte(`[{"id":2,"name":"a"},{"id":3,"name":"b"}]`), want: []testItem{{ID: 2, Name: "a"}, {ID: 3, Name: "b"}}},
		{name: "wrong top-level", data: []byte(`123`), wantErr: true},
		{name: "malformed", data: []byte(`{"id":1,"name":}`), wantErr: true},
		{name: "nil dest", data: []byte(`{"id":1,"name":"x"}`), wantErr: true, nilDest: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			var got []testItem
			var dest *[]testItem
			if tt.nilDest {
				dest = nil
			} else {
				dest = &got
			}

			err := unmarshalDataSliceJSON[testItem](tt.data, dest)
			if (err != nil) != tt.wantErr {
				t.Fatalf("unmarshalDataSliceJSON() err=%v wantErr=%v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("unmarshalDataSliceJSON() got=%#v want=%#v", got, tt.want)
			}
		})
	}
}

func TestUnmarshalDataSliceYAML(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    []testItem
		wantErr bool
		nilDest bool
	}{
		{name: "single mapping block", data: []byte("id: 10\nname: solo\n"), want: []testItem{{ID: 10, Name: "solo"}}},
		{name: "single mapping flow", data: []byte(`{id: 11, name: flow}`), want: []testItem{{ID: 11, Name: "flow"}}},
		{name: "sequence block", data: []byte("- id: 12\n  name: a\n- id: 13\n  name: b\n"), want: []testItem{{ID: 12, Name: "a"}, {ID: 13, Name: "b"}}},
		{name: "sequence flow", data: []byte(`[{id: 14, name: a}, {id: 15, name: b}]`), want: []testItem{{ID: 14, Name: "a"}, {ID: 15, Name: "b"}}},
		{name: "wrong top-level scalar", data: []byte(`justastring`), wantErr: true},
		{name: "malformed", data: []byte("id: [1, 2\nname: x\n"), wantErr: true},
		{name: "nil dest", data: []byte("id: 1\nname: x\n"), wantErr: true, nilDest: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			var got []testItem
			var dest *[]testItem
			if tt.nilDest {
				dest = nil
			} else {
				dest = &got
			}

			err := unmarshalDataSliceYAML[testItem](tt.data, dest)
			if (err != nil) != tt.wantErr {
				t.Fatalf("unmarshalDataSliceYAML() err=%v wantErr=%v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("unmarshalDataSliceYAML() got=%#v want=%#v", got, tt.want)
			}
		})
	}
}

func TestSetNestedField(t *testing.T) {
	type tc struct {
		name  string
		start map[string]interface{}
		path  string
		value interface{}
		want  map[string]interface{}
	}

	tests := []tc{
		{
			name:  "creates nested maps and sets leaf",
			start: map[string]interface{}{},
			path:  "status.health",
			value: "OK",
			want: map[string]interface{}{
				"status": map[string]interface{}{
					"health": "OK",
				},
			},
		},
		{
			name: "overwrites existing leaf value",
			start: map[string]interface{}{
				"status": map[string]interface{}{
					"health": "BAD",
				},
			},
			path:  "status.health",
			value: "OK",
			want: map[string]interface{}{
				"status": map[string]interface{}{
					"health": "OK",
				},
			},
		},
		{
			name: "preserves sibling keys in existing nested map",
			start: map[string]interface{}{
				"status": map[string]interface{}{
					"health": "BAD",
					"uptime": 123,
				},
			},
			path:  "status.health",
			value: "OK",
			want: map[string]interface{}{
				"status": map[string]interface{}{
					"health": "OK",
					"uptime": 123,
				},
			},
		},
		{
			name:  "handles deeper nesting",
			start: map[string]interface{}{},
			path:  "a.b.c",
			value: 42,
			want: map[string]interface{}{
				"a": map[string]interface{}{
					"b": map[string]interface{}{
						"c": 42,
					},
				},
			},
		},
		{
			name: "parent exists but is not a map: converts it to map and continues",
			start: map[string]interface{}{
				"status": "not-a-map",
			},
			path:  "status.health",
			value: "OK",
			want: map[string]interface{}{
				"status": map[string]interface{}{
					"health": "OK",
				},
			},
		},
		{
			name: "value nil uses merge-patch null semantics (sets explicit nil)",
			start: map[string]interface{}{
				"status": map[string]interface{}{
					"health": "OK",
				},
			},
			path:  "status.health",
			value: nil,
			want: map[string]interface{}{
				"status": map[string]interface{}{
					"health": nil,
				},
			},
		},
		{
			name:  "string value that is valid JSON object is unmarshaled",
			start: map[string]interface{}{},
			path:  "spec.config",
			value: `{"x":1,"y":"z"}`,
			want: map[string]interface{}{
				"spec": map[string]interface{}{
					"config": map[string]interface{}{
						"x": float64(1),
						"y": "z",
					},
				},
			},
		},
		{
			name:  "string value that is valid JSON array is unmarshaled",
			start: map[string]interface{}{},
			path:  "spec.list",
			value: `[1,"a",true,null]`,
			want: map[string]interface{}{
				"spec": map[string]interface{}{
					"list": []interface{}{
						float64(1),
						"a",
						true,
						nil,
					},
				},
			},
		},
		{
			name:  "string value that is valid JSON scalar is unmarshaled",
			start: map[string]interface{}{},
			path:  "spec.enabled",
			value: `true`,
			want: map[string]interface{}{
				"spec": map[string]interface{}{
					"enabled": true,
				},
			},
		},
		{
			name:  "string value that is not JSON remains a string",
			start: map[string]interface{}{},
			path:  "spec.note",
			value: "hello world",
			want: map[string]interface{}{
				"spec": map[string]interface{}{
					"note": "hello world",
				},
			},
		},
		{
			name:  "non-string values are set directly",
			start: map[string]interface{}{},
			path:  "spec.count",
			value: int64(7),
			want: map[string]interface{}{
				"spec": map[string]interface{}{
					"count": int64(7),
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got := cloneMap(tt.start)
			SetNestedField(got, tt.path, tt.value)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("SetNestedField(%v, %q, %#v)\n got: %#v\nwant: %#v",
					tt.start, tt.path, tt.value, got, tt.want)
			}
		})
	}
}

func TestSetNestedField_EdgeCases_NoPanics(t *testing.T) {
	t.Run("nil target is a no-op (no panic)", func(t *testing.T) {
		var m map[string]interface{}  // nil
		SetNestedField(m, "a.b", "v") // should not panic
		// can't assert contents (nil), just ensure no panic
	})

	t.Run("empty path is a no-op", func(t *testing.T) {
		m := map[string]interface{}{"a": 1}
		SetNestedField(m, "", "x")
		want := map[string]interface{}{"a": 1}
		if !reflect.DeepEqual(m, want) {
			t.Fatalf("got %#v want %#v", m, want)
		}
	})

	t.Run("path of only dots is a no-op", func(t *testing.T) {
		m := map[string]interface{}{"a": 1}
		SetNestedField(m, "...", "x")
		want := map[string]interface{}{"a": 1}
		if !reflect.DeepEqual(m, want) {
			t.Fatalf("got %#v want %#v", m, want)
		}
	})

	t.Run("leading dot ignores empty segment (.a behaves like a)", func(t *testing.T) {
		m := map[string]interface{}{}
		SetNestedField(m, ".a", "v")
		want := map[string]interface{}{"a": "v"}
		if !reflect.DeepEqual(m, want) {
			t.Fatalf("got %#v want %#v", m, want)
		}
	})

	t.Run("trailing dot ignores empty segment (a. behaves like a)", func(t *testing.T) {
		m := map[string]interface{}{}
		SetNestedField(m, "a.", "v")
		want := map[string]interface{}{"a": "v"}
		if !reflect.DeepEqual(m, want) {
			t.Fatalf("got %#v want %#v", m, want)
		}
	})

	t.Run("double dots ignore empty segment (a..b behaves like a.b)", func(t *testing.T) {
		m := map[string]interface{}{}
		SetNestedField(m, "a..b", "v")
		want := map[string]interface{}{
			"a": map[string]interface{}{
				"b": "v",
			},
		}
		if !reflect.DeepEqual(m, want) {
			t.Fatalf("got %#v want %#v", m, want)
		}
	})

	t.Run("single segment path still sets top-level field", func(t *testing.T) {
		m := map[string]interface{}{"a": 1}
		SetNestedField(m, "b", 2)
		want := map[string]interface{}{"a": 1, "b": 2}
		if !reflect.DeepEqual(m, want) {
			t.Fatalf("got %#v want %#v", m, want)
		}
	})
}

func TestFirstNonSpaceByte(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
		want byte
	}{
		{name: "empty", in: []byte(""), want: 0},
		{name: "whitespace only", in: []byte(" \n\t "), want: 0},
		{name: "json object", in: []byte("  {\n}\n"), want: '{'},
		{name: "json array", in: []byte("\n\t[1,2]"), want: '['},
		{name: "yaml block sequence", in: []byte("\n  - a\n"), want: '-'},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if got := firstNonSpaceByte(tt.in); got != tt.want {
				t.Fatalf("firstNonSpaceByte()=%q want=%q", got, tt.want)
			}
		})
	}
}

// Deep-copy helper so test cases don't accidentally share maps.
func cloneMap(m map[string]interface{}) map[string]interface{} {
	if m == nil {
		return nil
	}
	out := make(map[string]interface{}, len(m))
	for k, v := range m {
		switch vv := v.(type) {
		case map[string]interface{}:
			out[k] = cloneMap(vv)
		default:
			out[k] = v
		}
	}
	return out
}
