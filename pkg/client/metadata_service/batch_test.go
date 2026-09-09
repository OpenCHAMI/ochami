// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package metadata_service

import (
	"context"
	"errors"
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
