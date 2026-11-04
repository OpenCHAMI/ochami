<!--
SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC

SPDX-License-Identifier: MIT
-->

# Changelog

<!-- Text width is 80, only use spaces and use 4 spaces instead of tabs -->
<!-- vim: set et sta tw=80 ts=4 sw=4 sts=0: -->

All notable changes to this project will be documented in this file.

The format is based on [Keep a
Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- This changelog, covering every tagged version.
- Exit codes 130 and 143 for commands stopped by SIGINT and SIGTERM. Ctrl+C and
  SIGTERM now cancel the running command so that its cleanup runs, and a second
  Ctrl+C kills a stuck cleanup. See the SIGNALS section of `ochami(1)`.
- `bss boot params` payloads given with `-d` can hold one object or an array of
  them, sending one request per element.
- `pkg/config`, a public Go API for loading configuration without global state
  and for editing config files as `ochami config` does (`config.File`).
- `pkg/client/bss` batch methods (`PostBootParamsBatch`, `PutBootParamsBatch`,
  `PatchBootParamsBatch`, and `DeleteBootParamsBatch`).
- `smd.GroupPatch` and `SMDClient.PatchGroupFields`, which send only the group
  fields that are set.

### Changed

- **BREAKING:** Failures now exit with a code for their cause instead of 1: 2
  for usage, 3 for configuration, 4 for authentication, 5 for payload or output
  errors, 6 for unsuccessful HTTP responses, and 7 for network errors. See the
  EXIT STATUS section of `ochami(1)`.
- **BREAKING:** Declining a confirmation prompt exits with 8 instead of 0 (or 1
  for some config file prompts).
- **BREAKING:** A multi-item command whose failures differ in cause exits with
  the new code 9.
- **BREAKING:** A command whose output cannot be written exits with 5 instead of
  0.
- **BREAKING:** The per-user config file is read from
  `$XDG_CONFIG_HOME/ochami/config.yaml` when `XDG_CONFIG_HOME` is set to an
  absolute path, instead of `~/.config/ochami/config.yaml`.
- **BREAKING:** `-d` can no longer be combined with the flags or positional
  arguments that would also describe the request, which is now a usage error.
  This covers `bss boot params add|set|update|delete` with `--xname`, `--nid`,
  `--mac`, `--kernel`, `--initrd`, or `--params`, `smd group update` with
  `--description` or `--tag`, and the `smd` add, delete, and `group update`
  commands with positional arguments.
- **BREAKING:** `discover static` no longer accepts positional arguments.
- **BREAKING:** `smd compep|component|iface|rfe delete --all` can no longer be
  combined with IDs or `-d`. Those selectors were ignored, and every resource
  was deleted.
- **BREAKING:** `--cacert` and `--insecure` are now mutually exclusive.
  Previously, `--cacert` silently turned certificate verification back on.
- **BREAKING:** A timeout of zero or less is rejected, with exit code 3 from
  configuration and 2 from `--timeout`.
- **BREAKING:** `config cluster set <cluster> name <new-name>` fails with exit
  code 3 if the cluster does not exist, instead of creating an empty cluster.
  Cluster names must be non-empty and cannot contain `.`.
- **BREAKING:** The `rfc7386` value of `--patch-method` is now `rfc7396`, with
  no alias for the old value. Requests are unchanged.
- **BREAKING:** Container releases no longer publish architecture-suffixed tags
  such as `v0.11.0-amd64`. Use the multi-platform tag and select an architecture
  with `--platform`.
- **BREAKING:** Go API changes in `pkg/client`:
  - Service client methods and the low-level request methods take a
    `context.Context` as their first parameter.
  - Multi-item methods return `client.BatchResult[T]`.
  - `PatchMethodRFC7386` is now `PatchMethodRFC7396`.
  - `metadata_service` patch methods take their data as `interface{}`.
  - `OchamiClient` and `RCSClient` values are no longer comparable.
  - `ReadPayloadWithReader`, `ReadPayloadFileWithReader`, and their slice
    variants take an explicit reader for standard input.
  - `pkg/client/rcs` no longer handles signals. Programs must cancel the context
    passed to `ConnectConsole`, which then returns nil.
- **BREAKING:** `NodeXnameToBMCXname` rejects non-node xnames, and the
  `pkg/discover` functions `DiscoveryInfoV2` and `DiscoveryInfoV2Deprecated`
  take variadic options.
- **BREAKING:** Commands run outside the root command need a `cli.Runtime` on
  their context.
- Config edits are written to a temporary file and renamed into place, so a
  crash cannot truncate the file. The config directory must be writable, and an
  edit fails if the file's owner cannot be kept.
- `--log-level`, `--log-format`, `--log-color`, and `--verbose` also apply to
  the commands that edit the config file.
- Errors that occur before logging is configured print as `ochami: <message>`.
- `bss boot params` reports failures per item, as other batch commands do, and a
  failure does not stop the remaining requests.
- `--help` and the man pages for `boot` and `metadata` patch commands explain
  how the patch flags behave when combined, citing RFC 7396 instead of RFC 7386.
- Minimum Go version is now 1.27.3.

### Fixed

- `smd <resource> delete -d` now deletes what its payload names. It previously
  deleted nothing and reported success.
- Resource IDs that cannot name a single resource are rejected before any
  request is sent, in every service client. This covers blank IDs, `.`, `..`,
  and IDs containing `/`, `?`, or `#`, including percent-encoded forms.
  Previously, `boot node delete ../bmcs/abc` sent `DELETE /bmcs/abc`, and a
  blank SMD ID could delete a whole collection.
- Batch commands validate their whole selection first, so one bad ID exits with
  2 before any request is sent, and reads no longer print partial output.
- `--flag=false` now behaves like omitting the flag for boolean flags, including
  `--ignore-config`, `--no-token`, `--overwrite`, `--by-ip`, and the service
  status flags. `smd compep|component|iface|rfe delete --all=false` previously
  deleted every resource of the type.
- `--cluster` now overrides a configured `default-cluster`.
- `--cacert` now applies to `rcs console show` and `rcs console connect`, and a
  file with no valid certificates is rejected.
- `smd component get` with `--xname` or `--nid` and `smd iface get --id` honor
  `--no-token` and a cluster's `enable-auth: false`.
- `smd group update` sends only the given fields, so updating the tags no longer
  clears the description.
- `smd iface add` accepts multiple `network,IPv4` pairs as its help describes,
  and malformed pairs are a usage error instead of a panic.
- `bss boot params delete` can select by any one of its flags, as documented,
  and payload MAC addresses are validated.
- `metadata` patch commands send `--add` and `--remove` as RFC 6902 operations,
  and JSON Patch operations keep null values.
- `metadata` delete commands prompt before resolving the client and token.
- `--log-format json` now emits JSON.
- `rcs console connect` treats a normal close as success, forwards the final
  bytes of input, and no longer risks a concurrent websocket write when
  interrupted.
- Base64-encoded content decoded by `cloud-init` commands no longer has trailing
  NUL bytes.
- Response body read and close failures are reported instead of dropped.
- `metadata --uri` help names the metadata service.

### Security

- Updated `github.com/go-jose/go-jose/v4` to 4.1.6 for
  [GHSA-mgcc-cww5-fp2q](https://github.com/advisories/GHSA-mgcc-cww5-fp2q).
- Updated Go to 1.27.3 for
  [GO-2026-6809](https://pkg.go.dev/vuln/GO-2026-6809).
- Updated `golang.org/x/net` to 0.60.0 for HTTP/2 vulnerabilities reported by
  govulncheck.

## [0.10.4] - 2026-10-05

### Changed

- Minimum Go version is now 1.26.7.

### Fixed

- `metadata defaults|group|instance|peer patch` no longer panics when given
  `--set`, `--unset`, `--add`, or `--remove`.

## [0.10.3] - 2026-08-26

### Added

- `--show-token` to print full access tokens in debug logs.

### Changed

- **BREAKING:** Debug logs truncate access tokens by default.
- **BREAKING:** `OchamiClient` constructors take functional options.
- Minimum Go version is now 1.26.6.

## [0.10.2] - 2026-08-11

### Changed

- **BREAKING:** The Go module path is now `github.com/openchami/ochami`.
- Releases publish to the AUR again.

## [0.10.1] - 2026-08-07

### Added

- `list` alias for the `get` commands under `bss`, `cloud-init`, and `smd`.

### Changed

- Updated boot-service to v0.3.1 and metadata-service to v0.2.1.
- Releases temporarily stopped publishing to the AUR.

### Fixed

- Flat YAML payloads for `boot` and `metadata` commands no longer lose their
  fields and send empty specs.
- `boot` and `metadata` `add` commands report only the resources they created,
  by name.

## [0.10.0] - 2026-07-30

### Added

- `rcs` commands for the
  [remote-console](https://github.com/OpenCHAMI/remote-console) service.
- `--envelope`/`-e` on `boot` and `metadata` commands to pass the full data
  envelope (spec, metadata, labels, and annotations).
- `log.color` config option and `--log-color` flag, which accept `auto`, `on`,
  and `off`.
- `-v` prints each loaded config file and the merged result.

### Changed

- `boot` and `metadata` commands take only the data spec by default, with a
  mandatory `name` for `add` and `set`.
- Log messages are not colored when output is piped, by default.
- Minimum Go version is now 1.26.5.

### Fixed

- `discover static` no longer reads past EOF on standard input.
- `metadata` `set` commands no longer require `--data` to read standard input.
- Commands in `ochami-metadata(1)` are under their correct headings.

## [0.9.1] - 2026-07-20

### Fixed

- YAML output for `boot` and `metadata` resources uses the same keys as JSON,
  such as `public_keys` instead of `publickeys`.
- Corrected a typo in the `metadata` command description.

## [0.9.0] - 2026-07-01

### Added

- `metadata` commands (alias `md`) for
  [metadata-service](https://github.com/OpenCHAMI/metadata-service): `defaults`,
  `group`, `instance`, `peer`, and `service`.
- Access tokens are sent to boot-service and metadata-service.

### Changed

- **BREAKING:** `--remove` on `boot` patch commands removes by array index
  instead of by value.
- Minimum Go version is now 1.26.4.

### Fixed

- `boot` patch commands send `--add` and `--remove` as RFC 6902 operations so
  that array updates work.
- `bss`, `cloud-init`, and `smd` delete commands prompt before any client
  action, so the user confirms before being asked for a token.

## [0.8.1] - 2026-06-17

### Changed

- The default boot-service path is now `/boot-service` instead of `/boot`, to
  match the default release configuration.

### Fixed

- `--uri` and `--cluster-uri` are merged for boot-service.
- `boot` commands no longer require `-d` to read standard input.
- All `boot` delete commands prompt for confirmation before any client action,
  and `--no-confirm=false` no longer confirms automatically.
- `smd group member set` sends `group` instead of `label` in its payload.
- `smd group update` reports which group failed.

## [0.8.0] - 2026-06-02

### Added

- `smd group membership` to view the group memberships of nodes, with filters.
- `default-input-format` and `default-output-format` config options.
- `--extra-vars` on `cloud-init group render` to inject variables missing from
  the metadata.
- `--uri` on `discover static`.
- Debug log messages for boot-service requests.

### Fixed

- `smd` requests apply their query parameters.
- `--cluster-uri` no longer panics.
- arm64 container images no longer target amd64.

## [0.7.3] - 2026-05-14

### Added

- `boot service status`.

### Changed

- Updated boot-service to v0.1.5.
- Minimum Go version is now 1.26.3.

## [0.7.2] - 2026-04-21

### Changed

- Updated boot-service to v0.1.1.

### Security

- Updated `github.com/go-jose/go-jose/v4` to 4.1.4 for
  [GHSA-78h2-9frx-2jm8](https://github.com/advisories/GHSA-78h2-9frx-2jm8).

## [0.7.1] - 2026-03-25

### Fixed

- The Makefile no longer duplicates the `/usr` prefix in `libexecdir` and
  `sharedir`.

### Security

- Updated `github.com/buger/jsonparser` to 1.1.2 for
  [GO-2026-4514](https://pkg.go.dev/vuln/GO-2026-4514).
- Updated `golang.org/x/crypto` to 0.45.0 for
  [GO-2025-4116](https://pkg.go.dev/vuln/GO-2025-4116) and
  [GO-2025-4134](https://pkg.go.dev/vuln/GO-2025-4134).

## [0.7.0] - 2026-03-05

### Added

- `boot` commands for [boot-service](https://github.com/OpenCHAMI/boot-service):
  `bmc`, `config`, and `node`, each with `add`, `delete`, `get`, `list`,
  `patch`, and `set`. See `ochami-boot(1)`.
- Default value for the `timeout` config option.

### Fixed

- `--token` and `--no-token` are mutually exclusive.
- Corrected a token copy-paste typo in help output.

## [0.6.1] - 2026-02-05

### Added

- `pcs status list` and `pcs status show` to display power status.

## [0.6.0] - 2025-11-25

### Added

- New static discovery format that specifies BMCs and nodes separately, allowing
  several nodes on one BMC without repeating BMC details. See
  `ochami-discover(1)`.
- `ochami-discovery-old2new.py` to convert from the old format.

### Deprecated

- The old static discovery format.

## [0.5.6] - 2025-11-04

### Added

- Static discovery creates one RedfishEndpoint for multiple nodes on the same
  BMC.

## [0.5.5] - 2025-10-09

### Fixed

- `smd rfe add` payloads use the `RedfishEndpoints` object that SMD uses.
- `bss` and `smd` commands report missing arguments and flags as errors instead
  of printing usage and exiting successfully.

## [0.5.4] - 2025-09-19

### Added

- `groups` in static discovery nodes, for membership in multiple SMD groups.
- `bmc_fqdn` in static discovery nodes.
- Static discovery sets a default set of power actions on RedfishEndpoints,
  since PCS rejects transitions for actions SMD does not know.

### Deprecated

- `group` in static discovery nodes. Use `groups` instead.

### Fixed

- Static discovery sends EthernetInterfaces with discovery version 1 instead of
  version 2.

## [0.5.3] - 2025-08-27

### Added

- `service status` under `bss`, `smd`, and `cloud-init`.
- `service version` under `bss` and `cloud-init`.

### Changed

- **BREAKING:** `pcs status` is now `pcs service status`.

### Deprecated

- `bss status` and `smd status`. Use `service status` instead.

## [0.5.2] - 2025-08-26

### Added

- `cloud-init group render` supports `dict.items()` in templates.

### Changed

- Minimum Go version is now 1.24.4.

## [0.5.1] - 2025-08-15

### Fixed

- Corrected spacing in `ochami-discover(1)`.

## [0.5.0] - 2025-08-14

### Added

- `--no-token` to disable token handling for one command.
- `cluster.enable-auth` config option to disable it persistently.

### Fixed

- `config set` and `config cluster set` store booleans as booleans instead of
  strings.

## [0.4.0] - 2025-08-14

### Added

- `--discovery-version` on `discover static`. Version 1 POSTs EthernetInterfaces
  to SMD, for older SMD versions that do not create them.

### Changed

- `discover static` no longer POSTs EthernetInterfaces by default, avoiding 409
  conflicts with SMD versions that create them.

## [0.3.6] - 2025-08-01

### Fixed

- Corrected `--format-input` shell completion for `discover`.

## [0.3.5] - 2025-07-22

### Fixed

- `-c` no longer fails when the config file exists.
- Yes/no prompts use a lowercase `n`.
- `Iface.String()` in `pkg/discover` includes `mac_addr`.
- Corrected errors when merging URI configuration.

### Security

- Updated `github.com/go-viper/mapstructure/v2` to 2.3.0 for
  [GHSA-fv92-fjc5-jj9h](https://github.com/advisories/GHSA-fv92-fjc5-jj9h).

## [0.3.4] - 2025-06-02

### Changed

- Updated cloud-init to v1.2.3.
- Minimum Go version is now 1.24.0.

## [0.3.3] - 2025-04-28

### Added

- `smd rfe` in `ochami-smd(1)`.

### Fixed

- `smd rfe add` help uses `-d` instead of `-f`.

## [0.3.2] - 2025-04-11

### Fixed

- YAML input and output use the same keys as JSON for structures that were
  missing YAML tags.

## [0.3.1] - 2025-04-10

### Added

- `bss boot image set` to set the boot image (`root=`) without repeating the
  whole kernel command line.

### Fixed

- `pcs transition` commands log their help output.

## [0.3.0] - 2025-04-09

### Changed

- **BREAKING:** `discover` is now a metacommand, and its functionality moved to
  `discover static`, whose payload options match other commands (`-d` and `-f`).

## [0.2.1] - 2025-04-08

### Security

- Updated `github.com/go-jose/go-jose/v4` to 4.0.5 for
  [GHSA-c6gw-w398-hv78](https://github.com/advisories/GHSA-c6gw-w398-hv78).
- Updated `golang.org/x/net` to 0.36.0 for
  [GO-2025-3503](https://pkg.go.dev/vuln/GO-2025-3503).

## [0.2.0] - 2025-04-08

### Changed

- **BREAKING:** `cloud-init` commands now match the v1 API of the OpenCHAMI
  cloud-init server. See `ochami-cloud-init(1)`.
- **BREAKING:** `--force` is now `--no-confirm` in `delete` commands.
- Minimum Go version is now 1.23.

### Removed

- **BREAKING:** The old `cloud-init` `config` and `data` commands.

### Fixed

- File payloads are marshalled correctly into request bodies.

## [0.1.4] - 2025-04-02

### Fixed

- Payload files are unmarshalled according to their format.

## [0.1.3] - 2025-03-24

### Changed

- `--format-input` and `--format-output` only accept supported formats.

### Fixed

- HTTP bodies are marshalled correctly.
- `cloud-init` commands use the format flag values.

## [0.1.2] - 2025-03-24

### Added

- Token-based authentication for PCS.

## [0.1.1] - 2025-03-19

### Added

- `pcs transition` commands: `start`, `monitor`, `show`, `list`, and `abort`.

### Changed

- Minimum Go version is now 1.22.

## [0.1.0] - 2025-03-15

### Added

- `config cluster show` and `config cluster unset`.
- `--help` hints and man page references in error messages.

### Changed

- **BREAKING:** `cluster.base-uri` is now `cluster.uri`, and
  `cluster.<service>-uri` is now `cluster.<service>.uri`. See
  `ochami-config(5)`.
- **BREAKING:** `--base-uri` is replaced by `--cluster-uri` and per-service
  `--uri` flags.
- **BREAKING:** `config set` and `config cluster set` set values by config key
  instead of by flag.
- **BREAKING:** `-f`/`--payload` is replaced by `-d`/`--data`, which takes data
  directly or a file with `@`, cURL style. `-f`/`--format-input` and
  `-F`/`--format-output` set the payload and output formats.

### Fixed

- `--config` is mutually exclusive with `--system` and `--user`.
- `config set` and `config cluster set` create a nonexistent config file given
  with `--config`, and read-only commands such as `config show` report an error
  instead of offering to create it.
- Merged config lists take a conflicting entry's value from the source being
  merged in.

## [0.0.13] - 2025-02-28

### Added

- `pcs status`.

## [0.0.12] - 2025-02-27

### Added

- `-F` as a short flag for `--payload-format`.

## [0.0.11] - 2025-01-31

### Changed

- The default log format is now `rfc3339`.

## [0.0.10] - 2025-01-30

### Added

- `smd group member set`.

## [0.0.9] - 2025-01-27

### Added

- `--overwrite` on `discover`.

### Fixed

- Corrected copy-paste errors in SMD client errors and log messages.

## [0.0.8] - 2025-01-08

### Fixed

- Request errors are checked correctly.
- BSS and SMD log messages name the correct service.

## [0.0.7] - 2024-12-31

### Added

- Compiler in `version` output.

## [0.0.6] - 2024-12-30

### Changed

- The Makefile checks for the `hostname` command.

## [0.0.5] - 2024-12-30

### Added

- Detailed build information in `version` output.
- `clean`, `install`, `uninstall`, and shell completion targets in the Makefile.
- AUR packaging.

## [0.0.4] - 2024-12-16

### Fixed

- Log messages use the base names of source files and are colorized.
- Packages install to `/usr` instead of `/usr/local`, with the correct zsh
  completion path.

### Security

- Updated `golang.org/x/crypto` to 0.31.0 for
  [GO-2024-3321](https://pkg.go.dev/vuln/GO-2024-3321).

## [0.0.3] - 2024-12-10

### Fixed

- `cloud-init` commands marshal the correct variable for secure data.

## [0.0.2] - 2024-12-09

### Fixed

- The user is asked to create the config file if it does not exist.

## [0.0.1] - 2024-12-09

### Added

- `config set` and `config unset`.
- `--output-format` for commands that output data.
- Shell completions in release packages.
- `ochami-cloud-init(1)` and an example config file.

### Changed

- `cloud-init` commands match the API of the cloud-init server v0.1.1.

## [0.0.1-beta.11] - 2024-11-22

### Added

- Man pages, included in release packages: `ochami(1)`, `ochami-bss(1)`,
  `ochami-config(1)`, `ochami-config(5)`, `ochami-discover(1)`, and
  `ochami-smd(1)`.

### Fixed

- Corrected the long description of `config cluster set`.

## [0.0.1-beta.10-1] - 2024-11-20

### Changed

- Release binaries are built with CGO disabled.

## [0.0.1-beta.10] - 2024-11-16

### Added

- Payloads can be read from standard input.
- Access token expiration is checked and validated locally.
- Linux packages in releases.

### Changed

- **BREAKING:** Commands that send one request to a service moved under that
  service. `boot` is now `bss boot`, and `component`, `compep`, `group`,
  `iface`, and `rfe` are now under `smd`.
- **BREAKING:** `discover` payloads hold nodes under a `nodes` key instead of a
  plain YAML list.

### Fixed

- `discover` appends xnames to groups correctly.

## [0.0.1-beta.9] - 2024-11-08

### Changed

- The Go module moved from the `synackd` GitHub organization to `OpenCHAMI`.
- Release binaries and containers are attested.

### Fixed

- `discover` adds components before redfish endpoints.

## [0.0.1-beta.8] - 2024-11-07

### Added

- Windows builds.
- Debug message showing the path of the loaded config file.

### Fixed

- `config cluster set` checks that the cluster list exists before editing it.
- Usage strings consistently follow docopt.

## [0.0.1-beta.7] - 2024-11-05

### Added

- `cloud-init config` commands: `add`, `delete`, `get`, and `set`.

## [0.0.1-beta.6] - 2024-11-04

### Added

- `group` commands: `add`, `delete`, `get`, and `update`.
- `group member` commands: `add`, `delete`, and `get`.
- Group assignment in `discover` payloads.
- arm64 builds.

### Fixed

- `-` converts to `_` in the cluster name used for the token environment
  variable.

## [0.0.1-beta.5] - 2024-10-28

### Added

- `discover` to populate SMD from a file of nodes.
- Broken or dirty indicator in the version string.

## [0.0.1-beta.4] - 2024-10-24

### Added

- `smd status`.
- `component`, `compep`, `iface`, and `rfe` commands for SMD.
- HTTP response code in debug output.

### Fixed

- Unknown subcommands produce an error.
- Corrected the error message for unsuccessful `bss status` responses.

## [0.0.1-beta.3] - 2024-10-08

### Added

- `boot params` commands accept an HTTP payload from a file.

### Changed

- **BREAKING:** `bss bootparams` is now `boot params`, and `bss bootscript` is
  now `boot script`.
- **BREAKING:** `OCHAMI_ACCESS_TOKEN` is replaced by cluster-specific
  `<CLUSTER_NAME>_ACCESS_TOKEN` variables.

## [0.0.1-beta.2] - 2024-09-27

### Fixed

- Container images hold the binary in `/bin` and create the config file
  directory.

## [0.0.1-beta.1] - 2024-09-27

### Added

- Initial release of the `ochami` command line interface.
- `bss` commands: `bootparams`, `bootscript`, `dumpstate`, `history`, `hosts`,
  and `status`.
- `config` commands: `show` and `cluster`, with a default cluster and a config
  file created at the default location on request.
- `version` command and `--version` flag.
- Global flags for cluster selection, tokens, and TLS (`--cacert` and
  `--insecure`), plus `--ignore-config`, `--log-format`, and `--log-level`.
- Release builds and container images.

[Unreleased]: https://github.com/OpenCHAMI/ochami/compare/v0.10.4...HEAD
[0.10.4]: https://github.com/OpenCHAMI/ochami/compare/v0.10.3...v0.10.4
[0.10.3]: https://github.com/OpenCHAMI/ochami/compare/v0.10.2...v0.10.3
[0.10.2]: https://github.com/OpenCHAMI/ochami/compare/v0.10.1...v0.10.2
[0.10.1]: https://github.com/OpenCHAMI/ochami/compare/v0.10.0...v0.10.1
[0.10.0]: https://github.com/OpenCHAMI/ochami/compare/v0.9.1...v0.10.0
[0.9.1]: https://github.com/OpenCHAMI/ochami/compare/v0.9.0...v0.9.1
[0.9.0]: https://github.com/OpenCHAMI/ochami/compare/v0.8.1...v0.9.0
[0.8.1]: https://github.com/OpenCHAMI/ochami/compare/v0.8.0...v0.8.1
[0.8.0]: https://github.com/OpenCHAMI/ochami/compare/v0.7.3...v0.8.0
[0.7.3]: https://github.com/OpenCHAMI/ochami/compare/v0.7.2...v0.7.3
[0.7.2]: https://github.com/OpenCHAMI/ochami/compare/v0.7.1...v0.7.2
[0.7.1]: https://github.com/OpenCHAMI/ochami/compare/v0.7.0...v0.7.1
[0.7.0]: https://github.com/OpenCHAMI/ochami/compare/v0.6.1...v0.7.0
[0.6.1]: https://github.com/OpenCHAMI/ochami/compare/v0.6.0...v0.6.1
[0.6.0]: https://github.com/OpenCHAMI/ochami/compare/v0.5.6...v0.6.0
[0.5.6]: https://github.com/OpenCHAMI/ochami/compare/v0.5.5...v0.5.6
[0.5.5]: https://github.com/OpenCHAMI/ochami/compare/v0.5.4...v0.5.5
[0.5.4]: https://github.com/OpenCHAMI/ochami/compare/v0.5.3...v0.5.4
[0.5.3]: https://github.com/OpenCHAMI/ochami/compare/v0.5.2...v0.5.3
[0.5.2]: https://github.com/OpenCHAMI/ochami/compare/v0.5.1...v0.5.2
[0.5.1]: https://github.com/OpenCHAMI/ochami/compare/v0.5.0...v0.5.1
[0.5.0]: https://github.com/OpenCHAMI/ochami/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/OpenCHAMI/ochami/compare/v0.3.6...v0.4.0
[0.3.6]: https://github.com/OpenCHAMI/ochami/compare/v0.3.5...v0.3.6
[0.3.5]: https://github.com/OpenCHAMI/ochami/compare/v0.3.4...v0.3.5
[0.3.4]: https://github.com/OpenCHAMI/ochami/compare/v0.3.3...v0.3.4
[0.3.3]: https://github.com/OpenCHAMI/ochami/compare/v0.3.2...v0.3.3
[0.3.2]: https://github.com/OpenCHAMI/ochami/compare/v0.3.1...v0.3.2
[0.3.1]: https://github.com/OpenCHAMI/ochami/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/OpenCHAMI/ochami/compare/v0.2.1...v0.3.0
[0.2.1]: https://github.com/OpenCHAMI/ochami/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/OpenCHAMI/ochami/compare/v0.1.4...v0.2.0
[0.1.4]: https://github.com/OpenCHAMI/ochami/compare/v0.1.3...v0.1.4
[0.1.3]: https://github.com/OpenCHAMI/ochami/compare/v0.1.2...v0.1.3
[0.1.2]: https://github.com/OpenCHAMI/ochami/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/OpenCHAMI/ochami/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/OpenCHAMI/ochami/compare/v0.0.13...v0.1.0
[0.0.13]: https://github.com/OpenCHAMI/ochami/compare/v0.0.12...v0.0.13
[0.0.12]: https://github.com/OpenCHAMI/ochami/compare/v0.0.11...v0.0.12
[0.0.11]: https://github.com/OpenCHAMI/ochami/compare/v0.0.10...v0.0.11
[0.0.10]: https://github.com/OpenCHAMI/ochami/compare/v0.0.9...v0.0.10
[0.0.9]: https://github.com/OpenCHAMI/ochami/compare/v0.0.8...v0.0.9
[0.0.8]: https://github.com/OpenCHAMI/ochami/compare/v0.0.7...v0.0.8
[0.0.7]: https://github.com/OpenCHAMI/ochami/compare/v0.0.6...v0.0.7
[0.0.6]: https://github.com/OpenCHAMI/ochami/compare/v0.0.5...v0.0.6
[0.0.5]: https://github.com/OpenCHAMI/ochami/compare/v0.0.4...v0.0.5
[0.0.4]: https://github.com/OpenCHAMI/ochami/compare/v0.0.3...v0.0.4
[0.0.3]: https://github.com/OpenCHAMI/ochami/compare/v0.0.2...v0.0.3
[0.0.2]: https://github.com/OpenCHAMI/ochami/compare/v0.0.1...v0.0.2
[0.0.1]: https://github.com/OpenCHAMI/ochami/compare/v0.0.1-beta.11...v0.0.1
[0.0.1-beta.11]: https://github.com/OpenCHAMI/ochami/compare/v0.0.1-beta.10-1...v0.0.1-beta.11
[0.0.1-beta.10-1]: https://github.com/OpenCHAMI/ochami/compare/v0.0.1-beta.10...v0.0.1-beta.10-1
[0.0.1-beta.10]: https://github.com/OpenCHAMI/ochami/compare/v0.0.1-beta.9...v0.0.1-beta.10
[0.0.1-beta.9]: https://github.com/OpenCHAMI/ochami/compare/v0.0.1-beta.8...v0.0.1-beta.9
[0.0.1-beta.8]: https://github.com/OpenCHAMI/ochami/compare/v0.0.1-beta.7...v0.0.1-beta.8
[0.0.1-beta.7]: https://github.com/OpenCHAMI/ochami/compare/v0.0.1-beta.6...v0.0.1-beta.7
[0.0.1-beta.6]: https://github.com/OpenCHAMI/ochami/compare/v0.0.1-beta.5...v0.0.1-beta.6
[0.0.1-beta.5]: https://github.com/OpenCHAMI/ochami/compare/v0.0.1-beta.4...v0.0.1-beta.5
[0.0.1-beta.4]: https://github.com/OpenCHAMI/ochami/compare/v0.0.1-beta.3...v0.0.1-beta.4
[0.0.1-beta.3]: https://github.com/OpenCHAMI/ochami/compare/v0.0.1-beta.2...v0.0.1-beta.3
[0.0.1-beta.2]: https://github.com/OpenCHAMI/ochami/compare/v0.0.1-beta.1...v0.0.1-beta.2
[0.0.1-beta.1]: https://github.com/OpenCHAMI/ochami/commits/v0.0.1-beta.1
