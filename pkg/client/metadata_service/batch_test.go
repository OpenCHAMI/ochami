// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package metadata_service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	metadata_service_client "github.com/openchami/metadata-service/pkg/client"
)

// TestMetadataServiceBatch_PropagatesCancellation verifies caller cancellation reaches every item request.
func TestMetadataServiceBatch_PropagatesCancellation(t *testing.T) {
	var requests atomic.Int32
	c, srv := newTestClient(t, func(http.ResponseWriter, *http.Request) {
		requests.Add(1)
	})
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	results := c.AddGroups(ctx, "", []metadata_service_client.CreateGroupRequest{{}, {}})
	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}
	for i, result := range results {
		if !errors.Is(result.Err, context.Canceled) {
			t.Errorf("results[%d].Err = %v, want context.Canceled", i, result.Err)
		}
	}
	if got := requests.Load(); got != 0 {
		t.Errorf("server requests = %d, want 0", got)
	}
}

// TestMetadataServiceBatch_EmptyInput verifies empty input produces empty results.
func TestMetadataServiceBatch_EmptyInput(t *testing.T) {
	var requests atomic.Int32
	c, srv := newTestClient(t, func(http.ResponseWriter, *http.Request) {
		requests.Add(1)
	})
	defer srv.Close()

	ctx := context.Background()
	results := c.AddGroups(ctx, "", []metadata_service_client.CreateGroupRequest{})
	if len(results) != 0 {
		t.Fatalf("len(results) = %d, want 0", len(results))
	}
	if got := requests.Load(); got != 0 {
		t.Errorf("server requests = %d, want 0", got)
	}
}

// TestMetadataServiceBatch_AllSuccess verifies that AddGroups sends one request
// per item and reports no error when every request succeeds.
func TestMetadataServiceBatch_AllSuccess(t *testing.T) {
	var requests atomic.Int32
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"ID": "g` + fmt.Sprint(requests.Load()) + `"}`))
	})
	defer srv.Close()

	ctx := context.Background()
	reqs := []metadata_service_client.CreateGroupRequest{{}, {}}
	results := c.AddGroups(ctx, "", reqs)

	if len(results) != len(reqs) {
		t.Fatalf("len(results) = %d, want %d", len(results), len(reqs))
	}
	for i, result := range results {
		if result.Err != nil {
			t.Errorf("results[%d].Err = %v, want nil", i, result.Err)
		}
	}
	if got := requests.Load(); got != int32(len(reqs)) {
		t.Errorf("server requests = %d, want %d", got, len(reqs))
	}
}

// TestMetadataServiceBatch_MixedResults verifies that AddGroups reports the
// failure at the failed item's position when only the second of three requests
// fails.
func TestMetadataServiceBatch_MixedResults(t *testing.T) {
	var requestCount atomic.Int32
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		count := requestCount.Add(1)
		if count == 2 {
			http.Error(w, "error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"ID": "g` + fmt.Sprint(count) + `"}`))
	})
	defer srv.Close()

	ctx := context.Background()
	reqs := []metadata_service_client.CreateGroupRequest{{}, {}, {}}
	results := c.AddGroups(ctx, "", reqs)

	if len(results) != len(reqs) {
		t.Fatalf("len(results) = %d, want %d", len(results), len(reqs))
	}
	if results[0].Err != nil {
		t.Errorf("results[0].Err = %v, want nil", results[0].Err)
	}
	if results[1].Err == nil {
		t.Errorf("results[1].Err = nil, want non-nil")
	}
	if results[2].Err != nil {
		t.Errorf("results[2].Err = %v, want nil", results[2].Err)
	}
	if got := requestCount.Load(); got != int32(len(reqs)) {
		t.Errorf("server requests = %d, want %d", got, len(reqs))
	}
}

// TestMetadataServiceBatch_DeadlineStopsBatch verifies that a caller deadline
// that expires during the first item's request stops the batch: every item
// fails with context.DeadlineExceeded and no request is sent for the second.
func TestMetadataServiceBatch_DeadlineStopsBatch(t *testing.T) {
	var requests atomic.Int32
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		time.Sleep(10 * time.Millisecond) // Outlast the caller's deadline.
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"ID": "g0"}`))
	})
	defer srv.Close()

	reqs := []metadata_service_client.CreateGroupRequest{{}, {}}
	results := c.AddGroups(ctx, "", reqs)

	if len(results) != len(reqs) {
		t.Fatalf("len(results) = %d, want %d", len(results), len(reqs))
	}
	for i, result := range results {
		if !errors.Is(result.Err, context.DeadlineExceeded) {
			t.Errorf("results[%d].Err = %v, want context.DeadlineExceeded", i, result.Err)
		}
	}
	if got := requests.Load(); got > 1 {
		t.Errorf("server requests = %d, want at most 1", got)
	}
}

// TestMetadataServiceDeleteGroups_Batch verifies that DeleteGroups sends one
// request per ID and reports no error when every request succeeds.
func TestMetadataServiceDeleteGroups_Batch(t *testing.T) {
	var requests atomic.Int32
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	})
	defer srv.Close()

	ctx := context.Background()
	ids := []string{"g0", "g1"}
	results := c.DeleteGroups(ctx, "", ids)

	if len(results) != len(ids) {
		t.Fatalf("len(results) = %d, want %d", len(results), len(ids))
	}
	for i, result := range results {
		if result.Err != nil {
			t.Errorf("results[%d].Err = %v, want nil", i, result.Err)
		}
	}
	if got := requests.Load(); got != int32(len(ids)) {
		t.Errorf("server requests = %d, want %d", got, len(ids))
	}
}
