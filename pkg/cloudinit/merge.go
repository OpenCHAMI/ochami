// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cloudinit

import (
	"fmt"
	"regexp"
	"strings"

	gj "github.com/jryberg/gojinja"
)

// This file ports cloud-init's cloudinit.mergers package and
// util.mergemanydict, including their behavior for unusual settings, so that
// merged configuration matches a node's.

// defaultPartMergers is CloudConfigPartHandler's default for cloud-config
// parts.
const defaultPartMergers = "dict(replace)+list()+str()"

// defaultManyMergers is mergers.DEF_MERGE_TYPE, used by mergemanydict.
const defaultManyMergers = "list()+dict()+str()"

// mergerSpec is a merger name and its options. Options is whatever the
// configuration supplied: a list, a string, or a mapping. Membership tests on
// it follow Python's in operator, so a string option matches substrings.
type mergerSpec struct {
	name    any
	options any
}

var mergerPattern = regexp.MustCompile(`^([a-zA-Z_][A-Za-z0-9_]*)\((.*?)\)$`)

// parseMergeHow mirrors mergers.string_extract_mergers.
func parseMergeHow(mergeHow string) ([]mergerSpec, error) {
	var specs []mergerSpec
	for _, name := range strings.Split(mergeHow, "+") {
		name = strings.ReplaceAll(strings.TrimSpace(strings.ToLower(name)), "-", "_")
		if name == "" {
			continue
		}
		m := mergerPattern.FindStringSubmatch(name)
		if m == nil {
			return nil, fmt.Errorf("matcher identifier '%s' is not in the right format", name)
		}
		options := []any{}
		for _, opt := range strings.Split(strings.TrimSpace(m[2]), ",") {
			if opt = strings.TrimSpace(opt); opt != "" {
				options = append(options, strings.ToLower(opt))
			}
		}
		specs = append(specs, mergerSpec{name: m[1], options: options})
	}
	return specs, nil
}

// extractMergers mirrors mergers.dict_extract_mergers. It removes merge_how,
// or merge_type when merge_how is absent or null, from cfg.
func extractMergers(cfg *gj.OrderedDict) ([]mergerSpec, error) {
	raw, _ := cfg.Get("merge_how")
	cfg.Delete("merge_how")
	if raw == nil {
		raw, _ = cfg.Get("merge_type")
		cfg.Delete("merge_type")
	}
	if mapping, ok := raw.(*gj.OrderedDict); ok {
		// Python iterates a mapping's keys, each then treated as a string
		// specification.
		raw = mapping.Keys()
	}
	switch v := raw.(type) {
	case nil:
		return nil, nil
	case string:
		return parseMergeHow(v)
	case []any:
		var specs []mergerSpec
		for _, m := range v {
			var spec mergerSpec
			switch item := m.(type) {
			case *gj.OrderedDict:
				name, ok := item.Get("name")
				s, isString := name.(string)
				if !ok || !isString {
					return nil, fmt.Errorf("merger name must be a string, got %T", name)
				}
				settings, ok := item.Get("settings")
				if !ok {
					return nil, fmt.Errorf("merger %q has no settings", s)
				}
				spec = mergerSpec{name: strings.TrimSpace(strings.ReplaceAll(s, "-", "_")), options: settings}
			case []any:
				if len(item) == 0 {
					return nil, fmt.Errorf("merger list is empty")
				}
				spec = mergerSpec{name: item[0], options: []any{}}
				if len(item) >= 2 {
					spec.options = item[1:]
				}
			case string:
				// Python indexes the string: its first character is the name
				// and the rest are the options.
				runes := []rune(item)
				if len(runes) == 0 {
					return nil, fmt.Errorf("merger string is empty")
				}
				spec = mergerSpec{name: string(runes[0]), options: []any{}}
				if len(runes) >= 2 {
					spec.options = string(runes[1:])
				}
			default:
				return nil, fmt.Errorf("unsupported merger specification %T", m)
			}
			if pyTruthy(spec.name) {
				specs = append(specs, spec)
			}
		}
		return specs, nil
	default:
		return nil, fmt.Errorf("'%s' object is not iterable", pyTypeName(raw))
	}
}

// pyContains mirrors Python's item in container for merger options.
func pyContains(container any, item string) (bool, error) {
	switch c := container.(type) {
	case []any:
		for _, v := range c {
			if s, ok := v.(string); ok && s == item {
				return true, nil
			}
		}
		return false, nil
	case string:
		return strings.Contains(c, item), nil
	case *gj.OrderedDict:
		_, ok := c.Get(item)
		return ok, nil
	case nil:
		return false, fmt.Errorf("argument of type 'NoneType' is not iterable")
	default:
		return false, fmt.Errorf("merger settings of type %T are not a container", container)
	}
}

