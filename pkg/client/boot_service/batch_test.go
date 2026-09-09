// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package boot_service

import (
	"context"
	"errors"
	"testing"
	"time"

	boot_service_client "github.com/openchami/boot-service/pkg/client"
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

// TestBootServiceBatchPropagatesCancellation verifies caller cancellation reaches every item request.
func TestBootServiceBatchPropagatesCancellation(t *testing.T) {
	c, srv := newTestClient(t, nil)
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	results := c.AddNodes(ctx, "", []boot_service_client.CreateNodeRequest{{}, {}})
	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}
	for i, result := range results {
		if !errors.Is(result.Err, context.Canceled) {
			t.Errorf("results[%d].Err = %v, want context.Canceled", i, result.Err)
		}
	}
}
