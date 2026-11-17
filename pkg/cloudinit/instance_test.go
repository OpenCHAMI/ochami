// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cloudinit

import (
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"

	gj "github.com/jryberg/gojinja"
)

// v1For returns the v1 mapping built for a JSON metadata document.
func v1For(t *testing.T, metadata string) *gj.OrderedDict {
	t.Helper()
	data, err := buildInstanceData(RenderRequest{Metadata: Document{Data: []byte(metadata)}})
	if err != nil {
		t.Fatal(err)
	}
	v1, _ := data.Get("v1")
	return v1.(*gj.OrderedDict)
}

// TestStandardizedMetadata verifies the v1 keys derived from metadata as
// cloud-init's NoCloud datasource derives them.
func TestStandardizedMetadata(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		metadata string
		key      string
		want     any
		absent   bool
	}{
		{name: "instance id", metadata: `{"instance-id":"i-1"}`, key: "instance_id", want: "i-1"},
		{name: "default instance id", metadata: `{}`, key: "instance-id", want: "iid-datasource"},
		{name: "short hostname", metadata: `{"local-hostname":"n1.example.com"}`, key: "local_hostname", want: "n1"},
		{name: "address hostname", metadata: `{"local-hostname":"10.1.2.3"}`, key: "local-hostname", want: "ip-10-1-2-3"},
		{name: "missing hostname", metadata: `{}`, key: "local_hostname", absent: true},
		{name: "empty hostname", metadata: `{"local-hostname":""}`, key: "local_hostname", absent: true},
		{name: "unknown cloud", metadata: `{}`, key: "cloud_name", want: "unknown"},
		{name: "named cloud", metadata: `{"cloud-name":"OpenCHAMI"}`, key: "cloud-name", want: "openchami"},
		{name: "cloud id from platform", metadata: `{"region":"r1"}`, key: "cloud_id", want: "nocloud"},
		{name: "cloud id from name", metadata: `{"cloud-name":"demo"}`, key: "cloud_id", want: "demo"},
		{name: "cloud id region prefix", metadata: `{"cloud-name":"aws","region":"cn-north-1"}`, key: "cloud_id", want: "aws-china"},
		{name: "region prefix other cloud", metadata: `{"cloud-name":"demo","region":"cn-north-1"}`, key: "cloud_id", want: "demo"},
		{name: "zone", metadata: `{"availability-zone":"z1"}`, key: "availability_zone", want: "z1"},
		{name: "underscore zone", metadata: `{"availability_zone":"z2"}`, key: "availability-zone", want: "z2"},
		{name: "placement zone", metadata: `{"placement":{"availability-zone":"z3"}}`, key: "availability_zone", want: "z3"},
		{name: "no zone", metadata: `{}`, key: "availability_zone", want: nil},
		{name: "region", metadata: `{"region":"r1"}`, key: "region", want: "r1"},
		{name: "keys string", metadata: `{"public-keys":"a\nb\r\nc"}`, key: "public_ssh_keys", want: []any{"a", "b", "c"}},
		{name: "keys list", metadata: `{"public-keys":["a","b"]}`, key: "public_ssh_keys", want: []any{"a", "b"}},
		{name: "keys mapping", metadata: `{"public-keys":{"x":"a","y":["b",""]}}`, key: "public_ssh_keys", want: []any{"a", "b"}},
		{name: "no keys", metadata: `{}`, key: "public_ssh_keys", want: []any{}},
		{name: "platform", metadata: `{}`, key: "platform", want: "nocloud"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := v1For(t, tt.metadata).Get(tt.key)
			if tt.absent {
				if ok {
					t.Fatalf("v1.%s = %#v, want absent", tt.key, got)
				}
				return
			}
			if !ok || !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("v1.%s = %#v (present %v), want %#v", tt.key, got, ok, tt.want)
			}
		})
	}
}

