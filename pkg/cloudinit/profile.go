// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cloudinit

import (
	"crypto/rand"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"sync"
	"unicode"

	gj "github.com/jryberg/gojinja"
	gjerrors "github.com/jryberg/gojinja/pkg/errors"
	"github.com/jryberg/gojinja/pkg/lexer"
	"github.com/jryberg/gojinja/pkg/runtime"
)

// MissingVariablePrefix begins the text cloud-init prints in place of an
// undefined variable, followed by the variable's name.
const MissingVariablePrefix = "CI_MISSING_JINJA_VAR/"

// jinjaPartPrefix is the lowercase prefix cloud-init uses to classify a part
// as a Jinja template (handlers.INCLUSION_TYPES_MAP).
const jinjaPartPrefix = "## template: jinja"

var (
	// templateHeader matches the first line that templater.detect_template
	// treats as a template header.
	templateHeader = regexp.MustCompile(`(?i)^##\s*template:(.*)`)

	// versionKey matches the instance-data keys whose children
	// convert_jinja_instance_data copies to the enclosing mapping.
	versionKey = regexp.MustCompile(`^v\p{Nd}+\n?$`)

	// jinjaOperator matches Jinja2 operators, longest first, as in Jinja2's
	// lexer.operator_re. cloud-init replaces each match with an underscore to
	// form key aliases.
	jinjaOperator = func() *regexp.Regexp {
		ops := []string{"//", "**", "==", "!=", ">=", "<=",
			"+", "-", "/", "*", "%", "~", "[", "]", "(", ")", "{", "}",
			">", "<", "=", ".", ":", "|", ",", ";"}
		for i, op := range ops {
			ops[i] = regexp.QuoteMeta(op)
		}
		return regexp.MustCompile(strings.Join(ops, "|"))
	}()
)

// HasJinjaHeader reports whether cloud-init would process template as a Jinja
// part. cloud-init renders only parts that begin with "## template: jinja",
// ignoring case and leading whitespace. It uses other parts verbatim.
func HasJinjaHeader(template []byte) bool {
	text := strings.ToLower(strings.TrimLeftFunc(string(template), unicode.IsSpace))
	return strings.HasPrefix(text, jinjaPartPrefix)
}

// templateBody removes a template header line, as templater.detect_template
// does, and reports whether it removed one. Header types other than jinja
// are rejected.
func templateBody(template string) (string, bool, error) {
	ident, rest, _ := strings.Cut(template, "\n")
	match := templateHeader.FindStringSubmatch(ident)
	if match == nil {
		return template, false, nil
	}
	if kind := strings.ToLower(strings.TrimSpace(match[1])); kind != "jinja" {
		return "", false, fmt.Errorf("unsupported template type %q: only jinja templates are rendered", kind)
	}
	return rest, true, nil
}

// lexerOptions returns cloud-init's whitespace settings: trim_blocks enabled,
// lstrip_blocks disabled, LF newlines, and one final source newline removed.
func lexerOptions() lexer.Options {
	opts := lexer.DefaultOptions()
	opts.TrimBlocks = true
	opts.LstripBlocks = false
	opts.KeepTrailingNewline = false
	opts.NewlineSequence = "\n"
	return opts
}

// instanceData builds template variables the way cloud-init does for a part.
// Metadata becomes ds.meta_data and extra variables replace top-level keys of
// that instance data. The result then goes through the sorted JSON round trip
// of instance-data.json and convert_jinja_instance_data.
func instanceData(metadata, extra *gj.OrderedDict) *gj.OrderedDict {
	ds := gj.NewOrderedDict()
	ds.Set("meta_data", metadata)
	data := gj.NewOrderedDict()
	data.Set("ds", ds)
	for _, key := range extra.Keys() {
		value, _ := extra.Get(key)
		data.Set(key, value)
	}
	return convertInstanceData(data)
}

// convertInstanceData mirrors convert_jinja_instance_data. It visits keys in
// sorted order, copies the children of version keys such as v1 to the
// enclosing mapping, and adds an alias for each key containing a Jinja
// operator, such as local_hostname for local-hostname. Mappings inside lists
// are sorted but not converted.
func convertInstanceData(data *gj.OrderedDict) *gj.OrderedDict {
	result := gj.NewOrderedDict()
	for _, key := range sortedKeys(data) {
		value, _ := data.Get(key)
		if mapping, ok := value.(*gj.OrderedDict); ok {
			converted := convertInstanceData(mapping)
			result.Set(key, converted)
			if versionKey.MatchString(key) {
				for _, child := range converted.Keys() {
					childValue, _ := converted.Get(child)
					result.Set(child, copyValue(childValue))
				}
			}
		} else {
			result.Set(key, sortMappings(value))
		}
		if alias := jinjaOperator.ReplaceAllString(key, "_"); alias != key {
			current, _ := result.Get(key)
			result.Set(alias, copyValue(current))
		}
	}
	return result
}

