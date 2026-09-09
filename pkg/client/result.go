// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package client

// Result pairs the value and error produced for one item in a batch operation.
type Result[T any] struct {
	Value T
	Err   error
}

// BatchResult contains one Result per input item, in input order.
type BatchResult[T any] []Result[T]

// HasErrors reports whether any item failed.
func (r BatchResult[T]) HasErrors() bool {
	for _, result := range r {
		if result.Err != nil {
			return true
		}
	}
	return false
}

// Errors returns non-nil item errors in input order.
func (r BatchResult[T]) Errors() []error {
	errs := make([]error, 0)
	for _, result := range r {
		if result.Err != nil {
			errs = append(errs, result.Err)
		}
	}
	return errs
}

// Values returns values for successful items only, in input order.
func (r BatchResult[T]) Values() []T {
	values := make([]T, 0, len(r))
	for _, result := range r {
		if result.Err == nil {
			values = append(values, result.Value)
		}
	}
	return values
}