// TestBuildInstanceData_Sources verifies instance data, metadata, and extra
// variable precedence.
func TestBuildInstanceData_Sources(t *testing.T) {
	t.Parallel()
	t.Run("instance data used as given", func(t *testing.T) {
		t.Parallel()
		data, err := buildInstanceData(RenderRequest{InstanceData: Document{Data: []byte(`{"v1":{"distro":"rocky"},"b":1,"a":2}`)}})
		if err != nil {
			t.Fatal(err)
		}
		if got := sortedKeys(data); !reflect.DeepEqual(got, []string{"a", "b", "v1"}) {
			t.Fatalf("keys = %v", got)
		}
		v1, _ := data.Get("v1")
		if keys := sortedKeys(v1.(*gj.OrderedDict)); !reflect.DeepEqual(keys, []string{"distro"}) {
			t.Fatalf("v1 keys = %v, want only distro", keys)
		}
	})
	t.Run("extra variables replace v1", func(t *testing.T) {
		t.Parallel()
		data, err := buildInstanceData(RenderRequest{
			Metadata:  Document{Data: []byte(`{"instance-id":"i-1"}`)},
			ExtraVars: Document{Data: []byte(`{"v1":{"variant":"suse"}}`)},
		})
		if err != nil {
			t.Fatal(err)
		}
		v1, _ := data.Get("v1")
		if keys := sortedKeys(v1.(*gj.OrderedDict)); !reflect.DeepEqual(keys, []string{"variant"}) {
			t.Fatalf("v1 keys = %v, want only variant", keys)
		}
	})
	t.Run("instance data excludes metadata", func(t *testing.T) {
		t.Parallel()
		_, err := buildInstanceData(RenderRequest{
			Metadata:     Document{Data: []byte(`{}`)},
			InstanceData: Document{Data: []byte(`{}`)},
		})
		var renderErr *RenderError
		if !errors.As(err, &renderErr) || renderErr.Stage != StageInstanceData {
			t.Fatalf("error = %v, want instance-data stage", err)
		}
	})
}

// TestPyJSONDumps verifies the json_dumps format of instance-data.json.
func TestPyJSONDumps(t *testing.T) {
	t.Parallel()
	value := gj.NewOrderedDict()
	value.Set("z", []any{int64(1), 2.5, true, nil})
	value.Set("a", gj.NewOrderedDict())
	value.Set("s", "é\"\x7f😀")
	value.Set("e", []any{})
	want := strings.Join([]string{
		"{",
		` "a": {},`,
		` "e": [],`,
		` "s": "\u00e9\"\u007f\ud83d\ude00",`,
		` "z": [`,
		`  1,`,
		`  2.5,`,
		`  true,`,
		`  null`,
		` ]`,
		"}",
	}, "\n")
	if got := pyJSONDumps(value); got != want {
		t.Fatalf("pyJSONDumps() =\n%s\nwant\n%s", got, want)
	}
}

// TestPyFloatRepr verifies Python's float repr.
func TestPyFloatRepr(t *testing.T) {
	t.Parallel()
	tests := map[float64]string{
		0:                    "0.0",
		math.Copysign(0, -1): "-0.0",
		1:                    "1.0",
		1.5:                  "1.5",
		-2.25:                "-2.25",
		100:                  "100.0",
		0.0001:               "0.0001",
		0.00001:              "1e-05",
		1e15:                 "1000000000000000.0",
		1e16:                 "1e+16",
		1.5e16:               "1.5e+16",
		123456789.5:          "123456789.5",
		1e-300:               "1e-300",
		0.1:                  "0.1",
	}
	for in, want := range tests {
		if got := pyFloatRepr(in); got != want {
			t.Errorf("pyFloatRepr(%v) = %q, want %q", in, got, want)
		}
	}
	if got := pyJSONFloat(math.Inf(-1)); got != "-Infinity" {
		t.Errorf("pyJSONFloat(-Inf) = %q", got)
	}
}

// TestPySplitlines verifies Python's str.splitlines.
func TestPySplitlines(t *testing.T) {
	t.Parallel()
	tests := map[string][]string{
		"":          nil,
		"a":         {"a"},
		"a\n":       {"a"},
		"a\n\nb":    {"a", "", "b"},
		"a\r\nb\rc": {"a", "b", "c"},
		"a b\x0cc":  {"a", "b", "c"},
	}
	for in, want := range tests {
		if got := pySplitlines(in); !reflect.DeepEqual(got, want) {
			t.Errorf("pySplitlines(%q) = %q, want %q", in, got, want)
		}
	}
}
