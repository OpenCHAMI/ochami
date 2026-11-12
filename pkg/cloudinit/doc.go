// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

// Package cloudinit renders cloud-init Jinja templates for ochami and external
// Go tools the way cloud-init renders a template part on a node. Render is
// independent of CLI state, HTTP clients, files, and process input/output. Each
// call builds fresh variables and an independent environment.
//
// # Template header
//
// cloud-init renders only parts that begin with "## template: jinja" and uses
// other parts verbatim. Use [HasJinjaHeader] to check whether a node would
// render a template. Render removes a jinja header line before rendering and
// reports syntax errors against the original line numbers. It rejects other
// header types, such as basic. Templates without a header are still rendered,
// as a convenience for testing.
//
// # Variables and input documents
//
// Variables follow cloud-init's instance data. Metadata is exposed as
// ds.meta_data. The standardized v1 keys that cloud-init's NoCloud datasource
// derives from metadata are added, such as v1.instance_id, v1.local_hostname,
// v1.cloud_name, v1.availability_zone, v1.region, and v1.public_ssh_keys. Keys
// that only a booted node knows, such as v1.distro and sys_info, are absent.
// A node's own instance data can be supplied instead of metadata. Extra
// variables replace top-level keys of the instance data. Supplying ds or v1
// replaces the entire value. [InstanceDataJSON] returns the instance data as
// cloud-init writes instance-data.json. As on a node, the data then takes a
// JSON round trip, so every mapping is ordered by sorted keys, including
// mappings inside lists. Each key that contains a Jinja operator also gets an
// alias with underscores, so ds.meta_data.local_hostname reads local-hostname.
// The children of a version key such as v1 are copied to the enclosing mapping. Base64-encoded keys are
// not decoded.
//
// Documents accept JSON (including json-pretty) and YAML. Empty formats mean
// JSON, and empty or null documents mean empty mappings. Other top-level values
// must be mappings. YAML is decoded as PyYAML's safe_load decodes it, which is
// how cloud-init loads NoCloud metadata: YAML 1.1 scalars (yes, on, and off as
// booleans, 0644 as octal, 1:20 as base 60, and timestamps), duplicate keys
// keeping the last value, and tabs rejected where PyYAML rejects them. Binary
// scalars and timestamps become what cloud-init writes to instance-data.json
// for them. Scalar keys become strings as in instance-data.json, so a yes key
// reads as "true". Recursive aliases, explicitly tagged collections other
// than maps and sequences, custom tags, multiple documents, and integers
// outside signed 64-bit range are rejected. JSON duplicate keys keep the last
// value. YAML merge keys expand, earlier merge sources win, and explicit keys
// override merged values. Fractional and exponent JSON numbers are floats.
//
// # Rendering profile
//
// The profile is cloud-init 26.2's: a sandboxed environment with autoescape
// disabled, trim_blocks enabled, lstrip_blocks disabled, LF newlines, and only
// the do extension, so break and continue are syntax errors. An undefined
// value prints as [MissingVariablePrefix] followed by its name. Passing one
// through a filter that changes its text, such as upper, is an execute error,
// because that text cannot be reproduced. The final
// newline Jinja2 removes from the template is restored. The supported corpus is
// checked byte-for-byte against cloud-init's own rendering code with Python
// 3.15.0, Jinja2 3.1.6, and MarkupSafe 3.0.4. This is corpus compatibility, not
// universal Jinja2 equivalence, and older cloud-init releases differ, for
// example by rendering without the sandbox.
//
// Host environment access is disabled and no filesystem loader is configured.
// Render renders a single template. Results are returned verbatim without
// YAML normalization.
//
// # Full rendering
//
// [RenderFull] is experimental. It processes a node's user-data and
// vendor-data as cloud-init does: #include and #include-once URLs are
// fetched with a caller-supplied [Fetcher] and processed recursively, Jinja
// template parts are rendered with a [PartRenderer] and then handled by
// their rendered type, cloud-config parts are merged with cloud-init's
// mergers and every merge_how form, and user cloud-config is merged over
// vendor cloud-config. Parts a node would skip are reported with their reason
// rather than returned as errors, and a failed include aborts processing that
// data as on a node. vendor_data.enabled and vendor_data.disabled_handlers in
// user cloud-config are honored. Shell scripts and boothooks are listed, not
// run. MIME multipart data, cloud-config archives, JSON patches, and part
// handlers are reported as unsupported, because only cloud-init itself can
// process them. The merged result omits the node's system configuration,
// which cloud-init merges beneath it.
//
// The pkg/cloudinit/helper directory holds a Python helper that processes
// the same data with cloud-init's own code, without running boothooks or
// loading part handlers. It is checked against RenderFull in
// testdata/full.json, and [HelperScript] returns it.
//
// # Render bundles
//
// A [Bundle] holds gathered render inputs: a template or a node's user-data,
// vendor-data, and fetched includes, with instance-data.json. [WriteBundle]
// writes one to a private directory together with the helper and a run.sh
// that renders it with cloud-init wherever cloud-init is installed.
// [LoadBundle] reads it back, and [Bundle.Fetcher] serves its includes to
// RenderFull. Bundles may contain sensitive metadata but never credentials.
//
// # Errors
//
// Errors are *RenderError values with a RenderStage and an unwrap-able cause.
// Callers can use errors.As to identify stages and errors.Is to inspect context
// cancellation. A template that fails at the compile or execute stage would be
// skipped by cloud-init on a node, which logs a warning and continues. No
// partial result is returned on failure. Render never writes output. Callers
// control persistence and output error handling.
package cloudinit
