# SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
# SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
#
# SPDX-License-Identifier: MIT

#
# STAGE 1: Build
#

FROM golang:1.24 AS builder
ARG CGO_ENABLED=0
WORKDIR /ochami

COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN make clean
RUN make

#
# STAGE 2: Application
#

FROM cgr.dev/chainguard/wolfi-base

RUN apk add --no-cache tini

COPY --from=builder /ochami/ochami /bin/ochami

# Make dir for config file
RUN mkdir -p /.config/ochami
RUN chown -R 65534:65534 /.config

# nobody 65534:65534
USER 65534:65534

CMD [ "/bin/ochami" ]
ENTRYPOINT [ "/sbin/tini", "--" ]
