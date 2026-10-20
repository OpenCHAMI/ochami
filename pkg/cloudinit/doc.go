// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

// Package cloudinit renders cloud-init Jinja templates for ochami and external
// Go tools. Render is independent of CLI state, HTTP clients, files, and process
// input/output. Each call builds fresh variables and an independent environment.
//
// # Variables and input documents
//
// Metadata is exposed as ds.meta_data. Extra variables replace top-level keys;
// supplying ds replaces the entire value. Mapping insertion order is preserved
// recursively. Replacement retains a key's position; new keys append in input
// order. JSON duplicate keys use the last value at the first key's position.
// YAML merge keys expand at their source position, earlier merge sources win,
// and explicit keys override merged values without moving their position.
//
// Documents accept JSON (including json-pretty) and YAML. Empty formats mean
// JSON, and empty or null documents mean empty mappings. Other top-level values
// must be mappings. YAML uses yaml.v3 scalar resolution, including timestamps
// and binary scalars. Mapping keys must be strings. Duplicate YAML keys, cyclic
// aliases, unsupported tags, multiple documents, and integers outside signed
// 64-bit range are rejected. Fractional and exponent JSON numbers are floats.
//
// # Rendering profile
//
// The supported cloud-init corpus is checked byte-for-byte against Python
// 3.15.0, Jinja2 3.1.6, and MarkupSafe 3.0.4 using a regular Environment:
// autoescape disabled, default Undefined, trim_blocks and lstrip_blocks disabled,
// newline_sequence LF, keep_trailing_newline disabled, and the do and loopcontrols
// extensions enabled. The engine also retains its bounded range and parser
// limits. This is corpus compatibility, not universal Python Jinja2 equivalence.
//
// Templates are trusted code: rendering is non-sandboxed. Host environment
// access is disabled and no filesystem loader is configured. The package renders
// a single local template, rather than cloud-init's full processing pipeline.
// Results are returned verbatim without YAML normalization or an added newline.
//
// # Errors
//
// Errors are *RenderError values with a RenderStage and an unwrap-able cause.
// Callers can use errors.As to identify stages and errors.Is to inspect context
// cancellation. No partial result is returned on failure. Render never writes
// output; callers control persistence and output error handling.
package cloudinit
