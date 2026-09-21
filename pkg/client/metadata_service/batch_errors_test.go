// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package metadata_service

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"

	metadata_service_client "github.com/openchami/metadata-service/pkg/client"
)

// TestMetadataServiceBatch_AllFailure verifies that AddGroups sends one request
// per item and reports an error for every item when every request fails.
func TestMetadataServiceBatch_AllFailure(t *testing.T) {
	var requests atomic.Int32
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		http.Error(w, "not found", http.StatusNotFound)
	})
	defer srv.Close()

	ctx := context.Background()
	reqs := []metadata_service_client.CreateGroupRequest{{}, {}}
	results := c.AddGroups(ctx, "", reqs)

	if len(results) != len(reqs) {
		t.Fatalf("len(results) = %d, want %d", len(results), len(reqs))
	}
	for i, result := range results {
		if result.Err == nil {
			t.Errorf("results[%d].Err = nil, want non-nil", i)
		}
	}
	if got := requests.Load(); got != int32(len(reqs)) {
		t.Errorf("server requests = %d, want %d", got, len(reqs))
	}
}
