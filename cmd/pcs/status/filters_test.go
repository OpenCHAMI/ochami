// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package status

// filters_test.go directly exercises the PowerFilter and MgmtFilter flag types
// and their shell-completion helpers, which are only reached via cobra's
// completion machinery during normal execution.

import (
	"testing"
)

func TestPowerFilterSetAndType(t *testing.T) {
	var pf PowerFilter
	for _, v := range []string{"on", "OFF", "Undefined"} {
		if err := pf.Set(v); err != nil {
			t.Errorf("PowerFilter.Set(%q) = %v, want nil", v, err)
		}
	}
	if err := pf.Set("bogus"); err == nil {
		t.Error("PowerFilter.Set(bogus) = nil, want error")
	}
	if pf.Type() != "PowerFilter" {
		t.Errorf("PowerFilter.Type() = %q, want PowerFilter", pf.Type())
	}
	_ = pf.String()

	comps, _ := pcsStatusListPowerFilterCompletion(nil, nil, "")
	if len(comps) == 0 {
		t.Error("power filter completion returned no entries")
	}
}

func TestMgmtFilterSetAndType(t *testing.T) {
	var mf MgmtFilter
	for _, v := range []string{"available", "Unavailable"} {
		if err := mf.Set(v); err != nil {
			t.Errorf("MgmtFilter.Set(%q) = %v, want nil", v, err)
		}
	}
	if err := mf.Set("bogus"); err == nil {
		t.Error("MgmtFilter.Set(bogus) = nil, want error")
	}
	if mf.Type() != "MgmtFilter" {
		t.Errorf("MgmtFilter.Type() = %q, want MgmtFilter", mf.Type())
	}
	_ = mf.String()

	comps, _ := pcsStatusListMgmtFilterCompletion(nil, nil, "")
	if len(comps) == 0 {
		t.Error("mgmt filter completion returned no entries")
	}
}