func containsAny(container any, items ...string) (bool, error) {
	for _, item := range items {
		ok, err := pyContains(container, item)
		if err != nil || ok {
			return ok, err
		}
	}
	return false, nil
}

// firstOption returns the first of methods that options contains, or def.
func firstOption(options any, def string, methods ...string) (string, error) {
	for _, m := range methods {
		ok, err := pyContains(options, m)
		if err != nil {
			return "", err
		}
		if ok {
			return m, nil
		}
	}
	return def, nil
}

// typedMerger merges values whose source has a particular Python type.
type typedMerger interface {
	handles(kind string) bool
	merge(root *merger, value, with any) (any, error)
}

// merger mirrors mergers.LookupMerger: the first typed merger that handles
// the source value's type merges it, and otherwise the source is kept.
type merger struct {
	lookups []typedMerger
}

// newMerger mirrors mergers.construct.
func newMerger(specs []mergerSpec) (*merger, error) {
	root := &merger{}
	for _, spec := range specs {
		name, ok := spec.name.(string)
		if !ok {
			return nil, fmt.Errorf("merger name must be a string, got %T", spec.name)
		}
		if !strings.HasPrefix(name, "m_") {
			name = "m_" + name
		}
		var m typedMerger
		var err error
		switch name {
		case "m_dict":
			m, err = newDictMerger(spec.options)
		case "m_list":
			m, err = newListMerger(spec.options)
		case "m_str":
			m, err = newStrMerger(spec.options)
		default:
			return nil, fmt.Errorf("could not find merger module named '%s'", name)
		}
		if err != nil {
			return nil, err
		}
		root.lookups = append(root.lookups, m)
	}
	return root, nil
}

// pyKind returns the lowercase Python type name that dispatches merging.
func pyKind(value any) string {
	switch value.(type) {
	case *gj.OrderedDict:
		return "dict"
	case []any:
		return "list"
	case string:
		return "str"
	default:
		return ""
	}
}

func (r *merger) merge(source, with any) (any, error) {
	kind := pyKind(source)
	for _, m := range r.lookups {
		if m.handles(kind) {
			return m.merge(r, source, with)
		}
	}
	return source, nil
}

// dictMerger mirrors mergers.m_dict.Merger.
type dictMerger struct {
	replace, recurseStr, recurseArray, allowDelete bool
}

func newDictMerger(options any) (*dictMerger, error) {
	method, err := firstOption(options, "no_replace", "replace", "no_replace")
	if err != nil {
		return nil, err
	}
	m := &dictMerger{replace: method == "replace"}
	if m.recurseStr, err = pyContains(options, "recurse_str"); err != nil {
		return nil, err
	}
	if m.recurseArray, err = containsAny(options, "recurse_array", "recurse_list"); err != nil {
		return nil, err
	}
	if m.allowDelete, err = pyContains(options, "allow_delete"); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *dictMerger) handles(kind string) bool { return kind == "dict" }

func (m *dictMerger) merge(root *merger, value, with any) (any, error) {
	other, ok := with.(*gj.OrderedDict)
	if !ok {
		return value, nil
	}
	merged := value.(*gj.OrderedDict).Copy()
	for _, key := range other.Keys() {
		newValue, _ := other.Get(key)
		oldValue, exists := merged.Get(key)
		if !exists {
			merged.Set(key, newValue)
			continue
		}
		if newValue == nil && m.allowDelete {
			merged.Delete(key)
			continue
		}
		result := oldValue
		switch {
		case m.replace:
			result = newValue
		case pyKind(newValue) == "list" && m.recurseArray,
			pyKind(newValue) == "str" && m.recurseStr,
			pyKind(newValue) == "dict":
			var err error
			if result, err = root.merge(oldValue, newValue); err != nil {
				return nil, err
			}
		}
		merged.Set(key, result)
	}
	return merged, nil
}

// listMerger mirrors mergers.m_list.Merger.
type listMerger struct {
	method                               string
	recurseStr, recurseDict, recurseList bool
}

func newListMerger(options any) (*listMerger, error) {
	method, err := firstOption(options, "replace", "append", "prepend", "replace", "no_replace")
	if err != nil {
		return nil, err
	}
	m := &listMerger{method: method}
	if m.recurseStr, err = pyContains(options, "recurse_str"); err != nil {
		return nil, err
	}
	if m.recurseDict, err = pyContains(options, "recurse_dict"); err != nil {
		return nil, err
	}
	if m.recurseList, err = containsAny(options, "recurse_array", "recurse_list"); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *listMerger) handles(kind string) bool { return kind == "list" }

// pyIterate returns the items Python's list.extend takes from value.
func pyIterate(value any) ([]any, error) {
	switch v := value.(type) {
	case []any:
		return v, nil
	case string:
		items := []any{}
		for _, r := range v {
			items = append(items, string(r))
		}
		return items, nil
	case *gj.OrderedDict:
		return v.Keys(), nil
	default:
		return nil, fmt.Errorf("'%s' object is not iterable", pyTypeName(value))
	}
}

