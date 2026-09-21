// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package client

import (
	"context"
	"time"
)

// RunBatch executes operation once per item, in input order, and returns one
// aligned Result per item.
//
// Before each item, ctx.Err() is checked: once it is non-nil (the caller
// cancelled or its deadline expired), execution stops and every remaining
// item — including the one that was about to run — is filled with that error.
// Only Err is set for those items; their Value is the zero value.
func RunBatch[T, R any](ctx context.Context, items []T, operation func(context.Context, T) (R, error)) BatchResult[R] {
	results := make(BatchResult[R], len(items))
	for i, item := range items {
		if err := ctx.Err(); err != nil {
			for ; i < len(results); i++ {
				results[i].Err = err
			}
			break
		}
		results[i].Value, results[i].Err = operation(ctx, item)
	}
	return results
}

// RunBatchWithTimeout is RunBatch with a per-item timeout: each operation
// gets a context derived from ctx that expires after timeout and is released
// as soon as that item finishes. Cancellation or expiry of ctx itself stops
// the batch as in RunBatch.
func RunBatchWithTimeout[T, R any](ctx context.Context, timeout time.Duration, items []T, operation func(context.Context, T) (R, error)) BatchResult[R] {
	return RunBatch(ctx, items, func(ctx context.Context, item T) (R, error) {
		requestCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		return operation(requestCtx, item)
	})
}
