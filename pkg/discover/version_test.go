// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package discover

import "testing"

func TestDiscoveryVersion(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    DiscoveryVersion
		wantErr bool
	}{
		{name: "v1", value: "1", want: DiscoveryMethodV1},
		{name: "v2", value: "2", want: DiscoveryMethodV2},
		{name: "non-numeric", value: "latest", want: DiscoveryMethodV1, wantErr: true},
		{name: "unsupported", value: "3", want: DiscoveryMethodV1, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DiscoveryMethodV1
			err := got.Set(tt.value)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Set(%q) error = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("Set(%q) version = %v, want %v", tt.value, got, tt.want)
			}
			if got.String() != tt.want.String() {
				t.Errorf("String() = %q, want %q", got.String(), tt.want.String())
			}
			if got.Type() != "DiscoveryVersion" {
				t.Errorf("Type() = %q, want DiscoveryVersion", got.Type())
			}
		})
	}
}