// sortMappings returns a deep copy of value with every mapping's keys in
// sorted order, matching json.dumps(sort_keys=True) followed by json.loads.
func sortMappings(value any) any {
	switch v := value.(type) {
	case *gj.OrderedDict:
		out := gj.NewOrderedDict()
		for _, key := range sortedKeys(v) {
			child, _ := v.Get(key)
			out.Set(key, sortMappings(child))
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, child := range v {
			out[i] = sortMappings(child)
		}
		return out
	default:
		return value
	}
}

// copyValue returns a deep copy of value that keeps mapping order, like
// Python's copy.deepcopy.
func copyValue(value any) any {
	switch v := value.(type) {
	case *gj.OrderedDict:
		out := gj.NewOrderedDict()
		for _, key := range v.Keys() {
			child, _ := v.Get(key)
			out.Set(key, copyValue(child))
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, child := range v {
			out[i] = copyValue(child)
		}
		return out
	default:
		return value
	}
}

// sortedKeys returns a mapping's keys in code point order, as Python sorts
// strings. Decoded documents only contain string keys.
func sortedKeys(data *gj.OrderedDict) []string {
	keys := make([]string, 0, data.Len())
	for _, key := range data.Keys() {
		keys = append(keys, fmt.Sprint(key))
	}
	slices.Sort(keys)
	return keys
}

// rejectLoopControls rejects the break and continue tags. gojinja always
// parses them, but cloud-init does not enable Jinja2's loopcontrols extension,
// so on a node these templates fail to compile.
func rejectLoopControls(source string, opts lexer.Options) error {
	lx, err := lexer.New(opts)
	if err != nil {
		return err
	}
	stream, err := lx.Tokenize(source, "", "")
	if err != nil {
		return err
	}
	afterBlockBegin := false
	for !stream.EOS() {
		tok := stream.Next()
		if afterBlockBegin && tok.Kind == lexer.TokenName && (tok.Value == "break" || tok.Value == "continue") {
			msg := fmt.Sprintf("Encountered unknown tag '%s'. cloud-init does not enable the loopcontrols extension.", tok.Value)
			return gjerrors.NewTemplateSyntaxError(msg, tok.Lineno, "", "")
		}
		afterBlockBegin = tok.Kind == lexer.TokenBlockBegin
	}
	return nil
}

// headerLineOffset reports syntax errors against the original template after
// its header line was removed, as cloud-init does.
func headerLineOffset(err error, template string) error {
	var syntaxErr *gjerrors.TemplateSyntaxError
	if errors.As(err, &syntaxErr) {
		syntaxErr.Lineno++
		syntaxErr.Source = template
	}
	return err
}

// missingVariables reproduces cloud-init's UndefinedJinjaVariable, which
// prints MissingVariablePrefix and the variable name. gojinja cannot change
// the text of an undefined value, so each render uses debug undefined values
// whose names carry a random token and then replaces the printed placeholders.
// The token is digits only, so case filters leave it intact and replace can
// detect placeholders it cannot restore.
type missingVariables struct {
	token string
	mu    sync.Mutex
	names map[string]struct{}
}

func newMissingVariables() *missingVariables {
	digits := make([]byte, 24)
	_, _ = rand.Read(digits) // crypto/rand.Read never returns an error
	for i, b := range digits {
		digits[i] = '0' + b%10
	}
	return &missingVariables{token: string(digits), names: map[string]struct{}{}}
}

// factory creates undefined values that keep their original error messages.
func (m *missingVariables) factory(hint, name string, obj any, exc func(string) error) runtime.Undefined {
	message := runtime.Undefined{Hint: hint, Name: name, Obj: obj}.Message()
	if exc == nil {
		exc = func(msg string) error { return gjerrors.NewUndefinedError(msg) }
	}
	m.mu.Lock()
	m.names[name] = struct{}{}
	m.mu.Unlock()
	return runtime.Undefined{
		Mode:           runtime.ModeDebug,
		Name:           m.token + name,
		ExcConstructor: func(string) error { return exc(message) },
	}
}

// replace restores cloud-init's text for each printed undefined value.
func (m *missingVariables) replace(out string) (string, error) {
	if !strings.Contains(out, m.token) {
		return out, nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for name := range m.names {
		printed := name
		if printed == "" {
			printed = "None"
		}
		out = strings.ReplaceAll(out, "{{ "+m.token+name+" }}", MissingVariablePrefix+printed)
	}
	if strings.Contains(out, m.token) {
		return "", fmt.Errorf("an undefined variable was transformed before output, so its %s text cannot be reproduced: define the variable or use the default filter", MissingVariablePrefix)
	}
	return out, nil
}
