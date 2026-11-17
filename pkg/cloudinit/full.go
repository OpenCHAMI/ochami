// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cloudinit

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	gj "github.com/jryberg/gojinja"
)

// This file follows cloud-init's user-data processing (user_data.py), its
// part handlers (handlers/*.py), and how stages.py consumes user-data and
// vendor-data, so that ochami can render the configuration a node ends up
// with. It is experimental.

// Content types cloud-init assigns to parts (handlers.INCLUSION_TYPES_MAP).
const (
	TypeInclude      = "text/x-include-url"
	TypeIncludeOnce  = "text/x-include-once-url"
	TypeShellScript  = "text/x-shellscript"
	TypeCloudConfig  = "text/cloud-config"
	TypePartHandler  = "text/part-handler"
	TypeBoothook     = "text/cloud-boothook"
	TypeArchive      = "text/cloud-config-archive"
	TypeJSONPatch    = "text/cloud-config-jsonp"
	TypeJinja        = "text/jinja2"
	TypeNotMultipart = "text/x-not-multipart"
	TypeMultipart    = "multipart/mixed"
)

// inclusionPrefixes mirrors handlers.INCLUSION_SRCH: the prefixes in reverse
// sorted order, so longer variants of a prefix are tried first.
var inclusionPrefixes = []struct{ prefix, ctype string }{
	{"#part-handler", TypePartHandler},
	{"#include-once", TypeIncludeOnce},
	{"#include", TypeInclude},
	{"#cloud-config-jsonp", TypeJSONPatch},
	{"#cloud-config-archive", TypeArchive},
	{"#cloud-config", TypeCloudConfig},
	{"#cloud-boothook", TypeBoothook},
	{"#!", TypeShellScript},
	{"## template: jinja", TypeJinja},
}

// typeFromStartsWith mirrors handlers.type_from_starts_with.
func typeFromStartsWith(payload string) string {
	lower := strings.TrimLeftFunc(strings.ToLower(payload), unicode.IsSpace)
	for _, p := range inclusionPrefixes {
		if strings.HasPrefix(lower, p.prefix) {
			return p.ctype
		}
	}
	return ""
}

// ErrIncludeUnresolved marks an include that a Fetcher cannot resolve, for
// example a URL ochami cannot map to the impersonation API. A node would
// still fetch it, so the part is reported as unresolved and processing
// continues.
var ErrIncludeUnresolved = errors.New("include cannot be resolved")

// Fetcher returns the content of an #include URL. Other errors are treated as
// a failed fetch, which makes cloud-init abort processing that data.
type Fetcher func(ctx context.Context, url string) ([]byte, error)

// PartRenderer renders a Jinja template part with the given instance data,
// serialized as instance-data.json.
type PartRenderer interface {
	RenderPart(ctx context.Context, part Part, payload string, instanceData []byte) (string, error)
}

// InternalRenderer renders parts with [Render].
type InternalRenderer struct{}

// RenderPart implements PartRenderer.
func (InternalRenderer) RenderPart(ctx context.Context, _ Part, payload string, instanceData []byte) (string, error) {
	return Render(ctx, RenderRequest{Template: []byte(payload), InstanceData: Document{Data: instanceData}})
}

// PartStatus describes what cloud-init does with a part.
type PartStatus string

const (
	// PartMerged is a cloud-config part merged into the configuration.
	PartMerged PartStatus = "merged"
	// PartDropped is a part cloud-init skips with a warning, such as a
	// template that fails to render or invalid cloud-config.
	PartDropped PartStatus = "dropped"
	// PartScript is a shell script, which cloud-init runs rather than merges.
	PartScript PartStatus = "script"
	// PartBoothook is a boothook, which cloud-init runs rather than merges.
	PartBoothook PartStatus = "boothook"
	// PartExcluded is a part whose type vendor_data.disabled_handlers excludes.
	PartExcluded PartStatus = "excluded"
	// PartIgnored is a part with no handler, which cloud-init ignores.
	PartIgnored PartStatus = "ignored"
	// PartUnsupported is a part ochami's engine cannot process, such as MIME
	// multipart data. The cloud-init engine processes it.
	PartUnsupported PartStatus = "unsupported"
	// PartUnresolved is an include the Fetcher could not resolve.
	PartUnresolved PartStatus = "unresolved"
	// PartFailed is an include that could not be fetched, which makes
	// cloud-init abort processing its data.
	PartFailed PartStatus = "failed"
)

