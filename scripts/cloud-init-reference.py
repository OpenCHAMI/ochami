#!/usr/bin/env python3
# SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
#
# SPDX-License-Identifier: MIT

"""Verify (or explicitly regenerate) cloud-init reference cases.

These cases cover the parts of cloud-init's processing that ochami ports to
Go, other than Jinja rendering, which scripts/jinja-reference.py covers:

- yaml11.json: values as yaml.safe_load builds them, which is how cloud-init
  loads cloud-config and NoCloud meta-data.

Each expected value is computed by the pinned cloud-init release and its
dependencies. Pass the source tree with --cloud-init-src. See
doc/jinja-compatibility.md.
"""

import argparse
import base64
import datetime
import importlib.metadata
import json
import math
from pathlib import Path
import platform
import re
import sys


def load_cloud_init(src):
    """Put a cloud-init source tree on sys.path and return its version."""
    src = Path(src).resolve()
    meson = src / "meson.build"
    if not (src / "cloudinit" / "__init__.py").is_file() or not meson.is_file():
        raise SystemExit(f"{src} is not a cloud-init source tree")
    match = re.search(r"^\s*version:\s*'([^']+)'", meson.read_text(), re.MULTILINE)
    if not match:
        raise SystemExit(f"{meson} does not declare a version")
    sys.path.insert(0, str(src))
    return match.group(1)


def json_key(key):
    """Return the string json.dumps writes for a mapping key."""
    if isinstance(key, str):
        return key
    if key is True:
        return "true"
    if key is False:
        return "false"
    if key is None:
        return "null"
    if isinstance(key, float):
        return json.dumps(key)
    return str(key)


def tagged(value):
    """Return a JSON description of a loaded value that keeps its type."""
    if value is None:
        return {"t": "null"}
    if isinstance(value, bool):
        return {"t": "bool", "v": value}
    if isinstance(value, int):
        return {"t": "int", "v": str(value)}
    if isinstance(value, float):
        if math.isnan(value):
            return {"t": "float", "v": "nan"}
        return {"t": "float", "v": repr(value)}
    if isinstance(value, str):
        return {"t": "str", "v": value}
    if isinstance(value, bytes):
        return {"t": "bytes", "v": base64.b64encode(value).decode()}
    if isinstance(value, datetime.datetime):
        return {"t": "datetime", "v": str(value)}
    if isinstance(value, datetime.date):
        return {"t": "date", "v": str(value)}
    if isinstance(value, list):
        return {"t": "list", "v": [tagged(item) for item in value]}
    if isinstance(value, dict):
        items = sorted((json_key(k), tagged(v)) for k, v in value.items())
        return {"t": "dict", "v": [list(item) for item in items]}
    raise TypeError(f"unexpected type {type(value)}")


def check_yaml(root, write):
    import yaml

    path = root / "yaml11.json"
    doc = json.loads(path.read_text())
    failed = False
    for case in doc["cases"]:
        try:
            expected = tagged(yaml.safe_load(case["yaml"]))
        except Exception:  # cloud-init treats any load failure as unusable data
            expected = {"error": True}
        if write:
            case["expected"] = expected
        elif case.get("expected") != expected:
            print(f"yaml11 {case['name']}: expected {expected}, file has {case.get('expected')}", file=sys.stderr)
            failed = True
    if write:
        path.write_text(json.dumps(doc, indent=2, ensure_ascii=False) + "\n")
    return failed


def main():
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--cloud-init-src", required=True, help="source tree of the pinned cloud-init release")
    parser.add_argument("--write", action="store_true", help="replace expected values")
    args = parser.parse_args()
    root = Path(__file__).resolve().parents[1] / "pkg/cloudinit/testdata"
    manifest = json.loads((root / "reference.json").read_text())
    versions = {
        "python": platform.python_version(),
        "cloud_init": load_cloud_init(args.cloud_init_src),
        "pyyaml": importlib.metadata.version("pyyaml"),
    }
    for key, actual in versions.items():
        if actual != manifest[key]:
            parser.error(f"{key}: require {manifest[key]}, got {actual}")

    failed = check_yaml(root, args.write)
    return int(failed)


if __name__ == "__main__":
    sys.exit(main())
