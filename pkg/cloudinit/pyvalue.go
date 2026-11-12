// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cloudinit

import (
	"encoding/base64"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	gj "github.com/jryberg/gojinja"
)

// pyTruthy reports whether value is true in Python's sense.
func pyTruthy(value any) bool {
	switch v := value.(type) {
	case nil:
		return false
	case bool:
		return v
	case string:
		return v != ""
	case int:
		return v != 0
	case int64:
		return v != 0
	case float64:
		return v != 0
	case []any:
		return len(v) > 0
	case *gj.OrderedDict:
		return v.Len() > 0
	case map[string]any:
		return len(v) > 0
	case []byte:
		return len(v) > 0
	default:
		return true
	}
}

// pySplitlines mirrors Python's str.splitlines without keepends.
func pySplitlines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		switch r {
		case '\n', '\v', '\f', '\x1c', '\x1d', '\x1e', '\u0085', ' ', ' ':
			lines = append(lines, s[start:i])
			i += size
			start = i
			continue
		case '\r':
			lines = append(lines, s[start:i])
			i += size
			if i < len(s) && s[i] == '\n' {
				i++
			}
			start = i
			continue
		}
		i += size
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

// jsonRoundTrip returns value as json.loads(atomic_helper.json_dumps(value))
// would: mappings sorted by key, bytes as ci-b64 strings, and other values
// JSON cannot represent replaced by cloud-init's warning text.
func jsonRoundTrip(value any) any {
	switch v := value.(type) {
	case *gj.OrderedDict:
		out := gj.NewOrderedDict()
		for _, key := range sortedKeys(v) {
			child, _ := v.Get(key)
			out.Set(key, jsonRoundTrip(child))
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, child := range v {
			out[i] = jsonRoundTrip(child)
		}
		return out
	case nil, bool, string, int64, float64:
		return v
	case int:
		return int64(v)
	case []byte:
		return "ci-b64:" + base64.StdEncoding.EncodeToString(v)
	default:
		return "Warning: redacted unserializable type " + pyTypeRepr(v)
	}
}

// pyTypeRepr returns the repr of the Python type that value stands for.
func pyTypeRepr(value any) string {
	switch value.(type) {
	case time.Time:
		return "<class 'datetime.datetime'>"
	default:
		return fmt.Sprintf("<class '%T'>", value)
	}
}

// pyJSONDumps serializes value as cloud-init's atomic_helper.json_dumps does:
// indent 1, sorted keys, ASCII-only strings, and Python float formatting.
func pyJSONDumps(value any) string {
	var b strings.Builder
	writePyJSON(&b, jsonRoundTrip(value), 0)
	return b.String()
}

func writePyJSON(b *strings.Builder, value any, depth int) {
	indent := func(d int) {
		b.WriteByte('\n')
		b.WriteString(strings.Repeat(" ", d))
	}
	switch v := value.(type) {
	case *gj.OrderedDict:
		if v.Len() == 0 {
			b.WriteString("{}")
			return
		}
		b.WriteByte('{')
		for i, key := range sortedKeys(v) {
			if i > 0 {
				b.WriteByte(',')
			}
			indent(depth + 1)
			writePyJSONString(b, key)
			b.WriteString(": ")
			child, _ := v.Get(key)
			writePyJSON(b, child, depth+1)
		}
		indent(depth)
		b.WriteByte('}')
	case []any:
		if len(v) == 0 {
			b.WriteString("[]")
			return
		}
		b.WriteByte('[')
		for i, child := range v {
			if i > 0 {
				b.WriteByte(',')
			}
			indent(depth + 1)
			writePyJSON(b, child, depth+1)
		}
		indent(depth)
		b.WriteByte(']')
	case nil:
		b.WriteString("null")
	case bool:
		b.WriteString(strconv.FormatBool(v))
	case int64:
		b.WriteString(strconv.FormatInt(v, 10))
	case float64:
		b.WriteString(pyJSONFloat(v))
	case string:
		writePyJSONString(b, v)
	default:
		writePyJSONString(b, fmt.Sprint(v))
	}
}

// writePyJSONString writes s as json.dumps does with ensure_ascii enabled.
func writePyJSONString(b *strings.Builder, s string) {
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		case '\b':
			b.WriteString(`\b`)
		case '\f':
			b.WriteString(`\f`)
		default:
			switch {
			case r < 0x20 || (r >= 0x7f && r < 0x10000):
				fmt.Fprintf(b, `\u%04x`, r)
			case r >= 0x10000:
				r -= 0x10000
				fmt.Fprintf(b, `\u%04x\u%04x`, 0xd800+(r>>10), 0xdc00+(r&0x3ff))
			default:
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('"')
}

// pyJSONFloat formats f as json.dumps does: float.__repr__, with NaN and
// infinities spelled as JavaScript literals.
func pyJSONFloat(f float64) string {
	switch {
	case math.IsNaN(f):
		return "NaN"
	case math.IsInf(f, 1):
		return "Infinity"
	case math.IsInf(f, -1):
		return "-Infinity"
	}
	return pyFloatRepr(f)
}

// pyFloatRepr mirrors Python's repr(float): the shortest round-tripping
// digits, in positional notation for decimal exponents from -4 to 15, and
// otherwise in scientific notation with a two-digit minimum exponent.
func pyFloatRepr(f float64) string {
	switch {
	case math.IsNaN(f):
		return "nan"
	case math.IsInf(f, 1):
		return "inf"
	case math.IsInf(f, -1):
		return "-inf"
	}
	sci := strconv.FormatFloat(f, 'e', -1, 64)
	sign := ""
	if strings.HasPrefix(sci, "-") {
		sign, sci = "-", sci[1:]
	}
	mantissa, expText, _ := strings.Cut(sci, "e")
	exp, err := strconv.Atoi(expText)
	if err != nil {
		// FormatFloat always writes a decimal exponent.
		panic(err)
	}
	digits := strings.Replace(mantissa, ".", "", 1)
	if exp < -4 || exp >= 16 {
		m := digits[:1]
		if len(digits) > 1 {
			m += "." + digits[1:]
		}
		expSign := "+"
		if exp < 0 {
			expSign, exp = "-", -exp
		}
		return fmt.Sprintf("%s%se%s%02d", sign, m, expSign, exp)
	}
	if exp < 0 {
		return sign + "0." + strings.Repeat("0", -exp-1) + digits
	}
	if len(digits) <= exp+1 {
		return sign + digits + strings.Repeat("0", exp+1-len(digits)) + ".0"
	}
	return sign + digits[:exp+1] + "." + digits[exp+1:]
}