// Part describes one part of user-data or vendor-data.
type Part struct {
	// Data is "user-data" or "vendor-data".
	Data string `json:"data"`
	// Name is the name cloud-init gives the part, such as part-001. Includes
	// that are not attached as parts have no name.
	Name string `json:"name,omitempty"`
	// Source is the include URL the part came from, or Data.
	Source string `json:"source"`
	// Type is the content type cloud-init assigns.
	Type string `json:"type"`
	// Status is what cloud-init does with the part.
	Status PartStatus `json:"status"`
	// Message explains a status other than merged.
	Message string `json:"message,omitempty"`
	// Warnings are problems that do not change the status.
	Warnings []string `json:"warnings,omitempty"`
}

// FullRequest contains a node's user-data and vendor-data, its instance data,
// and how to fetch includes and render templates.
type FullRequest struct {
	UserData   []byte
	VendorData []byte
	// InstanceData is the node's instance-data.json, such as the output of
	// InstanceDataJSON.
	InstanceData []byte
	// Fetch resolves #include URLs. Without it, every include is unresolved.
	Fetch Fetcher
	// Renderer renders Jinja template parts. Without it, InternalRenderer is
	// used.
	Renderer PartRenderer
}

// FullResult is the configuration a node ends up with.
type FullResult struct {
	Parts []Part `json:"parts"`
	// user and vendor are the merged cloud-config of each data source, or nil
	// when it produced none. merged is user cloud-config merged over vendor
	// cloud-config, as cloud-init combines them.
	user, vendor, merged *gj.OrderedDict
	// VendorDisabled is set when user-data disables vendor-data.
	VendorDisabled bool `json:"vendor_disabled,omitempty"`
	// UserFailed and VendorFailed explain why cloud-init would abort
	// processing that data, for example because an include failed.
	UserFailed   string `json:"user_failed,omitempty"`
	VendorFailed string `json:"vendor_failed,omitempty"`
}

// maxIncludeDepth bounds nested includes, which cloud-init does not limit.
const maxIncludeDepth = 32

// builtinVendorData is the vendor_data setting in cloud-init's built-in
// configuration (settings.CFG_BUILTIN).
func builtinVendorData() *gj.OrderedDict {
	vd := gj.NewOrderedDict()
	vd.Set("enabled", true)
	vd.Set("prefix", []any{})
	cfg := gj.NewOrderedDict()
	cfg.Set("vendor_data", vd)
	return cfg
}

