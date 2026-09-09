// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package boot_service

import (
	"context"
	"time"

	"github.com/openchami/ochami/pkg/client"
)

// runBatch executes each operation sequentially with a per-item timeout derived
// from the caller's context and preserves input order in the returned results.
func runBatch[T, R any](ctx context.Context, timeout time.Duration, items []T, operation func(context.Context, T) (R, error)) client.BatchResult[R] {
	results := make(client.BatchResult[R], len(items))
	for i, item := range items {
		if err := ctx.Err(); err != nil {
			for ; i < len(results); i++ {
				results[i].Err = err
			}
			break
		}
		requestCtx, cancel := context.WithTimeout(ctx, timeout)
		results[i].Value, results[i].Err = operation(requestCtx, item)
		cancel()
	}
	return results
}