func (m *listMerger) merge(root *merger, value, with any) (any, error) {
	old := value.([]any)
	if _, isList := with.([]any); m.method == "replace" && !isList {
		return with, nil
	}
	switch m.method {
	case "prepend", "append":
		items, err := pyIterate(with)
		if err != nil {
			return nil, err
		}
		if m.method == "prepend" {
			return append(append([]any{}, items...), old...), nil
		}
		return append(append([]any{}, old...), items...), nil
	}
	merged := append([]any{}, old...)
	items, err := pyIterate(with)
	if err != nil {
		return nil, err
	}
	if _, isMapping := with.(*gj.OrderedDict); isMapping && len(items) > 0 && len(merged) > 0 {
		// Python indexes the mapping by position, which raises KeyError.
		return nil, fmt.Errorf("KeyError: 0")
	}
	for i := 0; i < len(merged) && i < len(items); i++ {
		if m.method == "no_replace" {
			continue
		}
		newValue := items[i]
		switch {
		case pyKind(newValue) == "list" && m.recurseList,
			pyKind(newValue) == "str" && m.recurseStr,
			pyKind(newValue) == "dict" && m.recurseDict:
			if merged[i], err = root.merge(merged[i], newValue); err != nil {
				return nil, err
			}
		default:
			merged[i] = newValue
		}
	}
	return merged, nil
}

// strMerger mirrors mergers.m_str.Merger.
type strMerger struct {
	appendMode bool
}

func newStrMerger(options any) (*strMerger, error) {
	appendMode, err := pyContains(options, "append")
	if err != nil {
		return nil, err
	}
	return &strMerger{appendMode: appendMode}, nil
}

func (m *strMerger) handles(kind string) bool { return kind == "str" }

func (m *strMerger) merge(_ *merger, value, with any) (any, error) {
	if !m.appendMode {
		return with, nil
	}
	s, ok := with.(string)
	if !ok {
		return nil, fmt.Errorf("can only concatenate str (not \"%s\") to str", pyTypeName(with))
	}
	return value.(string) + s, nil
}

// pyTypeName returns the Python type name for a decoded value.
func pyTypeName(value any) string {
	switch value.(type) {
	case nil:
		return "NoneType"
	case bool:
		return "bool"
	case int64:
		return "int"
	case float64:
		return "float"
	case string:
		return "str"
	case []any:
		return "list"
	case *gj.OrderedDict:
		return "dict"
	case []byte:
		return "bytes"
	case Timestamp:
		return "datetime"
	default:
		return fmt.Sprintf("%T", value)
	}
}

// mergePart merges a cloud-config part into buf as CloudConfigPartHandler
// does: mergers from the part's merge_how or merge_type, then from its
// Merge-Type header, defaulting to dict(replace)+list()+str(). The part's
// merger keys are removed from payload.
func mergePart(buf, payload *gj.OrderedDict, header string) (*gj.OrderedDict, error) {
	specs, err := extractMergers(payload)
	if err != nil {
		return nil, err
	}
	headerSpecs, err := parseMergeHow(header)
	if err != nil {
		return nil, err
	}
	specs = append(specs, headerSpecs...)
	if len(specs) == 0 {
		if specs, err = parseMergeHow(defaultPartMergers); err != nil {
			return nil, err
		}
	}
	m, err := newMerger(specs)
	if err != nil {
		return nil, err
	}
	merged, err := m.merge(buf, payload)
	if err != nil {
		return nil, err
	}
	mapping, ok := merged.(*gj.OrderedDict)
	if !ok {
		return nil, fmt.Errorf("merging produced %s, not a mapping", pyTypeName(merged))
	}
	return mapping, nil
}

// mergeMany mirrors util.mergemanydict: sources in priority order, each
// merged with its own merge_how or merge_type, or list()+dict()+str().
func mergeMany(sources []*gj.OrderedDict) (*gj.OrderedDict, error) {
	merged := gj.NewOrderedDict()
	for _, cfg := range sources {
		if !pyTruthy(cfg) {
			continue
		}
		cfg = cfg.Copy()
		specs, err := extractMergers(cfg)
		if err != nil {
			return nil, err
		}
		if len(specs) == 0 {
			if specs, err = parseMergeHow(defaultManyMergers); err != nil {
				return nil, err
			}
		}
		m, err := newMerger(specs)
		if err != nil {
			return nil, err
		}
		result, err := m.merge(merged, cfg)
		if err != nil {
			return nil, err
		}
		mapping, ok := result.(*gj.OrderedDict)
		if !ok {
			return nil, fmt.Errorf("merging produced %s, not a mapping", pyTypeName(result))
		}
		merged = mapping
	}
	return merged, nil
}
