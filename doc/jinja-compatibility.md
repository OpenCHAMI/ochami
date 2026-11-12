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

The renderer reproduces how cloud-init renders a Jinja template part on a node,
in `cloudinit.handlers.jinja_template` and `cloudinit.templater`. The target is
cloud-init 26.2. The engine is `github.com/jryberg/gojinja v1.0.4` (upstream
commit `13975a9b5689cc0ee58c7706497d844a576721a2`). The checked-in oracle uses:

| Setting | Value |
| --- | --- |
| cloud-init | 26.2 (source tarball, SHA-256 pinned in CI) |
| Python | 3.15.0 |
| Jinja2 | 3.1.6 |
| MarkupSafe | 3.0.4 |
| Environment | Sandboxed (`SandboxedEnvironment`) |
| Autoescape | Disabled |
| Undefined | `CI_MISSING_JINJA_VAR/<name>` (cloud-init's `UndefinedJinjaVariable`) |
| `trim_blocks` | Enabled |
| `lstrip_blocks` | Disabled |
| `newline_sequence` | LF |
| Final newline | Removed by Jinja2, then restored if the template ended with one |
| Extensions | `do` only |
| Header | A leading `## template: jinja` line is removed before rendering |

gojinja always parses `break` and `continue`, so the renderer rejects those tags
itself, as Jinja2 does without the `loopcontrols` extension. gojinja cannot
change the text of an undefined value, so each render uses debug undefined
values whose names carry a random token and replaces the printed placeholders.
Host-environment access is disabled and no filesystem loader is configured. The
sandbox's 100,000-element range limit applies. Rendering runs inside the Go
executable, without Python, CGO, or subprocesses. Context cancellation is passed
into execution. Parsing and decoding do not provide preemption within those
operations.

cloud-init renders only parts that begin with `## template: jinja`, compared
without case after leading whitespace. `HasJinjaHeader` reports that check, and
the CLI warns when a template would be used verbatim on a node. `Render` still
renders a template without a header, and rejects header types other than
`jinja`. After removing the header, syntax errors are reported against the
original line numbers, as cloud-init does.

Variables follow cloud-init's instance data. Metadata becomes `ds.meta_data`,
and the standardized `v1` keys that cloud-init's NoCloud datasource derives
from metadata are added, as `sources.DataSource._get_standardized_metadata`
builds them: `instance_id` (or `iid-datasource`), `local_hostname` (the first
label of `local-hostname`, or `ip-a-b-c-d` for an IPv4 address),
`availability_zone`, `region`, `cloud_name` (`unknown` unless metadata names
it), `cloud_id`, `platform` (`nocloud`), and `public_ssh_keys`, each with its
dashed alias where cloud-init writes one. Keys only a booted node knows, such
as `v1.distro`, `v1.kernel_release`, and `sys_info`, are absent. A node's own
instance data can replace all of this (`--instance-data`). Extra variables
replace top-level keys without deep merging. As on a node, where variables are
read back from `instance-data.json`, the data then takes a JSON round trip, so
every mapping is ordered by sorted keys, including mappings inside lists, and
values JSON cannot hold become the text cloud-init writes for them (binary
values become `ci-b64:` strings, and timestamps become cloud-init's
"redacted unserializable type" warning). The reference fixtures check
`InstanceDataJSON` output byte for byte against cloud-init's `json_dumps`. cloud-init's
`convert_jinja_instance_data` then adds an alias with underscores for each key
containing a Jinja operator (`local-hostname` gains `local_hostname`) and
copies the children of version keys such as `v1` to the enclosing mapping.
Base64-encoded keys listed in `base64-encoded-keys` are not decoded.

Inputs are decoded into ordered containers before that conversion. JSON
duplicate keys keep the last value. YAML is decoded as PyYAML's `safe_load`
decodes it, which is how cloud-init loads NoCloud metadata and cloud-config:
PyYAML's implicit resolvers for YAML 1.1 booleans, integers (including octal
`0644` and base 60 `1:20`), floats, nulls, and timestamps, its constructors,
duplicate keys keeping the last value, and its rejection of tabs outside
quoted scalars, block scalar content, and comments. Scalar keys become the
strings JSON gives them. YAML merge keys expand, earlier merge sources win,
and explicit keys override merged values. Integer values must fit signed
64-bit range, a limit PyYAML does not have. `pkg/cloudinit/testdata/yaml11.json`
records PyYAML's result for each case.

