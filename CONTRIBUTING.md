<!--
SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC

SPDX-License-Identifier: MIT
-->

# Contributing to ochami

<!-- Text width is 80, only use spaces and use 4 spaces instead of tabs -->
<!-- vim: set et sta tw=80 ts=4 sw=4 sts=0: -->

Welcome to **ochami** — part of the OpenCHAMI community project under the Linux
Foundation Projects (LFP), High Performance Software Foundation (HPSF).

For general contribution guidelines, code of conduct, DCO requirements, and
community standards, please see:

**[OpenCHAMI Contributing
Guidelines](https://github.com/OpenCHAMI/.github/blob/main/CONTRIBUTING.md)**

This document provides ochami-specific guidance for building, testing, and
submitting contributions.

## Building

### Prerequisites

- Go (see minimum version in [go.mod](go.mod))
- `make` (for local builds)
- `goreleaser` (for official builds)
- `scdoc` (for man page generation)
- Container runtime: Docker, Podman, or compatible (optional, for container
  builds)

### Local Build

The fastest way to build during development:

```bash
make
```

This uses the Makefile with linker flags to embed build metadata. The binary
will be created in the current directory.

Run tests:

```bash
make test
```

Run linting:

```bash
make lint
```

Check REUSE compliance:

```bash
make check-reuse
```

### Local Container Builds

**This is the recommended approach for testing container changes before
submitting a PR.**

#### Multi-Stage

Build a local container from source using a multi-stage build:

```bash
make container
```

To use a different container runtime (e.g. Podman):

```bash
make CONTAINER_PROG=$(which podman) container
```

#### Goreleaser

Build local containers for all supported architectures using the Makefile
goreleaser target:

```bash
# Build for your current platform
make GORELEASER_OPTS='--clean --snapshot' goreleaser-release
```

> [!NOTE]
> Goreleaser as of writing does not have the capability of building only a
> single container for the current architecture. To do this using Goreleaser,
> modification of `.goreleaser.yaml` is necessary (just be sure to revert
> changes after).
>
> If this is not desired, use the multi-stage build method above.

To use a different container runtime (e.g. Podman):

```bash
make CONTAINER_PROG=$(which podman) GORELEASER_OPTS='--clean --snapshot' goreleaser-release
```

The Makefile supports these environment variables for goreleaser builds:

- **`IS_PR_BUILD`** - Whether this is a PR build (default: `false`)
- **`GORELEASER_OPTS`** - Additional flags to pass to goreleaser

### Testing Local Container

The container will be available as `ghcr.io/openchami/ochami:latest` locally.
You can test it:

```bash
# Using Docker
docker run ghcr.io/openchami/ochami:latest ochami --version

# Using Podman
podman run ghcr.io/openchami/ochami:latest ochami --version
```

## Submitting Pull Requests

### Container Builds on PRs

When you submit a PR from a fork, GitHub Actions will automatically build
binaries, packages, and containers. **Container images are built but not
published for fork PRs** due to GitHub security restrictions.

#### Building Containers Locally

The best workflow is to **build and test locally first** before submitting a PR:

1. **Make your changes**
2. **Build locally** using the Makefile approach above
3. **Test the container** locally to validate your changes
4. **Submit your PR** - binaries/packages/containers will build automatically
   (containers are built locally but not published to registries for forks)

This ensures your changes work before CI runs.

#### Fork PR Containers

**Container images are built locally but not published for fork PRs due to GitHub
security restrictions.**

GitHub Actions workflows from fork PRs can build containers locally within the
workflow but cannot push them to any registry. This is a security feature and
cannot be bypassed.

**Your fork PR will still:**
- Build binaries for all platforms
- Build packages (deb, rpm, apk, archlinux)
- Build container images locally (for validation)
- Run all tests and linting
- Be fully reviewable and mergeable

To test containers before submitting, build and test locally using the Makefile.

**For reviewers:**

Check out the fork branch locally to test containers:

```bash
# Fetch PR branch
PR_NUMBER=<pr_number>
git fetch origin "pull/${PR_NUMBER}/head:pr-${PR_NUMBER}"
git checkout "pr-${PR_NUMBER}"

# Build and test
make GORELEASER_OPTS='--clean --snapshot' goreleaser-release
docker run ghcr.io/openchami/ochami:latest ochami --version
```

**Note:** PRs from upstream branches (not forks) will build containers normally
at `ghcr.io/openchami/ochami:pr-<pr_number>`.

### General PR Guidelines

1. **Fork the repository** and create a feature branch
2. **Build and test locally** using `make` or `make goreleaser-build`
3. **Run all checks** before submitting:
   ```bash
   make test
   make lint
   make check-reuse
   ```
4. **Commit with DCO sign-off** (see [OpenCHAMI Contributing
   Guidelines](https://github.com/OpenCHAMI/.github/blob/main/CONTRIBUTING.md#developer-certificate-of-origin-dco))
5. **Open a Pull Request** and ensure it passes CI checks
6. **Respond to review feedback** promptly

## Additional Resources

- **[OpenCHAMI Contributing
  Guidelines](https://github.com/OpenCHAMI/.github/blob/main/CONTRIBUTING.md)**
  - Code of conduct, DCO, REUSE compliance, quality standards
- **[OpenCHAMI Community](https://github.com/OpenCHAMI/community)** - Values,
  principles, charter, governance
- **[ochami Documentation](man/)** - Manual pages in scdoc format
- **[OpenCHAMI Slack](https://openchami.slack.com)** - Community chat
- **[ochami Issues](https://github.com/openchami/ochami/issues)** - Bug reports
  and feature requests

## Quick Links

- [Our Values](https://github.com/OpenCHAMI/community/blob/main/VALUES.md)
- [Architectural Principles](https://github.com/OpenCHAMI/community/blob/main/TSC/Principles.md)
- [Charter](https://github.com/OpenCHAMI/community/blob/main/Charter.md)
- [Governance](https://github.com/OpenCHAMI/community/blob/main/Governance.md)
- [Copyright Guidelines](https://github.com/OpenCHAMI/community/blob/main/Copyright.md)

Thank you for contributing to ochami and the OpenCHAMI community!
