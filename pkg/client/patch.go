// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package client

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/OpenCHAMI/ochami/pkg/format"
)

// JSONPatchOperation is a single operation in an RFC 6902 JSON Patch document.
type JSONPatchOperation struct {
	Op    string      `json:"op" yaml:"op"`
	Path  string      `json:"path" yaml:"path"`
	Value interface{} `json:"value,omitempty" yaml:"value,omitempty"`
}

// PatchMethod represents the supported patch type
type PatchMethod string

const (
	PatchMethodRFC6902 PatchMethod = "rfc6902" // JSON Patch
	PatchMethodRFC7386 PatchMethod = "rfc7386" // JSON Merge Patch
	PatchMethodKeyVal  PatchMethod = "keyval"  // Key-Value patching in dot notation
)

var (
	PatchMethodHelp = map[string]string{
		string(PatchMethodRFC6902): "JSON Patch format according to RFC 6902",
		string(PatchMethodRFC7386): "JSON Merge Patch format according to RFC 7386",
		string(PatchMethodKeyVal):  "Key-Value patching using dot notation (key.subkey=value)",
	}
)

func (pm PatchMethod) String() string {
	return string(pm)
}

func (pm *PatchMethod) Set(v string) error {
	switch PatchMethod(v) {
	case PatchMethodRFC6902,
		PatchMethodRFC7386,
		PatchMethodKeyVal:
		*pm = PatchMethod(v)
		return nil
	default:
		return fmt.Errorf("must be one of %v", []PatchMethod{
			PatchMethodRFC6902,
			PatchMethodRFC7386,
			PatchMethodKeyVal,
		})
	}
}

func (pm PatchMethod) Type() string {
	return "PatchMethod"
}

// NewKeyValPatch takes slices of items to set/unset and returns a map that can
// be marshaled and used as PatchMethodKeyVal/PatchMethodRFC7386 data. addList
// and removeList are accepted for backward compatibility but are ignored here;
// use NewKeyValPatchData to produce RFC 6902 add/remove operations.
func NewKeyValPatch(setList, unsetList, addList, removeList []string) (map[string]interface{}, error) {
	patch := make(map[string]interface{})

	// Populate keys to set with their values
	for _, setPair := range setList {
		parts := strings.SplitN(setPair, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid format %q for set list (expected key=value or key.subkey=value)", setPair)
		}
		format.SetNestedField(patch, parts[0], parts[1])
	}

	// Populate keys to unset
	for _, unsetKey := range unsetList {
		format.SetNestedField(patch, unsetKey, nil)
	}

	_ = addList
	_ = removeList

	return patch, nil
}

// NewKeyValPatchData takes set/unset/add/remove arguments and returns patch
// data plus the patch method to use. If add/remove operations are present, it
// returns an RFC 6902 JSON Patch document. Otherwise it returns an RFC 7386 JSON
// Merge Patch document using keyval dot notation.
func NewKeyValPatchData(setList, unsetList, addList, removeList []string) (PatchMethod, interface{}, error) {
	if len(addList) == 0 && len(removeList) == 0 {
		patch, err := NewKeyValPatch(setList, unsetList, nil, nil)
		return PatchMethodKeyVal, patch, err
	}

	patch := []JSONPatchOperation{}
	for _, setPair := range setList {
		parts := strings.SplitN(setPair, "=", 2)
		if len(parts) != 2 {
			return "", nil, fmt.Errorf("invalid format %q for set list (expected key=value or key.subkey=value)", setPair)
		}
		patch = append(patch, JSONPatchOperation{Op: "add", Path: DotPathToJSONPointer(parts[0]), Value: parsePatchValue(parts[1])})
	}
	for _, unsetKey := range unsetList {
		patch = append(patch, JSONPatchOperation{Op: "remove", Path: DotPathToJSONPointer(unsetKey)})
	}
	for _, addPair := range addList {
		parts := strings.SplitN(addPair, "=", 2)
		if len(parts) != 2 {
			return "", nil, fmt.Errorf("invalid format %q for add list (expected key=value or key.subkey=value)", addPair)
		}
		patch = append(patch, JSONPatchOperation{Op: "add", Path: DotPathToJSONPointer(parts[0]) + "/-", Value: parsePatchValue(parts[1])})
	}
	for _, removePair := range removeList {
		parts := strings.SplitN(removePair, "=", 2)
		if len(parts) != 2 {
			return "", nil, fmt.Errorf("invalid format %q for remove list (expected key=index or key.subkey=index)", removePair)
		}
		if parts[1] == "-" || strings.TrimSpace(parts[1]) == "" {
			return "", nil, fmt.Errorf("invalid index %q for remove list item %q", parts[1], removePair)
		}
		patch = append(patch, JSONPatchOperation{Op: "remove", Path: DotPathToJSONPointer(parts[0]) + "/" + escapeJSONPointerSegment(parts[1])})
	}

	return PatchMethodRFC6902, patch, nil
}

// DotPathToJSONPointer converts dot notation (e.g. a.b) to an RFC 6901 JSON
// Pointer (e.g. /a/b).
func DotPathToJSONPointer(path string) string {
	raw := strings.Split(path, ".")
	parts := make([]string, 0, len(raw))
	for _, p := range raw {
		if p != "" {
			parts = append(parts, escapeJSONPointerSegment(p))
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return "/" + strings.Join(parts, "/")
}

func escapeJSONPointerSegment(s string) string {
	s = strings.ReplaceAll(s, "~", "~0")
	s = strings.ReplaceAll(s, "/", "~1")
	return s
}

func parsePatchValue(value string) interface{} {
	var jsonValue interface{}
	if err := json.Unmarshal([]byte(value), &jsonValue); err == nil {
		return jsonValue
	}
	return value
}
