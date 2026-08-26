// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package config

import (
	"reflect"
	"testing"
)

// TestClusterAccumulator_Order verifies that clusters are emitted in the order
// their names are first seen, regardless of how many sources contribute or in
// what order names recur. This is the deterministic-ordering guarantee relied
// upon by both LoadGlobalConfigMerged and ReadConfigWithDefaults.
func TestClusterAccumulator_Order(t *testing.T) {
	// Run repeatedly to guard against Go map iteration nondeterminism.
	for iter := 0; iter < 10; iter++ {
		ca := newClusterAccumulator()

		// Simulate three sources (default/system/user) contributing
		// clusters, with some names recurring across sources.
		mustAdd(t, ca, "zeta", map[string]any{"uri": "https://zeta-1"})
		mustAdd(t, ca, "alpha", map[string]any{"uri": "https://alpha-1"})
		mustAdd(t, ca, "mu", map[string]any{"uri": "https://mu-1"})
		// Recurrences should not change ordering.
		mustAdd(t, ca, "alpha", map[string]any{"uri": "https://alpha-2"})
		mustAdd(t, ca, "beta", map[string]any{"uri": "https://beta-1"})
		mustAdd(t, ca, "zeta", map[string]any{"uri": "https://zeta-2"})

		want := []string{"zeta", "alpha", "mu", "beta"}
		got := make([]string, 0, len(want))
		for _, c := range ca.slice() {
			got = append(got, c["name"].(string))
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("iteration %d: order = %v, want %v", iter, got, want)
		}
	}
}

// TestClusterAccumulator_Precedence verifies that later adds for the same
// cluster name take precedence (higher-priority sources are added last).
func TestClusterAccumulator_Precedence(t *testing.T) {
	ca := newClusterAccumulator()
	mustAdd(t, ca, "foo", map[string]any{"uri": "https://low"})
	mustAdd(t, ca, "foo", map[string]any{"uri": "https://high"})

	sl := ca.slice()
	if len(sl) != 1 {
		t.Fatalf("got %d clusters, want 1", len(sl))
	}
	cl := sl[0]["cluster"].(map[string]any)
	if cl["uri"] != "https://high" {
		t.Errorf("uri = %v, want https://high (last add should win)", cl["uri"])
	}
	// The default (enable-auth: true) should still be present.
	if b, ok := cl["enable-auth"].(bool); !ok || !b {
		t.Errorf("enable-auth = %v, want true (default applied)", cl["enable-auth"])
	}
}

func mustAdd(t *testing.T, ca *clusterAccumulator, name string, cluster map[string]any) {
	t.Helper()
	if err := ca.add(name, cluster); err != nil {
		t.Fatalf("add(%q): %v", name, err)
	}
}
