// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cloudinit

import (
	"encoding/base64"
	"fmt"
	"math"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Timestamp is a YAML timestamp as PyYAML constructs it: a datetime.date when
// DateOnly is set, and otherwise a datetime.datetime, which is timezone-aware
// when Aware is set.
type Timestamp struct {
	Time     time.Time
	DateOnly bool
	Aware    bool
}

// String formats t as Python's str does for a date or datetime.
func (t Timestamp) String() string {
	if t.DateOnly {
		return t.Time.Format("2006-01-02")
	}
	s := t.Time.Format("2006-01-02 15:04:05")
	if usec := t.Time.Nanosecond() / 1000; usec != 0 {
		s += fmt.Sprintf(".%06d", usec)
	}
	if t.Aware {
		_, offset := t.Time.Zone()
		sign := "+"
		if offset < 0 {
			sign, offset = "-", -offset
		}
		s += fmt.Sprintf("%s%02d:%02d", sign, offset/3600, offset/60%60)
		if offset%60 != 0 {
			s += fmt.Sprintf(":%02d", offset%60)
		}
	}
	return s
}

// YAML 1.1 tags that PyYAML's SafeLoader resolves for plain scalars.
const (
	tagBool      = "!!bool"
	tagFloat     = "!!float"
	tagInt       = "!!int"
	tagMerge     = "!!merge"
	tagNull      = "!!null"
	tagTimestamp = "!!timestamp"
	tagValue     = "!!value"
	tagStr       = "!!str"
	tagBinary    = "!!binary"
)

// implicitResolvers mirrors PyYAML's Resolver.add_implicit_resolver calls, in
// registration order. Each applies only to plain scalars whose first character
// is listed. An empty scalar uses the resolvers registered for "".
var implicitResolvers = []struct {
	tag    string
	first  string
	empty  bool
	regexp *regexp.Regexp
}{
	{tagBool, "yYnNtTfFoO", false, regexp.MustCompile(`^(?:yes|Yes|YES|no|No|NO|true|True|TRUE|false|False|FALSE|on|On|ON|off|Off|OFF)$`)},
	{tagFloat, "-+0123456789.", false, regexp.MustCompile(`^(?:[-+]?(?:[0-9][0-9_]*)\.[0-9_]*(?:[eE][-+][0-9]+)?|\.[0-9][0-9_]*(?:[eE][-+][0-9]+)?|[-+]?[0-9][0-9_]*(?::[0-5]?[0-9])+\.[0-9_]*|[-+]?\.(?:inf|Inf|INF)|\.(?:nan|NaN|NAN))$`)},
	{tagInt, "-+0123456789", false, regexp.MustCompile(`^(?:[-+]?0b[0-1_]+|[-+]?0[0-7_]+|[-+]?(?:0|[1-9][0-9_]*)|[-+]?0x[0-9a-fA-F_]+|[-+]?[1-9][0-9_]*(?::[0-5]?[0-9])+)$`)},
	{tagMerge, "<", false, regexp.MustCompile(`^(?:<<)$`)},
	{tagNull, "~nN", true, regexp.MustCompile(`^(?:~|null|Null|NULL|)$`)},
	{tagTimestamp, "0123456789", false, regexp.MustCompile(`^(?:[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]|[0-9][0-9][0-9][0-9]-[0-9][0-9]?-[0-9][0-9]?(?:[Tt]|[ \t]+)[0-9][0-9]?:[0-9][0-9]:[0-9][0-9](?:\.[0-9]*)?(?:[ \t]*(?:Z|[-+][0-9][0-9]?(?::[0-9][0-9])?))?)$`)},
	{tagValue, "=", false, regexp.MustCompile(`^(?:=)$`)},
}

// timestampPattern mirrors SafeConstructor.timestamp_regexp.
var timestampPattern = regexp.MustCompile(`^([0-9][0-9][0-9][0-9])-([0-9][0-9]?)-([0-9][0-9]?)(?:(?:[Tt]|[ \t]+)([0-9][0-9]?):([0-9][0-9]):([0-9][0-9])(?:\.([0-9]*))?(?:[ \t]*(Z|([-+])([0-9][0-9]?)(?::([0-9][0-9]))?))?)?$`)

// resolvePlain returns the tag PyYAML's SafeLoader assigns to a plain scalar.
func resolvePlain(value string) string {
	for _, r := range implicitResolvers {
		if value == "" {
			if !r.empty {
				continue
			}
		} else if !strings.ContainsRune(r.first, []rune(value)[0]) {
			continue
		}
		if r.regexp.MatchString(value) {
			return r.tag
		}
	}
	return tagStr
}

// constructScalar converts a scalar with a resolved or explicit tag as
// PyYAML's SafeConstructor does.
func constructScalar(tag, value string) (any, error) {
	switch tag {
	case tagStr:
		return value, nil
	case tagNull:
		return nil, nil
	case tagBool:
		switch strings.ToLower(value) {
		case "yes", "true", "on":
			return true, nil
		case "no", "false", "off":
			return false, nil
		}
		return nil, fmt.Errorf("invalid boolean %q", value)
	case tagInt:
		return constructInt(value)
	case tagFloat:
		return constructFloat(value)
	case tagTimestamp:
		return constructTimestamp(value)
	case tagBinary:
		decoded, err := base64.StdEncoding.DecodeString(strings.Join(strings.Fields(value), ""))
		if err != nil {
			return nil, fmt.Errorf("invalid binary data: %w", err)
		}
		return decoded, nil
	case tagValue:
		return nil, fmt.Errorf("could not determine a constructor for the tag 'tag:yaml.org,2002:value'")
	default:
		return nil, fmt.Errorf("unsupported YAML tag %q", tag)
	}
}

// constructInt mirrors SafeConstructor.construct_yaml_int. Values outside
// signed 64-bit range are rejected.
func constructInt(value string) (int64, error) {
	s := strings.ReplaceAll(value, "_", "")
	negative := false
	if s != "" && (s[0] == '-' || s[0] == '+') {
		negative = s[0] == '-'
		s = s[1:]
	}
	n := new(big.Int)
	var ok bool
	switch {
	case s == "0":
		ok = true
	case strings.HasPrefix(s, "0b"):
		_, ok = n.SetString(s[2:], 2)
	case strings.HasPrefix(s, "0x"):
		_, ok = n.SetString(s[2:], 16)
	case strings.HasPrefix(s, "0"):
		_, ok = n.SetString(s, 8)
	case strings.Contains(s, ":"):
		ok = true
		base := big.NewInt(1)
		parts := strings.Split(s, ":")
		for i := len(parts) - 1; i >= 0; i-- {
			digit, valid := new(big.Int).SetString(parts[i], 10)
			if !valid {
				ok = false
				break
			}
			n.Add(n, digit.Mul(digit, base))
			base.Mul(base, big.NewInt(60))
		}
	default:
		_, ok = n.SetString(s, 10)
	}
	if !ok {
		return 0, fmt.Errorf("invalid integer %q", value)
	}
	if negative {
		n.Neg(n)
	}
	if !n.IsInt64() {
		return 0, fmt.Errorf("integer outside signed 64-bit range: %q", value)
	}
	return n.Int64(), nil
}

// constructFloat mirrors SafeConstructor.construct_yaml_float.
func constructFloat(value string) (float64, error) {
	s := strings.ToLower(strings.ReplaceAll(value, "_", ""))
	sign := 1.0
	if s != "" && s[0] == '-' {
		sign = -1
	}
	if s != "" && (s[0] == '-' || s[0] == '+') {
		s = s[1:]
	}
	switch {
	case s == ".inf":
		return sign * math.Inf(1), nil
	case s == ".nan":
		return math.NaN(), nil
	case strings.Contains(s, ":"):
		parts := strings.Split(s, ":")
		total, base := 0.0, 1.0
		for i := len(parts) - 1; i >= 0; i-- {
			digit, err := strconv.ParseFloat(parts[i], 64)
			if err != nil {
				return 0, fmt.Errorf("invalid float %q", value)
			}
			total += digit * base
			base *= 60
		}
		return sign * total, nil
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil && !strings.Contains(err.Error(), "value out of range") {
		return 0, fmt.Errorf("invalid float %q", value)
	}
	return sign * f, nil
}

// constructTimestamp mirrors SafeConstructor.construct_yaml_timestamp,
// including Python's validation of each field.
func constructTimestamp(value string) (Timestamp, error) {
	m := timestampPattern.FindStringSubmatch(value)
	if m == nil {
		return Timestamp{}, fmt.Errorf("invalid timestamp %q", value)
	}
	atoi := func(s string) int {
		n, _ := strconv.Atoi(s) //nolint:errcheck // the pattern only matches digits
		return n
	}
	year, month, day := atoi(m[1]), atoi(m[2]), atoi(m[3])
	valid := func(t time.Time) bool {
		return t.Year() == year && int(t.Month()) == month && t.Day() == day
	}
	if m[4] == "" {
		t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
		if year < 1 || !valid(t) {
			return Timestamp{}, fmt.Errorf("invalid date %q", value)
		}
		return Timestamp{Time: t, DateOnly: true}, nil
	}
	hour, minute, second := atoi(m[4]), atoi(m[5]), atoi(m[6])
	if hour > 23 || minute > 59 || second > 59 {
		return Timestamp{}, fmt.Errorf("invalid time %q", value)
	}
	usec := 0
	if m[7] != "" {
		fraction := m[7]
		if len(fraction) > 6 {
			fraction = fraction[:6]
		}
		usec = atoi(fraction + strings.Repeat("0", 6-len(fraction)))
	}
	loc, aware := time.UTC, false
	switch {
	case m[9] != "":
		offset := (atoi(m[10])*60 + atoi(m[11])) * 60
		if offset >= 24*3600 {
			return Timestamp{}, fmt.Errorf("invalid time zone offset %q", value)
		}
		if m[9] == "-" {
			offset = -offset
		}
		loc, aware = time.FixedZone("", offset), true
	case m[8] == "Z":
		aware = true
	}
	t := time.Date(year, time.Month(month), day, hour, minute, second, usec*1000, loc)
	if year < 1 || !valid(t) {
		return Timestamp{}, fmt.Errorf("invalid date %q", value)
	}
	return Timestamp{Time: t, Aware: aware}, nil
}

// pyKeyString returns the string a non-string mapping key becomes when
// cloud-init writes it to JSON, so instance data and merged configuration keep
// string keys.
func pyKeyString(key any) string {
	switch k := key.(type) {
	case string:
		return k
	case nil:
		return "null"
	case bool:
		return strconv.FormatBool(k)
	case int64:
		return strconv.FormatInt(k, 10)
	case float64:
		return pyJSONFloat(k)
	default:
		return fmt.Sprint(k)
	}
}

// checkTabs rejects tab characters where PyYAML's scanner does. PyYAML allows
// tabs only inside quoted scalars, block scalar content after its
// indentation, and comments. yaml.v3 also accepts tabs as separators, so
// without this check a document that cloud-init cannot load would decode.
func checkTabs(data []byte) error {
	var quote byte
	block, blockIndent, parentIndent := false, -1, 0
	for lineno, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSuffix(line, "\r")
		indent := len(line) - len(strings.TrimLeft(line, " "))
		if block {
			rest := line[indent:]
			switch {
			case rest == "":
				continue
			case blockIndent < 0 && indent > parentIndent && rest[0] != '\t':
				blockIndent = indent
				continue
			case blockIndent >= 0 && indent >= blockIndent:
				continue
			}
			block, blockIndent = false, -1
		}
		significant := line
		for i := 0; i < len(line); i++ {
			c := line[i]
			if quote != 0 {
				switch {
				case quote == '"' && c == '\\':
					i++
				case quote == '\'' && c == '\'' && i+1 < len(line) && line[i+1] == '\'':
					i++
				case c == quote:
					quote = 0
				}
				continue
			}
			switch {
			case c == '\t':
				return fmt.Errorf("line %d: found a tab character where YAML 1.1 does not allow one", lineno+1)
			case c == '#' && (i == 0 || line[i-1] == ' '):
				significant = line[:i]
				i = len(line)
			case (c == '"' || c == '\'') && (i == indent || strings.ContainsRune(" [{,", rune(line[i-1]))):
				quote = c
			}
		}
		if quote == 0 && blockHeader.MatchString(significant) {
			block, blockIndent, parentIndent = true, -1, indent
		}
	}
	return nil
}

// blockHeader matches a line that ends with a block scalar indicator.
var blockHeader = regexp.MustCompile(`(?:^|[\s:\-?])[|>][-+0-9]*\s*$`)
