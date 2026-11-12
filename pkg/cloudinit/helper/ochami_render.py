#!/usr/bin/env python3
# SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
#
# SPDX-License-Identifier: MIT

"""Render cloud-init data with cloud-init's own code, for ochami.

ochami embeds this helper and writes it into render bundles. It processes
user-data and vendor-data with cloud-init's UserDataProcessor, part handlers,
and mergers, as a node does, and prints the result as JSON. Includes are read
from the bundle instead of the network, and nothing is executed: boothooks are
recorded instead of run, and part handlers, which load Python code from the
data, are not loaded.

It supports cloud-init 23.4 and later on Python 3.9 and later. Run it with
python3 -I so that files next to it cannot shadow Python modules.
"""

import argparse
import base64
import datetime
import inspect
import json
import logging
import math
import os
from pathlib import Path
import re
import sys
import tempfile

PART_HANDLER = "text/part-handler"


def bootstrap(src):
    """Make a cloud-init source tree importable, if one is given."""
    if src:
        sys.path.insert(0, str(Path(src).resolve()))


def cloud_init_version(src=None):
    """Return the cloud-init version, reading meson.build for source trees."""
    from cloudinit import version

    found = version.version_string()
    if src and not re.match(r"^[0-9]", found):
        meson = Path(src) / "meson.build"
        match = re.search(r"^\s*version:\s*'([^']+)'", meson.read_text(), re.MULTILINE)
        if match:
            return match.group(1)
    return found


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
    """Return a JSON description of a value that keeps its Python type."""
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
    if isinstance(value, (list, tuple)):
        return {"t": "list", "v": [tagged(item) for item in value]}
    if isinstance(value, dict):
        items = sorted((json_key(k), tagged(v)) for k, v in value.items())
        return {"t": "dict", "v": [list(item) for item in items]}
    return {"t": "unknown", "v": str(value)}


class _Datasource:
    """The parts of a datasource that handlers use."""

    def __init__(self, instance_id):
        self._instance_id = instance_id

    def get_instance_id(self):
        return self._instance_id


class _Response:
    """A url_helper response for bundled include content."""

    def __init__(self, contents, code=200):
        self.contents = contents
        self.code = code

    def ok(self, *args, **kwargs):
        return 200 <= self.code < 300

    def __str__(self):
        return "" if self.contents is None else self.contents.decode("utf-8", "replace")


class _Capture(logging.Handler):
    """Collect warnings that cloud-init logs."""

    def __init__(self):
        logging.Handler.__init__(self, logging.WARNING)
        self.messages = []

    def emit(self, record):
        self.messages.append(record.getMessage())


def _instance_id(instance_data):
    v1 = instance_data.get("v1") or {}
    meta = (instance_data.get("ds") or {}).get("meta_data") or {}
    return v1.get("instance_id") or meta.get("instance-id") or "iid-datasource"


def _make(cls, options):
    """Construct a handler with the keyword arguments its release accepts."""
    params = inspect.signature(cls.__init__).parameters
    if any(p.kind == p.VAR_KEYWORD for p in params.values()):
        return cls(**options)
    return cls(**{k: v for k, v in options.items() if k in params})


def _consume(paths, datasource, message, cloud_config_path, excluded):
    """Run the default handlers over message, as Init._do_handlers does,
    and return the merged cloud-config and each part's status."""
    from cloudinit import handlers, helpers, util
    from cloudinit.handlers.cloud_config import CloudConfigPartHandler
    from cloudinit.handlers.jinja_template import JinjaTemplatePartHandler
    from cloudinit.handlers.shell_script import ShellScriptPartHandler
    from cloudinit.settings import PER_ALWAYS, PER_INSTANCE

    calls = {}

    def record(handler, label):
        original = handler.handle_part

        def handle_part(data, ctype, filename, *args, **kwargs):
            if ctype not in (handlers.CONTENT_START, handlers.CONTENT_END):
                calls.setdefault(filename, []).append(label)
            return original(data, ctype, filename, *args, **kwargs)

        handler.handle_part = handle_part
        return handler

    class Boothooks(handlers.Handler):
        """Record boothooks without running them."""

        prefixes = ["#cloud-boothook"]

        def __init__(self):
            handlers.Handler.__init__(self, PER_ALWAYS)

        def handle_part(self, data, ctype, filename, payload, frequency):
            pass

    options = {
        "paths": paths,
        "datasource": datasource,
        "cloud_config_path": cloud_config_path,
        "script_path": "vendor_scripts" if cloud_config_path.startswith("vendor") else "scripts",
    }
    cloud_config = record(_make(CloudConfigPartHandler, options), "cloud-config")
    scripts = record(_make(ShellScriptPartHandler, options), "script")
    boothooks = record(Boothooks(), "boothook")
    jinja = record(
        _make(JinjaTemplatePartHandler, dict(options, sub_handlers=[cloud_config, scripts, boothooks])),
        "jinja",
    )
    content = helpers.ContentHandlers()
    for handler in (cloud_config, scripts, boothooks, jinja):
        content.register(handler, overwrite=False)
    for handler in (cloud_config, scripts, boothooks, jinja):
        handlers.call_begin(handler, None, PER_INSTANCE)
    merged = []
    original = cloud_config.handle_part

    def track(data, ctype, filename, *args, **kwargs):
        result = original(data, ctype, filename, *args, **kwargs)
        if filename and filename.strip() in getattr(cloud_config, "file_names", []):
            merged.append(filename.strip())
        return result

    cloud_config.handle_part = track
    part_data = {
        "handlers": content,
        "handlerdir": None,
        "data": None,
        "frequency": PER_INSTANCE,
        "handlercount": 0,
        "excluded": list(excluded) + [PART_HANDLER],
    }
    try:
        handlers.walk(message, handlers.walker_callback, data=part_data)
    finally:
        for handler in (cloud_config, scripts, boothooks, jinja):
            handlers.call_end(handler, None, PER_INSTANCE)

    errors = set(n.strip() for n in getattr(cloud_config, "error_file_names", []))
    parts = []
    for part in message.walk():
        if part.get_content_maintype() == "multipart":
            continue
        name = part.get_filename()
        ctype = part.get_content_type()
        seen = calls.get(name, [])
        if ctype == PART_HANDLER:
            status = "unsupported"
        elif ctype in excluded:
            status = "excluded"
        elif name in merged:
            status = "merged"
        elif name in errors:
            status = "dropped"
        elif "script" in seen:
            status = "script"
        elif "boothook" in seen:
            status = "boothook"
        elif seen:
            status = "dropped"
        else:
            status = "ignored"
        parts.append({"name": name, "type": ctype, "status": status})

    path = paths.get_ipath(cloud_config_path)
    config = None
    if os.path.isfile(path):
        load = getattr(util, "load_text_file", None) or util.load_file
        text = load(path)
        if text.strip():
            config = util.load_yaml(text, default={})
    return config, parts


