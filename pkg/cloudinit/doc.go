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
// ds.meta_data, and extra variables replace top-level keys of the instance
// data. Supplying ds replaces the entire value. As on a node, every mapping is
// then ordered by sorted keys, including mappings inside lists. Each key that
// contains a Jinja operator also gets an alias with underscores, so
// ds.meta_data.local_hostname reads local-hostname. The children of a version
// key such as v1 are copied to the enclosing mapping. Base64-encoded keys are
// not decoded.
//
// Documents accept JSON (including json-pretty) and YAML. Empty formats mean
// JSON, and empty or null documents mean empty mappings. Other top-level values
// must be mappings. YAML uses yaml.v3 scalar resolution, including timestamps
// and binary scalars, which differs from the YAML 1.1 rules cloud-init uses.
// Mapping keys must be strings. Duplicate YAML keys, cyclic aliases, unsupported
// tags, multiple documents, and integers outside signed 64-bit range are
// rejected. JSON duplicate keys keep the last value. YAML merge keys expand,
// earlier merge sources win, and explicit keys override merged values.
// Fractional and exponent JSON numbers are floats.
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
// The package renders a single local template, rather than cloud-init's full
// processing pipeline: it does not follow #include, merge parts, or validate
// the result. Results are returned verbatim without YAML normalization.
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