// RenderFull processes user-data and vendor-data as cloud-init does on a node
// and returns each part's status and the merged cloud-config. Errors are
// returned only for invalid requests and cancellation. Problems a node would
// tolerate are reported in the result.
func RenderFull(ctx context.Context, req FullRequest) (FullResult, error) {
	var result FullResult
	renderer := req.Renderer
	if renderer == nil {
		renderer = InternalRenderer{}
	}

	user := &processor{ctx: ctx, data: "user-data", fetch: req.Fetch, renderer: renderer, instanceData: req.InstanceData}
	if err := user.process(req.UserData, "user-data", 0); err != nil {
		if ctx.Err() != nil {
			return result, ctx.Err()
		}
		user.abort(err)
		result.Parts = user.parts
		result.UserFailed = err.Error()
		return result, nil
	}
	result.user = user.handle(nil)

	var vendorParts []Part
	if len(req.VendorData) > 0 {
		enabled, excluded, warning := vendorDataSettings(result.user)
		switch {
		case !enabled:
			result.VendorDisabled = true
		default:
			vendor := &processor{ctx: ctx, data: "vendor-data", fetch: req.Fetch, renderer: renderer, instanceData: req.InstanceData}
			if err := vendor.process(req.VendorData, "vendor-data", 0); err != nil {
				if ctx.Err() != nil {
					return result, ctx.Err()
				}
				vendor.abort(err)
				result.VendorFailed = err.Error()
			} else {
				result.vendor = vendor.handle(excluded)
			}
			vendorParts = vendor.parts
			if warning != "" && len(vendorParts) > 0 {
				vendorParts[0].Warnings = append(vendorParts[0].Warnings, warning)
			}
		}
	}
	result.Parts = append(user.parts, vendorParts...)
	if err := ctx.Err(); err != nil {
		return result, err
	}

	var sources []*gj.OrderedDict
	for _, cfg := range []*gj.OrderedDict{result.user, result.vendor} {
		if cfg != nil {
			sources = append(sources, cfg)
		}
	}
	merged, err := mergeMany(sources)
	if err != nil {
		// A node logs the failure and continues without the merged
		// configuration. Report it on the last part.
		result.Parts = append(result.Parts, Part{Data: "vendor-data", Source: "merge", Status: PartDropped, Message: "failed to merge user-data over vendor-data: " + err.Error()})
		merged = gj.NewOrderedDict()
	}
	result.merged = merged
	return result, nil
}

// vendorDataSettings mirrors Init._consume_vendordata: the vendor_data
// setting from user cloud-config merged over cloud-init's built-in default.
func vendorDataSettings(user *gj.OrderedDict) (enabled bool, excluded []string, warning string) {
	sources := []*gj.OrderedDict{}
	if user != nil {
		sources = append(sources, user)
	}
	cfg, err := mergeMany(append(sources, builtinVendorData()))
	if err != nil {
		return true, nil, ""
	}
	raw, _ := cfg.Get("vendor_data")
	vd, ok := raw.(*gj.OrderedDict)
	if !ok {
		return false, nil, "invalid vendor_data setting. resetting to: {'enabled': False}"
	}
	value, _ := vd.Get("enabled")
	if !pyIsTrue(value) {
		return false, nil, ""
	}
	if handlers, ok := vd.Get("disabled_handlers"); ok {
		if list, ok := handlers.([]any); ok {
			for _, h := range list {
				if s, ok := h.(string); ok {
					excluded = append(excluded, s)
				}
			}
		}
	}
	return true, excluded, ""
}

// pyIsTrue mirrors util.is_true.
func pyIsTrue(value any) bool {
	if b, ok := value.(bool); ok {
		return b
	}
	var s string
	switch v := value.(type) {
	case nil:
		s = "none"
	case string:
		s = v
	default:
		s = fmt.Sprint(v)
	}
	switch strings.TrimSpace(strings.ToLower(s)) {
	case "true", "1", "on", "yes":
		return true
	}
	return false
}

// attached is a part cloud-init attaches for its handlers.
type attached struct {
	index   int
	ctype   string
	payload string
}

// processor processes one data source.
type processor struct {
	ctx          context.Context
	data         string
	fetch        Fetcher
	renderer     PartRenderer
	instanceData []byte
	parts        []Part
	attached     []attached
}

func (p *processor) addPart(part Part) int {
	part.Data = p.data
	p.parts = append(p.parts, part)
	return len(p.parts) - 1
}

// abort marks the parts gathered before a failure. cloud-init raises before
// attaching any part, so none keeps its name.
func (p *processor) abort(err error) {
	for i := range p.parts {
		part := &p.parts[i]
		if part.Status == "" {
			part.Name = ""
			part.Status, part.Message = PartDropped, "not processed, because cloud-init aborts processing "+p.data+": "+err.Error()
		}
	}
}

