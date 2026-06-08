// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package client

import (
	"reflect"
	"testing"
)

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

func TestNewKeyValPatchDataMergePatch(t *testing.T) {
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

func TestNewKeyValPatchDataRFC6902(t *testing.T) {
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

func TestNewKeyValPatchDataRejectsInvalidRemoveIndex(t *testing.T) {
	_, _, err := NewKeyValPatchData(nil, nil, nil, []string{"groups=-"})
	if err == nil {
		t.Fatalf("NewKeyValPatchData accepted invalid remove index")
	}
}