def full_render(user_data, vendor_data, instance_data, includes, src=None):
    """Process user-data and vendor-data as a node does.

    includes maps each include URL to a dict with "status" (resolved,
    unresolved, or failed) and, when resolved, "content" bytes.
    """
    from cloudinit import helpers, settings, user_data as ud, util
    import cloudinit.user_data

    capture = _Capture()
    logging.getLogger().addHandler(capture)
    tmp = tempfile.mkdtemp(prefix="ochami-render-")
    datasource = _Datasource(_instance_id(instance_data))
    paths = helpers.Paths({"cloud_dir": os.path.join(tmp, "cloud"), "run_dir": os.path.join(tmp, "run")}, ds=datasource)
    os.makedirs(paths.run_dir, mode=0o700)
    util.write_file(paths.get_runpath("instance_data_sensitive"), json.dumps(instance_data), mode=0o600)

    def read_file_or_url(url, *args, **kwargs):
        entry = includes.get(url)
        if entry is None or entry.get("status") == "unresolved":
            # ochami could not fetch it, so leave it out as ochami does.
            return _Response(None)
        if entry.get("status") == "failed":
            return _Response(None, code=int(entry.get("code") or 404))
        return _Response(entry["content"])

    cloudinit.user_data.read_file_or_url = read_file_or_url
    result = {"cloud_init_version": cloud_init_version(src)}

    def process(name, raw, cloud_config_path, excluded):
        section = {"parts": [], "config": None, "failed": None}
        try:
            message = ud.UserDataProcessor(paths).process(raw)
        except Exception as e:  # cloud-init aborts consuming data
            section["failed"] = str(e)
            return section
        config, parts = _consume(paths, datasource, message, cloud_config_path, excluded)
        section["parts"] = parts
        section["config"] = config
        return section

    user = process("user-data", user_data, "cloud_config", [])
    result["user"] = {"parts": user["parts"], "failed": user["failed"], "config": tagged(user["config"]) if user["config"] is not None else None}
    vendor = {"parts": [], "config": None, "failed": None}
    disabled = False
    if user["failed"] is None and vendor_data:
        sources = [user["config"]] if user["config"] else []
        cfg = util.mergemanydict(sources + [{"vendor_data": settings.CFG_BUILTIN["vendor_data"]}])
        vdcfg = cfg.get("vendor_data", {})
        if not isinstance(vdcfg, dict):
            vdcfg = {"enabled": False}
        if util.is_true(vdcfg.get("enabled")):
            vendor = process("vendor-data", vendor_data, "vendor_cloud_config", vdcfg.get("disabled_handlers") or [])
        else:
            disabled = True
    result["vendor"] = {
        "parts": vendor["parts"],
        "failed": vendor["failed"],
        "disabled": disabled,
        "config": tagged(vendor["config"]) if vendor["config"] is not None else None,
    }
    merged = {}
    if user["failed"] is None:
        merged = util.mergemanydict([c for c in (user["config"], vendor["config"]) if c])
    result["merged"] = tagged(merged)
    result["text"] = _text(result, merged)
    result["warnings"] = capture.messages
    logging.getLogger().removeHandler(capture)
    return result


def _text(result, merged):
    """Format merged configuration as cloud-config, as ochami prints it."""
    from cloudinit import safeyaml

    names = []
    for data in ("user", "vendor"):
        for part in result[data]["parts"]:
            if part["status"] == "merged":
                names.append("# %s-data %s" % (data, part["name"]))
    lines = ["#cloud-config", ""]
    if names:
        lines.append("# from %d files" % len(names))
        lines.extend(names)
        lines.append("")
    lines.append(safeyaml.dumps(merged))
    return "\n".join(lines)


def render_template(payload, instance_data):
    """Render one template as cloud-init devel render does."""
    from cloudinit.handlers.jinja_template import NotJinjaError, render_jinja_payload_from_file

    tmp = tempfile.mkdtemp(prefix="ochami-render-")
    path = os.path.join(tmp, "instance-data.json")
    with open(path, "w") as f:
        json.dump(instance_data, f)
    try:
        return render_jinja_payload_from_file(payload, "template", path)
    except NotJinjaError:
        return payload
