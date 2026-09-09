// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package smd

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/openchami/ochami/pkg/client"
)

func TestExecuteBatch(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		results := executeBatch(context.Background(), []int(nil), func(context.Context, int) (client.HTTPEnvelope, error) {
			t.Fatal("operation called for empty input")
			return client.HTTPEnvelope{}, nil
		})
		if len(results) != 0 {
			t.Fatalf("len(results) = %d, want 0", len(results))
		}
	})

	t.Run("mixed results preserve order", func(t *testing.T) {
		itemErr := errors.New("item failed")
		results := executeBatch(context.Background(), []int{1, 2, 3}, func(_ context.Context, item int) (client.HTTPEnvelope, error) {
			result := client.HTTPEnvelope{StatusCode: item}
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
		results := executeBatch(ctx, []int{1, 2, 3}, func(_ context.Context, item int) (client.HTTPEnvelope, error) {
			calls++
			cancel()
			return client.HTTPEnvelope{StatusCode: item}, nil
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
		results := executeBatch(ctx, []int{1, 2}, func(context.Context, int) (client.HTTPEnvelope, error) {
			calls++
			return client.HTTPEnvelope{}, nil
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
