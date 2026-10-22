#!/usr/bin/env python3
# SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
#
# SPDX-License-Identifier: MIT

"""Verify (or explicitly regenerate) the pinned cloud-init rendering oracle."""

import argparse
import importlib.metadata
import json
from pathlib import Path
import platform
import sys

import jinja2


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--write", action="store_true", help="replace expected output")
    args = parser.parse_args()
    root = Path(__file__).resolve().parents[1] / "pkg/cloudinit/testdata"
    manifest = json.loads((root / "reference.json").read_text())
    versions = {
        "python": platform.python_version(),
        "jinja2": importlib.metadata.version("jinja2"),
        "markupsafe": importlib.metadata.version("markupsafe"),
    }
    for key, actual in versions.items():
        if actual != manifest[key]:
            parser.error(f"{key}: require {manifest[key]}, got {actual}")
    env = jinja2.Environment(
        autoescape=False,
        undefined=jinja2.Undefined,
        trim_blocks=False,
        lstrip_blocks=False,
        newline_sequence="\n",
        keep_trailing_newline=False,
        extensions=["jinja2.ext.do", "jinja2.ext.loopcontrols"],
    )
    failed = False
    for fixture in manifest["fixtures"]:
        variables = {"ds": {"meta_data": json.loads((root / fixture["metadata"]).read_bytes())}}
        variables.update(json.loads((root / fixture["extra_vars"]).read_bytes()))
        source = (root / fixture["template"]).read_bytes().decode("utf-8")
        rendered = env.from_string(source).render(variables).encode("utf-8")
        target = root / fixture["expected"]
        if args.write:
            target.write_bytes(rendered)
        elif target.read_bytes() != rendered:
            print(f"{fixture['name']}: Python output differs from {target}", file=sys.stderr)
            failed = True
    return int(failed)


if __name__ == "__main__":
    sys.exit(main())
