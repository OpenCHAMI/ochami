// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package config

import (
	"errors"
	"testing"
)

// TestMergeURIConfig_IncludesEveryService verifies that MergeURIConfig also
// merges the boot-service and metadata-service URIs.
func TestMergeURIConfig_IncludesEveryService(t *testing.T) {
	old := ClusterConfig{
		MetadataService: ClusterMetadataService{URI: "/old-metadata"},
		BootService:     ClusterBootService{URI: "/old-boot"},
	}
	got := old.MergeURIConfig(ClusterConfig{
		MetadataService: ClusterMetadataService{URI: "/new-metadata"},
		BootService:     ClusterBootService{URI: "/new-boot"},
	})
	if got.MetadataService.URI != "/new-metadata" || got.BootService.URI != "/new-boot" {
		t.Fatalf("MergeURIConfig() = %#v", got)
	}
}

// TestGetServiceBaseURI_Boundaries verifies the typed error GetServiceBaseURI
// returns for a missing, invalid, malformed, opaque, or unanchored service URI
// and for an unknown service.
func TestGetServiceBaseURI_Boundaries(t *testing.T) {
	tests := []struct {
		name string
		cfg  ClusterConfig
		svc  ServiceName
		as   any
	}{
		{name: "missing boot URI", svc: ServiceBoot, as: &ErrMissingURI{}},
		{name: "missing metadata URI", svc: ServiceMetadata, as: &ErrMissingURI{}},
		{name: "missing PCS URI", svc: ServicePCS, as: &ErrMissingURI{}},
		{name: "missing SMD URI", svc: ServiceSMD, as: &ErrMissingURI{}},
		{name: "missing RCS URI", svc: ServiceRCS, as: &ErrMissingURI{}},
		{name: "invalid cluster URI", cfg: ClusterConfig{URI: "relative"}, svc: ServiceSMD, as: &ErrInvalidURI{}},
		{name: "malformed service URI", cfg: ClusterConfig{SMD: ClusterSMD{URI: "%zz"}}, svc: ServiceSMD, as: &ErrInvalidServiceURI{}},
		{name: "opaque service URI", cfg: ClusterConfig{SMD: ClusterSMD{URI: "mailto:user@example.com"}}, svc: ServiceSMD, as: &ErrInvalidServiceURI{}},
		{name: "relative service without cluster", cfg: ClusterConfig{SMD: ClusterSMD{URI: "/smd"}}, svc: ServiceSMD, as: &ErrInvalidServiceURI{}},
		{name: "empty service URL", cfg: ClusterConfig{SMD: ClusterSMD{URI: "?query=yes"}}, svc: ServiceSMD, as: &ErrInvalidServiceURI{}},
		{name: "unknown service", cfg: ClusterConfig{URI: "https://cluster.example"}, svc: ServiceName("unknown"), as: &ErrUnknownService{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.cfg.GetServiceBaseURI(tt.svc)
			if err == nil || !errors.As(err, tt.as) {
				t.Fatalf("GetServiceBaseURI() error = %v, want %T", err, tt.as)
			}
		})
	}
}
