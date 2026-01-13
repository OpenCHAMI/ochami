#!/usr/bin/env python3

# SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
#
# SPDX-License-Identifier: MIT

"""
Convert old node discovery format that consists of only 'nodes' to the new
format that contains both 'nodes' and 'bmcs'.

Reads JSON or YAML from STDIN and writes the converted document (same
format, JSON or YAML, as input by default) to STDOUT.

- Old format: nodes contain bmc_* fields (bmc_mac, bmc_ip, bmc_fqdn, bmc_xname)
  and may use 'group' (string) or 'groups' (string/array).
- New format: top-level 'bmcs' array; each node has 'bmc' that references a BMC
  by its xname (string). 'group' is replaced by 'groups' (array).

External deps: optional PyYAML for YAML support. If PyYAML is not installed,
the tool will still work for JSON input.
"""
import sys
import json
import re
import argparse

try:
    import yaml  # optional
    HAVE_YAML = True
except Exception:
    HAVE_YAML = False

BMC_FIELDS = ("bmc_mac", "bmc_ip", "bmc_fqdn", "bmc_xname")


def parse_args():
    parser = argparse.ArgumentParser(
        prog="old2new.py",
        description=(
            "Convert old node inventory format (JSON or YAML) to the new format.\n"
            "Reads from STDIN and writes to STDOUT."
        ),
        epilog=(
            "Examples:\n"
            "  # Default: detect input, match output format\n"
            "  python3 old2new.py < nodes.yaml > nodes-new.yaml\n"
            "  python3 old2new.py < nodes.json > nodes-new.json\n\n"
            "  # Force output JSON regardless of input\n"
            "  python3 old2new.py -o json < nodes.yaml > nodes-new.json\n\n"
            "  # Force output YAML regardless of input\n"
            "  python3 old2new.py -o yaml < nodes.json > nodes-new.yaml\n\n"
            "  # Force the input parser (rarely needed; defaults to auto)\n"
            "  python3 old2new.py -i yaml < nodes.yaml > nodes-new.yaml\n"
        ),
        formatter_class=argparse.RawDescriptionHelpFormatter,
    )
    parser.add_argument(
        "-i", "--in-format",
        dest="in_format",
        choices=["auto", "json", "yaml"],
        default="auto",
        help="Input format (default: auto)."
    )
    parser.add_argument(
        "-o", "--out-format",
        dest="out_format",
        choices=["match", "json", "yaml"],
        default="match",
        help="Output format (default: match input format)."
    )
    args = parser.parse_args()

    return args


def read_input(fmt_hint):
    """Return ('json'|'yaml', python_obj)."""
    raw = sys.stdin.read()

    if fmt_hint == "json":
        try:
            return "json", json.loads(raw)
        except Exception as e:
            sys.stderr.write(f"Failed to parse input as JSON: {e}\n")
            sys.exit(2)
    elif fmt_hint == "yaml":
        if not HAVE_YAML:
            sys.stderr.write("YAML parsing requested but PyYAML is not installed.\n")
            sys.exit(2)
        try:
            return "yaml", yaml.safe_load(raw)
        except Exception as e:
            sys.stderr.write(f"Failed to parse input as YAML: {e}\n")
            sys.exit(2)
    else:
        # auto: try JSON first, then YAML
        try:
            return "json", json.loads(raw)
        except Exception:
            pass
        if not HAVE_YAML:
            sys.stderr.write(
                "Input does not appear to be JSON and PyYAML is not installed.\n"
                "Please install PyYAML (pip install pyyaml) or provide JSON input.\n"
            )
            sys.exit(2)
        try:
            return "yaml", yaml.safe_load(raw)
        except Exception as e:
            sys.stderr.write(f"Failed to parse input as YAML: {e}\n")
            sys.exit(2)


def _derive_bmc_xname(node_xname, bmc_fqdn, bmc_xname):
    """Best-effort derivation of the BMC xname.
    Priority:
      1. explicit bmc_xname
      2. trim node xname suffix 'n<digits>...'
      3. leftmost label of bmc_fqdn
    """
    if bmc_xname:
        return bmc_xname
    if node_xname:
        # Example: x1000c0s0b0n0 -> x1000c0s0b0
        return re.sub(r"n\d+.*$", "", node_xname)
    if bmc_fqdn:
        return bmc_fqdn.split(".", 1)[0]
    return None