Output is compared as UTF-8 bytes before any parsing. YAML parsing additionally
checks the expected mounts and commands. Output is never trimmed, reindented,
or serialized again. Compared with Gonja, block trimming, the restored final
newline, and undefined text change output. Gonja already iterated mappings in
sorted order, because metadata decoded into Go maps.

## Full rendering

`RenderFull` and `cloud-init render --full` process a node's user-data and
vendor-data as cloud-init does, following `user_data.py`, the default part
handlers, and `stages.py`:

- Content types come from the lowercased, left-trimmed prefix
  (`handlers.type_from_starts_with`). Gzip data is decompressed.
- `#include` and `#include-once` lines are fetched and processed recursively.
  A failed fetch makes cloud-init abort processing that data, which is
  reported. The CLI fetches each `{base}/{group}.yaml` include as the node's
  impersonated group template, and reports other URLs as unresolved.
- Jinja template parts are rendered and then handled by their rendered type.
  Only cloud-config, shell scripts, and boothooks are accepted, as by
  cloud-init's Jinja handler. A part that fails to render is dropped.
- Cloud-config parts are merged in order with a port of `cloudinit.mergers`,
  honoring `merge_how` and `merge_type` in every form and their unusual cases,
  with the default `dict(replace)+list()+str()`. Invalid, empty, or
  non-mapping cloud-config is dropped.
- `vendor_data.enabled` and `vendor_data.disabled_handlers` in user
  cloud-config, merged over cloud-init's built-in default, are honored.
- User cloud-config is merged over vendor cloud-config as `util.mergemanydict`
  does. The node's system configuration, which cloud-init merges beneath them,
  is not included.

The internal engine reports MIME multipart data, cloud-config archives, JSON
patches, and part handlers as unsupported. `pkg/cloudinit/testdata/merge.json`
and `pkg/cloudinit/testdata/full.json` record cloud-init's results for the
merger port and for full rendering.

## Delegating to cloud-init

The CLI can render with a real cloud-init instead of, or as well as, the
internal engine (`--engine cloud-init`, `--compare`). Templates go through
`cloud-init devel render`, run from a private directory with relative file
arguments, so a wrapper that runs cloud-init in a node image works. Full
rendering runs the embedded helper `pkg/cloudinit/helper/ochami_render.py`
when a Python interpreter that imports cloud-init is available. It drives
cloud-init's `UserDataProcessor`, part handlers, and mergers over bundled
includes, records boothooks without running them, and never loads part
handlers. Without such an interpreter, cloud-init renders each template and
ochami merges the parts.

CI runs the delegation tests against cloud-init 23.4 and 24.4 on Python 3.9 with
Jinja2 2.11.3, as Rocky Linux 9 images ship them, and against the 26.2 reference.
Results must match the reference exactly. Other releases must run, and their
differences are logged. For example, Jinja2 2.11.3 has no `items` filter, so
the items filter fixture fails to render on those releases, as it would on
their nodes.

## Supported corpus and known differences

The five fixtures in `pkg/cloudinit/testdata/reference.json` cover the original
multi-group mount template, list extension, single-group variants, and the items
filter variant. Their input order deliberately differs from lexical order, so
the corpus checks cloud-init's key sorting. Each template begins with the
`## template: jinja` header.
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

