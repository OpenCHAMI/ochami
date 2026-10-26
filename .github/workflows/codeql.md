<!--
SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC

SPDX-License-Identifier: MIT
-->

# CodeQL scanning and fork pull requests

The explicit workflow in `.github/workflows/codeql.yml` scans Go, Python,
and GitHub Actions on pull requests targeting `main`, pushes to `main`,
and a weekly schedule. It can also be run manually to refresh the baseline.
Go uses the version in `go.mod` and builds all packages for analysis.

## Why fork pull requests could wait forever

GitHub's [CodeQL default
setup](https://docs.github.com/en/code-security/concepts/code-scanning/setup-types)
excludes pull requests from forks. The repository's **Pass CI before merge**
ruleset requires CodeQL results, so a fork pull request could wait for results
that would never be produced. PR #214 was an example: neither its head commit
1e6304f nor its merge commit e8375b0 had a CodeQL run.

The explicit workflow uses `pull_request` to analyze the merge commit,
including for forks. GitHub permits code scanning uploads from this event
even with a [read-only pull request
token](https://docs.github.com/en/code-security/reference/code-scanning/troubleshoot-analysis-errors/resource-not-accessible).
First-time contributors may still need the usual Actions approval.

## Migrating from default setup

Coordinate these steps when deploying the workflow:

1. In repository **Settings → Advanced Security → CodeQL analysis**, select
   **Switch to advanced** and disable default setup. Default setup
   [blocks uploads from explicit CodeQL workflows](https://docs.github.com/en/code-security/reference/code-scanning/troubleshoot-analysis-errors/results-different-than-expected).
2. Merge `.github/workflows/codeql.yml` into `main`. If the migration PR itself
   is blocked waiting for CodeQL, an administrator may need to use the existing
   ruleset bypass to land this initial workflow.
3. Verify the push run on `main` succeeds for all three languages and uploads
   a new baseline. If needed, run the workflow manually on `main`.
4. Update existing fork PR branches with `main` so they include the workflow
   and trigger a new `pull_request` run. Approve runs when GitHub requests it.
5. Verify all three analyses are uploaded for each PR and its CodeQL gate
   completes. Keep the existing CodeQL ruleset requirement enabled.

Rerunning an old build or test job cannot create the missing CodeQL run.
Check CodeQL under **Actions** and **Security → Code scanning** if a PR still
waits after its workflow completes. Do not use `pull_request_target` to build
fork code or grant fork workflows write tokens to work around missing scans.
