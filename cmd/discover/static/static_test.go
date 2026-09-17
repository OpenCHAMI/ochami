// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package static

// static_test.go covers the pure helper functions extracted from the static
// discovery command: buildGroupList (group assembly and deduplication) and
// discoverStaticDeprecatedFormat (deprecated-format detection).

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"testing"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/pkg/client"
	"github.com/openchami/ochami/pkg/client/smd"
)

func TestSingleBatchResult(t *testing.T) {
	want := errors.New("batch failed")
	if _, err := singleBatchResult(nil, nil, want); !errors.Is(err, want) {
		t.Fatalf("singleBatchResult() error = %v, want batch error", err)
	}
	if _, err := singleBatchResult(nil, nil, nil); err == nil {
		t.Fatal("singleBatchResult() accepted misaligned empty results")
	}
	henv := client.HTTPEnvelope{StatusCode: http.StatusCreated}
	got, err := singleBatchResult([]client.HTTPEnvelope{henv}, []error{nil}, nil)
	if err != nil || got.StatusCode != http.StatusCreated {
		t.Fatalf("singleBatchResult() = (%v, %v)", got, err)
	}
}

func TestUpsertOnConflict(t *testing.T) {
	conflict := fmt.Errorf("%w: conflict", client.UnsuccessfulHTTPError)
	wantUpdateErr := errors.New("update failed")
	updates := 0
	errs := upsertOnConflict([]string{"create", "replace", "fail"},
		func(item string) (client.HTTPEnvelope, error) {
			switch item {
			case "create":
				return client.HTTPEnvelope{StatusCode: http.StatusCreated}, nil
			case "replace":
				return client.HTTPEnvelope{StatusCode: http.StatusConflict}, conflict
			default:
				return client.HTTPEnvelope{StatusCode: http.StatusBadRequest}, fmt.Errorf("%w: bad request", client.UnsuccessfulHTTPError)
			}
		},
		func(string) (client.HTTPEnvelope, error) {
			updates++
			return client.HTTPEnvelope{}, wantUpdateErr
		})
	if updates != 1 {
		t.Errorf("updates = %d, want 1", updates)
	}
	if len(errs) != 2 || !errors.Is(errs[0], wantUpdateErr) || !errors.Is(errs[1], client.UnsuccessfulHTTPError) {
		t.Errorf("errors = %v, want update and create errors", errs)
	}
}

func groupByLabel(groups []smd.Group) map[string]smd.Group {
	m := make(map[string]smd.Group, len(groups))
	for _, g := range groups {
		m[g.Label] = g
	}
	return m
}

func TestBuildGroupList(t *testing.T) {
	tests := []struct {
		name  string
		nodes []nodeCommon
		// wantMembers maps group label -> sorted expected members.
		wantMembers map[string][]string
	}{
		{
			name:        "no nodes yields no groups",
			nodes:       nil,
			wantMembers: map[string][]string{},
		},
		{
			name: "single node single group",
			nodes: []nodeCommon{
				{Name: "n0", Xname: "x0", Groups: []string{"compute"}},
			},
			wantMembers: map[string][]string{
				"compute": {"x0"},
			},
		},
		{
			name: "multiple nodes shared group deduplicated per member",
			nodes: []nodeCommon{
				{Name: "n0", Xname: "x0", Groups: []string{"compute"}},
				{Name: "n1", Xname: "x1", Groups: []string{"compute", "slurm"}},
			},
			wantMembers: map[string][]string{
				"compute": {"x0", "x1"},
				"slurm":   {"x1"},
			},
		},
		{
			name: "deprecated group field merged with groups",
			nodes: []nodeCommon{
				{Name: "n0", Xname: "x0", Group: "legacy", Groups: []string{"compute"}},
			},
			wantMembers: map[string][]string{
				"legacy":  {"x0"},
				"compute": {"x0"},
			},
		},
		{
			name: "deprecated group with blank name still added",
			nodes: []nodeCommon{
				{Name: "", Xname: "x9", Group: "legacy"},
			},
			wantMembers: map[string][]string{
				"legacy": {"x9"},
			},
		},
	}

	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			got := buildGroupList(tc.nodes)
			if len(got) != len(tc.wantMembers) {
				t.Fatalf("group count = %d, want %d (%v)", len(got), len(tc.wantMembers), got)
			}
			byLabel := groupByLabel(got)
			for label, wantMembers := range tc.wantMembers {
				g, ok := byLabel[label]
				if !ok {
					t.Errorf("missing group %q", label)
					continue
				}
				members := append([]string(nil), g.Members.IDs...)
				sort.Strings(members)
				sort.Strings(wantMembers)
				if len(members) != len(wantMembers) {
					t.Errorf("group %q members = %v, want %v", label, members, wantMembers)
					continue
				}
				for i := range members {
					if members[i] != wantMembers[i] {
						t.Errorf("group %q members = %v, want %v", label, members, wantMembers)
						break
					}
				}
			}
		})
	}
}

func TestDiscoverStaticDeprecatedFormat(t *testing.T) {
	tests := []struct {
		name string
		data map[string][]map[string]any
		want bool
	}{
		{
			name: "no nodes",
			data: map[string][]map[string]any{},
			want: false,
		},
		{
			name: "new format node",
			data: map[string][]map[string]any{
				"nodes": {{"name": "n0", "xname": "x0"}},
			},
			want: false,
		},
		{
			name: "deprecated bmc_ip key",
			data: map[string][]map[string]any{
				"nodes": {{"name": "n0", "bmc_ip": "172.16.0.1"}},
			},
			want: true,
		},
		{
			name: "deprecated bmc_mac key",
			data: map[string][]map[string]any{
				"nodes": {{"name": "n0", "bmc_mac": "de:ca:fc:0f:ee:ee"}},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			got := discoverStaticDeprecatedFormat(cmd, tc.data)
			if got != tc.want {
				t.Errorf("discoverStaticDeprecatedFormat = %v, want %v", got, tc.want)
			}
		})
	}
}
