#!/usr/bin/env python3
# SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
#
# SPDX-License-Identifier: MIT

"""Verify (or explicitly regenerate) the pinned cloud-init rendering oracle.

Each fixture is rendered by cloud-init's own Jinja part handling, not by a
separately configured Jinja2 environment. The fixture's instance data is the
instance-data.json that ochami builds for its metadata and extra variables.
The script checks that the file is formatted exactly as cloud-init's
atomic_helper.json_dumps writes it, and then renders with
cloudinit.handlers.jinja_template.render_jinja_payload, which converts
variables, removes the template header, and renders with cloud-init's
environment.

cloud-init is not published to PyPI. Pass the source tree of the pinned
release with --cloud-init-src. See doc/jinja-compatibility.md.
"""

import argparse
import importlib.metadata
import json
import logging
from pathlib import Path
import platform
import re
import sys


def load_cloud_init(src):
    """Import cloud-init's rendering functions from a source tree.

    An unbuilt tree has no version module, so the version comes from its
    meson.build project declaration.
    """
    src = Path(src).resolve()
    meson = src / "meson.build"
    if not (src / "cloudinit" / "__init__.py").is_file() or not meson.is_file():
        raise SystemExit(f"{src} is not a cloud-init source tree")
    match = re.search(r"^\s*version:\s*'([^']+)'", meson.read_text(), re.MULTILINE)
    if not match:
        raise SystemExit(f"{meson} does not declare a version")
    sys.path.insert(0, str(src))
    from cloudinit import atomic_helper
    from cloudinit.handlers.jinja_template import render_jinja_payload

    return match.group(1), atomic_helper.json_dumps, render_jinja_payload


def main():
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--cloud-init-src", required=True, help="source tree of the pinned cloud-init release")
    parser.add_argument("--write", action="store_true", help="replace expected output")
    args = parser.parse_args()
    root = Path(__file__).resolve().parents[1] / "pkg/cloudinit/testdata"
    manifest = json.loads((root / "reference.json").read_text())
    cloud_init_version, json_dumps, render_jinja_payload = load_cloud_init(args.cloud_init_src)
    versions = {
        "python": platform.python_version(),
        "cloud_init": cloud_init_version,
        "jinja2": importlib.metadata.version("jinja2"),
        "markupsafe": importlib.metadata.version("markupsafe"),
    }
    for key, actual in versions.items():
        if actual != manifest[key]:
            parser.error(f"{key}: require {manifest[key]}, got {actual}")

    # cloud-init logs, rather than raises, when it ignores a template part.
    logging.basicConfig(level=logging.WARNING, format="%(levelname)s: %(message)s")
    failed = False
    for fixture in manifest["fixtures"]:
        # ochami builds this file. A node writes it with json_dumps and reads
        # its variables back from it.
        written = (root / fixture["instance_data"]).read_text()
        instance_data = json.loads(written)
        if json_dumps(instance_data) + "\n" != written:
            print(f"{fixture['name']}: {fixture['instance_data']} is not formatted as cloud-init writes it", file=sys.stderr)
            failed = True
        source = (root / fixture["template"]).read_bytes().decode("utf-8")
        rendered = render_jinja_payload(source, fixture["template"], instance_data)
        if rendered is None:
            print(f"{fixture['name']}: cloud-init ignored the template", file=sys.stderr)
            failed = True
            continue
        target = root / fixture["expected"]
        if args.write:
            target.write_bytes(rendered.encode("utf-8"))
        elif target.read_bytes() != rendered.encode("utf-8"):
            print(f"{fixture['name']}: cloud-init output differs from {target}", file=sys.stderr)
            failed = True
    return int(failed)


if __name__ == "__main__":
    sys.exit(main())
