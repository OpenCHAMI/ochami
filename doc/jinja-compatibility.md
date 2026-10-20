<!--
SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC

SPDX-License-Identifier: MIT
-->

# Cloud-init rendering compatibility

`pkg/cloudinit.Render` is the public, state-independent API used by
`ochami cloud-init render`. It accepts a template and in-memory metadata
and extra-variable documents. It returns complete text or a `*RenderError`
with an inspectable stage and cause. The package never performs HTTP requests,
reads files or process input, or writes output. See its package documentation
and runnable examples for the full API and input contract.

The CLI reads local files or fetches documents, reads inline/`@file`/`@-` extra
variables, calls the same public API, and maps library and output errors to
`CodePayload`. Empty group configuration retains its existing early return
without fetching metadata.
The API renders an empty template normally. Neither interface implements the
rest of cloud-init's processing pipeline.

## Local and hybrid CLI testing

To verify the checked-in mount fixture without any service or token, run from
the repository root:

```sh
make
./ochami --ignore-config cloud-init render \
  --template pkg/cloudinit/testdata/cloud-init-original.j2 \
  --metadata pkg/cloudinit/testdata/cloud-init-original.metadata.json \
  > /tmp/ochami-rendered.yaml
cmp pkg/cloudinit/testdata/cloud-init-original.yaml /tmp/ochami-rendered.yaml
```

`cmp` exits successfully without output when every byte matches. The metadata
file contains the node metadata itself, without a `ds` wrapper; the renderer
places it under `ds.meta_data`. YAML and JSON metadata are accepted regardless
of `--format-input`, which controls only extra variables.

Omit `--metadata` to use an empty metadata mapping and provide any needed
variables with the existing `--extra-vars` option. Entirely local rendering
does not initialize a service client, resolve a service URI, handle tokens, or
send HTTP requests. Global configuration and logging still apply;
`--ignore-config` avoids reading configuration files for isolated tests.

To combine a local template with live metadata, use `--node`:

```sh
./ochami cloud-init render --template template.j2 --node x1000c0s0b0n0
```

This mode uses normal service configuration and authentication, fetching only
node metadata. With `--template`, `--node` and `--metadata` cannot be combined.
For a node-specific service template, use `--group GROUP --node NODE`. Add
`--metadata FILE` to replace rendering metadata while keeping node-specific
template selection; this skips the live metadata request.

The existing `cloud-init group render GROUP NODE` remains a compatibility
wrapper over the same renderer. It requires both positional arguments and
fetches both the node-specific group template and live node metadata. Its
original `--extra-vars` and `--format-input` options remain available; its help
points to the canonical command. New source-selection flags (`--template`,
`--metadata`, `--group`, and `--node`) are exclusive to `cloud-init render`.
The canonical command accepts no positional arguments. Both commands use the
gojinja rendering profile documented below.

Both file flags accept `-` for invocation-owned stdin. For example:

```sh
./ochami --ignore-config cloud-init render \
  --template - --metadata metadata.yaml < template.j2
```

Only one source may consume stdin across `--template -`, `--metadata -`, and
`--extra-vars @-`. Invalid combinations fail with `CodeUsage` before reading
inputs. File and rendering failures use `CodePayload`, and template failures
publish no partial output. Template and metadata paths are literal filenames;
they do not use the extra-variable `@file` syntax. The template's directory
does not establish an include-loader root. The public Go API is unchanged.

## Template selection with and without a node

`--group` and `--template` are mutually exclusive template sources. With
`--group`, omitting `--node` selects the raw stored template:

```sh
./ochami cloud-init render --group compute --metadata metadata.yaml
```

This requests `/admin/groups/{group}` and extracts `file.content` using
`file.encoding`: `plain` bytes are used directly, and `base64` content is decoded
once with the existing cloud-config decoder. Nonempty content requires a supported
encoding. Invalid group JSON, unknown encodings, or invalid base64 produce
`CodePayload`; HTTP failures retain `CodeHTTP`. A group without template content
returns successfully with an empty-template warning.

Without `--node`, metadata comes from `--metadata` or defaults to an empty
mapping. Stored group metadata is not merged into `ds.meta_data`. Extra variables
retain their normal top-level replacement behavior. No node metadata or
impersonation request is made, even if the metadata file contains a node ID.
The raw template may not apply to any particular node; this mode does not check
node membership or reproduce a node's effective configuration.

Provide `--node` to retain node-specific template selection:

```sh
./ochami cloud-init render --group compute --node x1000c0s0b0n0 \
  --metadata metadata.yaml
```

The node selects the service's impersonated group template through
`/admin/impersonation/{id}/{group}.yaml`. The metadata file replaces variables
under `ds.meta_data`; it does not change which template the service selects or
its group-membership requirement. `--node` therefore has a purpose even when
no live metadata is requested.

