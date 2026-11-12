// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cloud_init

import (
	"fmt"
	"strings"
)

// diffContext is the number of unchanged lines shown around each change.
const diffContext = 3

// UnifiedDiff returns a unified diff from a to b, or an empty string when
// they are equal. It is meant for the short documents cloud-init renders.
func UnifiedDiff(nameA, nameB, a, b string) string {
	if a == b {
		return ""
	}
	x, y := splitLines(a), splitLines(b)
	ops := diffLines(x, y)

	var out strings.Builder
	fmt.Fprintf(&out, "--- %s\n+++ %s\n", nameA, nameB)
	for start := 0; start < len(ops); {
		// Find the next change.
		for start < len(ops) && ops[start].kind == ' ' {
			start++
		}
		if start == len(ops) {
			break
		}
		// Extend the hunk while changes are within twice the context.
		end := start
		for i := start; i < len(ops); i++ {
			if ops[i].kind != ' ' {
				end = i + 1
				continue
			}
			if i-end >= 2*diffContext {
				break
			}
		}
		from := max(start-diffContext, 0)
		to := min(end+diffContext, len(ops))
		ai, bi, an, bn := ops[from].a, ops[from].b, 0, 0
		var body strings.Builder
		for _, op := range ops[from:to] {
			body.WriteByte(op.kind)
			body.WriteString(op.line)
			body.WriteByte('\n')
			if op.kind != '+' {
				an++
			}
			if op.kind != '-' {
				bn++
			}
		}
		fmt.Fprintf(&out, "@@ -%s +%s @@\n%s", hunkRange(ai, an), hunkRange(bi, bn), body.String())
		start = to
	}
	return out.String()
}

func hunkRange(start, n int) string {
	if n == 0 {
		return fmt.Sprintf("%d,0", start)
	}
	if n == 1 {
		return fmt.Sprintf("%d", start+1)
	}
	return fmt.Sprintf("%d,%d", start+1, n)
}

func splitLines(s string) []string {
	s = strings.TrimSuffix(s, "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

type diffOp struct {
	kind byte // ' ', '-', or '+'
	line string
	a, b int // line indexes before this operation
}

// diffLines returns the edit script from x to y using a longest common
// subsequence table.
func diffLines(x, y []string) []diffOp {
	n, m := len(x), len(y)
	lcs := make([][]int, n+1)
	for i := range lcs {
		lcs[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if x[i] == y[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else {
				lcs[i][j] = max(lcs[i+1][j], lcs[i][j+1])
			}
		}
	}
	var ops []diffOp
	i, j := 0, 0
	for i < n || j < m {
		switch {
		case i < n && j < m && x[i] == y[j]:
			ops = append(ops, diffOp{' ', x[i], i, j})
			i++
			j++
		case i < n && (j == m || lcs[i+1][j] >= lcs[i][j+1]):
			ops = append(ops, diffOp{'-', x[i], i, j})
			i++
		default:
			ops = append(ops, diffOp{'+', y[j], i, j})
			j++
		}
	}
	return ops
}
