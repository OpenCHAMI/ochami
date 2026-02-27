// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package client

import (
	"fmt"
	"strings"

	"github.com/OpenCHAMI/ochami/pkg/format"
)

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

// NewKeyValPatch takes slices of items to set/unset/add/remove and returns a
// map that can be marshaled and used as PatchMethodKeyVal data.
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

	// Populate keys to add with their values
	for _, addPair := range addList {
		parts := strings.SplitN(addPair, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid format %q for add list (expected key=value or key.subkey=value)", addPair)
		}
		// For arrays, use JSON Merge Patch append syntax, if possible,
		// otherwise, convert to JSON Patch
		format.SetNestedField(patch, parts[0], parts[1])
	}

	// Populate keys to remove with their values
	for _, removePair := range removeList {
		parts := strings.SplitN(removePair, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid format %q for remove list (expected key=value or key.subkey=value)", removePair)
		}
		// Remove operations are complex and might need JSON Patch; for now,
		// handle simple cases
		format.SetNestedField(patch, parts[0], parts[1])
	}

	return patch, nil
}