def normalize_groups_inplace(node: dict) -> None:
    """Convert 'group' (string), if it exists, to 'groups' (list) in-place for a
    node.
    """
    groups = []
    if "group" in node and node["group"] is not None:
        g = node.pop("group")
        if isinstance(g, str) and g:
            groups.append(g)
        elif isinstance(g, list):
            groups.extend([str(x) for x in g if x])
    if "groups" in node and node["groups"] is not None:
        g = node["groups"]
        if isinstance(g, str):
            groups.append(g)
        elif isinstance(g, list):
            groups.extend([str(x) for x in g if x])
        # Deduplicate groups
        seen = set()
        deduped = []
        for x in groups:
            if x not in seen:
                seen.add(x)
                deduped.append(x)
        node["groups"] = deduped or None
    else:
        node["groups"] = groups or None


def convert(data):
    if not isinstance(data, dict):
        raise SystemExit("Top-level document must be a mapping/dict containing 'nodes'.")
    nodes = data.get("nodes")
    if not isinstance(nodes, list):
        raise SystemExit("Input must contain a top-level 'nodes' array.")

    # Aggregate unique BMCs. Key primarily by xname; fall back to (mac, ip, fqdn) tuple.
    bmcs_by_key = {}
    node_records = []

    for raw_node in nodes:
        if not isinstance(raw_node, dict):
            continue
        node = dict(raw_node)  # shallow copy
        # Extract BMC fields from the node
        bmc_vals = {fld: node.pop(fld, None) for fld in BMC_FIELDS}
        # Normalize groups
        normalize_groups_inplace(node)

        # Determine BMC xname
        bmc_xname = _derive_bmc_xname(node.get("xname"), bmc_vals.get("bmc_fqdn"), bmc_vals.get("bmc_xname"))

        # If any BMC info is present, create/merge a BMC record and add node['bmc']
        has_any_bmc_info = any(v for v in bmc_vals.values())
        if has_any_bmc_info or bmc_xname:
            if bmc_xname:
                key = ("xname", bmc_xname)
            else:
                key = ("triplet", bmc_vals.get("bmc_mac"), bmc_vals.get("bmc_ip"), bmc_vals.get("bmc_fqdn"))
            rec = bmcs_by_key.get(key, {})
            # Merge what we know
            if bmc_xname and not rec.get("xname"):
                rec["xname"] = bmc_xname
            if bmc_vals.get("bmc_mac") and not rec.get("mac"):
                rec["mac"] = bmc_vals["bmc_mac"]
            if bmc_vals.get("bmc_ip") and not rec.get("ip"):
                rec["ip"] = bmc_vals["bmc_ip"]
            if bmc_vals.get("bmc_fqdn") and not rec.get("fqdn"):
                rec["fqdn"] = bmc_vals["bmc_fqdn"]
            bmcs_by_key[key] = rec

            # Only set node['bmc'] when we know the BMC xname (target format requirement)
            if bmc_xname:
                node["bmc"] = bmc_xname

        node_records.append(node)

    # Build stable bmcs list. Assign deterministic 'name' fields (bmc1, bmc2, ...), though mapping uses xname.
    def sort_key(item):
        key, rec = item
        if "xname" in rec and rec["xname"]:
            return ("0", rec["xname"])
        return ("1", rec.get("ip") or "", rec.get("mac") or "", rec.get("fqdn") or "")

    bmcs_list = []
    for idx, (_, rec) in enumerate(sorted(bmcs_by_key.items(), key=sort_key), start=1):
        out = {}
        if rec.get("xname") is not None:
            out["xname"] = rec["xname"]
        if rec.get("mac") is not None:
            out["mac"] = rec["mac"]
        if rec.get("ip") is not None:
            out["ip"] = rec["ip"]
        if rec.get("fqdn") is not None:
            out["fqdn"] = rec["fqdn"]
        bmcs_list.append(out)

    out_doc = {}
    out_doc["bmcs"] = bmcs_list
    out_doc["nodes"] = node_records
    return out_doc


def write_output(fmt, obj):
    if fmt == "json":
        json.dump(obj, sys.stdout, ensure_ascii=False, indent=2)
        sys.stdout.write("\n")
    else:
        if not HAVE_YAML:
            sys.stderr.write("YAML output requested but PyYAML not available.\n")
            sys.exit(2)
        yaml.safe_dump(obj, sys.stdout, sort_keys=False, indent=2, default_flow_style=False)


def main():
    args = parse_args()
    in_fmt, data = read_input(args.in_format)
    converted = convert(data)

    out_fmt = args.out_format
    if out_fmt == "match":
        out_fmt = in_fmt

    write_output(out_fmt, converted)


if __name__ == "__main__":
    main()
