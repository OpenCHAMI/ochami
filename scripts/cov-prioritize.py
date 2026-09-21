#!/usr/bin/env python3
# SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
#
# SPDX-License-Identifier: MIT

"""Deduplicate Go coverpkg profiles and classify uncovered blocks."""

from __future__ import annotations

import argparse
import collections
import csv
import dataclasses
import pathlib
import re
import sys


PROFILE_LINE = re.compile(
    r"^(?P<file>.+):(?P<sl>\d+)\.(?P<sc>\d+),(?P<el>\d+)\.(?P<ec>\d+) "
    r"(?P<statements>\d+) (?P<count>\d+)$"
)

CATEGORIES = {
    1: "reachable and behaviorally important",
    2: "reachable but low-risk",
    3: "process-boundary integration",
    4: "platform/terminal/signal-specific",
    5: "generated-client invariant",
    6: "unreachable defensive code",
    7: "obsolete compatibility code",
}


@dataclasses.dataclass
class Block:
    file: str
    start_line: int
    start_col: int
    end_line: int
    end_col: int
    statements: int
    count: int

    @property
    def key(self) -> tuple[object, ...]:
        return (
            self.file,
            self.start_line,
            self.start_col,
            self.end_line,
            self.end_col,
            self.statements,
        )


def parse_profile(path: pathlib.Path) -> dict[tuple[object, ...], Block]:
    blocks: dict[tuple[object, ...], Block] = {}
    with path.open(encoding="utf-8") as profile:
        mode = profile.readline().strip()
        if not mode.startswith("mode:"):
            raise ValueError(f"{path}: missing Go coverage mode header")
        for line_number, raw_line in enumerate(profile, start=2):
            line = raw_line.strip()
            if not line:
                continue
            match = PROFILE_LINE.match(line)
            if match is None:
                raise ValueError(f"{path}:{line_number}: malformed coverage block")
            values = match.groupdict()
            block = Block(
                file=values["file"],
                start_line=int(values["sl"]),
                start_col=int(values["sc"]),
                end_line=int(values["el"]),
                end_col=int(values["ec"]),
                statements=int(values["statements"]),
                count=int(values["count"]),
            )
            previous = blocks.get(block.key)
            if previous is None or block.count > previous.count:
                blocks[block.key] = block
    return blocks


def classify(block: Block) -> tuple[int, str]:
    relative = block.file.removeprefix("github.com/openchami/ochami/")
    basename = pathlib.PurePosixPath(relative).name

    if relative == "main.go" or (
        relative == "cmd/root.go" and block.start_line >= 197
    ):
        return 3, "requires subprocess exit or top-level process execution"

    if basename.endswith(("_unix.go", "_windows.go")):
        return 4, "platform-specific operating-system behavior"
    if relative == "pkg/client/rcs/rcs.go" and (
        52 <= block.start_line <= 54 or 250 <= block.start_line <= 405
    ):
        return 4, "terminal or interactive console behavior"

    if any(
        marker in relative
        for marker in ("completion", "/version/", "/service/service.go")
    ) or basename in {"version.go"}:
        return 2, "low-risk completion, command wiring, or informational output"

    return 1, "reachable command, configuration, or client behavior"


def write_inventory(path: pathlib.Path, blocks: list[Block]) -> None:
    with path.open("w", encoding="utf-8", newline="") as inventory:
        inventory.write("# SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC\n")
        inventory.write("# SPDX-" + "License-Identifier: MIT\n")
        writer = csv.writer(inventory, delimiter="\t", lineterminator="\n")
        writer.writerow(
            ("source", "start", "end", "statements", "category", "classification", "rationale")
        )
        for block in blocks:
            category, rationale = classify(block)
            writer.writerow(
                (
                    block.file.removeprefix("github.com/openchami/ochami/"),
                    f"{block.start_line}.{block.start_col}",
                    f"{block.end_line}.{block.end_col}",
                    block.statements,
                    category,
                    CATEGORIES[category],
                    rationale,
                )
            )


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("profile", nargs="?", default="coverage.out")
    parser.add_argument("--top", type=int, default=20, help="number of hotspot files")
    parser.add_argument("--inventory", type=pathlib.Path, help="write all uncovered blocks as TSV")
    args = parser.parse_args()

    try:
        blocks = parse_profile(pathlib.Path(args.profile))
    except (OSError, ValueError) as error:
        parser.error(str(error))

    total = sum(block.statements for block in blocks.values())
    covered = sum(block.statements for block in blocks.values() if block.count > 0)
    uncovered = sorted(
        (block for block in blocks.values() if block.count == 0),
        key=lambda block: (block.file, block.start_line, block.start_col),
    )
    percent = 100 * covered / total if total else 0
    print(
        f"deduplicated coverage: {covered}/{total} statements ({percent:.1f}%); "
        f"{len(uncovered)} uncovered blocks"
    )

    categories: collections.Counter[int] = collections.Counter()
    hotspots: collections.Counter[str] = collections.Counter()
    for block in uncovered:
        category, _ = classify(block)
        categories[category] += block.statements
        hotspots[block.file.removeprefix("github.com/openchami/ochami/")] += block.statements

    print("\nuncovered statements by category:")
    for category in sorted(CATEGORIES):
        print(f"  {category}: {categories[category]:4d}  {CATEGORIES[category]}")

    print("\nhotspot files:")
    for source, statements in hotspots.most_common(max(args.top, 0)):
        print(f"  {statements:4d}  {source}")

    if args.inventory is not None:
        write_inventory(args.inventory, uncovered)
        print(f"\nwrote {len(uncovered)} blocks to {args.inventory}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