// gunzip mirrors util.decomp_gzip with quiet failure: content that is not
// gzip data is returned unchanged.
func gunzip(raw []byte) []byte {
	if len(raw) < 2 || raw[0] != 0x1f || raw[1] != 0x8b {
		return raw
	}
	r, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		return raw
	}
	out, err := io.ReadAll(r)
	if err != nil {
		return raw
	}
	return out
}

// process mirrors UserDataProcessor.process for content that came from
// source. It returns an error when cloud-init would abort processing.
func (p *processor) process(raw []byte, source string, depth int) error {
	if err := p.ctx.Err(); err != nil {
		return err
	}
	if depth > maxIncludeDepth {
		return fmt.Errorf("includes nested more than %d deep at %s", maxIncludeDepth, source)
	}
	raw = gunzip(raw)
	head := raw
	if len(head) > 4096 {
		head = head[:4096]
	}
	if bytes.Contains(bytes.ToLower(head), []byte("mime-version:")) {
		p.addPart(Part{Source: source, Type: TypeMultipart, Status: PartUnsupported,
			Message: "MIME multipart data is processed only by the cloud-init engine"})
		return nil
	}
	text := string(raw)
	ctype := ""
	if utf8.Valid(raw) {
		ctype = typeFromStartsWith(text)
	}
	if ctype == "" {
		ctype = TypeNotMultipart
	}
	switch ctype {
	case TypeInclude, TypeIncludeOnce:
		return p.include(text, depth)
	case TypeArchive:
		p.addPart(Part{Source: source, Type: ctype, Status: PartUnsupported,
			Message: "cloud-config archives are processed only by the cloud-init engine"})
		return nil
	}
	index := p.addPart(Part{Name: fmt.Sprintf("part-%03d", len(p.attached)+1), Source: source, Type: ctype})
	p.attached = append(p.attached, attached{index: index, ctype: ctype, payload: text})
	return nil
}

// include mirrors UserDataProcessor._do_include.
func (p *processor) include(content string, depth int) error {
	for _, line := range pySplitlines(content) {
		lower := strings.ToLower(line)
		switch {
		case strings.HasPrefix(lower, "#include-once"):
			line = strings.TrimLeftFunc(line[len("#include-once"):], unicode.IsSpace)
		case strings.HasPrefix(lower, "#include"):
			line = strings.TrimLeftFunc(line[len("#include"):], unicode.IsSpace)
		}
		if strings.HasPrefix(line, "#") {
			continue
		}
		url := strings.TrimSpace(line)
		if url == "" {
			continue
		}
		if p.fetch == nil {
			p.addPart(Part{Source: url, Type: TypeInclude, Status: PartUnresolved, Message: "no fetcher for includes"})
			continue
		}
		body, err := p.fetch(p.ctx, url)
		switch {
		case errors.Is(err, ErrIncludeUnresolved):
			p.addPart(Part{Source: url, Type: TypeInclude, Status: PartUnresolved, Message: err.Error()})
			continue
		case err != nil:
			p.addPart(Part{Source: url, Type: TypeInclude, Status: PartFailed, Message: err.Error()})
			return fmt.Errorf("fetching from %s failed: %w", url, err)
		}
		if err := p.process(body, url, depth+1); err != nil {
			return err
		}
	}
	return nil
}

