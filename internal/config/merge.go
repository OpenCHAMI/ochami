// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package config

import (
	"fmt"
	"reflect"
)

// MergeMaps merges src into dst (in-place), left to right. Keys only in src are
// copied into dst. Where both have the same key:
//
//   - If both values are maps, they are merged recursively.
//   - If both are slices, they are merged via MergeSlices (dst elements win on
//     conflicts).
//   - Otherwise, dst's value is left intact (dst always "overwrites" src).
//
// Mismatched types for the same key will trigger an error.
//
// If slices are encountered and the slice contains map elements, MergeMaps will
// use mergeKey to compare elements to determine if the element in dstMap should
// overwrite the corresponding element in srcMap. Otherwise, elements in the
// source slice will be appended to the slice in the destination.
func MergeMaps(src, dst map[string]interface{}, mergeKey string) error {
	for k, sv := range src {
		dv, exists := dst[k]
		if !exists {
			// Item not present in dst, add it using value from src
			dst[k] = sv
			continue
		}

		// Key exists in both src and dst, check that type matches
		if reflect.TypeOf(sv) != reflect.TypeOf(dv) {
			// Type of key's value has differing type in each map, err
			return fmt.Errorf("type mismatch for key %q: %T (src) vs %T (dst)", k, sv, dv)
		}

		// Type matches, determine how to resolve conflict
		switch svTyped := sv.(type) {
		case map[string]interface{}:
			// Items are maps, recurse to merge them
			if err := MergeMaps(svTyped, dv.(map[string]interface{}), mergeKey); err != nil {
				return err
			}
		case []interface{}:
			// Items are slices, use MergeSlices to merge
			srcSlice := svTyped
			dstSlice := dv.([]interface{})
			MergeSlices(&srcSlice, &dstSlice, mergeKey)
			dst[k] = dstSlice

		default:
			// Items are scalars, keep the existing value in dst and do nothing
		}
	}

	// Done merging and no error occurred
	return nil
}

// MergeSlices merges *src into *dst, appending any src items that don't
// conflict, and leaving dst's items intact where mergeKey matches. For map
// elements, a matching mergeKey causes a deep‐merge of that pair using
// MergeMap, again with dst-values taking precedence.
//
// If a slice element is a map, MergeSlices will check if mergeKey is present
// in the maps and, if so, will call MergeMaps to perform a deep merge on the
// maps.
func MergeSlices(src, dst *[]interface{}, mergeKey string) {
	if src == nil || dst == nil {
		return
	}
	for _, s := range *src {
		matched := false

		// Check if src element is a map
		if sMap, ok := s.(map[string]interface{}); ok {
			// Element is a map; check if src and dst maps have
			// mergeKey and, if so, call MergeMaps to deep merge
			// them
			if keyVal, hasKey := sMap[mergeKey]; hasKey {
				// src map has mergeKey, iterate over elements
				// in dst to find map also with mergeKey
				for _, d := range *dst {
					if dMap, ok2 := d.(map[string]interface{}); ok2 {
						if dv, has := dMap[mergeKey]; has && dv == keyVal {
							// mergeKey found in dst map element (conflicting
							// map between src and dst); perform deep merge
							MergeMaps(sMap, dMap, mergeKey)
							matched = true
							break
						}
					}
				}
			}
		} else {
			// Element is not a map; check if it is exists also in
			// dst and do nothing if so
			for _, d := range *dst {
				// Use DeepEqual just to be safe...
				if reflect.DeepEqual(s, d) {
					matched = true
					break
				}
			}
		}

		// src element is not a map and does not already exist in dst;
		// append it to dst
		if !matched {
			*dst = append(*dst, s)
		}
	}
}