Endpoint selection depends only on the presence of `--node`. Metadata never
infers a node or changes the selected endpoint. A failed impersonation request
returns its error; it does not fall back to the raw-group endpoint. The
compatibility form `cloud-init group render GROUP NODE` always retains
node-specific selection. Both service modes use normal configuration and
authentication; entirely local rendering still uses `--template`.

## Reference profile

The initial dependency is `github.com/jryberg/gojinja v1.0.4` (upstream commit
`13975a9b5689cc0ee58c7706497d844a576721a2`). The checked-in oracle uses:

| Setting | Value |
| --- | --- |
| Python | 3.15.0 |
| Jinja2 | 3.1.6 |
| MarkupSafe | 3.0.4 |
| Environment | Regular, non-sandboxed; trusted templates |
| Autoescape | Disabled |
| Undefined | Default `Undefined` |
| `trim_blocks`, `lstrip_blocks` | Disabled |
| `newline_sequence` | LF |
| `keep_trailing_newline` | Disabled |
| Extensions | `do`, `loopcontrols` |

Host-environment access is disabled and no filesystem loader is configured.
gojinja's existing parser and range limits remain enabled (including the default
100,000-element range limit). Rendering runs inside the Go executable, without
Python, CGO, or subprocesses. Context cancellation is passed into execution;
parsing and decoding do not provide preemption within those operations.

Inputs are decoded directly into ordered containers. Metadata becomes
`ds.meta_data`, and extra variables replace top-level values without deep merging.
JSON duplicate keys retain their first position with the last value; duplicate
YAML keys are rejected. YAML merge keys expand at their source position, earlier
merge sources win, and explicit keys override merged values. Integer values must
fit signed 64-bit range. YAML scalar resolution follows yaml.v3; the reference
corpus uses JSON-shaped values and does not establish parity for every YAML tag.

Output is compared as UTF-8 bytes before any parsing. YAML parsing additionally
checks the expected mounts and commands. Output is never trimmed, reindented, or
serialized again. The default Jinja2 newline policy, insertion order, and engine
semantics can change output previously produced by Gonja.

## Supported corpus and known differences

The five fixtures in `pkg/cloudinit/testdata/reference.json` cover the original
multi-group mount template, list extension, single-group variants, and the items
filter variant. Their input order deliberately differs from lexical order.
There are three mounts, seven commands, a group without mounts, and a singleton
random choice. Go tests also cover JSON and YAML inputs, scalars, variable
precedence, errors, cancellation, and independent concurrent calls.

The earlier 157-probe native-engine audit reported 104 matching successful
outputs, 11 cases rejected by both implementations, and 42 differences at this
engine pin. Those counts are not a universal compatibility percentage. A shared
error outcome alone does not establish matching failure reasons.

The audit identified differences involving loop assignment scope, strict
undefined control flow, keyword argument binding, counting and Unicode methods,
include error semantics, sorting, recursive loops, formatting, safe-string
operations, JSON serialization policy, and arbitrary-precision integers.
Loader-name policy also differs. These remain upstream work when needed by a
supported template. In particular, mutable lists created in templates support
the audited `.extend()` fixture, while input-decoded Go slices do not expose
the same mutable-list methods at this pin. Do not infer support for mutating
input lists from the literal-list fixture.

Add a focused reproduction and Python expectation when a real template hits a
difference. A supported-fixture mismatch blocks adoption until an upstream fix
passes the entire corpus. Do not silently select a different engine, rewrite
templates, or alter the Python oracle to hide a difference. There is no engine
selector, compatibility fork, or custom filter API in this release.

## Reproducing and updating the oracle

With Python 3.15.0 available:

```sh
python3.15 -m venv /tmp/ochami-jinja-reference
/tmp/ochami-jinja-reference/bin/python -m pip install -r scripts/jinja-reference-requirements.txt
/tmp/ochami-jinja-reference/bin/python scripts/jinja-reference.py
go test ./pkg/cloudinit ./cmd ./internal/conventions
```

The script checks all three reference versions and fails if generated bytes
differ. To intentionally regenerate reviewed fixtures, run it with `--write`.
Keep fixtures self-contained; they do not depend on the adjacent Gonja checkout
used for the original audit. Normal Go tests require no Python. The CI reference
job checks the Python output independently of the Go golden tests.

For engine updates, retain the pin until the corpus, public API tests, and CLI
tests pass, and inspect any differences against the broader known backlog.
Run `make test`, `make lint`, `make reuse`, `make`, and `make man`, then verify
`CGO_ENABLED=0` builds on Linux, Darwin, and Windows for amd64 and arm64. Cross
compilation establishes build compatibility; it does not replace runtime tests
on those operating systems. Keep all third-party types behind the public API's
own declarations, as enforced by the source-convention tests.