// handle runs cloud-init's default part handlers over the attached parts and
// returns the merged cloud-config, or nil when no cloud-config part was
// handled.
func (p *processor) handle(excluded []string) *gj.OrderedDict {
	var buf *gj.OrderedDict
	merged := 0
	sawCloudConfig := false
	mergeCloudConfig := func(part *Part, payload string) {
		sawCloudConfig = true
		if buf == nil || merged == 0 {
			buf = gj.NewOrderedDict()
		}
		value, err := parseYAML11([]byte(payload))
		mapping, ok := value.(*gj.OrderedDict)
		if err != nil || !ok {
			part.Status, part.Message = PartDropped, "failed at merging in cloud config part: empty cloud config"
			if err != nil {
				part.Message += " (" + err.Error() + ")"
			}
			return
		}
		result, err := mergePart(buf, mapping, "")
		if err != nil {
			part.Status, part.Message = PartDropped, "failed at merging in cloud config part: "+err.Error()
			return
		}
		buf = result
		merged++
		part.Status = PartMerged
	}
	for _, a := range p.attached {
		part := &p.parts[a.index]
		if containsString(excluded, a.ctype) {
			part.Status, part.Message = PartExcluded, "excluded by vendor_data.disabled_handlers"
			continue
		}
		if a.ctype != TypeJinja && (strings.Contains(a.payload, "{{") || strings.Contains(a.payload, "{%")) {
			part.Warnings = append(part.Warnings, "contains Jinja syntax without a '## template: jinja' header, so cloud-init uses it verbatim")
		}
		switch a.ctype {
		case TypeCloudConfig:
			mergeCloudConfig(part, a.payload)
		case TypeJinja:
			rendered, err := p.renderer.RenderPart(p.ctx, *part, a.payload, p.instanceData)
			if err != nil {
				part.Status, part.Message = PartDropped, "ignoring jinja template: failed to render template: "+err.Error()
				continue
			}
			if rendered == "" {
				part.Status, part.Message = PartDropped, "template rendered to nothing"
				continue
			}
			switch sub := typeFromStartsWith(rendered); sub {
			case TypeCloudConfig:
				mergeCloudConfig(part, rendered)
			case TypeShellScript:
				part.Status = PartScript
			case TypeBoothook:
				part.Status = PartBoothook
			case TypeJSONPatch:
				part.Status, part.Message = PartUnsupported, "rendered cloud-config JSON patches are processed only by the cloud-init engine"
			default:
				if sub == "" {
					sub = "None"
				}
				part.Status, part.Message = PartDropped, "ignoring jinja template: could not find supported sub-handler for type "+sub
			}
		case TypeShellScript:
			part.Status = PartScript
		case TypeBoothook:
			part.Status = PartBoothook
		case TypePartHandler, TypeJSONPatch:
			part.Status, part.Message = PartUnsupported, a.ctype+" parts are processed only by the cloud-init engine"
		default:
			part.Status, part.Message = PartIgnored, "unhandled non-multipart userdata"
			if a.payload == "" {
				part.Message = "empty payload"
			}
		}
	}
	if !sawCloudConfig {
		return nil
	}
	return buf
}

func containsString(list []string, s string) bool {
	for _, item := range list {
		if item == s {
			return true
		}
	}
	return false
}

// Text formats the merged configuration as cloud-config, with a header that
// lists the merged parts as cloud-init does when it writes cloud-config.
func (r FullResult) Text() string {
	var b strings.Builder
	b.WriteString("#cloud-config\n\n")
	var names []string
	for _, part := range r.Parts {
		if part.Status == PartMerged {
			names = append(names, fmt.Sprintf("# %s %s (%s)", part.Data, part.Name, part.Source))
		}
	}
	if len(names) > 0 {
		fmt.Fprintf(&b, "# from %d files\n", len(names))
		b.WriteString(strings.Join(names, "\n"))
		b.WriteString("\n\n")
	}
	merged := r.merged
	if merged == nil {
		merged = gj.NewOrderedDict()
	}
	b.WriteString(dumpYAML(merged))
	return b.String()
}

// Config returns the merged configuration: user cloud-config merged over
// vendor cloud-config, as cloud-init combines them. A node also merges its
// system configuration beneath them, which is not included. Mappings are
// map[string]any, sequences []any, and scalars are nil, bool, int64,
// float64, string, []byte, or Timestamp.
func (r FullResult) Config() map[string]any {
	if r.merged == nil {
		return map[string]any{}
	}
	return plainValue(r.merged).(map[string]any)
}

