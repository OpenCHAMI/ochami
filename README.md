# ochami: OpenCHAMI Command Line Interface

<!-- Text width is 80, only use spaces and use 4 spaces instead of tabs -->
<!-- vim: set et sta tw=80 ts=4 sw=4 sts=0: -->

`ochami` is the command line interface to interact with the API of OpenCHAMI
services, especially the [State Management Database
(SMD)](https://github.com/OpenCHAMI/smd) and the [Boot Script Service
(BSS)](https://github.com/OpenCHAMI/bss). The tool is meant to ease interaction
with the API so one need not be proficient in `curl`.

## Documentation

There are manual pages in the [man directory](/man), which contain the most
complete usage documentation available. While they are quite readable on the
web, they are in [scdoc format](https://man.archlinux.org/man/scdoc.5.en) and
require [scdoc](https://git.sr.ht/~sircmpwn/scdoc) to build. There is a `make`
target to do build them:

```
make man
man man/ochami.1
```

## Getting Started

See [**Building**](#building) for instructions on how to build `ochami`. Then,
continue here with how to use it.

### 1. Creating a Configuration File

There are two configuration files in YAML format that `ochami` reads, in order:

1. System-Wide: `/etc/ochami/config.yaml`
2. User: `${HOME}/.config/ochami/config.yaml`

If neither exist, it will use compiled defaults. Configuration in the second
file override configuration in the first. Alternatively, the `-c`/`--config`
flag can be used to manually specify a config file path.

Let's generate a user-level configuration:

```bash
mkdir -p ~/.config/ochami/
ochami config show > ~/.config/ochami/config.yaml
```

This will generate a default configuration at `~/.config/ochami/config.yaml`.

> [!NOTE]
> The `ochami config show` command will read in any existing config files. If it
> is desired to use only the compiled defaults, use the `--ignore-config` flag.

### 2. Adding a Cluster

Next, `ochami` needs to be told how to contact the Ochami services. The
configuration file could be edited to do this, but `ochami` provides the
`cluster config` command to edit cluster configuration.

> [!NOTE]
> `ochami cluster config` is specific to configuring clusters in the config
> files. If global configuration in these files need to be managed, use `ochami
> config set/unset`.


Run the following command to add a default cluster configuration to the user
configuration file called `foobar` whose base URI is
`https://foobar.openchami.cluster`:

```bash
ochami config cluster set --user foobar --default --base-uri https://foobar.openchami.cluster
```

> [!NOTE]
> Since `ochami` supports multiple cluster configurations, `--default` makes
> this cluster the default, meaning if `--cluster` is not specified on the
> command line, this cluster's configuration will be used.

Now, when the configuration is shown, we should see the new cluster's details:

```bash
$ ochami config show
clusters:
    - cluster:
        base-uri: https://foobar.openchami.cluster
      name: foobar
default-cluster: foobar
log:
    format: rfc3339
    level: warning

```

### 3. Testing Unauthenticated Cluster Access

Test access by contacting an API endpoint not requiring an access token:

```bash
$ ochami bss status
{"bss-status":"running"}

```

### 4. Setting Access Token for Cluster

Since `ochami` supports multiple cluster configurations, it supports reading
environment variables corresponding to the cluster for the access token. This
can be overridden by using `--token`. Since our cluster is named "foobar", we
need to set `FOOBAR_ACCESS_TOKEN` to the value of the token so `ochami` can read
it when communicating with this cluster.

```bash
export FOOBAR_ACCESS_TOKEN=eyJhbGc...
```

Note that the format of the environment variable that `ochami` reads for the
access token is `<CLUSTER_NAME>_ACCESS_TOKEN` where `<CLUSTER_NAME>` is the
value of the cluster name (`name` in the config file specified with `--cluster`,
or `default-cluster` in the config file, the former taking precedence over the
latter) in all capitals and with dashes (-) and spaces substituted with
underscores (_).

### 5. Testing Authenticated Cluster Access

Now, we should be able to contact the API on an endpoint that requires
authentication:

```bash
$ ochami bss boot params get
null

```

## Building

### Goreleaser

Goreleaser is the way ochami gets built for releases, and is the officially
supported build method for troubleshooting.

```bash
export GOVERSION=$(go env GOVERSION)
export BUILD_HOST=$(hostname)
export BUILD_USER=$(whoami)
goreleaser build --clean --snapshot --single-target
```

Remove `--single-target` to build for all targets.

### Make

Make provides convenient and quick building for fast iteration and development.

Linker flags are used to embed build metadata into the binary. Building can
simply be done via:

```bash
make
```

On Unix-like systems, one can also install the binary, man pages, and
completions:

```bash
sudo make install
```

## Container

### Pulling

```bash
docker pull ghcr.io/synackd/ochami:latest
```

### Building

There are two dockerfiles for two different purposes.

- **Dockerfile** is for manual building and is intended for building locally. It
  uses a multi-stage build, the first stage building from source and the second
  stage copying the binary from the first stage.
- **goreleaser.dockerfile** is used by Goreleaser for CI. It assumes the binary
  has already been built and copies it into the container.

To build the ochami container (with `dirty` tag):

```bash
docker build . --tag ochami:dirty
```

### Running

```bash
docker run ghcr.io/synackd/ochami:latest ochami --ignore-config help
```
The above incantation will print out the command line's help message.
