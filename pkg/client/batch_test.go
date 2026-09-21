// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package client

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestRunBatch verifies that RunBatch returns one result per item in input
// order, and that once the context is canceled or its deadline has passed it
// stops calling the operation and fills the remaining results with the
// context's error.
func TestRunBatch(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		results := RunBatch(context.Background(), []int(nil), func(context.Context, int) (HTTPEnvelope, error) {
			t.Fatal("operation called for empty input")
			return HTTPEnvelope{}, nil
		})
		if len(results) != 0 {
			t.Fatalf("len(results) = %d, want 0", len(results))
		}
	})

	t.Run("mixed results preserve order", func(t *testing.T) {
		itemErr := errors.New("item failed")
		results := RunBatch(context.Background(), []int{1, 2, 3}, func(_ context.Context, item int) (HTTPEnvelope, error) {
			result := HTTPEnvelope{StatusCode: item}
			if item == 2 {
				return result, itemErr
			}
			return result, nil
		})
		if len(results) != 3 {
			t.Fatalf("len(results) = %d, want 3", len(results))
		}
		for i, result := range results {
			if result.Value.StatusCode != i+1 {
				t.Errorf("results[%d].Value.StatusCode = %d, want %d", i, result.Value.StatusCode, i+1)
			}
		}
		if !errors.Is(results[1].Err, itemErr) || results[0].Err != nil || results[2].Err != nil {
			t.Fatalf("result errors = %v, want [nil, item error, nil]", results.Errors())
		}
	})

	t.Run("cancellation stops operations", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		calls := 0
		results := RunBatch(ctx, []int{1, 2, 3}, func(_ context.Context, item int) (HTTPEnvelope, error) {
			calls++
			cancel()
			return HTTPEnvelope{StatusCode: item}, nil
		})
		if calls != 1 {
			t.Fatalf("operation calls = %d, want 1", calls)
		}
		if results[0].Value.StatusCode != 1 || results[0].Err != nil {
			t.Fatalf("results[0] = %#v, want successful first result", results[0])
		}
		for i := 1; i < len(results); i++ {
			if !errors.Is(results[i].Err, context.Canceled) {
				t.Errorf("results[%d].Err = %v, want context.Canceled", i, results[i].Err)
			}
		}
	})

	t.Run("preexpired deadline", func(t *testing.T) {
		ctx, cancel := context.WithDeadline(context.Background(), time.Time{})
		defer cancel()
		calls := 0
		results := RunBatch(ctx, []int{1, 2}, func(context.Context, int) (HTTPEnvelope, error) {
			calls++
			return HTTPEnvelope{}, nil
		})
		if calls != 0 {
			t.Fatalf("operation calls = %d, want 0", calls)
		}
		for i := range results {
			if !errors.Is(results[i].Err, context.DeadlineExceeded) {
				t.Errorf("results[%d].Err = %v, want context.DeadlineExceeded", i, results[i].Err)
			}
		}
	})
}

// TestRunBatchWithTimeout verifies that RunBatchWithTimeout preserves input
// order, gives each item a context bounded by the timeout and released when the
// item finishes, and stops after the caller's context is canceled.
func TestRunBatchWithTimeout(t *testing.T) {
	t.Run("preserves order", func(t *testing.T) {
		results := RunBatchWithTimeout(context.Background(), time.Second, []int{3, 1, 2}, func(_ context.Context, item int) (int, error) {
			return item * 10, nil
		})
		values := results.Values()
		want := []int{30, 10, 20}
		for i := range want {
			if values[i] != want[i] {
				t.Fatalf("values[%d] = %d, want %d", i, values[i], want[i])
			}
		}
	})

	t.Run("bounds each item by the timeout", func(t *testing.T) {
		const timeout = time.Minute
		start := time.Now()
		results := RunBatchWithTimeout(context.Background(), timeout, []int{1, 2}, func(ctx context.Context, _ int) (time.Time, error) {
			deadline, ok := ctx.Deadline()
			if !ok {
				return time.Time{}, errors.New("item context has no deadline")
			}
			return deadline, nil
		})
		for i, result := range results {
			if result.Err != nil {
				t.Fatalf("results[%d].Err = %v", i, result.Err)
			}
			if result.Value.Before(start.Add(timeout)) || result.Value.After(time.Now().Add(timeout)) {
				t.Errorf("results[%d] deadline = %v, want about %v from the item's start", i, result.Value, timeout)
			}
		}
	})

	t.Run("releases each item context", func(t *testing.T) {
		var itemCtxs []context.Context
		RunBatchWithTimeout(context.Background(), time.Minute, []int{1, 2}, func(ctx context.Context, _ int) (int, error) {
			itemCtxs = append(itemCtxs, ctx)
			return 0, nil
		})
		for i, ctx := range itemCtxs {
			if !errors.Is(ctx.Err(), context.Canceled) {
				t.Errorf("item %d context err = %v, want context.Canceled once the item finished", i, ctx.Err())
			}
		}
	})

	t.Run("stops after cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		calls := 0
		results := RunBatchWithTimeout(ctx, time.Second, []int{1, 2, 3}, func(_ context.Context, item int) (int, error) {
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
	})
}