// plainValue converts decoded values to plain Go maps and slices.
func plainValue(value any) any {
	switch v := value.(type) {
	case *gj.OrderedDict:
		out := make(map[string]any, v.Len())
		for _, key := range sortedKeys(v) {
			child, _ := v.Get(key)
			out[key] = plainValue(child)
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = plainValue(item)
		}
		return out
	default:
		return v
	}
}

// Comparable returns a description of the merged configuration that keeps
// each value's Python type, for comparing results from different engines.
func (r FullResult) Comparable() any {
	if r.merged == nil {
		return typedDescription(gj.NewOrderedDict())
	}
	return typedDescription(r.merged)
}

// typedDescription mirrors tagged() in scripts/cloud-init-reference.py.
func typedDescription(value any) any {
	switch v := value.(type) {
	case nil:
		return map[string]any{"t": "null"}
	case bool:
		return map[string]any{"t": "bool", "v": v}
	case int64:
		return map[string]any{"t": "int", "v": fmt.Sprint(v)}
	case float64:
		if v != v {
			return map[string]any{"t": "float", "v": "nan"}
		}
		return map[string]any{"t": "float", "v": pyFloatRepr(v)}
	case string:
		return map[string]any{"t": "str", "v": v}
	case []byte:
		return map[string]any{"t": "bytes", "v": base64.StdEncoding.EncodeToString(v)}
	case Timestamp:
		if v.DateOnly {
			return map[string]any{"t": "date", "v": v.String()}
		}
		return map[string]any{"t": "datetime", "v": v.String()}
	case []any:
		items := make([]any, len(v))
		for i, item := range v {
			items[i] = typedDescription(item)
		}
		return map[string]any{"t": "list", "v": items}
	case *gj.OrderedDict:
		keys := sortedKeys(v)
		sort.Strings(keys)
		items := make([]any, 0, len(keys))
		for _, key := range keys {
			child, _ := v.Get(key)
			items = append(items, []any{key, typedDescription(child)})
		}
		return map[string]any{"t": "dict", "v": items}
	default:
		return map[string]any{"t": "unknown", "v": fmt.Sprint(v)}
	}
}

// ComparableText formats a description from Comparable as YAML, so that two
// descriptions can be compared line by line. Strings that would load as
// other types are quoted, so type differences remain visible.
func ComparableText(description any) (string, error) {
	value, err := fromTypedDescription(description)
	if err != nil {
		return "", err
	}
	return dumpYAML(value), nil
}

// fromTypedDescription reverses typedDescription for formatting.
func fromTypedDescription(description any) (any, error) {
	m, ok := description.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid typed value %v", description)
	}
	switch m["t"] {
	case "null":
		return nil, nil
	case "bool":
		return m["v"], nil
	case "int":
		return constructInt(fmt.Sprint(m["v"]))
	case "float":
		if s := fmt.Sprint(m["v"]); s == "nan" || s == "inf" || s == "-inf" {
			return constructFloat("." + strings.TrimPrefix(s, "-"))
		}
		return constructFloat(fmt.Sprint(m["v"]))
	case "str", "date", "datetime", "unknown":
		return fmt.Sprint(m["v"]), nil
	case "bytes":
		return base64.StdEncoding.DecodeString(fmt.Sprint(m["v"]))
	case "list":
		items, _ := m["v"].([]any)
		out := make([]any, 0, len(items))
		for _, item := range items {
			v, err := fromTypedDescription(item)
			if err != nil {
				return nil, err
			}
			out = append(out, v)
		}
		return out, nil
	case "dict":
		items, _ := m["v"].([]any)
		out := gj.NewOrderedDict()
		for _, item := range items {
			pair, ok := item.([]any)
			if !ok || len(pair) != 2 {
				return nil, fmt.Errorf("invalid typed mapping item %v", item)
			}
			v, err := fromTypedDescription(pair[1])
			if err != nil {
				return nil, err
			}
			out.Set(fmt.Sprint(pair[0]), v)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("unknown typed value kind %v", m["t"])
	}
}
