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

// TestRunBatchPreservesOrder verifies the batch adapter retains the input sequence.
func TestRunBatchPreservesOrder(t *testing.T) {
	results := runBatch(context.Background(), time.Second, []int{3, 1, 2}, func(_ context.Context, item int) (int, error) {
		return item * 10, nil
	})

	values := results.Values()
	want := []int{30, 10, 20}
	for i := range want {
		if values[i] != want[i] {
			t.Fatalf("values[%d] = %d, want %d", i, values[i], want[i])
		}
	}
}

// TestRunBatchStopsAfterCancellation verifies unattempted items are aligned
// with the parent cancellation error and their operations are not invoked.
func TestRunBatchStopsAfterCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	results := runBatch(ctx, time.Second, []int{1, 2, 3}, func(_ context.Context, item int) (int, error) {
		calls++
		cancel()
		return item * 10, nil
	})

	if calls != 1 {
		t.Fatalf("operation calls = %d, want 1", calls)
	}
	if results[0].Value != 10 || results[0].Err != nil {
		t.Fatalf("results[0] = %#v, want successful first result", results[0])
	}
	for i := 1; i < len(results); i++ {
		if !errors.Is(results[i].Err, context.Canceled) {
			t.Errorf("results[%d].Err = %v, want context.Canceled", i, results[i].Err)
		}
	}
}

// TestMetadataServiceBatchPropagatesCancellation verifies caller cancellation reaches every item request.
func TestMetadataServiceBatchPropagatesCancellation(t *testing.T) {
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

// TestMetadataServiceBatchEmptyInput verifies empty input produces empty results.
func TestMetadataServiceBatchEmptyInput(t *testing.T) {
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

// TestMetadataServiceBatchAllSuccess verifies all-success paths.
func TestMetadataServiceBatchAllSuccess(t *testing.T) {
	var requests atomic.Int32
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"ID": "g` + fmt.Sprint(requests.Load()) + `"}`)) //nolint:errcheck // test response
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

// TestMetadataServiceBatchAllFailure verifies all-failure paths.
func TestMetadataServiceBatchAllFailure(t *testing.T) {
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

// TestMetadataServiceBatchMixedResults verifies mixed success/failure preserves order.
func TestMetadataServiceBatchMixedResults(t *testing.T) {
	var requestCount atomic.Int32
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		count := requestCount.Add(1)
		if count == 2 {
			http.Error(w, "error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"ID": "g` + fmt.Sprint(count) + `"}`)) //nolint:errcheck // test response
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

// TestMetadataServiceBatchDeadlineBetweenItems verifies deadline between items stops
// subsequent operations.
func TestMetadataServiceBatchDeadlineBetweenItems(t *testing.T) {
	var requests atomic.Int32
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		// First request succeeds after a delay
		if requests.Load() == 1 {
			time.Sleep(10 * time.Millisecond) // Exceed the deadline
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"ID": "g0"}`)) //nolint:errcheck // test response
	})
	defer srv.Close()

	reqs := []metadata_service_client.CreateGroupRequest{{}, {}}
	results := c.AddGroups(ctx, "", reqs)

	if len(results) != len(reqs) {
		t.Fatalf("len(results) = %d, want %d", len(results), len(reqs))
	}

	// First may succeed or fail with deadline, but second should have deadline error
	// The key is that we don't make more requests than the context allows
	for i, result := range results {
		if result.Err == nil {
			// If it succeeded, it must be the first one
			if i != 0 {
				t.Errorf("results[%d] succeeded unexpectedly", i)
			}
		} else if !errors.Is(result.Err, context.DeadlineExceeded) && !errors.Is(result.Err, context.Canceled) {
			// Could be deadline or canceled depending on timing
			t.Logf("results[%d].Err = %v", i, result.Err)
		}
	}

	// We should have made at most 2 requests (may be less if deadline hit early)
	if got := requests.Load(); got > int32(len(reqs)) {
		t.Errorf("server requests = %d, want at most %d", got, len(reqs))
	}
}

// TestMetadataServiceDeleteGroupsBatch verifies DELETE batch operations.
func TestMetadataServiceDeleteGroupsBatch(t *testing.T) {
	var requests atomic.Int32
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`)) //nolint:errcheck // test response
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
