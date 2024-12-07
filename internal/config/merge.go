package config

import (
	"fmt"
	"reflect"
)

// MergeMaps merges srcMap with dstMap, left to right. This means that items in
// dstMap overwrite items with the same key in srcMap. Items with the same key
// in either map must be the same type or this function will return an error.
// Also, maps (and any children maps) are assumed to have a string as the key
// since this function is meant to be used with data unmarshalled from data
// formats such as JSON or YAML.
//
// If slices are encountered and the slice contains map elements, MergeMaps will
// use mergeKey to compare elements to determine if the element in dstMap should
// overwrite the corresponding element in srcMap. Otherwise, elements in the
// source slice will be appended to the slice in the destination.
//
// MergeMaps edits dstMap in-place. This means that dstMap will contain the
// result of the merge.
func MergeMaps(srcMap, dstMap map[string]interface{}, mergeKey string) error {
	for skey, sval := range srcMap {
		dval, ok := dstMap[skey]
		if !ok {
			dstMap[skey] = sval
			continue
		}

		if reflect.TypeOf(dstMap[skey]) != reflect.TypeOf(sval) {
			return fmt.Errorf("type mismatch for key %s: %T != %T", skey, sval, dval)
		}

		switch sval.(type) {
		case map[string]interface{}:
			err := MergeMaps((srcMap[skey]).(map[string]interface{}), (dstMap[skey]).(map[string]interface{}), mergeKey)
			if err != nil {
				return err
			}
		case []interface{}:
			s1 := srcMap[skey].([]interface{})
			s2 := dstMap[skey].([]interface{})
			mergeSlices(&s1, &s2, mergeKey)
			srcMap[skey] = s1
			dstMap[skey] = s2
		default:
			dstMap[skey] = sval
		}
	}

	return nil
}

// mergeSlices merges srcSlice into dstSlice with items in dstSlice overwriting
// any conflicting items in srcSlice. If the slice contains maps, mergeKey is
// used to determine which source elements to overwrite with destination
// elements.
func mergeSlices(srcSlice, dstSlice *[]interface{}, mergeKey string) {
	if srcSlice == nil || dstSlice == nil {
		return
	}
	for _, sval := range *srcSlice {
		exists := false
		switch sv := sval.(type) {
		// Source item is a map
		case map[string]interface{}:
			for _, dval := range *dstSlice {
				switch dv := dval.(type) {
				// Dest item is a map
				case map[string]interface{}:
					if sv[mergeKey] == dv[mergeKey] {
						exists = true
						continue
					}
				}
			}
			if !exists {
				*dstSlice = append(*dstSlice, sv)
			}
		// Source item is not a map
		default:
			for _, dval := range *dstSlice {
				switch dv := dval.(type) {
				// Dest item is a map, skip
				case map[string]interface{}:
				// Dest item is not a map, compare
				default:
					if sval == dv {
						exists = true
						continue
					}
				}
			}
			if !exists {
				*dstSlice = append(*dstSlice, sv)
			}
		}
	}
}
