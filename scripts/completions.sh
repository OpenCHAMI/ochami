#!/bin/sh

# SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
# SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
#
# SPDX-License-Identifier: MIT

# scripts/completions.sh

set -e

if ! command -v go >/dev/null; then
  echo '"go" command not found' >&2
  exit 1
fi

scriptdir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)

rm -rf "${scriptdir}/../completions"
mkdir "${scriptdir}/../completions"
for sh in bash fish zsh; do
  go run "${scriptdir}/../main.go" --ignore-config completion "$sh" > "${scriptdir}/../completions/ochami.$sh"
done
