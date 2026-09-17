<!--
SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC

SPDX-License-Identifier: MIT
-->

# Coverage Completion Report

This report records the Part III coverage audit from `coverage-plan8.md`. The
authoritative profile was generated with `make coverage`; its `go tool cover`
statement total is **90.8%**.

## Result

- Plan 8 initial baseline: **90.8%**
- Final official total: **90.8%**
- Deduplicated `-coverpkg=./...` total: **9171/10099 statements (90.8%)**
- Deduplicated uncovered blocks: **901**
- 95% completion target: **not reached**
- 97–98% stretch target: **not reached**
- 90.8% non-regression floor: **met**

The numerical targets were not pursued with tests that merely execute lines.
The remaining gaps include command validation combinations, configuration
failure branches that need additional filesystem seams, HTTP-client edge
conditions, and interactive terminal behavior. The block-level classification
in [`coverage-gaps.tsv`](coverage-gaps.tsv) is the closure inventory for this
profile.

## Changes in the closure pass

Behavior tests added or corrected in this pass verify:

- malformed successful boot-service responses for add, get, patch, and set;
- HTTP failures for representative boot-service list and batch commands;
- simple and envelope boot-service set paths;
- input and output format flag application, including invalid values;
- propagation and exit-code mapping of output writer failures;
- short-write handling for zero-length and oversized writer results;
- prompt writer and input reader failures;
- merged default configuration loading and user-path resolution;
- default logging configuration; and
- successful and failed parent-directory synchronization.

Several coverage-only tests were rejected or rewritten because they asserted
only that code executed, logged unexpected success instead of failing, used an
invalid command shape, or purported to test malformed response handling without
making a request.

No production statements were removed during this final pass. Earlier commits
in the coverage series removed obsolete non-runtime client wrappers after call
site analysis showed they had no users.

## Remaining categories

The deduplicated uncovered statements are classified as follows:

| Category | Statements | Rationale |
| --- | ---: | --- |
| Reachable and behaviorally important | 889 | Command, configuration, and client branches remain candidates for focused tests. |
| Reachable but low-risk | 11 | Completion, command wiring, and informational output branches. |
| Process-boundary integration | 7 | `main` and top-level execution/exit behavior require subprocess tests. |
| Platform/terminal/signal-specific | 21 | Directory sync and interactive terminal handling require platform or terminal integration. |
| Generated-client invariant | 0 | No remaining block was assigned solely to a generated-client invariant. |
| Unreachable defensive code | 0 | No unproven defensive branch was labeled unreachable. |
| Obsolete compatibility code | 0 | No obsolete compatibility block remains in the inventory. |

Classification intentionally errs toward “reachable and behaviorally
important.” A block is not called unreachable or obsolete without a proven
contract. Each row in `coverage-gaps.tsv` records its source range, statement
count, category, and rationale.

## Largest remaining hotspots

| File | Uncovered statements |
| --- | ---: |
| `cmd/cloud_init/node/get.go` | 38 |
| `internal/configfile/configfile.go` | 34 |
| `pkg/client/smd/smd.go` | 33 |
| `internal/cli/runtime.go` | 32 |
| `pkg/client/rcs/rcs.go` | 30 |
| `pkg/client/client.go` | 24 |
| `cmd/cloud_init/group/get.go` | 21 |
| `cmd/bss/boot/image/set.go` | 17 |
| `cmd/discover/static/static.go` | 17 |
| `pkg/config/config.go` | 17 |

## Reproduction

```sh
make coverage
go tool cover -func=coverage.out
go tool cover -html=coverage.out -o coverage.html
scripts/cov-prioritize.py coverage.out \
  --inventory doc/coverage-gaps.tsv
```

The official percentage is always the `go tool cover` total. The helper merges
identical source ranges emitted by `-coverpkg=./...` only for prioritization and
inventory generation.
