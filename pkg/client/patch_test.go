// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package client

import (
	"reflect"
	"testing"
)

// TestDotPathToJSONPointer verifies that DotPathToJSONPointer converts a dotted
// key path to a JSON pointer, skipping empty segments and escaping "~" and "/".
func TestDotPathToJSONPointer(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "simple", path: "hostname", want: "/hostname"},
		{name: "nested", path: "interface.ip", want: "/interface/ip"},
		{name: "empty segments ignored", path: ".a..b.", want: "/a/b"},
		{name: "escape", path: "a~/b.c", want: "/a~0~1b/c"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DotPathToJSONPointer(tt.path); got != tt.want {
				t.Fatalf("DotPathToJSONPointer(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

// TestNewKeyValPatchData_MergePatch verifies that set and unset operations
// alone produce a key-value merge patch with typed values and nulls for unset
// keys.
func TestNewKeyValPatchData_MergePatch(t *testing.T) {
	method, data, err := NewKeyValPatchData([]string{"hostname=ex01", "nid=42"}, []string{"role"}, nil, nil)
	if err != nil {
		t.Fatalf("NewKeyValPatchData returned error: %v", err)
	}
	if method != PatchMethodKeyVal {
		t.Fatalf("method = %s, want %s", method, PatchMethodKeyVal)
	}
	want := map[string]interface{}{"hostname": "ex01", "nid": float64(42), "role": nil}
	if !reflect.DeepEqual(data, want) {
		t.Fatalf("data = %#v, want %#v", data, want)
	}
}

// TestNewKeyValPatchData_RFC6902 verifies that adding array add or remove
// operations produces an RFC 6902 patch containing every operation in order.
func TestNewKeyValPatchData_RFC6902(t *testing.T) {
	method, data, err := NewKeyValPatchData(
		[]string{"hostname=ex01"},
		[]string{"role"},
		[]string{"groups=compute"},
		[]string{"groups=0"},
	)
	if err != nil {
		t.Fatalf("NewKeyValPatchData returned error: %v", err)
	}
	if method != PatchMethodRFC6902 {
		t.Fatalf("method = %s, want %s", method, PatchMethodRFC6902)
	}
	want := []JSONPatchOperation{
		{Op: "add", Path: "/hostname", Value: "ex01"},
		{Op: "remove", Path: "/role"},
		{Op: "add", Path: "/groups/-", Value: "compute"},
		{Op: "remove", Path: "/groups/0"},
	}
	if !reflect.DeepEqual(data, want) {
		t.Fatalf("data = %#v, want %#v", data, want)
	}
}

// TestPatchMethod_Value verifies PatchMethod's pflag.Value implementation
// accepts its three known values and rejects anything else.
func TestPatchMethod_Value(t *testing.T) {
	var method PatchMethod
	for _, value := range []string{"rfc6902", "rfc7386", "keyval"} {
		if err := method.Set(value); err != nil {
			t.Errorf("Set(%q): %v", value, err)
		}
	}
	if err := method.Set("invalid"); err == nil {
		t.Error("Set(invalid) returned nil")
	}
	if method.Type() != "PatchMethod" {
		t.Errorf("Type = %q", method.Type())
	}
}
