// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package config

import (
	"reflect"
	"testing"
)

func TestMergeMaps(t *testing.T) {
	type args struct {
		srcMap   map[string]interface{}
		dstMap   map[string]interface{}
		mergeKey string
	}
	tests := []struct {
		name      string
		args      args
		mergedMap map[string]interface{}
		wantErr   bool
	}{
		{
			name: "copy missing key",
			args: args{
				srcMap:   map[string]interface{}{"a": 1},
				dstMap:   map[string]interface{}{},
				mergeKey: "",
			},
			mergedMap: map[string]interface{}{"a": 1},
			wantErr:   false,
		},
		{
			name: "dst scalar wins over src",
			args: args{
				srcMap:   map[string]interface{}{"a": 1},
				dstMap:   map[string]interface{}{"a": 2},
				mergeKey: "",
			},
			mergedMap: map[string]interface{}{"a": 2},
			wantErr:   false,
		},
		{
			name: "recursive map merge",
			args: args{
				srcMap: map[string]interface{}{
					"m": map[string]interface{}{"x": 1},
				},
				dstMap: map[string]interface{}{
					"m": map[string]interface{}{"y": 2},
				},
				mergeKey: "",
			},
			mergedMap: map[string]interface{}{
				"m": map[string]interface{}{"x": 1, "y": 2},
			},
			wantErr: false,
		},
		{
			name: "simple slice merge (scalars)",
			args: args{
				srcMap: map[string]interface{}{
					"s": []interface{}{1, 2},
				},
				dstMap: map[string]interface{}{
					"s": []interface{}{2, 3},
				},
				mergeKey: "",
			},
			mergedMap: map[string]interface{}{
				"s": []interface{}{2, 3, 1},
			},
			wantErr: false,
		},
		{
			name: "slice merge with maps and mergeKey",
			args: args{
				srcMap: map[string]interface{}{
					"list": []interface{}{
						map[string]interface{}{"id": 1, "val": "a"},
						map[string]interface{}{"id": 2, "val": "b"},
					},
				},
				dstMap: map[string]interface{}{
					"list": []interface{}{
						map[string]interface{}{"id": 2, "val": "B"},
					},
				},
				mergeKey: "id",
			},
			mergedMap: map[string]interface{}{
				"list": []interface{}{
					map[string]interface{}{"id": 2, "val": "B"},
					map[string]interface{}{"id": 1, "val": "a"},
				},
			},
			wantErr: false,
		},
		{
			name: "type mismatch error",
			args: args{
				srcMap:   map[string]interface{}{"a": 1},
				dstMap:   map[string]interface{}{"a": []interface{}{1}},
				mergeKey: "",
			},
			mergedMap: map[string]interface{}{"a": []interface{}{1}},
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := MergeMaps(tt.args.srcMap, tt.args.dstMap, tt.args.mergeKey); (err != nil) != tt.wantErr {
				t.Errorf("MergeMaps() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(tt.args.dstMap, tt.mergedMap) {
				t.Errorf("MergeMaps() = %v, want %v", tt.args.dstMap, tt.mergedMap)
			}
		})
	}
}

func Test_MergeSlices(t *testing.T) {
	type args struct {
		srcSlice *[]interface{}
		dstSlice *[]interface{}
		mergeKey string
	}
	tests := []struct {
		name        string
		args        args
		mergedSlice *[]interface{}
	}{
		{
			name: "simple scalar union",
			args: args{
				srcSlice: &[]interface{}{1, 2},
				dstSlice: &[]interface{}{2, 3},
				mergeKey: "",
			},
			mergedSlice: &[]interface{}{2, 3, 1},
		},
		{
			name: "map slice merge by key",
			args: args{
				srcSlice: &[]interface{}{
					map[string]interface{}{"id": 1, "v": "a"},
					map[string]interface{}{"id": 2, "v": "b"},
				},
				dstSlice: &[]interface{}{
					map[string]interface{}{"id": 2, "v": "B"},
				},
				mergeKey: "id",
			},
			mergedSlice: &[]interface{}{
				map[string]interface{}{"id": 2, "v": "B"},
				map[string]interface{}{"id": 1, "v": "a"},
			},
		},
		{
			name: "duplicate scalars aren’t re-appended",
			args: args{
				srcSlice: &[]interface{}{1, 1, 2},
				dstSlice: &[]interface{}{2},
				mergeKey: "",
			},
			mergedSlice: &[]interface{}{2, 1},
		},
		{
			name: "nil slices are no-ops",
			args: args{
				srcSlice: nil,
				dstSlice: nil,
				mergeKey: "id",
			},
			mergedSlice: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			MergeSlices(tt.args.srcSlice, tt.args.dstSlice, tt.args.mergeKey)
			if !reflect.DeepEqual(tt.args.dstSlice, tt.mergedSlice) {
				t.Errorf("MergeSlices() = %v, want %v", tt.args.dstSlice, tt.mergedSlice)
			}
		})
	}
}