The cloud-init profile has its own known differences. An undefined value passed
through a filter that changes its text, such as `upper`, is an execute error,
because its `CI_MISSING_JINJA_VAR/` text cannot be reproduced. An undefined
value inside a printed list shows the same text rather than Python's
`Undefined` representation. Error messages differ from cloud-init's, although
the same templates fail. Older cloud-init releases differ: before 26.2, for
example, rendering is not sandboxed, and nodes run the Jinja2 version their
distribution ships. Use `--compare` with a cloud-init from the node's image to
check a template against the release a node runs.

Add a focused reproduction and cloud-init expectation when a real template hits
a difference. A supported-fixture mismatch blocks adoption until an upstream fix
passes the entire corpus. Do not silently select a different engine, rewrite
templates, or alter the cloud-init oracle to hide a difference. The cloud-init
engine is for checking and for exact answers, not for hiding differences in
the internal engine. There is no compatibility fork or custom filter API.

## Reproducing and updating the oracle

The oracle renders each fixture with cloud-init's own code. It builds the same
instance data, takes the sorted JSON round trip of `instance-data.json`, and
calls `render_jinja_payload`, so it tests cloud-init's environment and variable
conversion rather than a copy of them. cloud-init is not published to PyPI, so
the script imports it from a release source tree. With Python 3.15.0 available:

```sh
curl -fsSL -o /tmp/cloud-init-26.2.tar.gz \
  https://github.com/canonical/cloud-init/archive/refs/tags/26.2.tar.gz
echo "16274a4102426871cad84ad07db41f82aa8f4d43d95e5947a62e199c2df775c6  /tmp/cloud-init-26.2.tar.gz" | sha256sum -c -
mkdir /tmp/cloud-init-26.2
tar -xzf /tmp/cloud-init-26.2.tar.gz -C /tmp/cloud-init-26.2 --strip-components=1
python3.15 -m venv /tmp/ochami-jinja-reference
/tmp/ochami-jinja-reference/bin/python -m pip install -r scripts/jinja-reference-requirements.txt
/tmp/ochami-jinja-reference/bin/python scripts/jinja-reference.py --cloud-init-src /tmp/cloud-init-26.2
go test ./pkg/cloudinit ./cmd ./internal/conventions
```

The script checks the cloud-init, Python, Jinja2, and MarkupSafe versions, and
fails if cloud-init ignores a template or generated bytes differ. To
intentionally regenerate reviewed fixtures, run it with `--write`. Keep fixtures
self-contained. They do not depend on the adjacent Gonja checkout used for the
original audit. Normal Go tests require no Python. The CI reference job checks
cloud-init's output independently of the Go golden tests. When moving to a new
cloud-init release, update the version and checksum in the CI job, this
document, and `reference.json` together, and review the release's changes to
`cloudinit/templater.py` and `cloudinit/handlers/jinja_template.py`.

`scripts/cloud-init-reference.py` verifies the YAML, merge, and full rendering
cases the same way, using the embedded helper for full rendering. It needs
the additional packages in `scripts/jinja-reference-requirements.txt` that
cloud-init's handlers import. To check delegation locally, point the
integration tests at the same tree:

```sh
/tmp/ochami-jinja-reference/bin/python scripts/cloud-init-reference.py --cloud-init-src /tmp/cloud-init-26.2
OCHAMI_TEST_CLOUD_INIT_SRC=/tmp/cloud-init-26.2 \
OCHAMI_TEST_CLOUD_INIT_PYTHON=/tmp/ochami-jinja-reference/bin/python \
  go test -count=1 -tags cloudinit_integration -run CloudInitIntegration ./internal/cli/cloud_init/
```

For engine updates, retain the pin until the corpus, public API tests, and CLI
tests pass, and inspect any differences against the broader known backlog.
Run `make test`, `make lint`, `make reuse`, `make`, and `make man`, then verify
`CGO_ENABLED=0` builds on Linux, Darwin, and Windows for amd64 and arm64. Cross
compilation establishes build compatibility; it does not replace runtime tests
on those operating systems. Keep all third-party types behind the public API's
own declarations, as enforced by the source-convention tests.
